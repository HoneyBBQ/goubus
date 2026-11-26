package blobmsg

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"maps"
	"math"
	"reflect"
	"slices"
	"strings"

	"github.com/honeybbq/goubus/v2/internal/errs"
)

// CreateBlobmsgData encodes a table of ubus method arguments.
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

// CreateBlobmsgTable encodes a blobmsg table.
func CreateBlobmsgTable(values map[string]any) ([]byte, error) {
	keys := GetSortedMapKeys(values)

	var entries [][]byte
	totalLen64 := int64(BlobHeaderBytes)

	for _, key := range keys {
		value := values[key]
		reflected := reflect.ValueOf(value)
		if reflected.IsValid() && reflected.Kind() == reflect.Pointer && reflected.IsNil() {
			continue
		}

		value, err := normalizeStructValue(value)
		if err != nil {
			return nil, err
		}
		item, err := CreateBlobmsgEntry(key, value)
		if err != nil {
			return nil, err
		}

		entries = append(entries, item)
		totalLen64 += int64(len(item))
	}

	if totalLen64 < 0 || totalLen64 > math.MaxUint32 {
		return nil, errs.ErrInvalidBlobLength
	}

	return BuildBlobBuffer(uint32(totalLen64), entries), nil
}

// GetSortedMapKeys returns map keys in deterministic order.
func GetSortedMapKeys(values map[string]any) []string {
	return slices.Sorted(maps.Keys(values))
}

// CreateBlobmsgArray encodes a blobmsg array.
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
		return nil, errs.ErrInvalidBlobLength
	}

	return BuildBlobBuffer(uint32(totalLen64), entries), nil
}

// CreateBlobmsgEntry encodes one named blobmsg value.
func CreateBlobmsgEntry(name string, value any) ([]byte, error) {
	blobType, valueData, err := EncodeBlobmsgValue(value)
	if err != nil {
		return nil, err
	}

	nameLen := len(name)
	if nameLen > math.MaxUint16 {
		return nil, errs.Wrapf(errs.ErrInvalidParameter, "name length %d exceeds uint16", nameLen)
	}

	attrLen, err := CalculateAttrLen(nameLen, len(valueData))
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	EncodeBlobmsgHeader(&buf, blobType, attrLen, name, uint16(nameLen))
	_, _ = buf.Write(valueData)

	return AlignBuffer(buf.Bytes()), nil
}

func CalculateAttrLen(nameLen, valueLen int) (uint32, error) {
	nameHeaderLen := Align4(2 + nameLen + 1)

	attrLen64 := int64(MinAttrLen) + int64(nameHeaderLen) + int64(valueLen)
	if attrLen64 < 0 || attrLen64 > math.MaxUint32 {
		return 0, errs.ErrInvalidBlobLength
	}

	return uint32(attrLen64), nil
}

func EncodeBlobmsgHeader(buf *bytes.Buffer, blobType uint8, attrLen uint32, name string, nameLen uint16) {
	idLen := (uint32(blobType) << AttrIDShift) | (attrLen & AttrLenMask) | AttrExtended
	var hdr [6]byte
	binary.BigEndian.PutUint32(hdr[0:4], idLen)
	binary.BigEndian.PutUint16(hdr[4:6], nameLen)
	_, _ = buf.Write(hdr[:])
	_, _ = buf.WriteString(name)
	_ = buf.WriteByte(StringTerminator)

	for buf.Len()%4 != 0 {
		_ = buf.WriteByte(0)
	}
}

func EncodeBlobmsgValue(value any) (uint8, []byte, error) {
	switch v := value.(type) {
	case nil:
		return TypeUnspec, []byte{}, nil
	case bool:
		return EncodeBoolValue(v)
	case string:
		return EncodeStringAttributeValue(v)
	case json.Number, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return EncodeNumericValue(v)
	case []byte:
		u8, b := EncodeBytesAttributeValue(v)

		return u8, b, nil
	case map[string]any:
		return EncodeMapValue(v)
	case []any:
		return EncodeArrayValue(v)
	default:
		return EncodeReflectValue(value)
	}
}

func EncodeNumericValue(value any) (uint8, []byte, error) {
	switch v := value.(type) {
	case json.Number:
		return EncodeJSONNumber(v)
	case int, int8, int16, int32, int64:
		return EncodeIntegerValue(reflect.ValueOf(v).Int())
	case uint, uint8, uint16, uint32, uint64:
		return EncodeUnsignedValue(reflect.ValueOf(v).Uint())
	default:
		return EncodeFloatValue(v)
	}
}

func EncodeBoolValue(value bool) (uint8, []byte, error) {
	data := []byte{0}
	if value {
		data[0] = 1
	}

	return TypeBool, data, nil
}

func EncodeStringAttributeValue(value string) (uint8, []byte, error) {
	data := append([]byte(value), StringTerminator)

	return TypeString, data, nil
}

func EncodeBytesAttributeValue(value []byte) (uint8, []byte) {
	data := append(slices.Clip(slices.Clone(value)), StringTerminator)

	return TypeString, data
}

func EncodeFloatValue(value any) (uint8, []byte, error) {
	var f64 float64

	switch v := value.(type) {
	case float32:
		f64 = float64(v)
	case float64:
		f64 = v
	default:
		return 0, nil, errs.Wrapf(errs.ErrInvalidParameter, "expected float, got %T", value)
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

func EncodeJSONNumber(value json.Number) (uint8, []byte, error) {
	i64, err := value.Int64()
	if err == nil {
		return EncodeIntegerValue(i64)
	}

	f64, err := value.Float64()
	if err == nil {
		return TypeDouble, EncodeFloat64(f64), nil
	}

	return 0, nil, errs.Wrapf(errs.ErrInvalidParameter, "invalid number: %s", value.String())
}

func normalizeStructValue(value any) (any, error) {
	reflected := reflect.ValueOf(value)
	for reflected.IsValid() && (reflected.Kind() == reflect.Pointer || reflected.Kind() == reflect.Interface) {
		if reflected.IsNil() {
			return value, nil
		}
		reflected = reflected.Elem()
	}
	if !reflected.IsValid() || reflected.Kind() != reflect.Struct {
		return value, nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return nil, errs.Wrapf(errs.ErrInvalidParameter, "marshal map struct %T: %v", value, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	var normalized any
	err = decoder.Decode(&normalized)
	if err != nil {
		return nil, errs.Wrapf(errs.ErrInvalidParameter, "decode map struct %T: %v", value, err)
	}

	return normalized, nil
}

func EncodeReflectValue(value any) (uint8, []byte, error) {
	reflected := reflect.ValueOf(value)
	if !reflected.IsValid() {
		return TypeUnspec, nil, ErrNilValue
	}
	if reflected.Kind() == reflect.Pointer && reflected.IsNil() {
		return TypeUnspec, nil, nil
	}
	if marshaler, ok := value.(json.Marshaler); ok {
		return encodeMarshaledValue(value, marshaler)
	}

	reflected = ReflectValue(value)
	switch reflected.Kind() {
	case reflect.Map:
		return EncodeReflectMap(reflected)
	case reflect.Slice, reflect.Array:
		return EncodeReflectSlice(reflected)
	case reflect.Struct:
		return EncodeReflectStruct(reflected)
	case reflect.Bool:
		return EncodeBoolValue(reflected.Bool())
	case reflect.String:
		return EncodeStringAttributeValue(reflected.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return EncodeIntegerValue(reflected.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return EncodeUnsignedValue(reflected.Uint())
	case reflect.Float32, reflect.Float64:
		return EncodeFloatValue(reflected.Float())
	case reflect.Invalid, reflect.Complex64, reflect.Complex128, reflect.Chan,
		reflect.Func, reflect.Interface, reflect.Pointer, reflect.UnsafePointer:
		return 0, nil, errs.Wrapf(errs.ErrUnsupportedAttributeType, "%T", value)
	default:
		return 0, nil, errs.Wrapf(errs.ErrUnsupportedAttributeType, "%T", value)
	}
}

func encodeMarshaledValue(value any, marshaler json.Marshaler) (uint8, []byte, error) {
	data, err := marshaler.MarshalJSON()
	if err != nil {
		return 0, nil, errs.Wrapf(errs.ErrInvalidParameter, "marshal %T: %v", value, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	var decoded any
	decodeErr := decoder.Decode(&decoded)
	if decodeErr != nil {
		return 0, nil, errs.Wrapf(errs.ErrInvalidParameter, "decode marshaled %T: %v", value, decodeErr)
	}

	return EncodeBlobmsgValue(decoded)
}

func EncodeReflectMap(value reflect.Value) (uint8, []byte, error) {
	if value.Type().Key().Kind() != reflect.String {
		return 0, nil,
			errs.Wrapf(
				errs.ErrUnsupportedAttributeType,
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
		data := make([]byte, value.Len()+1)
		reflect.Copy(reflect.ValueOf(data[:value.Len()]), value)
		if value.Len() == 0 {
			return TypeUnspec, []byte{}, nil
		}

		return TypeString, data, nil
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
		data := []byte{
			byte((value >> 24) & 0xff),
			byte((value >> 16) & 0xff),
			byte((value >> 8) & 0xff),
			byte(value & 0xff),
		}

		return TypeInt32, data, nil
	}

	data := []byte{
		byte((value >> 56) & 0xff),
		byte((value >> 48) & 0xff),
		byte((value >> 40) & 0xff),
		byte((value >> 32) & 0xff),
		byte((value >> 24) & 0xff),
		byte((value >> 16) & 0xff),
		byte((value >> 8) & 0xff),
		byte(value & 0xff),
	}

	return TypeInt64, data, nil
}

func EncodeUnsignedValue(value uint64) (uint8, []byte, error) {
	if value <= math.MaxInt32 {
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

func ReflectValue(value any) reflect.Value {
	v := reflect.ValueOf(value)
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Value{}
		}

		v = v.Elem()
	}

	return v
}

func ParseJSONTag(tag string) string {
	if tag == "-" {
		return ""
	}
	tag, _, _ = strings.Cut(tag, ",")

	return tag
}
