package rpc_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/rpc"
)

const (
	testAuthSession   = "00000000000000000000000000000000"
	testSystemService = "system"
	testInfoMethod    = "info"
)

type listTestState struct {
	params []any
	calls  int
}

func TestMain(m *testing.M) {
	code := m.Run()
	if code != 0 {
		os.Exit(code)
	}

	runtime.GC()
	runtime.GC()

	prof := pprof.Lookup("goroutineleak")
	if prof != nil && prof.Count() > 0 {
		var buf bytes.Buffer
		_ = prof.WriteTo(&buf, 2)
		fmt.Fprintf(os.Stderr, "detected %d leaked goroutine(s):\n%s\n", prof.Count(), buf.String())
		os.Exit(1)
	}
}

func TestClient_NewClientAndCall(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/custom-ubus" {
			t.Errorf("URL.Path = %q, want /custom-ubus", r.URL.Path)
		}
		reqBody := decodeRPCRequestBody(r)
		params, _ := reqBody["params"].([]any)
		if len(params) > 0 && params[0] == testAuthSession {
			_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":[0,{"ubus_rpc_session":"secret-sid","timeout":3600}]}`)

			return
		}
		_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":2,"result":[0,{"hostname":"OpenWrt"}]}`)
	}))
	defer server.Close()

	client, err := rpc.NewClient(t.Context(), server.URL, "root", "super-secret",
		rpc.WithEndpoint("custom-ubus"),
		rpc.WithLogger(logger),
		rpc.WithHTTPClient(nil),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	info, err := goubus.Call[struct {
		Hostname string `json:"hostname"`
	}](t.Context(), client, testSystemService, testInfoMethod, nil)
	if err != nil || info.Hostname != "OpenWrt" {
		t.Fatalf("Call() = %+v, err %v", info, err)
	}

	assertCredentialRedaction(t, client, logBuf.String())

	closeErr := client.Close()
	if closeErr != nil {
		t.Fatalf("Close(): %v", closeErr)
	}
	_, err = client.Call(t.Context(), testSystemService, testInfoMethod, nil)
	if !errors.Is(err, goubus.ErrClosed) {
		t.Errorf("Call after Close = %v, want ErrClosed", err)
	}
}

func assertCredentialRedaction(t *testing.T, client *rpc.Client, logs string) {
	t.Helper()

	formatted := fmt.Sprintf("%s | %#v", client, client)
	var attrBuf bytes.Buffer
	slog.New(slog.NewTextHandler(&attrBuf, nil)).Info("client", slog.Any("rpc", client))

	combined := formatted + "\n" + attrBuf.String() + "\n" + logs
	for _, secret := range []string{"super-secret", "secret-sid"} {
		if strings.Contains(combined, secret) {
			t.Errorf("leaked secret %q in output:\n%s", secret, combined)
		}
	}
}

func TestClient_SessionSlidingAndIdleExpiry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var authCount atomic.Int32
		server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqBody := decodeRPCRequestBody(r)
			params, _ := reqBody["params"].([]any)
			if len(params) > 0 && params[0] == testAuthSession {
				authCount.Add(1)
				_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":[0,{"ubus_rpc_session":"s1","timeout":3600}]}`)

				return
			}
			_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":2,"result":[0,{"ok":true}]}`)
		}))

		client, err := rpc.NewClient(t.Context(), server.URL, "user", "pass", rpc.WithHTTPClient(server.Client()))
		if err != nil {
			t.Fatal(err)
		}

		// Calling at t=2000s slides the 3600s idle window past t=4000s.
		time.Sleep(2000 * time.Second)
		_, callErr := client.Call(t.Context(), "s", "m", nil)
		if callErr != nil {
			t.Fatal(callErr)
		}
		time.Sleep(2000 * time.Second)
		_, callErr = client.Call(t.Context(), "s", "m", nil)
		if callErr != nil {
			t.Fatal(callErr)
		}
		if got := authCount.Load(); got != 1 {
			t.Fatalf("authCount after sliding calls = %d, want 1", got)
		}

		// Idle past 3600s triggers re-authentication.
		time.Sleep(3601 * time.Second)
		_, callErr = client.Call(t.Context(), "s", "m", nil)
		if callErr != nil {
			t.Fatal(callErr)
		}
		if got := authCount.Load(); got != 2 {
			t.Errorf("authCount after idle expiry = %d, want 2", got)
		}
	})
}

func TestClient_PermissionDeniedAndExplicitReauthenticate(t *testing.T) {
	var logins, calls atomic.Int32
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqBody := decodeRPCRequestBody(r)
		params, _ := reqBody["params"].([]any)
		if len(params) > 0 && params[0] == testAuthSession {
			n := logins.Add(1)
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":[0,{"ubus_rpc_session":"sid-%d","timeout":3600}]}`, n)

			return
		}
		calls.Add(1)
		if params[0] == "sid-1" {
			_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":2,"error":{"code":-32002,"message":"Access denied"}}`)

			return
		}
		_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":2,"result":[0,{"ok":true}]}`)
	}))

	client, err := rpc.NewClient(t.Context(), server.URL, "user", "password", rpc.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Call(t.Context(), testSystemService, testInfoMethod, nil)
	if !errors.Is(err, goubus.ErrPermissionDenied) || calls.Load() != 1 || logins.Load() != 1 {
		t.Fatalf("denied Call = %v, calls=%d, logins=%d", err, calls.Load(), logins.Load())
	}

	authErr := client.Authenticate(t.Context())
	if authErr != nil {
		t.Fatalf("Authenticate(): %v", authErr)
	}
	_, err = client.Call(t.Context(), testSystemService, testInfoMethod, nil)
	if err != nil {
		t.Fatalf("Call after Authenticate: %v", err)
	}
}

func TestClient_WithAutoReauthAndValidation(t *testing.T) {
	var logins atomic.Int32
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqBody := decodeRPCRequestBody(r)
		params, _ := reqBody["params"].([]any)
		if len(params) > 0 && params[0] == testAuthSession {
			n := logins.Add(1)
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":[0,{"ubus_rpc_session":"sid-%d","timeout":3600}]}`, n)

			return
		}
		if params[0] == "sid-1" || params[2] == "forbidden" {
			_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":2,"error":{"code":-32002,"message":"Access denied"}}`)

			return
		}
		_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":2,"result":[0,{"ok":true}]}`)
	}))

	client, err := rpc.NewClient(t.Context(), server.URL, "user", "password",
		rpc.WithHTTPClient(server.Client()),
		rpc.WithAutoReauth(true),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Call(t.Context(), "", "info", nil)
	if !errors.Is(err, goubus.ErrInvalidParameter) {
		t.Fatalf("empty service error = %v, want ErrInvalidParameter", err)
	}
	_, err = client.Call(t.Context(), testSystemService, "info", []int{1, 2})
	if !errors.Is(err, goubus.ErrInvalidParameter) {
		t.Fatalf("non-object payload error = %v, want ErrInvalidParameter", err)
	}

	var typedNil *struct{ A int }
	_, err = client.Call(t.Context(), testSystemService, testInfoMethod, typedNil)
	if err != nil || logins.Load() != 2 {
		t.Fatalf("auto-reauth Call = %v, logins = %d; want nil, 2", err, logins.Load())
	}

	// Genuine ACL denial on an already-fresh session returns ErrPermissionDenied after one re-auth attempt.
	_, err = client.Call(t.Context(), testSystemService, "forbidden", nil)
	if !errors.Is(err, goubus.ErrPermissionDenied) {
		t.Fatalf("ACL-denied Call error = %v, want ErrPermissionDenied", err)
	}
}

func TestClient_ConcurrentReauthAndClose(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var logins atomic.Int32
		server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqBody := decodeRPCRequestBody(r)
			params, _ := reqBody["params"].([]any)
			if len(params) > 0 && params[0] == testAuthSession {
				logins.Add(1)
				_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":[0,{"ubus_rpc_session":"sid","timeout":10}]}`)

				return
			}
			_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":2,"result":[0]}`)
		}))

		client, err := rpc.NewClient(t.Context(), server.URL, "u", "p", rpc.WithHTTPClient(server.Client()))
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(15 * time.Second)

		var wg sync.WaitGroup
		for range 12 {
			wg.Go(func() {
				_, _ = client.Call(t.Context(), testSystemService, "reboot", nil)
			})
		}
		wg.Wait()

		if got := logins.Load(); got != 2 {
			t.Errorf("concurrent expired calls triggered %d logins, want 2", got)
		}

		closeErr := client.Close()
		if closeErr != nil {
			t.Fatalf("Close(): %v", closeErr)
		}
		authErr := client.Authenticate(t.Context())
		if !errors.Is(authErr, goubus.ErrClosed) {
			t.Errorf("Authenticate after Close = %v, want ErrClosed", authErr)
		}
	})
}

func TestClient_ListAndOversizedCap(t *testing.T) {
	state := &listTestState{}
	client := newListTestClient(t, state)

	objects, err := goubus.List(t.Context(), client)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(state.params, []any{"*"}) {
		t.Errorf("wildcard params = %#v, want [\"*\"]", state.params)
	}
	if objects["system"]["info"]["uptime"] != goubus.ArgTypeNumber {
		t.Errorf("List normalized = %#v", objects)
	}

	_, listErr := client.List(t.Context(), "")
	if !errors.Is(listErr, goubus.ErrInvalidParameter) {
		t.Errorf("empty pattern error = %v, want ErrInvalidParameter", listErr)
	}

	assertOversizedRejected(t)
}

func TestClient_HoistsExplicitSessionAndHandles404(t *testing.T) {
	notFoundSrv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	_, err := rpc.NewClient(t.Context(), notFoundSrv.URL, "u", "p", rpc.WithHTTPClient(notFoundSrv.Client()))
	if !errors.Is(err, goubus.ErrConnectionFailed) || !strings.Contains(err.Error(), "endpoint not found") {
		t.Fatalf("404 login error = %v, want ErrConnectionFailed with endpoint not found", err)
	}

	assertSessionHoisting(t)
}

func assertSessionHoisting(t *testing.T) {
	t.Helper()

	var seenSID string
	var seenArgs map[string]any
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqBody := decodeRPCRequestBody(r)
		params, _ := reqBody["params"].([]any)
		if params[0] == testAuthSession {
			_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":[0,{"ubus_rpc_session":"default-sid","timeout":3600}]}`)

			return
		}
		seenSID, _ = params[0].(string)
		seenArgs, _ = params[3].(map[string]any)
		_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":2,"result":[0,{"access":true}]}`)
	}))

	client, err := rpc.NewClient(t.Context(), server.URL, "u", "p", rpc.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Call(t.Context(), "session", "access", map[string]any{
		"ubus_rpc_session": "explicit-sid",
		"scope":            "ubus",
	})
	if err != nil || seenSID != "explicit-sid" {
		t.Fatalf("explicit session Call = (%q, %v)", seenSID, err)
	}
	if _, exists := seenArgs["ubus_rpc_session"]; exists {
		t.Errorf("ubus_rpc_session not stripped from params[3]: %#v", seenArgs)
	}
}

func newListTestClient(t *testing.T, state *listTestState) *rpc.Client {
	t.Helper()

	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := decodeRPCRequestBody(r)
		if body["method"] == "call" {
			_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":[0,{"ubus_rpc_session":"sid","timeout":3600}]}`)

			return
		}
		state.calls++
		state.params, _ = body["params"].([]any)
		_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":2,"result":{`+
			`"system":{"info":{"hostname":"string","uptime":["number","integer"]}}}}`)
	}))

	client, err := rpc.NewClient(t.Context(), server.URL, "u", "p", rpc.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}

	return client
}

func assertOversizedRejected(t *testing.T) {
	t.Helper()

	const oversizedLen = (4 << 20) + 1024
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqBody := decodeRPCRequestBody(r)
		params, _ := reqBody["params"].([]any)
		if len(params) > 0 && params[0] == testAuthSession {
			_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":[0,{"ubus_rpc_session":"sid","timeout":3600}]}`)

			return
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":2,"result":[0,{"pad":"`))
		_, _ = w.Write([]byte(strings.Repeat("x", oversizedLen)))
		_, _ = w.Write([]byte(`"}]}`))
	}))

	client, err := rpc.NewClient(t.Context(), server.URL, "u", "p", rpc.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	_, callErr := client.Call(t.Context(), "s", "m", nil)
	if !errors.Is(callErr, goubus.ErrInvalidResponse) {
		t.Errorf("oversized Call error = %v, want ErrInvalidResponse", callErr)
	}
}

func decodeRPCRequestBody(request *http.Request) map[string]any {
	reqBody := map[string]any{}
	decodeErr := json.NewDecoder(request.Body).Decode(&reqBody)
	if decodeErr != nil {
		return nil
	}

	return reqBody
}

func startExampleRouter() (string, func()) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeRPCRequestBody(r)
		if req["method"] == "list" {
			_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":2,"result":{"system":{"board":{"hostname":"string"}}}}`)

			return
		}
		params, _ := req["params"].([]any)
		if len(params) > 0 && params[0] == testAuthSession {
			_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":[0,{"ubus_rpc_session":"sid","timeout":300}]}`)

			return
		}
		_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":2,"result":[0,{"hostname":"OpenWrt"}]}`)
	}))

	return srv.URL, srv.Close
}
