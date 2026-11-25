package rpc_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/rpc"
)

func TestClient_InitAndEndpointErrors(t *testing.T) {
	var nilCtx context.Context
	_, err := rpc.NewClient(nilCtx, "://bad-url", "u", "p")
	if !errors.Is(err, goubus.ErrInvalidParameter) {
		t.Errorf("invalid URL error = %v, want ErrInvalidParameter", err)
	}
	_, err = rpc.NewClient(t.Context(), "ftp://192.0.2.1", "u", "p")
	if !errors.Is(err, goubus.ErrInvalidParameter) {
		t.Errorf("unsupported scheme error = %v, want ErrInvalidParameter", err)
	}

	notFoundSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer notFoundSrv.Close()

	_, err = rpc.NewClient(t.Context(), notFoundSrv.URL, "u", "p", rpc.WithLogger(nil))
	if !errors.Is(err, goubus.ErrConnectionFailed) || !strings.Contains(err.Error(), "endpoint not found") {
		t.Errorf("404 login error = %v, want endpoint not found wrapping ErrConnectionFailed", err)
	}

	badSessionSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":[0,{"ubus_rpc_session":123}]}`)
	}))
	defer badSessionSrv.Close()

	_, err = rpc.NewClient(t.Context(), badSessionSrv.URL, "u", "p")
	if !errors.Is(err, goubus.ErrInvalidResponse) {
		t.Errorf("malformed session data error = %v, want ErrInvalidResponse", err)
	}
}

func TestClient_ProtocolAndErrorMapping(t *testing.T) {
	var mode atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeRPCRequestBody(r)
		params, _ := req["params"].([]any)
		if len(params) > 0 && params[0] == testAuthSession {
			_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":[0,{"ubus_rpc_session":"sid","timeout":300}]}`)

			return
		}
		writeModeResponse(w, mode.Load())
	}))
	defer srv.Close()

	cli, err := rpc.NewClient(t.Context(), srv.URL, "u", "p")
	if err != nil {
		t.Fatal(err)
	}

	assertPayloadInputVariants(t, cli)
	assertWireErrorResponses(t, cli, &mode)
}

func writeModeResponse(w http.ResponseWriter, m int32) {
	switch m {
	case 0:
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":2,"result":[0,{"big":%q}]}`, strings.Repeat("z", 600))
	case 1:
		_, _ = fmt.Fprint(w, `{invalid-json`)
	case 2:
		_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":2,"result":"not-array"}`)
	case 3:
		_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":2,"result":[1.5]}`)
	case 4:
		_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":2,"result":[4]}`)
	case 5:
		_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":2,"error":{"code":-32000,"message":"obj missing"}}`)
	case 6:
		_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":2,"error":{"code":-32601,"message":"no method"}}`)
	case 7:
		_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":2,"error":{"code":-32602,"message":"bad params"}}`)
	case 8:
		_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":2,"error":{"code":-32600,"message":"bad req"}}`)
	case 9:
		_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":2,"error":{"code":-32099,"message":"other"}}`)
	case 10:
		_, _ = w.Write(nil)
	}
}

func assertPayloadInputVariants(t *testing.T, cli *rpc.Client) {
	t.Helper()

	_, err := cli.Call(t.Context(), "s", "m", `{"nested":{"ubus_rpc_session":"inner"}}`)
	if err != nil {
		t.Fatalf("Call(string JSON with nested key): %v", err)
	}
	_, err = cli.Call(t.Context(), "s", "m", []byte(`{"a":1}`))
	if err != nil {
		t.Fatalf("Call([]byte JSON): %v", err)
	}
	_, err = cli.Call(t.Context(), "s", "m", `{bad`)
	if !errors.Is(err, goubus.ErrInvalidParameter) {
		t.Errorf("invalid string JSON = %v, want ErrInvalidParameter", err)
	}
	_, err = cli.Call(t.Context(), "s", "m", make(chan int))
	if !errors.Is(err, goubus.ErrInvalidParameter) {
		t.Errorf("unmarshalable arg = %v, want ErrInvalidParameter", err)
	}
}

func assertWireErrorResponses(t *testing.T, cli *rpc.Client, mode *atomic.Int32) {
	t.Helper()

	expected := []struct {
		want error
		mode int32
	}{
		{mode: 1, want: goubus.ErrInvalidResponse},
		{mode: 2, want: goubus.ErrInvalidResponse},
		{mode: 3, want: goubus.ErrInvalidResponse},
		{mode: 4, want: goubus.ErrNotFound},
		{mode: 5, want: goubus.ErrNotFound},
		{mode: 6, want: goubus.ErrMethodNotFound},
		{mode: 7, want: goubus.ErrInvalidParameter},
		{mode: 8, want: goubus.ErrInvalidCommand},
		{mode: 9, want: goubus.ErrUnknown},
		{mode: 10, want: goubus.ErrInvalidResponse},
	}
	for _, tc := range expected {
		mode.Store(tc.mode)
		_, err := cli.Call(t.Context(), "s", "m", nil)
		if !errors.Is(err, tc.want) {
			t.Errorf("mode %d error = %v, want %v", tc.mode, err, tc.want)
		}
	}
}

func TestClient_ListErrorsAndConcurrentAutoReauth(t *testing.T) {
	var listStep atomic.Int32
	var logins atomic.Int32
	gate := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeRPCRequestBody(r)
		if req["method"] == "list" {
			switch listStep.Add(1) {
			case 1:
				_, _ = fmt.Fprint(w, `{bad`)
			case 2:
				_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":2,"error":{"code":-32600,"message":"bad"}}`)
			case 3:
				_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":2,"result":null}`)
			default:
				_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":2,"result":"not-map"}`)
			}

			return
		}
		params, _ := req["params"].([]any)
		if len(params) > 0 && params[0] == testAuthSession {
			n := logins.Add(1)
			if n == 2 {
				<-gate
			}
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":[0,{"ubus_rpc_session":"sid-%d","timeout":300}]}`, n)

			return
		}
		if params[0] == "sid-1" {
			_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":2,"error":{"code":-32002,"message":"Access denied"}}`)

			return
		}
		_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":2,"result":[0]}`)
	}))
	defer srv.Close()

	cli, err := rpc.NewClient(t.Context(), srv.URL, "u", "p", rpc.WithAutoReauth(true))
	if err != nil {
		t.Fatal(err)
	}

	assertListEdgeCases(t, cli)
	assertDeduplicatedAutoReauth(t, cli, gate, &logins)

	_ = cli.Close()
	_ = cli.Close()
	if !errors.Is(cli.Authenticate(t.Context()), goubus.ErrClosed) {
		t.Error("expected ErrClosed from Authenticate after Close")
	}
	_, listErr := cli.List(t.Context())
	if !errors.Is(listErr, goubus.ErrClosed) {
		t.Error("expected ErrClosed from List after Close")
	}
}

func assertListEdgeCases(t *testing.T, cli *rpc.Client) {
	t.Helper()

	_, err := cli.List(t.Context(), "")
	if !errors.Is(err, goubus.ErrInvalidParameter) {
		t.Errorf("List(empty) = %v, want ErrInvalidParameter", err)
	}
	_, err = cli.List(t.Context())
	if !errors.Is(err, goubus.ErrInvalidResponse) {
		t.Errorf("List(bad JSON) = %v, want ErrInvalidResponse", err)
	}
	_, err = cli.List(t.Context())
	if !errors.Is(err, goubus.ErrInvalidCommand) {
		t.Errorf("List(RPC error) = %v, want ErrInvalidCommand", err)
	}
	empty, err := cli.List(t.Context())
	if err != nil || len(empty) != 0 {
		t.Errorf("List(null result) = (%v, %v), want empty map", empty, err)
	}
	_, err = cli.List(t.Context())
	if !errors.Is(err, goubus.ErrInvalidResponse) {
		t.Errorf("List(bad signature map) = %v, want ErrInvalidResponse", err)
	}
}

func assertDeduplicatedAutoReauth(t *testing.T, cli *rpc.Client, gate chan struct{}, logins *atomic.Int32) {
	t.Helper()

	var wg sync.WaitGroup
	for range 6 {
		wg.Go(func() {
			_, callErr := cli.Call(context.Background(), "s", "m", nil)
			if callErr != nil {
				t.Errorf("concurrent auto-reauth Call: %v", callErr)
			}
		})
	}
	close(gate)
	wg.Wait()

	if got := logins.Load(); got != 2 {
		t.Errorf("concurrent auto-reauth logins = %d, want 2 (1 initial + 1 deduplicated)", got)
	}
}

func TestClient_TransportFailureAfterConnect(t *testing.T) {
	var logins atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeRPCRequestBody(r)
		params, _ := req["params"].([]any)
		if len(params) > 0 && params[0] == testAuthSession {
			switch logins.Add(1) {
			case 1:
				_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":[0,{"ubus_rpc_session":"sid","timeout":300}]}`)
			case 2:
				w.WriteHeader(http.StatusBadGateway)
			default:
				_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":[0,{"ubus_rpc_session":"sid2","timeout":0}]}`)
			}

			return
		}
		_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":2,"error":{"code":-32002,"message":"Access denied"}}`)
	}))

	cli, err := rpc.NewClient(t.Context(), srv.URL, "u", "p", rpc.WithAutoReauth(true), rpc.WithHTTPClient(nil))
	if err != nil {
		srv.Close()
		t.Fatal(err)
	}

	_, callErr := cli.Call(t.Context(), "s", "m", nil)
	if !errors.Is(callErr, goubus.ErrConnectionFailed) {
		t.Errorf("auto-reauth 502 error = %v, want ErrConnectionFailed", callErr)
	}
	if cli.Authenticate(t.Context()) != nil {
		t.Fatal("Authenticate before server close failed")
	}

	srv.Close()
	_, expErr := cli.Call(t.Context(), "s", "m", nil)
	if !errors.Is(expErr, goubus.ErrConnectionFailed) {
		t.Errorf("Call with expired session after server close = %v, want ErrConnectionFailed", expErr)
	}
	_, listErr := cli.List(t.Context())
	if !errors.Is(listErr, goubus.ErrConnectionFailed) {
		t.Errorf("List after server close = %v, want ErrConnectionFailed", listErr)
	}
	if !errors.Is(cli.Authenticate(t.Context()), goubus.ErrConnectionFailed) {
		t.Error("expected ErrConnectionFailed from Authenticate after server close")
	}
	_ = cli.Close()
}
