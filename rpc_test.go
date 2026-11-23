package goubus_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/honeybbq/goubus/v2"
)

const (
	testUbusEndpointPath = "/ubus"
	testUbusAuthSession  = "00000000000000000000000000000000"
	testSystemService    = "system"
	testInfoMethod       = "info"
	testHostname         = "OpenWrt"
)

func TestRPCClient_NewRPCClient(t *testing.T) {
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

	client, err := goubus.NewRPCClient(ctx, host, "user", "pass")
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

func TestRPCClient_Call(t *testing.T) {
	sessionID := "12345678901234567890123456789012"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleRPCCall(t, w, r, sessionID)
	}))
	defer server.Close()

	host := strings.TrimPrefix(server.URL, "http://")
	ctx := context.Background()

	client, err := goubus.NewRPCClient(ctx, host, "user", "pass")
	if err != nil {
		t.Fatal(err)
	}

	res, err := client.Call(ctx, testSystemService, testInfoMethod, nil)
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

	if info.Hostname != testHostname {
		t.Errorf("expected hostname OpenWrt, got %s", info.Hostname)
	}
}

func handleRPCCall(t *testing.T, writer http.ResponseWriter, request *http.Request, sessionID string) {
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

	if service == testSystemService && method == testInfoMethod {
		_, _ = fmt.Fprint(writer, `{"jsonrpc":"2.0","id":2,"result":[0,{"hostname":"OpenWrt"}]}`)
	}
}

func TestRPCClient_SessionExpiry(t *testing.T) {
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

	client, err := goubus.NewRPCClient(ctx, host, "user", "pass")
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

func TestRPCClientPermissionDeniedRequiresExplicitAuthentication(t *testing.T) {
	server, logins, calls := newRPCSessionRefreshServer(t)
	client, err := goubus.NewRPCClient(t.Context(), server.URL, "user", "password")
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Call(t.Context(), "system", "info", nil)
	if !errors.Is(err, goubus.ErrPermissionDenied) {
		t.Fatalf("Call() error = %v, want ErrPermissionDenied", err)
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("business requests after denied Call() = %d, want 1", got)
	}
	if got := logins.Load(); got != 1 {
		t.Errorf("login requests after denied Call() = %d, want 1", got)
	}

	err = client.Authenticate(t.Context())
	if err != nil {
		t.Fatalf("Authenticate(): %v", err)
	}
	if got := client.String(); !strings.HasPrefix(got, "RPCClient{") {
		t.Errorf("String() = %q, want RPCClient prefix", got)
	}
	_, err = client.Call(t.Context(), "system", "info", nil)
	if err != nil {
		t.Fatalf("Call() after Authenticate(): %v", err)
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("business requests = %d, want 2", got)
	}
	if got := logins.Load(); got != 2 {
		t.Errorf("login requests = %d, want 2", got)
	}
}

func TestRPCClientPermissionDeniedMiddlewareReauthenticatesOnce(t *testing.T) {
	server, logins, calls := newRPCSessionRefreshServer(t)
	client, err := goubus.NewRPCClient(t.Context(), server.URL, "user", "password")
	if err != nil {
		t.Fatal(err)
	}

	reauthenticate := func(client *goubus.RPCClient) goubus.CallMiddleware {
		return func(next goubus.Caller) goubus.Caller {
			return goubus.CallerFunc(func(ctx context.Context, service, method string, data any) (goubus.Result, error) {
				result, callErr := next.Call(ctx, service, method, data)
				if !errors.Is(callErr, goubus.ErrPermissionDenied) {
					return result, callErr
				}
				authErr := client.Authenticate(ctx)
				if authErr != nil {
					return nil, authErr
				}

				return next.Call(ctx, service, method, data)
			})
		}
	}
	caller := goubus.ChainMiddleware(client, reauthenticate(client))
	_, err = caller.Call(t.Context(), "system", "info", nil)
	if err != nil {
		t.Fatalf("Call() through middleware: %v", err)
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("business requests = %d, want 2 (one retry)", got)
	}
	if got := logins.Load(); got != 2 {
		t.Errorf("login requests = %d, want 2 (initial + explicit refresh)", got)
	}
}

func newRPCSessionRefreshServer(t *testing.T) (*httptest.Server, *atomic.Int32, *atomic.Int32) {
	t.Helper()

	var logins, calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body := decodeRPCRequestBody(request)
		params, ok := body["params"].([]any)
		if !ok || len(params) < 3 {
			t.Errorf("unexpected request params: %#v", body["params"])

			return
		}
		if params[0] == testUbusAuthSession {
			sid := logins.Add(1)
			_, _ = fmt.Fprintf(writer,
				`{"jsonrpc":"2.0","result":[0,{"ubus_rpc_session":"session-%d","timeout":3600}]}`,
				sid)

			return
		}
		calls.Add(1)
		if params[0] == "session-1" {
			_, _ = fmt.Fprint(writer,
				`{"jsonrpc":"2.0","error":{"code":-32002,"message":"Access denied"}}`)

			return
		}
		_, _ = fmt.Fprint(writer, `{"jsonrpc":"2.0","result":[0,{"ok":true}]}`)
	}))
	t.Cleanup(server.Close)

	return server, &logins, &calls
}

func TestRPCClientRejectsHTTPErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "temporarily unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	_, err := goubus.NewRPCClient(context.Background(), server.URL, "user", "password")
	if !errors.Is(err, goubus.ErrConnectionFailed) {
		t.Fatalf("NewRPCClient() error = %v, want ErrConnectionFailed", err)
	}
	if !strings.Contains(err.Error(), "503") {
		t.Errorf("NewRPCClient() error = %v, want HTTP status code", err)
	}
}

func TestRPCClient_ErrorHandling(t *testing.T) {
	tests := []struct {
		wantErr       error
		name          string
		response      string
		wantCallError bool
	}{
		{
			name:          "Method Not Found",
			response:      `{"jsonrpc":"2.0","id":2,"error":{"code":-32601,"message":"Method not found"}}`,
			wantErr:       goubus.ErrMethodNotFound,
			wantCallError: true,
		},
		{
			name:          "Ubus Not Found",
			response:      `{"jsonrpc":"2.0","id":2,"result":[4]}`,
			wantErr:       goubus.ErrNotFound,
			wantCallError: true,
		},
		{
			name:     "Invalid Response",
			response: `invalid json`,
			wantErr:  goubus.ErrInvalidResponse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runRPCErrorHandlingCase(t, tt)
		})
	}
}

func runRPCErrorHandlingCase(t *testing.T, testCase struct {
	wantErr       error
	name          string
	response      string
	wantCallError bool
}) {
	t.Helper()

	server := newRPCErrorHandlingServer(t, testCase.response)
	defer server.Close()

	host := strings.TrimPrefix(server.URL, "http://")
	client, err := goubus.NewRPCClient(context.Background(), host, "u", "p")
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Call(context.Background(), "s", "m", nil)
	if err == nil {
		t.Fatal("expected Call to fail before Unmarshal")
	}
	if !errors.Is(err, testCase.wantErr) {
		t.Errorf("Call() error = %v, want errors.Is(_, %v)", err, testCase.wantErr)
	}
	if testCase.wantCallError {
		if _, ok := errors.AsType[*goubus.CallError](err); !ok {
			t.Fatalf("Call() error = %T, want *goubus.CallError", err)
		}
		if !errors.Is(err, goubus.ErrCall) {
			t.Errorf("Call() error %v does not include ErrCall", err)
		}
	}
}

func newRPCErrorHandlingServer(t *testing.T, response string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		reqBody := decodeRPCRequestBody(request)
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

func decodeRPCRequestBody(request *http.Request) map[string]any {
	reqBody := map[string]any{}

	err := json.NewDecoder(request.Body).Decode(&reqBody)
	if err != nil {
		return nil
	}

	return reqBody
}

func TestRPCClientRejectsOversizedResponse(t *testing.T) {
	const oversizedBodyLen = (4 << 20) + 1024

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		reqBody := decodeRPCRequestBody(request)
		if reqBody == nil {
			return
		}
		if reqBody["method"] == "list" {
			_, _ = writer.Write([]byte(`{"jsonrpc":"2.0","id":2,"result":{"pad":"`))
			_, _ = writer.Write([]byte(strings.Repeat("x", oversizedBodyLen)))
			_, _ = writer.Write([]byte(`"}}`))

			return
		}
		params, ok := reqBody["params"].([]any)
		if ok && len(params) > 0 && params[0] == testUbusAuthSession {
			_, _ = fmt.Fprint(writer, `{"jsonrpc":"2.0","id":1,"result":[0,`+
				`{"ubus_rpc_session":"test-session","timeout":3600}]}`)

			return
		}
		_, _ = writer.Write([]byte(`{"jsonrpc":"2.0","id":2,"result":[0,{"pad":"`))
		_, _ = writer.Write([]byte(strings.Repeat("x", oversizedBodyLen)))
		_, _ = writer.Write([]byte(`"}]}`))
	}))
	defer server.Close()

	client, err := goubus.NewRPCClient(t.Context(), server.URL, "user", "pass")
	if err != nil {
		t.Fatalf("NewRPCClient: %v", err)
	}

	_, err = client.Call(t.Context(), testSystemService, testInfoMethod, nil)
	if !errors.Is(err, goubus.ErrInvalidResponse) {
		t.Fatalf("Call() error = %v, want ErrInvalidResponse", err)
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Errorf("Call() error = %v, want size limit message", err)
	}

	_, err = client.List(t.Context(), testSystemService)
	if !errors.Is(err, goubus.ErrInvalidResponse) {
		t.Fatalf("List() error = %v, want ErrInvalidResponse", err)
	}
}

func TestRPCClientHoistsUbusRPCSessionParam(t *testing.T) {
	var seenSID string
	var seenArgs map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		reqBody := decodeRPCRequestBody(request)
		params, ok := reqBody["params"].([]any)
		if !ok || len(params) < 4 {
			t.Fatalf("unexpected params: %#v", reqBody)
		}
		if params[0] == testUbusAuthSession {
			_, _ = fmt.Fprint(writer, `{"jsonrpc":"2.0","id":1,"result":[0,`+
				`{"ubus_rpc_session":"default-session","timeout":3600}]}`)

			return
		}
		seenSID, _ = params[0].(string)
		seenArgs, _ = params[3].(map[string]any)
		_, _ = fmt.Fprint(writer, `{"jsonrpc":"2.0","id":2,"result":[0,{"access":true}]}`)
	}))
	defer server.Close()

	client, err := goubus.NewRPCClient(t.Context(), server.URL, "user", "pass")
	if err != nil {
		t.Fatalf("NewRPCClient: %v", err)
	}

	_, err = client.Call(t.Context(), "session", "access", map[string]any{
		"ubus_rpc_session": "explicit-session",
		"scope":            "ubus",
	})
	if err != nil {
		t.Fatalf("Call(explicit session): %v", err)
	}
	if seenSID != "explicit-session" {
		t.Errorf("params[0] = %q, want explicit-session", seenSID)
	}
	if _, exists := seenArgs["ubus_rpc_session"]; exists {
		t.Errorf("params[3] still contains ubus_rpc_session: %#v", seenArgs)
	}

	_, err = client.Call(t.Context(), "session", "access", map[string]any{
		"ubus_rpc_session": "",
		"scope":            "ubus",
	})
	if err != nil {
		t.Fatalf("Call(empty session): %v", err)
	}
	if seenSID != "default-session" {
		t.Errorf("params[0] = %q, want default-session", seenSID)
	}
	if _, exists := seenArgs["ubus_rpc_session"]; exists {
		t.Errorf("params[3] still contains ubus_rpc_session: %#v", seenArgs)
	}
}
