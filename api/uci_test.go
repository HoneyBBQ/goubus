package api

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/honeybbq/goubus/errdefs"
	"github.com/honeybbq/goubus/types"
)

type callRecord struct {
	service string
	method  string
	data    any
}

type queuedResponse struct {
	result types.Result
	err    error
}

type mockTransport struct {
	queue []queuedResponse
	calls []callRecord
}

func (m *mockTransport) Call(service, method string, data any) (types.Result, error) {
	m.calls = append(m.calls, callRecord{service: service, method: method, data: data})

	if len(m.queue) == 0 {
		return mockResult{}, nil
	}
	resp := m.queue[0]
	m.queue = m.queue[1:]
	return resp.result, resp.err
}

func (m *mockTransport) Close() error { return nil }

type mockResult struct {
	unmarshal func(target any) error
}

func (m mockResult) Unmarshal(target any) error {
	if m.unmarshal != nil {
		return m.unmarshal(target)
	}
	return nil
}

func jsonResult(body string) mockResult {
	return mockResult{
		unmarshal: func(target any) error {
			return json.Unmarshal([]byte(body), target)
		},
	}
}

func TestSetUciInvokesTransport(t *testing.T) {
	mt := &mockTransport{
		queue: []queuedResponse{{result: mockResult{}}},
	}
	req := types.UbusUciRequest{
		UbusUciRequestGeneric: types.UbusUciRequestGeneric{
			Config:  "network",
			Section: "lan",
		},
		Values: map[string]any{"proto": "dhcp"},
	}

	if err := SetUci(mt, req); err != nil {
		t.Fatalf("SetUci returned error: %v", err)
	}

	if len(mt.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mt.calls))
	}
	call := mt.calls[0]
	if call.service != ServiceUCI {
		t.Errorf("service = %s, want %s", call.service, ServiceUCI)
	}
	if call.method != UciMethodSet {
		t.Errorf("method = %s, want %s", call.method, UciMethodSet)
	}
	if !reflect.DeepEqual(call.data, req) {
		t.Errorf("request mismatch\n got  %#v\n want %#v", call.data, req)
	}
}

func TestGetUciReturnsResponse(t *testing.T) {
	wantValues := map[string]any{
		"proto": "static",
		"dns":   []any{"1.1.1.1", "8.8.8.8"},
	}
	mt := &mockTransport{
		queue: []queuedResponse{{
			result: mockResult{
				unmarshal: func(target any) error {
					resp := target.(*types.UbusUciGetResponse)
					resp.Values = wantValues
					return nil
				},
			},
		}},
	}
	req := types.UbusUciGetRequest{
		UbusUciRequestGeneric: types.UbusUciRequestGeneric{
			Config:  "network",
			Section: "lan",
		},
	}

	resp, err := GetUci(mt, req)
	if err != nil {
		t.Fatalf("GetUci returned error: %v", err)
	}
	if !reflect.DeepEqual(resp.Values, wantValues) {
		t.Errorf("values mismatch\n got  %#v\n want %#v", resp.Values, wantValues)
	}

	if len(mt.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mt.calls))
	}
}

func TestAddToUciListAppendsValue(t *testing.T) {
	mt := &mockTransport{
		queue: []queuedResponse{
			{result: jsonResult(`{"value":"foo"}`)},
			{
				result: mockResult{},
			},
		},
	}

	if err := AddToUciList(mt, "network", "lan", "dns", "bar"); err != nil {
		t.Fatalf("AddToUciList returned error: %v", err)
	}

	if len(mt.calls) != 2 {
		t.Fatalf("expected 2 calls, got %d: %#v", len(mt.calls), mt.calls)
	}

	setCall := mt.calls[1]
	req, ok := setCall.data.(types.UbusUciRequest)
	if !ok {
		t.Fatalf("set call data has type %T, want types.UbusUciRequest", setCall.data)
	}
	value, ok := req.Values["dns"].(string)
	if !ok {
		t.Fatalf("dns value has type %T, want string", req.Values["dns"])
	}
	if value != "foo bar" {
		t.Errorf("dns value = %q, want %q", value, "foo bar")
	}
}

func TestAddToUciListCreatesListWhenMissing(t *testing.T) {
	mt := &mockTransport{
		queue: []queuedResponse{
			{result: mockResult{}, err: errdefs.ErrNotFound},
			{result: mockResult{}},
		},
	}

	if err := AddToUciList(mt, "network", "lan", "dns", "1.1.1.1"); err != nil {
		t.Fatalf("AddToUciList returned error: %v", err)
	}

	setCall := mt.calls[1]
	req := setCall.data.(types.UbusUciRequest)
	value := req.Values["dns"].(string)
	if value != "1.1.1.1" {
		t.Errorf("dns value = %q, want %q", value, "1.1.1.1")
	}
}

func TestParseUciMetadataCastsValues(t *testing.T) {
	meta := ParseUciMetadata(map[string]any{
		".name":      "wan",
		".type":      "interface",
		".index":     float64(3),
		".anonymous": "true",
	})

	if meta.Name != "wan" {
		t.Errorf("Name = %s, want wan", meta.Name)
	}
	if meta.Type != "interface" {
		t.Errorf("Type = %s, want interface", meta.Type)
	}
	if meta.Index == nil || *meta.Index != 3 {
		t.Fatalf("Index = %v, want 3", meta.Index)
	}
	if !bool(meta.Anonymous) {
		t.Errorf("Anonymous = %v, want true", meta.Anonymous)
	}
}

func TestGetUciPropagatesTransportError(t *testing.T) {
	expectedErr := errors.New("boom")
	mt := &mockTransport{
		queue: []queuedResponse{{err: expectedErr}},
	}

	_, err := GetUci(mt, types.UbusUciGetRequest{})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("error = %v, want %v", err, expectedErr)
	}
}

func TestAddUciInvokesTransport(t *testing.T) {
	mt := &mockTransport{
		queue: []queuedResponse{{result: mockResult{}}},
	}
	req := types.UbusUciRequest{
		UbusUciRequestGeneric: types.UbusUciRequestGeneric{
			Config: "network",
			Type:   "interface",
		},
		Values: map[string]any{"proto": "static"},
	}

	if err := AddUci(mt, req); err != nil {
		t.Fatalf("AddUci returned error: %v", err)
	}

	if len(mt.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mt.calls))
	}
	if mt.calls[0].method != UciMethodAdd {
		t.Errorf("method = %s, want %s", mt.calls[0].method, UciMethodAdd)
	}
}

func TestDeleteFromUciListRemovesValue(t *testing.T) {
	mt := &mockTransport{
		queue: []queuedResponse{
			{result: jsonResult(`{"value":"foo bar baz"}`)},
			{result: mockResult{}},
		},
	}

	if err := DeleteFromUciList(mt, "network", "lan", "dns", "bar"); err != nil {
		t.Fatalf("DeleteFromUciList returned error: %v", err)
	}

	if len(mt.calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(mt.calls))
	}

	setCall := mt.calls[1]
	req, ok := setCall.data.(types.UbusUciRequest)
	if !ok {
		t.Fatalf("call data type %T, want types.UbusUciRequest", setCall.data)
	}
	value := req.Values["dns"].(string)
	if value != "foo baz" {
		t.Errorf("dns value = %q, want %q", value, "foo baz")
	}
}

func TestGetAllUciExtractsSections(t *testing.T) {
	mt := &mockTransport{
		queue: []queuedResponse{{
			result: mockResult{
				unmarshal: func(target any) error {
					resp := target.(*types.UbusUciGetResponse)
					resp.Values = map[string]any{
						"lan": map[string]any{".type": "interface"},
						"wan": map[string]any{".type": "interface"},
						"cfg": "not-a-map",
					}
					return nil
				},
			},
		}},
	}

	all, err := GetAllUci(mt, types.UbusUciGetRequest{})
	if err != nil {
		t.Fatalf("GetAllUci returned error: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(all))
	}
	if _, ok := all["lan"]; !ok {
		t.Errorf("lan missing from sections")
	}
	if _, ok := all["wan"]; !ok {
		t.Errorf("wan missing from sections")
	}
}

func TestGetUciSectionsFiltersByType(t *testing.T) {
	mt := &mockTransport{
		queue: []queuedResponse{{
			result: mockResult{
				unmarshal: func(target any) error {
					resp := target.(*types.UbusUciGetResponse)
					resp.Values = map[string]any{
						"lan":   map[string]any{".type": "interface"},
						"wan":   map[string]any{".type": "interface"},
						"wifi0": map[string]any{".type": "wifi-device"},
					}
					return nil
				},
			},
		}},
	}

	names, err := GetUciSections(mt, "network", "interface")
	if err != nil {
		t.Fatalf("GetUciSections returned error: %v", err)
	}
	want := map[string]struct{}{"lan": {}, "wan": {}}
	if len(names) != len(want) {
		t.Fatalf("unexpected names %v", names)
	}
	for _, name := range names {
		if _, ok := want[name]; !ok {
			t.Fatalf("unexpected section %s", name)
		}
	}
}

func TestGetUciSectionsListExtractsNames(t *testing.T) {
	mt := &mockTransport{
		queue: []queuedResponse{{
			result: mockResult{
				unmarshal: func(target any) error {
					data := target.(*map[string]any)
					*data = map[string]any{
						"sections": map[string]any{"lan": nil, "wan": nil},
					}
					return nil
				},
			},
		}},
	}

	names, err := GetUciSectionsList(mt, types.UbusUciGetRequest{})
	if err != nil {
		t.Fatalf("GetUciSectionsList returned error: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
}

func TestCommitUciInvokesTransport(t *testing.T) {
	mt := &mockTransport{
		queue: []queuedResponse{{result: mockResult{}}},
	}
	if err := CommitUci(mt, types.UbusUciRequestGeneric{Config: "network"}); err != nil {
		t.Fatalf("CommitUci returned error: %v", err)
	}
	if len(mt.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mt.calls))
	}
	if mt.calls[0].method != UciMethodCommit {
		t.Errorf("method = %s, want %s", mt.calls[0].method, UciMethodCommit)
	}
}

func TestGetUciValueFromJSON(t *testing.T) {
	mt := &mockTransport{
		queue: []queuedResponse{{result: jsonResult(`{"value":"foo bar baz"}`)}},
	}
	req := types.UbusUciGetRequest{
		UbusUciRequestGeneric: types.UbusUciRequestGeneric{
			Config:  "network",
			Section: "lan",
			Option:  "dns",
		},
	}
	resp, err := GetUci(mt, req)
	if err != nil {
		t.Fatalf("GetUci returned error: %v", err)
	}
	if resp.Value != "foo bar baz" {
		t.Fatalf("Value = %q", resp.Value)
	}
}

func TestJSONResultUnmarshal(t *testing.T) {
	var resp types.UbusUciGetResponse
	if err := jsonResult(`{"value":"foo"}`).Unmarshal(&resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp.Value != "foo" {
		t.Fatalf("Value = %q", resp.Value)
	}
}
