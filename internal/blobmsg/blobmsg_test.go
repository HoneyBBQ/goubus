package blobmsg_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"runtime"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/blobmsg"
	"github.com/honeybbq/goubus/v2/internal/errs"
)

const testName = "test"

type bodyForbiddenReader struct {
	header   *bytes.Reader
	bodyRead bool
}

func (r *bodyForbiddenReader) Read(p []byte) (int, error) {
	if r.header.Len() > 0 {
		return r.header.Read(p)
	}
	r.bodyRead = true

	return 0, io.EOF
}

func TestEncodeDecodeHeader(t *testing.T) {
	original := &blobmsg.UbusMessageHeader{
		Version: 1,
		Type:    blobmsg.UbusMsgInvoke,
		Seq:     123,
		Peer:    0x12345678,
	}

	var buf bytes.Buffer

	err := blobmsg.EncodeHeader(&buf, original)
	if err != nil {
		t.Fatalf("EncodeHeader failed: %v", err)
	}

	// Complete blob header to satisfy ReadMessage logic (4 bytes length).
	_ = binary.Write(&buf, binary.BigEndian, uint32(4))

	// Simulate reading from connection.
	reader := bytes.NewReader(buf.Bytes())

	decoded, _, err := blobmsg.ReadMessage(reader)
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	if decoded.Version != original.Version ||
		decoded.Type != original.Type ||
		decoded.Seq != original.Seq ||
		decoded.Peer != original.Peer {
		t.Errorf("Decoded header mismatch. Got %+v, want %+v", decoded, original)
	}
}

func TestBlobmsgComprehensive(t *testing.T) {
	data := map[string]any{
		"string": "hello world",
		"int32":  int32(12345),
		"int64":  int64(1234567890123),
		"bool":   true,
		"double": 3.14159,
		"table": map[string]any{
			"nested_str": "nested",
			"nested_int": 99,
		},
		"array": []any{1, "two", false},
	}

	encoded, err := blobmsg.CreateBlobmsgTable(data)
	if err != nil {
		t.Fatalf("CreateBlobmsgTable failed: %v", err)
	}

	// CreateBlobmsgTable returns a full blob (header + payload)
	decoded, err := blobmsg.ParseBlobmsgContainer(encoded[4:], blobmsg.TypeTable)
	if err != nil {
		t.Fatalf("ParseBlobmsgContainer failed: %v", err)
	}

	decodedMap, ok := decoded.(map[string]any)
	if !ok {
		t.Fatalf("decoded is not map[string]any")
	}

	t.Run("BasicTypes", func(t *testing.T) {
		checkBasicTypes(t, data, decodedMap)
	})

	t.Run("NestedTable", func(t *testing.T) {
		nested, ok := decodedMap["table"].(map[string]any)
		if !ok {
			t.Fatalf("decodedMap['table'] is not map[string]any")
		}

		if nested["nested_str"] != "nested" {
			t.Errorf("nested_str mismatch: got %v", nested["nested_str"])
		}
	})

	t.Run("Array", func(t *testing.T) {
		array, ok := decodedMap["array"].([]any)
		if !ok {
			t.Fatalf("decodedMap['array'] is not []any")
		}

		if len(array) != 3 {
			t.Errorf("array length mismatch: got %d", len(array))
		}
	})
}

func checkBasicTypes(t *testing.T, data, decodedMap map[string]any) {
	t.Helper()

	if decodedMap["string"] != data["string"] {
		t.Errorf("string mismatch: got %v, want %v", decodedMap["string"], data["string"])
	}

	vInt32, ok := data["int32"].(int32)
	if !ok {
		t.Fatalf("data['int32'] is not int32")
	}

	if decodedMap["int32"] != int64(vInt32) {
		t.Errorf("int32 mismatch: got %v, want %v", decodedMap["int32"], data["int32"])
	}

	if decodedMap["int64"] != data["int64"] {
		t.Errorf("int64 mismatch: got %v, want %v", decodedMap["int64"], data["int64"])
	}

	if decodedMap["bool"] != true { // BLOBMSG_TYPE_BOOL (TypeInt8 = 7) decodes to bool, matching libubox/blobmsg_json.c
		t.Errorf("bool mismatch: got %v (%T), want true", decodedMap["bool"], decodedMap["bool"])
	}

	if decodedMap["double"] != data["double"] {
		t.Errorf("double mismatch: got %v, want %v", decodedMap["double"], data["double"])
	}
}

func TestSmallIntegers(t *testing.T) {
	data := map[string]any{
		"int8":  int8(-123),
		"int16": int16(-12345),
	}

	encoded, err := blobmsg.CreateBlobmsgTable(data)
	if err != nil {
		t.Fatalf("CreateBlobmsgTable failed: %v", err)
	}

	decoded, err := blobmsg.ParseBlobmsgContainer(encoded[4:], blobmsg.TypeTable)
	if err != nil {
		t.Fatalf("ParseBlobmsgContainer failed: %v", err)
	}

	decodedMap, isMap := decoded.(map[string]any)
	if !isMap {
		t.Fatalf("decoded is not map[string]any")
	}

	vInt8, isInt8 := data["int8"].(int8)
	if !isInt8 {
		t.Fatalf("data['int8'] is not int8")
	}

	if decodedMap["int8"] != int64(vInt8) {
		t.Errorf("int8 mismatch: got %v, want %v", decodedMap["int8"], data["int8"])
	}

	vInt16, isInt16 := data["int16"].(int16)
	if !isInt16 {
		t.Fatalf("data['int16'] is not int16")
	}

	if decodedMap["int16"] != int64(vInt16) {
		t.Errorf("int16 mismatch: got %v, want %v", decodedMap["int16"], data["int16"])
	}
}

func TestStringEncodingCompatibility(t *testing.T) {
	name := testName
	value := "a" // len 1 + null = 2
	// C implementation would have:
	// namelen = 4, headerlen = Align4(2 + 4 + 1) = 8
	// valueLen = 2
	// id_len = 4 (header) + 8 (name header) + 2 (value) = 14
	// Total bytes used = Align4(14) = 16

	entry, err := blobmsg.CreateBlobmsgEntry(name, value)
	if err != nil {
		t.Fatalf("CreateBlobmsgEntry failed: %v", err)
	}

	if len(entry) != 16 {
		t.Errorf("Expected total length 16, got %d", len(entry))
	}

	idLen := binary.BigEndian.Uint32(entry[:4])

	length := idLen & blobmsg.AttrLenMask
	if length != 14 {
		t.Errorf("Expected id_len length 14, got %d. This might indicate a discrepancy with libubox.", length)
	}
}

func TestBoolEncodingCompatibility(t *testing.T) {
	name := "b"
	value := true
	// C: namelen = 1, headerlen = Align4(2 + 1 + 1) = 4
	// valueLen = 1
	// id_len = 4 + 4 + 1 = 9
	// Total bytes = Align4(9) = 12

	entry, err := blobmsg.CreateBlobmsgEntry(name, value)
	if err != nil {
		t.Fatalf("CreateBlobmsgEntry failed: %v", err)
	}

	if len(entry) != 12 {
		t.Errorf("Expected total length 12, got %d", len(entry))
	}

	idLen := binary.BigEndian.Uint32(entry[:4])

	length := idLen & blobmsg.AttrLenMask
	if length != 9 {
		t.Errorf("Expected id_len length 9, got %d. This might indicate a discrepancy with libubox.", length)
	}
}

func TestBlobmsgEmpty(t *testing.T) {
	data := map[string]any{}

	encoded, err := blobmsg.CreateBlobmsgTable(data)
	if err != nil {
		t.Fatalf("CreateBlobmsgTable failed: %v", err)
	}

	if len(encoded) != 4 {
		t.Errorf("Expected empty table length 4, got %d", len(encoded))
	}

	decoded, err := blobmsg.ParseBlobmsgContainer(encoded[4:], blobmsg.TypeTable)
	if err != nil {
		t.Fatalf("ParseBlobmsgContainer failed: %v", err)
	}

	decodedMap, ok := decoded.(map[string]any)
	if !ok {
		t.Fatalf("decoded is not map[string]any")
	}

	if len(decodedMap) != 0 {
		t.Errorf("Expected empty map, got %v", decoded)
	}
}

func TestUintConversion(t *testing.T) {
	val := uint32(0x12345678)
	if v, ok := blobmsg.ReadUint(val); !ok || v != val {
		t.Errorf("ReadUint failed for uint32: got %v, %v", v, ok)
	}

	if v, ok := blobmsg.ReadUint(float64(12345)); !ok || v != 12345 {
		t.Errorf("ReadUint failed for float64: got %v, %v", v, ok)
	}
}

func TestReadMessageRejectsInvalidBlobLength(t *testing.T) {
	invalidLengths := []uint32{
		0,
		3,
		blobmsg.MaxMessageBytes + 1,
		0x7fffffff,
		0xffffffff,
	}

	for _, blobLen := range invalidLengths {
		var frame bytes.Buffer
		err := blobmsg.EncodeHeader(&frame, &blobmsg.UbusMessageHeader{
			Version: 0,
			Type:    blobmsg.UbusMsgData,
			Seq:     1,
			Peer:    1,
		})
		if err != nil {
			t.Fatalf("EncodeHeader(%d): %v", blobLen, err)
		}
		_ = binary.Write(&frame, binary.BigEndian, blobLen)

		reader := &bodyForbiddenReader{header: bytes.NewReader(frame.Bytes())}

		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		_, _, readErr := blobmsg.ReadMessage(reader)
		runtime.ReadMemStats(&after)

		if !errors.Is(readErr, errs.ErrInvalidBlobLength) {
			t.Fatalf("ReadMessage(blobLen=%d) error = %v, want ErrInvalidBlobLength", blobLen, readErr)
		}
		if reader.bodyRead {
			t.Fatalf("ReadMessage(blobLen=%d) attempted to read body after invalid header", blobLen)
		}
		if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 64<<10 {
			t.Fatalf("ReadMessage(blobLen=%d) allocated %d bytes, want bounded header allocation", blobLen, allocated)
		}
	}
}

func TestReadMessageAcceptsMaxMessageBytes(t *testing.T) {
	var frame bytes.Buffer
	err := blobmsg.EncodeHeader(&frame, &blobmsg.UbusMessageHeader{
		Version: 0,
		Type:    blobmsg.UbusMsgData,
		Seq:     7,
		Peer:    9,
	})
	if err != nil {
		t.Fatalf("EncodeHeader: %v", err)
	}
	_ = binary.Write(&frame, binary.BigEndian, uint32(blobmsg.MaxMessageBytes))
	frame.Write(make([]byte, blobmsg.MaxMessageBytes-blobmsg.BlobHeaderBytes))

	hdr, payload, err := blobmsg.ReadMessage(bytes.NewReader(frame.Bytes()))
	if err != nil {
		t.Fatalf("ReadMessage(MaxMessageBytes) failed: %v", err)
	}
	if hdr.Seq != 7 || len(payload) != blobmsg.MaxMessageBytes {
		t.Fatalf("ReadMessage(MaxMessageBytes) = seq %d len %d", hdr.Seq, len(payload))
	}
}

func TestBlobmsgUnspecNullRoundTrip(t *testing.T) {
	encoded, err := blobmsg.CreateBlobmsgTable(map[string]any{
		"null_field": nil,
		"items":      []any{nil, "ok"},
	})
	if err != nil {
		t.Fatalf("CreateBlobmsgTable failed: %v", err)
	}

	decoded, err := blobmsg.ParseBlobmsgContainer(encoded[blobmsg.BlobHeaderBytes:], blobmsg.TypeTable)
	if err != nil {
		t.Fatalf("ParseBlobmsgContainer failed: %v", err)
	}
	table, ok := decoded.(map[string]any)
	if !ok {
		t.Fatalf("decoded = %T, want map[string]any", decoded)
	}
	if val, exists := table["null_field"]; !exists || val != nil {
		t.Fatalf("table[null_field] = %#v (exists=%v), want nil", val, exists)
	}
	items, ok := table["items"].([]any)
	if !ok || len(items) != 2 || items[0] != nil || items[1] != "ok" {
		t.Fatalf("table[items] = %#v, want [nil, \"ok\"]", table["items"])
	}
}
