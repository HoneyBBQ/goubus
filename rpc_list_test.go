package goubus_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/honeybbq/goubus/v2"
)

type rpcListTestState struct {
	params []any
	calls  int
}

func TestRPCClientList(t *testing.T) {
	state := &rpcListTestState{}
	client := newRPCListTestClient(t, state)
	checkRPCListPatterns(t, client, state)
	checkRPCListInvalidPattern(t, client, state)
}

func newRPCListTestClient(t *testing.T, state *rpcListTestState) *goubus.RPCClient {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body := decodeRPCRequestBody(request)
		if body["method"] == "call" {
			_, _ = fmt.Fprint(writer, `{"jsonrpc":"2.0","id":1,"result":[0,{`+
				`"ubus_rpc_session":"session-secret","timeout":3600}]}`)

			return
		}
		state.calls++
		state.params, _ = body["params"].([]any)
		_, _ = fmt.Fprint(writer, `{
			"jsonrpc":"2.0",
			"id":2,
			"result":{
				"system":{
					"info":{"hostname":"string","uptime":["number","integer"]},
					"reboot":{}
				},
				"network":{"device":{"name":"string"}}
			}
		}`)
	}))
	t.Cleanup(server.Close)

	client, err := goubus.NewRPCClient(context.Background(), server.URL, "user", "password")
	if err != nil {
		t.Fatal(err)
	}

	return client
}

func checkRPCListPatterns(t *testing.T, client *goubus.RPCClient, state *rpcListTestState) {
	t.Helper()
	objects, err := client.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(state.params, []any{"*"}) {
		t.Errorf("wildcard list params = %#v, want [\"*\"]", state.params)
	}
	if objects["system"]["info"]["hostname"] != goubus.ArgTypeString ||
		objects["system"]["info"]["uptime"] != goubus.ArgTypeNumber {
		t.Errorf("normalized list signatures = %#v", objects)
	}

	objects, err = client.List(context.Background(), "system*", "network")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(state.params, []any{"system*", "network"}) {
		t.Errorf("pattern list params = %#v", state.params)
	}
	if len(objects) != 2 {
		t.Errorf("List() returned %d objects, want 2", len(objects))
	}

	_, err = client.List(context.Background(), "system*", "*")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(state.params, []any{"*"}) {
		t.Errorf("mixed wildcard list params = %#v", state.params)
	}
}

func checkRPCListInvalidPattern(t *testing.T, client *goubus.RPCClient, state *rpcListTestState) {
	t.Helper()
	before := state.calls
	_, err := client.List(context.Background(), "system", "")
	if !errors.Is(err, goubus.ErrInvalidParameter) {
		t.Errorf("empty pattern error = %v, want ErrInvalidParameter", err)
	}
	if state.calls != before {
		t.Errorf("empty pattern sent %d unexpected list request(s)", state.calls-before)
	}
}

func TestRPCClientListReturnsClosed(t *testing.T) {
	state := &rpcListTestState{}
	client := newRPCListTestClient(t, state)
	closeErr := client.Close()
	if closeErr != nil {
		t.Fatal(closeErr)
	}

	before := state.calls
	_, err := client.List(context.Background(), "system")
	if !errors.Is(err, goubus.ErrClosed) {
		t.Errorf("List() error = %v, want ErrClosed", err)
	}
	if state.calls != before {
		t.Errorf("closed List() sent %d request(s)", state.calls-before)
	}
}

func TestRPCClientListEmptyResultIsNonNil(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body := decodeRPCRequestBody(request)
		if body["method"] == "call" {
			_, _ = fmt.Fprint(writer, `{"jsonrpc":"2.0","result":[0,{"ubus_rpc_session":"s","timeout":3600}]}`)

			return
		}
		_, _ = fmt.Fprint(writer, `{"jsonrpc":"2.0","result":{}}`)
	}))
	defer server.Close()
	client, err := goubus.NewRPCClient(context.Background(), server.URL, "u", "p")
	if err != nil {
		t.Fatal(err)
	}
	objects, err := client.List(context.Background(), "absent")
	if err != nil || objects == nil || len(objects) != 0 {
		t.Errorf("List() = %#v, %v; want empty non-nil map", objects, err)
	}
}

func TestArgTypeUnmarshalJSON(t *testing.T) {
	cases := map[string]goubus.ArgType{
		`"string"`: goubus.ArgTypeString,
		`["number","integer","int32","int64","int16","double"]`: goubus.ArgTypeNumber,
		`["boolean","bool","int8"]`:                             goubus.ArgTypeBoolean,
		`"array"`:                                               goubus.ArgTypeArray,
		`["object","table"]`:                                    goubus.ArgTypeObject,
		`"other"`:                                               goubus.ArgTypeUnknown,
		`["other"]`:                                             goubus.ArgTypeUnknown,
	}
	for input, want := range cases {
		var got goubus.ArgType
		err := json.Unmarshal([]byte(input), &got)
		if err != nil || got != want {
			t.Errorf("Unmarshal(%s) = %q, %v; want %q", input, got, err, want)
		}
	}
}

func TestRPCClientRedactsCredentials(t *testing.T) {
	var logs strings.Builder
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(writer, `{"jsonrpc":"2.0","result":[0,{"ubus_rpc_session":"session-secret","timeout":3600}]}`)
	}))
	defer server.Close()

	client, err := goubus.NewRPCClient(
		context.Background(), server.URL, "user", "password-secret", goubus.WithRPCLogger(logger),
	)
	if err != nil {
		t.Fatal(err)
	}
	values := []string{client.String(), fmt.Sprintf("%#v", client), fmt.Sprint(client.LogValue()), logs.String()}
	for _, value := range values {
		if strings.Contains(value, "password-secret") || strings.Contains(value, "session-secret") {
			t.Errorf("credential leaked in %q", value)
		}
	}
	if !strings.Contains(client.String(), "<redacted>") {
		t.Errorf("String() did not show redaction marker: %s", client.String())
	}
}
