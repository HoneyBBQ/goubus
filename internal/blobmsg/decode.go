package blobmsg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"

	"github.com/honeybbq/goubus/v2/internal/errs"
)

func DecodeUint(payload []byte) (uint32, error) {
	if len(payload) < Uint32Size {
		return 0, errs.Wrapf(errs.ErrInvalidBlobLength, "payload too short for uint32: %d", len(payload))
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
			return nil, errs.ErrArrayEntryNotExtended
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
			return nil, errs.ErrTableEntryNotExtended
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
		return "", nil, errs.ErrBlobmsgPayloadTooShort
	}

	nameLen := int(binary.BigEndian.Uint16(payload[:Uint16Size]))
	headerLen := Align4(Uint16Size + nameLen + 1)
	if len(payload) < headerLen {
		return "", nil, errs.ErrInvalidBlobmsgHeaderLength
	}

	nameBytes := payload[Uint16Size : Uint16Size+nameLen]
	name := string(bytes.TrimRight(nameBytes, "\x00"))
	valueData := payload[headerLen:]
	if blobType == TypeUnspec {
		return name, nil, nil
	}

	value, err := ParseBlobmsgValue(blobType, valueData)
	if err != nil {
		return "", nil, err
	}

	return name, value, nil
}

func ParseBlobmsgValue(blobType uint32, data []byte) (any, error) {
	switch blobType {
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
	case TypeBool:
		return DecodeBool(data), nil
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

	return int64(data[0])<<56 | int64(data[1])<<48 | int64(data[2])<<40 | int64(data[3])<<32 |
		int64(data[4])<<24 | int64(data[5])<<16 | int64(data[6])<<8 | int64(data[7])
}

func DecodeInt32(data []byte) int64 {
	if len(data) < Uint32Size {
		return 0
	}
	v32 := int32(data[0])<<24 | int32(data[1])<<16 | int32(data[2])<<8 | int32(data[3])

	return int64(v32)
}

func DecodeInt16(data []byte) int64 {
	if len(data) < Uint16Size {
		return 0
	}
	v16 := int16(data[0])<<8 | int16(data[1])

	return int64(v16)
}

func DecodeBool(data []byte) bool {
	return len(data) > 0 && data[0] != 0
}
