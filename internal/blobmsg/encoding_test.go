// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package blobmsg_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/blobmsg"
)

func TestCreateBlobMessage(t *testing.T) {
	attrs := map[uint32]any{
		blobmsg.UbusAttrObjID:  uint32(123),
		blobmsg.UbusAttrMethod: "test",
	}
	ordered := []uint32{blobmsg.UbusAttrObjID, blobmsg.UbusAttrMethod}

	encoded, err := blobmsg.CreateBlobMessage(attrs, ordered)
	if err != nil {
		t.Fatalf("CreateBlobMessage failed: %v", err)
	}

	if len(encoded) < 8 {
		t.Fatalf("Encoded message too short: %d", len(encoded))
	}

	decoded, err := blobmsg.ParseTopLevelAttributes(encoded)
	if err != nil {
		t.Fatalf("Failed to decode back: %v", err)
	}

	if val, ok := decoded["objid"]; !ok {
		t.Errorf("objid not found")
	} else if v, ok := val.(uint32); !ok || v != 123 {
		t.Errorf("Decoded objid mismatch: got %v, want 123", val)
	}

	if val, ok := decoded["method"]; !ok {
		t.Errorf("method not found")
	} else if v, ok := val.(string); !ok || v != "test" {
		t.Errorf("Decoded method mismatch: got %v, want test", val)
	}
}

func TestNormalizeArgs(t *testing.T) {
	t.Run("nil", testNormalizeArgsNil)
	t.Run("map", testNormalizeArgsMap)
	t.Run("string", testNormalizeArgsString)
	t.Run("struct", testNormalizeArgsStruct)
}

func testNormalizeArgsNil(t *testing.T) {
	res, err := blobmsg.NormalizeArgs(nil)
	if err != nil || len(res) != 0 {
		t.Errorf("NormalizeArgs(nil) failed: %v, %v", res, err)
	}
}

func testNormalizeArgsMap(t *testing.T) {
	inputMap := map[string]any{"a": 1}

	res, err := blobmsg.NormalizeArgs(inputMap)
	if err != nil || res["a"] != 1 {
		t.Errorf("NormalizeArgs(map) failed: %v, %v", res, err)
	}
}

func testNormalizeArgsString(t *testing.T) {
	res, err := blobmsg.NormalizeArgs(`{"b": 2}`)
	if err != nil {
		t.Fatalf("NormalizeArgs failed: %v", err)
	}

	if val, ok := res["b"].(json.Number); !ok || val.String() != "2" {
		t.Errorf("NormalizeArgs(string) failed: %v", res)
	}
}

func testNormalizeArgsStruct(t *testing.T) {
	type S struct {
		C int `json:"c"`
	}

	res, err := blobmsg.NormalizeArgs(S{C: 3})
	if err != nil {
		t.Fatalf("NormalizeArgs failed: %v", err)
	}

	if val, ok := res["c"].(json.Number); !ok || val.String() != "3" {
		t.Errorf("NormalizeArgs(struct) failed: %v", res)
	}
}

const testValName = "test"

func TestReflectEncoding(t *testing.T) {
	t.Run("Struct", testReflectEncodingStruct)
}

func testReflectEncodingStruct(t *testing.T) {
	t.Helper()

	data := newReflectStructTestData()
	decodedMap := decodeReflectStructValue(t, data)
	assertReflectBasicTypes(t, decodedMap)
	assertReflectList(t, decodedMap)
	assertReflectObject(t, decodedMap)
}

func newReflectStructTestData() any {
	type Nested struct {
		Key string `json:"key"`
	}

	type MyStruct struct {
		Name   string   `json:"name"`
		Obj    Nested   `json:"obj"`
		List   []string `json:"list"`
		Age    int      `json:"age"`
		Active bool     `json:"active"`
	}

	return MyStruct{
		Name:   testValName,
		Age:    30,
		Active: true,
		List:   []string{"a", "b"},
		Obj:    Nested{Key: "val"},
	}
}

func decodeReflectStructValue(t *testing.T, data any) map[string]any {
	t.Helper()

	blobType, val, err := blobmsg.EncodeReflectValue(data)
	if err != nil {
		t.Fatalf("EncodeReflectValue failed: %v", err)
	}

	if blobType != blobmsg.TypeTable {
		t.Errorf("Expected TypeTable, got %d", blobType)
	}

	decoded, err := blobmsg.ParseBlobmsgContainer(val, blobmsg.TypeTable)
	if err != nil {
		t.Fatalf("ParseBlobmsgContainer failed: %v", err)
	}

	decodedMap, ok := decoded.(map[string]any)
	if !ok {
		t.Fatalf("decoded is not map[string]any")
	}

	return decodedMap
}

func assertReflectBasicTypes(t *testing.T, decodedMap map[string]any) {
	t.Helper()

	if decodedMap["name"] != "test" {
		t.Errorf("name mismatch: %v", decodedMap["name"])
	}

	if decodedMap["age"] != int64(30) {
		t.Errorf("age mismatch: %v", decodedMap["age"])
	}

	if decodedMap["active"] != int64(1) {
		t.Errorf("active mismatch: %v", decodedMap["active"])
	}
}

func assertReflectList(t *testing.T, decodedMap map[string]any) {
	t.Helper()

	list, ok := decodedMap["list"].([]any)
	if !ok {
		t.Fatalf("decodedMap['list'] is not []any")
	}

	if len(list) != 2 || list[0] != "a" {
		t.Errorf("list mismatch: %v", list)
	}
}

func assertReflectObject(t *testing.T, decodedMap map[string]any) {
	t.Helper()

	obj, ok := decodedMap["obj"].(map[string]any)
	if !ok {
		t.Fatalf("decodedMap['obj'] is not map[string]any")
	}

	if obj["key"] != "val" {
		t.Errorf("nested obj mismatch: %v", obj)
	}
}

func TestReadUintComprehensive(t *testing.T) {
	tests := []struct {
		input    any
		expected uint32
		ok       bool
	}{
		{uint8(8), 8, true},
		{uint16(16), 16, true},
		{uint32(32), 32, true},
		{uint64(64), 64, true},
		{uint64(0xFFFFFFFF + 1), 0, false},
		{int(10), 10, true},
		{int(-1), 0, false},
		{int32(32), 32, true},
		{int64(64), 64, true},
		{float64(12.0), 12, true},
		{float64(12.5), 0, false},
		{json.Number("100"), 100, true},
		{json.Number("-1"), 0, false},
		{json.Number("3.14"), 0, false},
		{"string", 0, false},
	}

	for _, tt := range tests {
		val, ok := blobmsg.ReadUint(tt.input)
		if ok != tt.ok || (ok && val != tt.expected) {
			t.Errorf("ReadUint(%v (%T)) = %v, %v; want %v, %v", tt.input, tt.input, val, ok, tt.expected, tt.ok)
		}
	}
}

func TestEncodeFloatValue(t *testing.T) {
	_, b32, err := blobmsg.EncodeFloatValue(float32(1.5))
	if err != nil {
		t.Fatal(err)
	}

	if len(b32) != 8 {
		t.Errorf("Expected 8 bytes for double, got %d", len(b32))
	}

	_, b64, err := blobmsg.EncodeFloatValue(float64(2.5))
	if err != nil {
		t.Fatal(err)
	}

	if len(b64) != 8 {
		t.Errorf("Expected 8 bytes for double, got %d", len(b64))
	}

	_, _, err = blobmsg.EncodeFloatValue("not a float")
	if err == nil {
		t.Error("Expected error for non-float")
	}
}

func TestEncodeIntAttributeValue(t *testing.T) {
	_, err := blobmsg.EncodeIntAttributeValue(100)
	if err != nil {
		t.Error(err)
	}

	_, err = blobmsg.EncodeIntAttributeValue(-1)
	if err == nil {
		t.Error("Expected error for negative int")
	}
}

func TestReflectUint64(t *testing.T) {
	if blobmsg.ReflectUint64(uint(10)) != 10 {
		t.Errorf("ReflectUint64(uint) failed")
	}

	if blobmsg.ReflectUint64(uint8(8)) != 8 {
		t.Errorf("ReflectUint64(uint8) failed")
	}

	if blobmsg.ReflectUint64(uint16(16)) != 16 {
		t.Errorf("ReflectUint64(uint16) failed")
	}

	if blobmsg.ReflectUint64(uint32(32)) != 32 {
		t.Errorf("ReflectUint64(uint32) failed")
	}

	if blobmsg.ReflectUint64(uint64(64)) != 64 {
		t.Errorf("ReflectUint64(uint64) failed")
	}

	if blobmsg.ReflectUint64("string") != 0 {
		t.Errorf("ReflectUint64(string) should be 0")
	}
}

func TestEncodeReflectMapFail(t *testing.T) {
	// Map with non-string key should fail
	badMap := map[int]string{1: "a"}

	_, _, err := blobmsg.EncodeReflectMap(reflect.ValueOf(badMap))
	if err == nil {
		t.Error("Expected error for non-string map key")
	}
}

func TestRemainingFunctions(t *testing.T) {
	t.Run("EncodeByte", testRemainingEncodeByte)
	t.Run("EncodeBytes", testRemainingEncodeBytes)
	t.Run("PadToAlign", func(t *testing.T) {
		if len(blobmsg.PadToAlign([]byte{1})) != 4 {
			t.Errorf("PadToAlign failed")
		}
	})
	t.Run("CreateBlobmsgData", testRemainingCreateBlobmsgData)
	t.Run("EncodeJsonNumber", testRemainingEncodeJsonNumber)
	t.Run("EncodeUnsignedValue", testRemainingEncodeUnsignedValue)
	t.Run("DecodeInt16", func(t *testing.T) {
		if blobmsg.DecodeInt16([]byte{0, 1}) != 1 {
			t.Errorf("DecodeInt16 failed")
		}
	})
	t.Run("ExtractDataSection", testRemainingExtractDataSection)
}

func testRemainingEncodeByte(t *testing.T) {
	b, err := blobmsg.EncodeByteAttributeValue(blobmsg.UbusAttrData, []byte{1, 2, 3})
	if err != nil || len(b) != 3 {
		t.Errorf("EncodeByteAttributeValue failed")
	}
}

func testRemainingEncodeBytes(t *testing.T) {
	bt, b2 := blobmsg.EncodeBytesAttributeValue([]byte{1, 2, 3})
	if bt != blobmsg.TypeString || len(b2) != 4 { // 3 + null
		t.Errorf("EncodeBytesAttributeValue failed")
	}
}

func testRemainingCreateBlobmsgData(t *testing.T) {
	res, err := blobmsg.CreateBlobmsgData(map[string]any{"foo": "bar"})
	if err != nil || len(res) == 0 {
		t.Errorf("CreateBlobmsgData failed")
	}
	// Empty data
	res, err = blobmsg.CreateBlobmsgData(nil)
	if err != nil || len(res) != 0 {
		t.Errorf("CreateBlobmsgData(nil) failed")
	}
}

func testRemainingEncodeJsonNumber(t *testing.T) {
	_, _, err := blobmsg.EncodeJsonNumber(json.Number("123"))
	if err != nil {
		t.Errorf("EncodeJsonNumber(int) failed")
	}

	_, _, err = blobmsg.EncodeJsonNumber(json.Number("123.45"))
	if err != nil {
		t.Errorf("EncodeJsonNumber(float) failed")
	}

	_, _, err = blobmsg.EncodeJsonNumber(json.Number("invalid"))
	if err == nil {
		t.Errorf("EncodeJsonNumber(invalid) should fail")
	}
}

func testRemainingEncodeUnsignedValue(t *testing.T) {
	_, b32, _ := blobmsg.EncodeUnsignedValue(uint64(100))
	if len(b32) != 4 {
		t.Errorf("EncodeUnsignedValue(32) failed")
	}

	_, b64, _ := blobmsg.EncodeUnsignedValue(uint64(0xFFFFFFFF + 1))
	if len(b64) != 8 {
		t.Errorf("EncodeUnsignedValue(64) failed")
	}
}

func testRemainingExtractDataSection(t *testing.T) {
	m := map[string]any{"data": map[string]any{"a": 1}}

	ext := blobmsg.ExtractDataSection(m)
	if ext["a"] != 1 {
		t.Errorf("ExtractDataSection failed for map")
	}

	m2 := map[string]any{"data": "simple"}

	ext2 := blobmsg.ExtractDataSection(m2)
	if ext2["value"] != "simple" {
		t.Errorf("ExtractDataSection failed for non-map")
	}

	m3 := map[string]any{"other": 1}

	ext3 := blobmsg.ExtractDataSection(m3)
	if ext3["other"] != 1 {
		t.Errorf("ExtractDataSection failed for no data key")
	}
}

func TestValidateSocketPath(t *testing.T) {
	// This is hard to test cross-platform without a real socket
	// but we can try a non-existent path
	err := blobmsg.ValidateSocketPath("/non/existent/path")
	if err == nil {
		t.Errorf("ValidateSocketPath should fail for non-existent path")
	}
}

func TestMoreReflectionAndErrors(t *testing.T) {
	t.Run("StructPointers", testMoreReflectionStructPointers)
	t.Run("EmptyByteSlice", testMoreReflectionEmptyByteSlice)
	t.Run("SliceOfAny", testMoreReflectionSliceOfAny)
	t.Run("InvalidReflectionType", testMoreReflectionInvalidType)
	t.Run("ParseBlobmsgValueDefaults", testMoreReflectionParseDefaults)
	t.Run("GetAttrNameDefault", testMoreReflectionGetAttrNameDefault)
	t.Run("ParseJSONTagCornerCases", testMoreReflectionParseJSONTag)
}

func testMoreReflectionStructPointers(t *testing.T) {
	type S struct{ V int }

	bt, b, err := blobmsg.EncodeBlobmsgValue(&S{V: 10})
	if err != nil || bt != blobmsg.TypeTable {
		t.Errorf("Struct pointer encoding failed: %v", err)
	}

	if len(b) == 0 {
		t.Errorf("Encoded bytes empty")
	}
}

func testMoreReflectionEmptyByteSlice(t *testing.T) {
	bt, _, err := blobmsg.EncodeBlobmsgValue([]byte{})
	if err != nil || bt != blobmsg.TypeString {
		t.Errorf("Empty byte slice should be TypeString, got %d", bt)
	}
}

func testMoreReflectionSliceOfAny(t *testing.T) {
	bt, _, err := blobmsg.EncodeBlobmsgValue([]any{1, "two"})
	if err != nil || bt != blobmsg.TypeArray {
		t.Errorf("Slice of any encoding failed")
	}
}

func testMoreReflectionInvalidType(t *testing.T) {
	_, _, err := blobmsg.EncodeReflectValue(make(chan int))
	if err == nil {
		t.Errorf("Channel encoding should fail")
	}
}

func testMoreReflectionParseDefaults(t *testing.T) {
	_, err := blobmsg.ParseBlobmsgValue(99, []byte{1}) // Unknown type
	if err == nil {
		t.Errorf("Parse unknown type should fail")
	}
}

func testMoreReflectionGetAttrNameDefault(t *testing.T) {
	if blobmsg.GetAttrName(99) != "attr_99" {
		t.Errorf("GetAttrName default failed")
	}
}

func testMoreReflectionParseJSONTag(t *testing.T) {
	if blobmsg.ParseJSONTag("-") != "" {
		t.Errorf("ParseJSONTag(-) failed")
	}

	if blobmsg.ParseJSONTag("name,omitempty") != "name" {
		t.Errorf("ParseJSONTag(name,omitempty) failed")
	}
}

func TestDecodeErrors(t *testing.T) {
	val, err := blobmsg.DecodeUint([]byte{1, 2, 3}) // too short
	if err == nil || val != 0 {
		t.Errorf("DecodeUint should fail for short payload")
	}

	if blobmsg.DecodeString([]byte{}) != "" {
		t.Errorf("DecodeString(empty) failed")
	}
}
