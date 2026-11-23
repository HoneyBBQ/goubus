package goubus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRPCClientAuthenticateSerializesRefresh(t *testing.T) {
	client, logins := newRPCSessionStateClient(t)
	client.rwMutex.Lock()
	client.sessionData.ExpireTime = time.Now().Add(-time.Second)
	client.rwMutex.Unlock()

	const callers = 16
	var wait sync.WaitGroup
	wait.Add(callers)
	errCh := make(chan error, callers)
	for range callers {
		go func() {
			defer wait.Done()
			errCh <- client.ensureAuthenticated(context.Background())
		}()
	}
	wait.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Errorf("ensureAuthenticated(): %v", err)
		}
	}
	if got := logins.Load(); got != 2 {
		t.Errorf("login requests = %d, want 2 (initial + one refresh)", got)
	}
}

func TestRPCClientCloseClearsCredentialsAndBlocksAuthentication(t *testing.T) {
	client, logins := newRPCSessionStateClient(t)
	closeErr := client.Close()
	if closeErr != nil {
		t.Fatalf("Close(): %v", closeErr)
	}
	if client.password != "" {
		t.Error("Close() did not clear the password")
	}

	loginCount := logins.Load()
	err := client.Authenticate(context.Background())
	if !errors.Is(err, ErrClosed) {
		t.Errorf("Authenticate() after Close() = %v, want ErrClosed", err)
	}
	sessionID, err := client.getValidSessionID(context.Background())
	if sessionID != "" || !errors.Is(err, ErrClosed) {
		t.Errorf("getValidSessionID() after Close() = %q, %v; want ErrClosed", sessionID, err)
	}
	if logins.Load() != loginCount {
		t.Error("Authenticate() after Close() sent another login")
	}
}

func TestRPCClientSuccessfulCallSlidesIdleTimeout(t *testing.T) {
	client, _ := newRPCSessionStateClient(t)
	client.rwMutex.RLock()
	originalExpiry := client.sessionData.ExpireTime
	client.rwMutex.RUnlock()

	_, err := client.Call(context.Background(), "system", "info", nil)
	if err != nil {
		t.Fatalf("Call(): %v", err)
	}

	client.rwMutex.RLock()
	updatedExpiry := client.sessionData.ExpireTime
	timeout := client.sessionData.Timeout
	client.rwMutex.RUnlock()
	if timeout <= 0 {
		t.Fatalf("session timeout = %d, want positive timeout", timeout)
	}
	if !updatedExpiry.After(originalExpiry) {
		t.Errorf("session expiry = %v, want after original expiry %v", updatedExpiry, originalExpiry)
	}
}

func newRPCSessionStateClient(t *testing.T) (*RPCClient, *atomic.Int32) {
	t.Helper()

	var logins atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var body struct {
			Params []any `json:"params"`
		}
		decodeErr := json.NewDecoder(request.Body).Decode(&body)
		if decodeErr != nil {
			t.Errorf("decode request: %v", decodeErr)

			return
		}
		if len(body.Params) == 0 {
			t.Error("request has no params")

			return
		}
		if body.Params[0] == ubusAuthSessionID {
			logins.Add(1)
			_, _ = fmt.Fprint(writer, `{"jsonrpc":"2.0","result":[0,{"ubus_rpc_session":"session","timeout":3600}]}`)

			return
		}
		_, _ = fmt.Fprint(writer, `{"jsonrpc":"2.0","result":[0]}`)
	}))
	t.Cleanup(server.Close)

	client, err := NewRPCClient(context.Background(), server.URL, "user", "password")
	if err != nil {
		t.Fatal(err)
	}

	return client, &logins
}
