// Package blobmsg encodes and decodes ubus and blobmsg wire frames.
package blobmsg

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"math"
	"os"
	"reflect"

	"github.com/honeybbq/goubus/v2/internal/errs"
)

// Align is the wire alignment used for blob attributes.
const (
	Align     = 4
	MinLen    = 4
	HeaderLen = 4
)

// Uint32Size and related constants are the encoded widths of unsigned integers.
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

// AttrIDMask and related constants describe blob attribute headers.
const (
	AttrIDMask       = 0x7f000000
	AttrIDShift      = 24
	AttrLenMask      = 0x00ffffff
	AttrExtended     = 0x80000000
	StringTerminator = byte(0)
	MinAttrLen       = 4
	HeaderBytes      = 8
	BlobHeaderBytes  = 4
	MaxMessageBytes  = 1 << 20
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

var (
	ErrNilValue   = errors.New("nil value")
	UbusAttrNames = map[uint32]string{
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
)

// UbusMessageHeader is the fixed header of an ubus message.
type UbusMessageHeader struct {
	Version uint8
	Type    uint8
	Seq     uint16
	Peer    uint32
}

// AttrHeader describes a parsed blob attribute header.
type AttrHeader struct {
	ID         uint32
	AttrType   uint32
	Length     int
	IsExtended bool
}

// BlobReader iterates over aligned blob attributes.
type BlobReader struct {
	Data   []byte
	Offset int
}

// HasNext reports whether the reader has another attribute.
func (r *BlobReader) HasNext() bool {
	return r.Offset < len(r.Data)
}

// Next reads the next attribute and its payload.
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
		return nil, nil, errs.Wrapf(errs.ErrInvalidBlobLength, "length %d", attrLen)
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

// AlignBuffer pads data to a blob alignment boundary.
func AlignBuffer(data []byte) []byte {
	paddedLen := Align4(len(data))
	if paddedLen == len(data) {
		return data
	}

	padding := make([]byte, paddedLen-len(data))

	return append(data, padding...)
}

// Align4 rounds a length up to a four-byte boundary.
func Align4(n int) int {
	return (n + (Align - 1)) &^ (Align - 1)
}

func NormalizeArgs(data any) (map[string]any, error) {
	if data == nil {
		return make(map[string]any), nil
	}

	switch value := data.(type) {
	case map[string]any:
		if len(value) == 0 {
			return make(map[string]any), nil
		}

		return maps.Clone(value), nil
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
	case uint, uint8, uint16, uint32, uint64:
		return readUint64(reflect.ValueOf(value).Uint())
	case int, int8, int16, int32, int64:
		return readInt64(reflect.ValueOf(value).Int())
	case float64:
		return readFloat64(value)
	case json.Number:
		return readJSONNumber(value)
	}

	return 0, false
}

func readUint64(value uint64) (uint32, bool) {
	if value <= math.MaxUint32 {
		return uint32(value), true
	}

	return 0, false
}

func readInt64(value int64) (uint32, bool) {
	if value >= 0 && value <= math.MaxUint32 {
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

func readJSONNumber(value json.Number) (uint32, bool) {
	i, err := value.Int64()
	if err == nil {
		return readInt64(i)
	}

	f, err := value.Float64()
	if err == nil {
		return readFloat64(f)
	}

	return 0, false
}

func ValidateSocketPath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.Mode()&os.ModeSocket == 0 {
		return errs.Wrapf(errs.ErrNotUnixSocket, "path '%s'", path)
	}

	return nil
}
