// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package blobmsg

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"os"
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/honeybbq/goubus/v2/errdefs"
)

const (
	Align     = 4
	MinLen    = 4
	HeaderLen = 4
)

const (
	Uint32Size = 4
	Uint16Size = 2
	Uint64Size = 8
)

// blobmsg type constants (aligned with libubox/blobmsg.h).
const (
	TypeUnspec = 0
	TypeArray  = 1
	TypeTable  = 2
	TypeString = 3
	TypeInt64  = 4
	TypeInt32  = 5
	TypeInt16  = 6
	TypeInt8   = 7
	TypeDouble = 8
	TypeBool   = TypeInt8
)

// ubus message types.
const (
	UbusMsgHello        = 0
	UbusMsgStatus       = 1
	UbusMsgData         = 2
	UbusMsgPing         = 3
	UbusMsgLookup       = 4
	UbusMsgInvoke       = 5
	UbusMsgAddObject    = 6
	UbusMsgRemoveObject = 7
	UbusMsgSubscribe    = 8
	UbusMsgUnsubscribe  = 9
	UbusMsgNotify       = 10
	UbusMsgMonitor      = 11
)

const (
	AttrIDMask       = 0x7f000000
	AttrIDShift      = 24
	AttrLenMask      = 0x00ffffff
	AttrExtended     = 0x80000000
	StringTerminator = byte(0)
	MinAttrLen       = 4
	HeaderBytes      = 8
	BlobHeaderBytes  = 4
)

// Ubus attribute ids.
const (
	UbusAttrUnspec      = 0
	UbusAttrStatus      = 1
	UbusAttrObjPath     = 2
	UbusAttrObjID       = 3
	UbusAttrMethod      = 4
	UbusAttrObjType     = 5
	UbusAttrSignature   = 6
	UbusAttrData        = 7
	UbusAttrTarget      = 8
	UbusAttrActive      = 9
	UbusAttrNoReply     = 10
	UbusAttrSubscribers = 11
	UbusAttrUser        = 12
	UbusAttrGroup       = 13
)

type UbusMessageHeader struct {
	Version uint8
	Type    uint8
	Seq     uint16
	Peer    uint32
}

type AttrHeader struct {
	ID         uint32
	AttrType   uint32
	Length     int
	IsExtended bool
}

type BlobReader struct {
	Data   []byte
	Offset int
}

func (r *BlobReader) HasNext() bool {
	return r.Offset < len(r.Data)
}

func (r *BlobReader) Next() (*AttrHeader, []byte, error) {
	if r.Offset+MinAttrLen > len(r.Data) {
		return nil, nil, io.EOF
	}

	raw := binary.BigEndian.Uint32(r.Data[r.Offset : r.Offset+4])

	attrLen := int(raw & AttrLenMask)
	if attrLen == 0 {
		r.Offset = len(r.Data)

		return nil, nil, io.EOF
	}

	if attrLen < MinAttrLen || r.Offset+attrLen > len(r.Data) {
		return nil, nil, errdefs.Wrapf(errdefs.ErrInvalidBlobLength, "length %d", attrLen)
	}

	header := &AttrHeader{
		ID:         (raw & AttrIDMask) >> AttrIDShift,
		AttrType:   (raw & AttrIDMask) >> AttrIDShift,
		Length:     attrLen,
		IsExtended: raw&AttrExtended != 0,
	}
	start := r.Offset + MinAttrLen
	end := r.Offset + attrLen
	payload := r.Data[start:end]
	r.Offset += Align4(attrLen)

	return header, payload, nil
}

func CreateBlobMessage(attrs map[uint32]any, ordered []uint32) ([]byte, error) {
	keys := GetSortedKeys(attrs, ordered)

	var items [][]byte

	totalLen64 := int64(BlobHeaderBytes)

	for _, key := range keys {
		item, err := EncodeUbusAttribute(key, attrs[key])
		if err != nil {
			return nil, err
		}

		items = append(items, item)
		totalLen64 += int64(len(item))
	}

	if totalLen64 < 0 || totalLen64 > math.MaxUint32 {
		return nil, errdefs.ErrInvalidBlobLength
	}

	totalLen := uint32(totalLen64)

	return BuildBlobBuffer(totalLen, items)
}

func GetSortedKeys(attrs map[uint32]any, ordered []uint32) []uint32 {
	keys := make([]uint32, 0, len(attrs))

	if len(ordered) != 0 {
		for _, k := range ordered {
			if _, ok := attrs[k]; ok {
				keys = append(keys, k)
			}
		}
	}

	for k := range attrs {
		if len(ordered) == 0 || !slices.Contains(keys, k) {
			keys = append(keys, k)
		}
	}

	return keys
}

func BuildBlobBuffer(totalLen uint32, items [][]byte) ([]byte, error) {
	var buf bytes.Buffer

	err := binary.Write(&buf, binary.BigEndian, totalLen)
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		_, err = buf.Write(item)
		if err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func EncodeUbusAttribute(attrID uint32, value any) ([]byte, error) {
	attrValue, err := EncodeAttributeValue(attrID, value)
	if err != nil {
		return nil, err
	}

	attrLen64 := int64(MinAttrLen) + int64(len(attrValue))
	if attrLen64 < 0 || attrLen64 > math.MaxUint32 {
		return nil, errdefs.ErrInvalidBlobLength
	}

	attrLen := uint32(attrLen64)
	idLen := (attrID << AttrIDShift) | (attrLen & AttrLenMask)

	var buf bytes.Buffer

	err = binary.Write(&buf, binary.BigEndian, idLen)
	if err != nil {
		return nil, err
	}

	_, err = buf.Write(attrValue)
	if err != nil {
		return nil, err
	}

	return AlignBuffer(buf.Bytes()), nil
}

func EncodeAttributeValue(attrID uint32, value any) ([]byte, error) {
	switch _value := value.(type) {
	case string:
		return EncodeStringValue(_value), nil
	case []byte:
		return EncodeByteAttributeValue(attrID, _value)
	case uint32, uint16, uint8:
		return EncodeUintAttributeValue(_value)
	case int:
		return EncodeIntAttributeValue(_value)
	default:
		return nil, errdefs.Wrapf(errdefs.ErrUnsupportedAttributeType, "%T", value)
	}
}

func EncodeByteAttributeValue(attrID uint32, value []byte) ([]byte, error) {
	return value, nil
}

func EncodeUintAttributeValue(value any) ([]byte, error) {
	switch _value := value.(type) {
	case uint32:
		return EncodeUint32(_value), nil
	case uint16:
		return EncodeUint32(uint32(_value)), nil
	case uint8:
		return EncodeUint32(uint32(_value)), nil
	default:
		return nil, errdefs.Wrapf(errdefs.ErrUnsupportedAttributeType, "%T", value)
	}
}

func EncodeIntAttributeValue(value int) ([]byte, error) {
	if value >= 0 && value <= math.MaxUint32 {
		return EncodeUint32(uint32(value)), nil
	}

	return nil, errdefs.Wrapf(errdefs.ErrInvalidParameter, "int value %d out of uint32 range", value)
}

func EncodeStringValue(value string) []byte {
	return append([]byte(value), StringTerminator)
}

func EncodeUint32(value uint32) []byte {
	data := make([]byte, Uint32Size)
	binary.BigEndian.PutUint32(data, value)

	return data
}

func PadToAlign(data []byte) []byte {
	paddedLen := Align4(len(data))
	if paddedLen == len(data) {
		return data
	}

	padding := make([]byte, paddedLen-len(data))

	return append(data, padding...)
}

func AlignBuffer(data []byte) []byte {
	paddedLen := Align4(len(data))
	if paddedLen == len(data) {
		return data
	}

	padding := make([]byte, paddedLen-len(data))

	return append(data, padding...)
}

func Align4(n int) int {
	return (n + (Align - 1)) &^ (Align - 1)
}

func CreateBlobmsgData(args map[string]any) ([]byte, error) {
	if len(args) == 0 {
		return []byte{}, nil
	}

	body, err := CreateBlobmsgTable(args)
	if err != nil {
		return nil, err
	}

	if len(body) <= BlobHeaderBytes {
		return []byte{}, nil
	}

	return body[BlobHeaderBytes:], nil
}

func CreateBlobmsgTable(values map[string]any) ([]byte, error) {
	keys := GetSortedMapKeys(values)

	var entries [][]byte

	totalLen64 := int64(BlobHeaderBytes)

	for _, key := range keys {
		item, err := CreateBlobmsgEntry(key, values[key])
		if err != nil {
			return nil, err
		}

		entries = append(entries, item)
		totalLen64 += int64(len(item))
	}

	if totalLen64 < 0 || totalLen64 > math.MaxUint32 {
		return nil, errdefs.ErrInvalidBlobLength
	}

	return BuildBlobBuffer(uint32(totalLen64), entries)
}

func GetSortedMapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

func CreateBlobmsgArray(values []any) ([]byte, error) {
	var entries [][]byte

	totalLen64 := int64(BlobHeaderBytes)

	for _, value := range values {
		item, err := CreateBlobmsgEntry("", value)
		if err != nil {
			return nil, err
		}

		entries = append(entries, item)
		totalLen64 += int64(len(item))
	}

	if totalLen64 < 0 || totalLen64 > math.MaxUint32 {
		return nil, errdefs.ErrInvalidBlobLength
	}

	return BuildBlobBuffer(uint32(totalLen64), entries)
}

func CreateBlobmsgEntry(name string, value any) ([]byte, error) {
	blobType, valueData, err := EncodeBlobmsgValue(value)
	if err != nil {
		return nil, err
	}

	nameLen := len(name)
	if nameLen > math.MaxUint16 {
		return nil, errdefs.Wrapf(errdefs.ErrInvalidParameter, "name length %d exceeds uint16", nameLen)
	}

	attrLen, err := CalculateAttrLen(nameLen, len(valueData))
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	err = EncodeBlobmsgHeader(&buf, blobType, attrLen, name, nameLen)
	if err != nil {
		return nil, err
	}

	_, err = buf.Write(valueData)
	if err != nil {
		return nil, err
	}

	return AlignBuffer(buf.Bytes()), nil
}

func CalculateAttrLen(nameLen, valueLen int) (uint32, error) {
	nameHeaderLen := Align4(2 + nameLen + 1)

	attrLen64 := int64(MinAttrLen) + int64(nameHeaderLen) + int64(valueLen)
	if attrLen64 < 0 || attrLen64 > math.MaxUint32 {
		return 0, errdefs.ErrInvalidBlobLength
	}

	return uint32(attrLen64), nil
}

func EncodeBlobmsgHeader(buf *bytes.Buffer, blobType uint8, attrLen uint32, name string, nameLen int) error {
	idLen := (uint32(blobType) << AttrIDShift) | (attrLen & AttrLenMask) | AttrExtended

	err := binary.Write(buf, binary.BigEndian, idLen)
	if err != nil {
		return err
	}

	var nameLen16 uint16
	if nameLen >= 0 && nameLen <= math.MaxUint16 {
		nameLen16 = uint16(nameLen)
	} else {
		return errdefs.Wrapf(errdefs.ErrInvalidParameter, "name length %d out of uint16 range", nameLen)
	}

	err = binary.Write(buf, binary.BigEndian, nameLen16)
	if err != nil {
		return err
	}

	_, err = buf.WriteString(name)
	if err != nil {
		return err
	}

	err = buf.WriteByte(StringTerminator)
	if err != nil {
		return err
	}

	for buf.Len()%4 != 0 {
		err = buf.WriteByte(0)
		if err != nil {
			return err
		}
	}

	return nil
}

func EncodeBlobmsgValue(value any) (uint8, []byte, error) {
	switch _value := value.(type) {
	case nil:
		return TypeUnspec, []byte{}, nil
	case bool, string:
		return EncodeBasicValue(_value)
	case json.Number, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return EncodeNumericValue(_value)
	case []byte:
		u8, b := EncodeBytesAttributeValue(_value)

		return u8, b, nil
	case map[string]any, []any:
		return EncodeComplexValue(_value)
	default:
		return EncodeReflectValue(value)
	}
}

var (
	ErrNilValue = errors.New("nil value")
)

func EncodeBasicValue(value any) (uint8, []byte, error) {
	if b, isBool := value.(bool); isBool {
		u8, buf := EncodeBoolValue(b)

		return u8, buf, nil
	}

	if s, isString := value.(string); isString {
		u8, buf := EncodeStringAttributeValue(s)

		return u8, buf, nil
	}

	return 0, nil, errdefs.Wrapf(errdefs.ErrUnsupportedAttributeType, "%T", value)
}

func EncodeNumericValue(value any) (uint8, []byte, error) {
	switch _value := value.(type) {
	case json.Number:
		return EncodeJsonNumber(_value)
	case int, int8, int16, int32, int64:
		return EncodeIntegerValue(ReflectInt64(_value))
	case uint, uint8, uint16, uint32, uint64:
		return EncodeUnsignedValue(ReflectUint64(_value))
	case float32, float64:
		return EncodeFloatValue(_value)
	default:
		return 0, nil, errdefs.Wrapf(errdefs.ErrUnsupportedAttributeType, "%T", value)
	}
}

func EncodeComplexValue(value any) (uint8, []byte, error) {
	if m, isMap := value.(map[string]any); isMap {
		return EncodeMapValue(m)
	}

	if a, isArray := value.([]any); isArray {
		return EncodeArrayValue(a)
	}

	return 0, nil, errdefs.Wrapf(errdefs.ErrUnsupportedAttributeType, "%T", value)
}

func EncodeBoolValue(value bool) (uint8, []byte) {
	data := []byte{0}
	if value {
		data[0] = 1
	}

	return TypeBool, data
}

func EncodeStringAttributeValue(value string) (uint8, []byte) {
	data := append([]byte(value), StringTerminator)

	return TypeString, data
}

func EncodeBytesAttributeValue(value []byte) (uint8, []byte) {
	data := append(append([]byte{}, value...), StringTerminator)

	return TypeString, data
}

func EncodeFloatValue(value any) (uint8, []byte, error) {
	var f64 float64

	switch _value := value.(type) {
	case float32:
		f64 = float64(_value)
	case float64:
		f64 = _value
	default:
		return 0, nil, errdefs.Wrapf(errdefs.ErrInvalidParameter, "expected float, got %T", value)
	}

	return TypeDouble, EncodeFloat64(f64), nil
}

func EncodeMapValue(value map[string]any) (uint8, []byte, error) {
	table, err := CreateBlobmsgTable(value)
	if err != nil {
		return 0, nil, err
	}

	return TypeTable, table[BlobHeaderBytes:], nil
}

func EncodeArrayValue(value []any) (uint8, []byte, error) {
	array, err := CreateBlobmsgArray(value)
	if err != nil {
		return 0, nil, err
	}

	return TypeArray, array[BlobHeaderBytes:], nil
}

func EncodeJsonNumber(value json.Number) (uint8, []byte, error) {
	i64, err := value.Int64()
	if err == nil {
		return EncodeIntegerValue(i64)
	}

	f64, err := value.Float64()
	if err == nil {
		return TypeDouble, EncodeFloat64(f64), nil
	}

	return 0, nil, errdefs.Wrapf(errdefs.ErrInvalidParameter, "invalid number: %s", value.String())
}

func EncodeReflectValue(value any) (uint8, []byte, error) {
	_value := ReflectValue(value)
	switch _value.Kind() {
	case reflect.Map:
		return EncodeReflectMap(_value)
	case reflect.Slice, reflect.Array:
		return EncodeReflectSlice(_value)
	case reflect.Struct:
		return EncodeReflectStruct(_value)
	case reflect.Invalid, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr, reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128,
		reflect.Chan, reflect.Func, reflect.Interface, reflect.Pointer,
		reflect.String, reflect.UnsafePointer:
		fallthrough
	default:
		return 0, nil, errdefs.Wrapf(errdefs.ErrUnsupportedAttributeType, "%T", value)
	}
}

func EncodeReflectMap(value reflect.Value) (uint8, []byte, error) {
	if value.Type().Key().Kind() != reflect.String {
		return 0, nil,
			errdefs.Wrapf(
				errdefs.ErrUnsupportedAttributeType,
				"map key must be string, got %s",
				value.Type().Key(),
			)
	}

	iter := value.MapRange()

	table := make(map[string]any, value.Len())

	for iter.Next() {
		table[iter.Key().String()] = iter.Value().Interface()
	}

	return EncodeBlobmsgValue(table)
}

func EncodeReflectSlice(value reflect.Value) (uint8, []byte, error) {
	if value.Type().Elem().Kind() == reflect.Uint8 {
		if value.Len() == 0 {
			return TypeUnspec, []byte{}, nil
		}

		data := make([]byte, 0, value.Len())
		data = append(data, value.Bytes()...)

		return TypeString, append(data, StringTerminator), nil
	}

	length := value.Len()

	items := make([]any, 0, length)

	for index := range length {
		items = append(items, value.Index(index).Interface())
	}

	return EncodeBlobmsgValue(items)
}

func EncodeReflectStruct(value reflect.Value) (uint8, []byte, error) {
	fields := make(map[string]any)

	typ := value.Type()
	for index := range value.NumField() {
		field := typ.Field(index)
		if !field.IsExported() {
			continue
		}

		name := field.Name
		if tag := field.Tag.Get("json"); tag != "" {
			name = ParseJSONTag(tag)
			if name == "" {
				continue
			}
		}

		fields[name] = value.Field(index).Interface()
	}

	return EncodeBlobmsgValue(fields)
}

func EncodeIntegerValue(value int64) (uint8, []byte, error) {
	if value >= math.MinInt32 && value <= math.MaxInt32 {
		var buf bytes.Buffer

		v32 := int32(value)

		err := binary.Write(&buf, binary.BigEndian, v32)
		if err != nil {
			return 0, nil, err
		}

		return TypeInt32, buf.Bytes(), nil
	}

	data := make([]byte, Uint64Size)
	binary.BigEndian.PutUint64(data, uint64(value))

	return TypeInt64, data, nil
}

func EncodeUnsignedValue(value uint64) (uint8, []byte, error) {
	if value <= math.MaxUint32 {
		data := make([]byte, Uint32Size)
		binary.BigEndian.PutUint32(data, uint32(value))

		return TypeInt32, data, nil
	}

	data := make([]byte, Uint64Size)
	binary.BigEndian.PutUint64(data, value)

	return TypeInt64, data, nil
}

func EncodeFloat64(value float64) []byte {
	data := make([]byte, Uint64Size)
	binary.BigEndian.PutUint64(data, math.Float64bits(value))

	return data
}

func ParseTopLevelAttributes(data []byte) (map[string]any, error) {
	if len(data) < BlobHeaderBytes {
		return make(map[string]any), nil
	}

	totalLen := binary.BigEndian.Uint32(data[:BlobHeaderBytes])
	if totalLen == 0 || int(totalLen) > len(data) {
		return nil, errdefs.ErrInvalidBlobLength
	}

	reader := BlobReader{Data: data[HeaderLen:int(totalLen)]}
	result := make(map[string]any)

	for reader.HasNext() {
		header, payload, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, err
		}

		value, err := ParseAttribute(header, payload)
		if err != nil {
			return nil, err
		}

		name := GetAttrName(header.ID)
		result[name] = value
	}

	return result, nil
}

func ParseAttribute(header *AttrHeader, payload []byte) (any, error) {
	switch header.ID {
	case UbusAttrStatus, UbusAttrObjID, UbusAttrObjType, UbusAttrSubscribers:
		return DecodeUint(payload)
	case UbusAttrObjPath, UbusAttrMethod, UbusAttrTarget, UbusAttrUser, UbusAttrGroup:
		return DecodeString(payload), nil
	case UbusAttrData, UbusAttrSignature:
		return ParseBlobmsgContainer(payload, TypeTable)
	default:
		if header.IsExtended {
			_, value, err := ParseBlobmsgEntry(header.AttrType, payload)

			return value, err
		}

		return payload, nil
	}
}

func DecodeUint(payload []byte) (uint32, error) {
	if len(payload) < Uint32Size {
		return 0, errdefs.Wrapf(errdefs.ErrInvalidBlobLength, "payload too short for uint32: %d", len(payload))
	}

	return binary.BigEndian.Uint32(payload[:Uint32Size]), nil
}

func DecodeString(payload []byte) string {
	if len(payload) == 0 {
		return ""
	}
	// Skip possible leading zero bytes (64-bit alignment padding).
	for len(payload) >= Uint64Size && bytes.Equal(payload[:Uint32Size], []byte{0, 0, 0, 0}) {
		payload = payload[Uint32Size:]
	}

	before, _, found := bytes.Cut(payload, []byte{StringTerminator})
	if !found {
		return string(payload)
	}

	return string(before)
}

func ParseBlobmsgContainer(payload []byte, expectedType uint8) (any, error) {
	if len(payload) == 0 {
		if expectedType == TypeArray {
			return []any{}, nil
		}

		return make(map[string]any), nil
	}
	// Skip possible leading zero bytes (64-bit alignment padding).
	for len(payload) >= Uint32Size && binary.BigEndian.Uint32(payload[:Uint32Size]) == 0 {
		payload = payload[Uint32Size:]
	}

	if len(payload) == 0 {
		if expectedType == TypeArray {
			return []any{}, nil
		}

		return make(map[string]any), nil
	}

	reader := BlobReader{Data: payload}

	switch expectedType {
	case TypeArray:
		return ParseBlobmsgArrayEntries(&reader)
	default:
		return ParseBlobmsgTableEntries(&reader)
	}
}

func ParseBlobmsgArrayEntries(reader *BlobReader) ([]any, error) {
	var items []any

	for reader.HasNext() {
		header, data, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, err
		}

		if !header.IsExtended {
			return nil, errdefs.ErrArrayEntryNotExtended
		}

		_, value, err := ParseBlobmsgEntry(header.AttrType, data)
		if err != nil {
			return nil, err
		}

		items = append(items, value)
	}

	return items, nil
}

func ParseBlobmsgTableEntries(reader *BlobReader) (map[string]any, error) {
	result := make(map[string]any)

	for reader.HasNext() {
		header, data, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, err
		}

		if !header.IsExtended {
			return nil, errdefs.ErrTableEntryNotExtended
		}

		name, value, err := ParseBlobmsgEntry(header.AttrType, data)
		if err != nil {
			return nil, err
		}

		result[name] = value
	}

	return result, nil
}

func ParseBlobmsgEntry(blobType uint32, payload []byte) (string, any, error) {
	if len(payload) < Uint16Size {
		return "", nil, errdefs.ErrBlobmsgPayloadTooShort
	}

	nameLen := int(binary.BigEndian.Uint16(payload[:Uint16Size]))

	headerLen := Align4(Uint16Size + nameLen + 1)
	if len(payload) < headerLen {
		return "", nil, errdefs.ErrInvalidBlobmsgHeaderLength
	}

	nameBytes := payload[Uint16Size : Uint16Size+nameLen]
	name := strings.TrimRight(string(nameBytes), "\x00")
	valueData := payload[headerLen:]

	value, err := ParseBlobmsgValue(blobType, valueData)
	if err != nil {
		return "", nil, err
	}

	return name, value, nil
}

func ParseBlobmsgValue(blobType uint32, data []byte) (any, error) {
	switch blobType {
	case TypeUnspec:
		return nil, ErrNilValue
	case TypeArray, TypeTable:
		return ParseBlobmsgContainer(data, uint8(blobType))
	case TypeString:
		return DecodeString(data), nil
	case TypeInt64:
		return DecodeInt64(data), nil
	case TypeInt32:
		return DecodeInt32(data), nil
	case TypeInt16:
		return DecodeInt16(data), nil
	case TypeInt8:
		return DecodeInt8(data), nil
	case TypeDouble:
		return DecodeFloat64Value(data), nil
	default:
		return nil, ErrNilValue
	}
}

func DecodeFloat64Value(data []byte) float64 {
	if len(data) < Uint64Size {
		return float64(0)
	}

	return math.Float64frombits(binary.BigEndian.Uint64(data[:Uint64Size]))
}

func DecodeInt64(data []byte) int64 {
	if len(data) < Uint64Size {
		return 0
	}

	var val int64

	err := binary.Read(bytes.NewReader(data[:Uint64Size]), binary.BigEndian, &val)
	if err != nil {
		return 0
	}

	return val
}

func DecodeInt32(data []byte) int64 {
	if len(data) < Uint32Size {
		return 0
	}

	var val int32

	err := binary.Read(bytes.NewReader(data[:Uint32Size]), binary.BigEndian, &val)
	if err != nil {
		return 0
	}

	return int64(val)
}

func DecodeInt16(data []byte) int64 {
	if len(data) < Uint16Size {
		return 0
	}

	var val int16

	err := binary.Read(bytes.NewReader(data[:Uint16Size]), binary.BigEndian, &val)
	if err != nil {
		return 0
	}

	return int64(val)
}

func DecodeInt8(data []byte) int64 {
	if len(data) < 1 {
		return 0
	}

	var val int8

	err := binary.Read(bytes.NewReader(data[:1]), binary.BigEndian, &val)
	if err != nil {
		return 0
	}

	return int64(val)
}

var UbusAttrNames = map[uint32]string{
	UbusAttrStatus:      "status",
	UbusAttrObjPath:     "objpath",
	UbusAttrObjID:       "objid",
	UbusAttrMethod:      "method",
	UbusAttrObjType:     "objtype",
	UbusAttrSignature:   "signature",
	UbusAttrData:        "data",
	UbusAttrTarget:      "target",
	UbusAttrActive:      "active",
	UbusAttrNoReply:     "no_reply",
	UbusAttrSubscribers: "subscribers",
	UbusAttrUser:        "user",
	UbusAttrGroup:       "group",
}

func GetAttrName(attrID uint32) string {
	if name, ok := UbusAttrNames[attrID]; ok {
		return name
	}

	return fmt.Sprintf("attr_%d", attrID)
}

func ReflectValue(value any) reflect.Value {
	_value := reflect.ValueOf(value)
	for _value.Kind() == reflect.Pointer || _value.Kind() == reflect.Interface {
		if _value.IsNil() {
			return reflect.Value{}
		}

		_value = _value.Elem()
	}

	return _value
}

func ReflectInt64(value any) int64 {
	switch _value := value.(type) {
	case int:
		return int64(_value)
	case int8:
		return int64(_value)
	case int16:
		return int64(_value)
	case int32:
		return int64(_value)
	case int64:
		return _value
	default:
		return 0
	}
}

func ReflectUint64(value any) uint64 {
	switch _value := value.(type) {
	case uint:
		return uint64(_value)
	case uint8:
		return uint64(_value)
	case uint16:
		return uint64(_value)
	case uint32:
		return uint64(_value)
	case uint64:
		return _value
	default:
		return 0
	}
}

func ParseJSONTag(tag string) string {
	if tag == "-" {
		return ""
	}

	if idx := strings.Index(tag, ","); idx != -1 {
		tag = tag[:idx]
	}

	return tag
}

func ReadMessage(reader io.Reader) (*UbusMessageHeader, []byte, error) {
	var err error

	headerBytesBuf := make([]byte, HeaderBytes)

	_, err = io.ReadFull(reader, headerBytesBuf)
	if err != nil {
		return nil, nil, errdefs.Wrapf(errdefs.ErrConnectionFailed, "read header: %v", err)
	}

	hdr := &UbusMessageHeader{}

	err = binary.Read(bytes.NewReader(headerBytesBuf), binary.BigEndian, hdr)
	if err != nil {
		return nil, nil, errdefs.Wrapf(errdefs.ErrInvalidResponse, "decode header: %v", err)
	}

	blobHeader := make([]byte, BlobHeaderBytes)

	_, err = io.ReadFull(reader, blobHeader)
	if err != nil {
		return nil, nil, errdefs.Wrapf(errdefs.ErrConnectionFailed, "read blob header: %v", err)
	}

	blobLen := binary.BigEndian.Uint32(blobHeader)
	payload := make([]byte, 0, blobLen)
	payload = append(payload, blobHeader...)

	if blobLen > BlobHeaderBytes {
		remaining := int(blobLen) - BlobHeaderBytes
		body := make([]byte, remaining)

		_, err = io.ReadFull(reader, body)
		if err != nil {
			return nil, nil, errdefs.Wrapf(errdefs.ErrConnectionFailed, "read blob body: %v", err)
		}

		payload = append(payload, body...)
	}

	return hdr, payload, nil
}

func NormalizeArgs(data any) (map[string]any, error) {
	if data == nil {
		return make(map[string]any), nil
	}

	switch value := data.(type) {
	case map[string]any:
		dst := make(map[string]any, len(value))
		maps.Copy(dst, value)

		return dst, nil
	case string:
		return decodeJSONMap([]byte(value))
	case []byte:
		return decodeJSONMap(value)
	default:
		raw, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}

		return decodeJSONMap(raw)
	}
}

func decodeJSONMap(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return make(map[string]any), nil
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()

	var out map[string]any

	err := dec.Decode(&out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func ReadUint(value any) (uint32, bool) {
	switch value := value.(type) {
	case uint8, uint16, uint32:
		return readUnsigned(value)
	case uint64:
		return readUint64(value)
	case int, int32, int64:
		return readSigned(value)
	case float64:
		return readFloat64(value)
	case json.Number:
		return readJsonNumber(value)
	}

	return 0, false
}

func readUint64(value uint64) (uint32, bool) {
	if value <= math.MaxUint32 {
		return uint32(value), true
	}

	return 0, false
}

func readFloat64(value float64) (uint32, bool) {
	if value >= 0 && value <= math.MaxUint32 && value == math.Trunc(value) {
		return uint32(value), true
	}

	return 0, false
}

func readJsonNumber(value json.Number) (uint32, bool) {
	i, err := value.Int64()
	if err == nil {
		return readSigned(i)
	}

	f, err := value.Float64()
	if err == nil {
		return readFloat64(f)
	}

	return 0, false
}

func readUnsigned(value any) (uint32, bool) {
	switch val := value.(type) {
	case uint8:
		return uint32(val), true
	case uint16:
		return uint32(val), true
	case uint32:
		return val, true
	default:
		return 0, false
	}
}

func readSigned(value any) (uint32, bool) {
	var i64 int64

	switch val := value.(type) {
	case int:
		i64 = int64(val)
	case int32:
		i64 = int64(val)
	case int64:
		i64 = val
	default:
		return 0, false
	}

	if i64 >= 0 && i64 <= math.MaxUint32 {
		return uint32(i64), true
	}

	return 0, false
}

func ExtractDataSection(attrs map[string]any) map[string]any {
	val, hasData := attrs["data"]
	if !hasData {
		return attrs
	}

	if m, isMap := val.(map[string]any); isMap {
		return m
	}

	return map[string]any{"value": val}
}

func ValidateSocketPath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.Mode()&os.ModeSocket == 0 {
		return errdefs.Wrapf(errdefs.ErrNotUnixSocket, "path '%s'", path)
	}

	return nil
}

func EncodeHeader(w io.Writer, header *UbusMessageHeader) error {
	return binary.Write(w, binary.BigEndian, header)
}
