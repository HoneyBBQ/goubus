package goubus_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/errdefs"
)

const (
	testUbusEndpointPath = "/ubus"
	testUbusAuthSession  = "00000000000000000000000000000000"
)

func TestRpcClient_NewRpcClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != testUbusEndpointPath {
			t.Errorf("expected path %s, got %s", testUbusEndpointPath, r.URL.Path)
		}

		_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":[0,`+
			`{"ubus_rpc_session":"12345678901234567890123456789012","timeout":3600}]}`)
	}))
	defer server.Close()

	host := strings.TrimPrefix(server.URL, "http://")
	ctx := context.Background()

	client, err := goubus.NewRpcClient(ctx, host, "user", "pass")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	defer func() {
		_ = client.Close()
	}()

	if client == nil {
		t.Fatal("client is nil")
	}
}

func TestRpcClient_Call(t *testing.T) {
	sessionID := "12345678901234567890123456789012"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleRpcCall(t, w, r, sessionID)
	}))
	defer server.Close()

	host := strings.TrimPrefix(server.URL, "http://")
	ctx := context.Background()

	client, err := goubus.NewRpcClient(ctx, host, "user", "pass")
	if err != nil {
		t.Fatal(err)
	}

	res, err := client.Call(ctx, "system", "info", nil)
	if err != nil {
		t.Fatal(err)
	}

	var info struct {
		Hostname string `json:"hostname"`
	}

	err = res.Unmarshal(&info)
	if err != nil {
		t.Fatal(err)
	}

	if info.Hostname != "OpenWrt" {
		t.Errorf("expected hostname OpenWrt, got %s", info.Hostname)
	}
}

func handleRpcCall(t *testing.T, writer http.ResponseWriter, request *http.Request, sessionID string) {
	t.Helper()

	var reqBody map[string]any

	errDecode := json.NewDecoder(request.Body).Decode(&reqBody)
	if errDecode != nil {
		t.Fatal(errDecode)
	}

	params, isParams := reqBody["params"].([]any)
	if !isParams {
		t.Fatal("params is not []any")
	}

	method, isMethod := reqBody["method"].(string)
	if !isMethod {
		t.Fatal("method is not string")
	}

	if method != "call" {
		t.Errorf("unexpected request: %v", reqBody)

		return
	}

	if params[0] == testUbusAuthSession {
		// Login
		_, _ = fmt.Fprint(writer, `{"jsonrpc":"2.0","id":1,"result":[0,`+
			`{"ubus_rpc_session":"12345678901234567890123456789012","timeout":3600}]}`)

		return
	}

	if params[0] == sessionID {
		handleActualCall(t, writer, params)

		return
	}

	t.Errorf("unexpected request: %v", reqBody)
}

func handleActualCall(t *testing.T, writer http.ResponseWriter, params []any) {
	t.Helper()

	service, isStr := params[1].(string)
	if !isStr {
		t.Fatal("service is not string")
	}

	method, isStr := params[2].(string)
	if !isStr {
		t.Fatal("method is not string")
	}

	if service == "system" && method == "info" {
		_, _ = fmt.Fprint(writer, `{"jsonrpc":"2.0","id":2,"result":[0,{"hostname":"OpenWrt"}]}`)
	}
}

func TestRpcClient_SessionExpiry(t *testing.T) {
	authCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var reqBody map[string]any

		errDecode := json.NewDecoder(request.Body).Decode(&reqBody)
		if errDecode != nil {
			return
		}

		params, ok := reqBody["params"].([]any)
		if !ok {
			return
		}

		if params[0] == testUbusAuthSession {
			// Login, return session with short timeout
			authCount++
			_, _ = fmt.Fprint(writer, `{"jsonrpc":"2.0","id":1,"result":[0,{"ubus_rpc_session":"s1","timeout":1}]}`)
		} else {
			// Normal call
			_, _ = fmt.Fprint(writer, `{"jsonrpc":"2.0","id":2,"result":[0,{"ok":true}]}`)
		}
	}))
	defer server.Close()

	host := strings.TrimPrefix(server.URL, "http://")
	ctx := context.Background()

	client, err := goubus.NewRpcClient(ctx, host, "user", "pass")
	if err != nil {
		t.Fatal(err)
	}

	// First call should use existing session
	_, _ = client.Call(ctx, "s", "m", nil)
	initialAuthCount := authCount

	// Second call should trigger re-authentication
	time.Sleep(2 * time.Second)

	_, _ = client.Call(ctx, "s", "m", nil)

	if authCount <= initialAuthCount {
		t.Errorf("expected re-authentication, but auth count only increased by %d", authCount-initialAuthCount)
	}
}

func TestRpcClient_ErrorHandling(t *testing.T) {
	tests := []struct {
		wantErr  error
		name     string
		response string
	}{
		{
			name:     "Method Not Found",
			response: `{"jsonrpc":"2.0","id":1,"error":{"code":3,"message":"Method not found"}}`,
			wantErr:  errdefs.ErrMethodNotFound,
		},
		{
			name:     "Ubus Not Found",
			response: `{"jsonrpc":"2.0","id":1,"result":[4]}`,
			wantErr:  errdefs.ErrNotFound,
		},
		{
			name:     "Invalid Response",
			response: `invalid json`,
			wantErr:  errdefs.ErrInvalidResponse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runRpcErrorHandlingCase(t, tt)
		})
	}
}

func runRpcErrorHandlingCase(t *testing.T, testCase struct {
	wantErr  error
	name     string
	response string
}) {
	t.Helper()

	server := newRpcErrorHandlingServer(t, testCase.response)
	defer server.Close()

	host := strings.TrimPrefix(server.URL, "http://")

	client, err := goubus.NewRpcClient(context.Background(), host, "u", "p")
	if err != nil {
		t.Fatal(err)
	}

	result, err := client.Call(context.Background(), "s", "m", nil)
	if err != nil {
		assertErrorContains(t, err, testCase.wantErr)

		return
	}

	err = result.Unmarshal(&struct{}{})
	if err == nil {
		t.Fatal("expected error from Unmarshal, got nil")
	}

	assertErrorContains(t, err, testCase.wantErr)
}

func newRpcErrorHandlingServer(t *testing.T, response string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		reqBody := decodeRpcRequestBody(request)
		if reqBody == nil {
			return
		}

		params, ok := reqBody["params"].([]any)
		if !ok {
			return
		}

		if params[0] == testUbusAuthSession {
			_, _ = fmt.Fprint(writer, `{"jsonrpc":"2.0","id":1,"result":[0,`+
				`{"ubus_rpc_session":"test-session","timeout":3600}]}`)

			return
		}

		_, _ = fmt.Fprint(writer, response)
	}))
}

func decodeRpcRequestBody(request *http.Request) map[string]any {
	reqBody := map[string]any{}

	err := json.NewDecoder(request.Body).Decode(&reqBody)
	if err != nil {
		return nil
	}

	return reqBody
}

func assertErrorContains(t *testing.T, got error, want error) {
	t.Helper()

	if want == nil {
		return
	}

	if !strings.Contains(strings.ToLower(got.Error()), strings.ToLower(want.Error())) {
		t.Errorf("expected error containing %v, got %v", want, got)
	}
}
