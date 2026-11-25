package blobmsg

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"slices"

	"github.com/honeybbq/goubus/v2/internal/errs"
)

// CreateBlobMessage encodes a top-level ubus message payload.
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
		return nil, errs.ErrInvalidBlobLength
	}

	return BuildBlobBuffer(uint32(totalLen64), items), nil
}

// GetSortedKeys orders attribute IDs using the requested prefix order.
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

// BuildBlobBuffer joins a length header and encoded attributes.
func BuildBlobBuffer(totalLen uint32, items [][]byte) []byte {
	out := make([]byte, 0, max(int(totalLen), BlobHeaderBytes))
	out = binary.BigEndian.AppendUint32(out, totalLen)

	for _, item := range items {
		out = append(out, item...)
	}

	return out
}

// EncodeUbusAttribute encodes a single top-level ubus attribute.
func EncodeUbusAttribute(attrID uint32, value any) ([]byte, error) {
	attrValue, err := EncodeAttributeValue(attrID, value)
	if err != nil {
		return nil, err
	}

	attrLen64 := int64(MinAttrLen) + int64(len(attrValue))
	if attrLen64 < 0 || attrLen64 > math.MaxUint32 {
		return nil, errs.ErrInvalidBlobLength
	}

	attrLen := uint32(attrLen64)
	idLen := (attrID << AttrIDShift) | (attrLen & AttrLenMask)

	raw := make([]byte, 0, Align4(int(attrLen)))
	raw = binary.BigEndian.AppendUint32(raw, idLen)
	raw = append(raw, attrValue...)

	return AlignBuffer(raw), nil
}

// EncodeUintAttributes encodes fixed uint32 ubus attributes into a blob payload.
func EncodeUintAttributes(pairs ...[2]uint32) []byte {
	const attrBytes = uint32(MinAttrLen + Uint32Size)
	totalLen := uint32(BlobHeaderBytes)
	for range pairs {
		totalLen += attrBytes
	}
	out := make([]byte, 0, totalLen)
	out = binary.BigEndian.AppendUint32(out, totalLen)
	for _, pair := range pairs {
		idLen := (pair[0] << AttrIDShift) | attrBytes
		out = binary.BigEndian.AppendUint32(out, idLen)
		out = binary.BigEndian.AppendUint32(out, pair[1])
	}

	return out
}

// EncodeAttributeValue encodes an attribute value without its header.
func EncodeAttributeValue(attrID uint32, value any) ([]byte, error) {
	switch v := value.(type) {
	case string:
		return EncodeStringValue(v), nil
	case []byte:
		return EncodeByteAttributeValue(attrID, v)
	case uint32:
		return EncodeUint32(v), nil
	case uint16:
		return EncodeUint32(uint32(v)), nil
	case uint8:
		return EncodeUint32(uint32(v)), nil
	case int:
		return EncodeIntAttributeValue(v)
	default:
		return nil, errs.Wrapf(errs.ErrUnsupportedAttributeType, "%T", value)
	}
}

// EncodeByteAttributeValue returns a byte attribute payload.
func EncodeByteAttributeValue(_ uint32, value []byte) ([]byte, error) {
	return value, nil
}

// EncodeIntAttributeValue encodes a non-negative integer attribute value.
func EncodeIntAttributeValue(value int) ([]byte, error) {
	v64 := int64(value)
	if v64 >= 0 && v64 <= math.MaxUint32 {
		return EncodeUint32(uint32(v64)), nil
	}

	return nil, errs.Wrapf(errs.ErrInvalidParameter, "int value %d out of uint32 range", value)
}

// EncodeStringValue encodes a null-terminated string.
func EncodeStringValue(value string) []byte {
	return append([]byte(value), StringTerminator)
}

// EncodeUint32 encodes an unsigned 32-bit integer in network byte order.
func EncodeUint32(value uint32) []byte {
	data := make([]byte, Uint32Size)
	binary.BigEndian.PutUint32(data, value)

	return data
}

func ParseTopLevelAttributes(data []byte) (map[string]any, error) {
	if len(data) < BlobHeaderBytes {
		return make(map[string]any), nil
	}

	totalLen := binary.BigEndian.Uint32(data[:BlobHeaderBytes])
	if totalLen == 0 || int(totalLen) > len(data) {
		return nil, errs.ErrInvalidBlobLength
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
	case UbusAttrStatus, UbusAttrObjID, UbusAttrObjType, UbusAttrSubscribers, UbusAttrTarget:
		return DecodeUint(payload)
	case UbusAttrObjPath, UbusAttrMethod, UbusAttrUser, UbusAttrGroup:
		return DecodeString(payload), nil
	case UbusAttrActive, UbusAttrNoReply:
		return len(payload) > 0 && payload[0] != 0, nil
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

func GetAttrName(attrID uint32) string {
	if name, ok := UbusAttrNames[attrID]; ok {
		return name
	}

	return fmt.Sprintf("attr_%d", attrID)
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
