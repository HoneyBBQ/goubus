//go:build integration

package goubus_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/rpc"
)

const (
	liveFileContent = "hello goubus live integration\n"
	liveFileMD5     = "d45da1d1728750d3d1ec27b24c00d06b"
)

var errConcurrentLiveCall = errors.New("concurrent live socket call failed")

type liveEnv struct {
	tlsHTTPClient *http.Client
	socketPath    string
	httpURL       string
	httpsURL      string
	caFile        string
	user          string
	password      string
}

type liveTransport struct {
	open func(t *testing.T) goubus.Transport
	name string
}

type uciConfigsResp struct {
	Configs []string `json:"configs"`
}

type luciTimeResp struct {
	Result int64 `json:"result"`
}

type luciProcessEntry struct {
	User string `json:"USER"`
}

type luciProcessesResp struct {
	Result []luciProcessEntry `json:"result"`
}

type rcScriptState struct {
	Start   int  `json:"start"`
	Enabled bool `json:"enabled"`
	Running bool `json:"running"`
}

type fileStatResp struct {
	Type string `json:"type"`
	Size uint64 `json:"size"`
}

type fileReadResp struct {
	Data string `json:"data"`
}

type fileMD5Resp struct {
	MD5 string `json:"md5"`
}

type fileListEntry struct {
	Name string `json:"name"`
}

type fileListResp struct {
	Entries []fileListEntry `json:"entries"`
}

type fileExecResp struct {
	Stdout string `json:"stdout"`
	Code   int    `json:"code"`
}

type sessionCreateResp struct {
	UbusRPCSession string `json:"ubus_rpc_session"`
}

type sessionAccessResp struct {
	Access bool `json:"access"`
}

type sessionValuesResp struct {
	Values map[string]any `json:"values"`
}

type callRecorder struct {
	methods []string
	mu      sync.Mutex
}

func mustLoadLiveEnv(t *testing.T) liveEnv {
	t.Helper()

	env := liveEnv{
		socketPath: mustGetenv(t, "UBUS_TEST_SOCKET"),
		httpURL:    mustGetenv(t, "UBUS_TEST_HTTP_URL"),
		httpsURL:   mustGetenv(t, "UBUS_TEST_HTTPS_URL"),
		caFile:     mustGetenv(t, "UBUS_TEST_CA_FILE"),
		user:       mustGetenv(t, "UBUS_TEST_USER"),
		password:   mustGetenv(t, "UBUS_TEST_PASSWORD"),
	}
	env.tlsHTTPClient = mustBuildLiveTLSClient(t, env.caFile)

	return env
}

func mustGetenv(t *testing.T, key string) string {
	t.Helper()

	val := os.Getenv(key)
	if val == "" {
		t.Fatalf(
			"missing required environment variable %s; start a lab first with: "+
				"./scripts/openwrt-lab.sh start <21-02|22-03|23-05|24-10|25-12> <work_dir> && source <work_dir>/env.sh",
			key,
		)
	}

	return val
}

func mustBuildLiveTLSClient(t *testing.T, caFile string) *http.Client {
	t.Helper()

	root, err := os.OpenRoot(filepath.Dir(caFile))
	if err != nil {
		t.Fatalf("open CA dir: %v", err)
	}
	caPEM, readErr := root.ReadFile(filepath.Base(caFile))
	closeErr := root.Close()
	err = errors.Join(readErr, closeErr)
	if err != nil {
		t.Fatalf("read CA file %s: %v", caFile, err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		t.Fatalf("append CA PEM from %s failed", caFile)
	}

	return &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:    pool,
				MinVersion: tls.VersionTLS12,
			},
		},
	}
}

func liveTransports(env liveEnv) []liveTransport {
	return []liveTransport{
		{
			name: "socket",
			open: func(t *testing.T) goubus.Transport {
				t.Helper()
				cli, err := goubus.NewClient(t.Context(), env.socketPath)
				if err != nil {
					t.Fatalf("NewSocketClient(%q): %v", env.socketPath, err)
				}
				t.Cleanup(func() { _ = cli.Close() })

				return cli
			},
		},
		{
			name: "rpc-http",
			open: func(t *testing.T) goubus.Transport {
				t.Helper()
				cli, err := rpc.NewClient(t.Context(), env.httpURL, env.user, env.password)
				if err != nil {
					t.Fatalf("NewRPCClient(HTTP %q): %v", env.httpURL, err)
				}
				t.Cleanup(func() { _ = cli.Close() })

				return cli
			},
		},
		{
			name: "rpc-https",
			open: func(t *testing.T) goubus.Transport {
				t.Helper()
				cli, err := rpc.NewClient(
					t.Context(),
					env.httpsURL,
					env.user,
					env.password,
					rpc.WithHTTPClient(env.tlsHTTPClient),
				)
				if err != nil {
					t.Fatalf("NewRPCClient(HTTPS %q): %v", env.httpsURL, err)
				}
				t.Cleanup(func() { _ = cli.Close() })

				return cli
			},
		},
	}
}

func cleanupContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

func TestLiveSocketClientFromConn(t *testing.T) {
	env := mustLoadLiveEnv(t)
	ctx := t.Context()

	dialer := net.Dialer{Timeout: 5 * time.Second}
	rawConn, err := dialer.DialContext(ctx, "unix", env.socketPath)
	if err != nil {
		t.Fatalf("dial unix %s: %v", env.socketPath, err)
	}

	cli, err := goubus.NewClientFromConn(ctx, rawConn)
	if err != nil {
		t.Fatalf("NewSocketClientFromConn: %v", err)
	}
	if cli.PeerID() == 0 {
		t.Errorf("PeerID() = 0, want non-zero ubus peer ID")
	}
	err = cli.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping(): %v", err)
	}

	resp, err := goubus.Call[uciConfigsResp](ctx, cli, "uci", "configs", nil)
	if err != nil {
		t.Fatalf("uci.configs over FromConn: %v", err)
	}
	if !slices.Contains(resp.Configs, "system") {
		t.Errorf("uci.configs = %v, missing system", resp.Configs)
	}

	err = cli.Close()
	if err != nil {
		t.Fatalf("Close(): %v", err)
	}
	_, err = cli.Call(ctx, "uci", "configs", nil)
	if !errors.Is(err, goubus.ErrClosed) {
		t.Errorf("Call after Close error = %v, want ErrClosed", err)
	}
}

func TestLiveAuthAndErrorSemantics(t *testing.T) {
	env := mustLoadLiveEnv(t)

	t.Run("rpc-wrong-password", func(t *testing.T) {
		_, err := rpc.NewClient(t.Context(), env.httpURL, env.user, env.password+"-wrong")
		if !errors.Is(err, goubus.ErrPermissionDenied) {
			t.Fatalf("HTTP wrong password error = %v, want ErrPermissionDenied", err)
		}

		_, err = rpc.NewClient(
			t.Context(),
			env.httpsURL,
			env.user,
			env.password+"-wrong",
			rpc.WithHTTPClient(env.tlsHTTPClient),
		)
		if !errors.Is(err, goubus.ErrPermissionDenied) {
			t.Fatalf("HTTPS wrong password error = %v, want ErrPermissionDenied", err)
		}
	})

	t.Run("rpc-reauthenticate", func(t *testing.T) {
		cli, err := rpc.NewClient(t.Context(), env.httpURL, env.user, env.password)
		if err != nil {
			t.Fatalf("NewRPCClient: %v", err)
		}
		t.Cleanup(func() { _ = cli.Close() })

		err = cli.Authenticate(t.Context())
		if err != nil {
			t.Fatalf("Authenticate(): %v", err)
		}
		resp, err := goubus.Call[uciConfigsResp](t.Context(), cli, "uci", "configs", nil)
		if err != nil || !slices.Contains(resp.Configs, "system") {
			t.Fatalf("Call after Authenticate() = %+v, err %v", resp, err)
		}
	})

	t.Run("https-untrusted-ca-rejected", func(t *testing.T) {
		_, err := rpc.NewClient(t.Context(), env.httpsURL, env.user, env.password)
		if !errors.Is(err, goubus.ErrConnectionFailed) {
			t.Fatalf("HTTPS without custom RootCAs error = %v, want ErrConnectionFailed", err)
		}
	})

	for _, tr := range liveTransports(env) {
		t.Run(tr.name, func(t *testing.T) {
			assertMissingObjectAndMethod(t, tr.open(t))
		})
	}
}

func assertMissingObjectAndMethod(t *testing.T, conn goubus.Caller) {
	t.Helper()

	_, err := conn.Call(t.Context(), "goubus.nonexistent.object", "ping", nil)
	if !errors.Is(err, goubus.ErrNotFound) {
		t.Fatalf("missing object error = %v, want ErrNotFound", err)
	}
	callErr, ok := errors.AsType[*goubus.CallError](err)
	if !ok {
		t.Fatalf("missing object error type = %T, want *goubus.CallError", err)
	}
	if !errors.Is(callErr, goubus.ErrCall) {
		t.Errorf("CallError %v does not match ErrCall", callErr)
	}

	_, err = conn.Call(t.Context(), "uci", "goubus_nonexistent_method", nil)
	if !errors.Is(err, goubus.ErrMethodNotFound) {
		t.Fatalf("missing method error = %v, want ErrMethodNotFound", err)
	}
	if _, isCallErr := errors.AsType[*goubus.CallError](err); !isCallErr {
		t.Fatalf("missing method error type = %T, want *goubus.CallError", err)
	}
}

func TestLiveListIntrospectionConsistency(t *testing.T) {
	env := mustLoadLiveEnv(t)

	targets := []string{"session", "uci", "file", "luci"}
	results := make(map[string]map[string]goubus.ObjectSignatures)

	for _, tr := range liveTransports(env) {
		t.Run(tr.name, func(t *testing.T) {
			conn := tr.open(t)
			listed, err := goubus.List(t.Context(), conn, targets...)
			if err != nil {
				t.Fatalf("List(%v): %v", targets, err)
			}
			assertCoreObjectSignatures(t, listed)
			results[tr.name] = listed
		})
	}

	if !reflect.DeepEqual(results["socket"], results["rpc-http"]) {
		t.Errorf("socket vs rpc-http List mismatch:\nsocket=%#v\nhttp=%#v", results["socket"], results["rpc-http"])
	}
	if !reflect.DeepEqual(results["socket"], results["rpc-https"]) {
		t.Errorf("socket vs rpc-https List mismatch:\nsocket=%#v\nhttps=%#v", results["socket"], results["rpc-https"])
	}
}

func assertCoreObjectSignatures(t *testing.T, listed map[string]goubus.ObjectSignatures) {
	t.Helper()

	for _, obj := range []string{"session", "uci", "file", "luci"} {
		if _, ok := listed[obj]; !ok {
			t.Fatalf("List missing core object %q: %#v", obj, listed)
		}
	}

	assertArgType(t, listed, "session", "login", "username", goubus.ArgTypeString)
	assertArgType(t, listed, "session", "login", "timeout", goubus.ArgTypeNumber)
	assertArgType(t, listed, "uci", "apply", "rollback", goubus.ArgTypeBoolean)
	assertArgType(t, listed, "uci", "apply", "timeout", goubus.ArgTypeNumber)
	assertArgType(t, listed, "file", "write", "path", goubus.ArgTypeString)
	assertArgType(t, listed, "file", "write", "append", goubus.ArgTypeBoolean)
	assertArgType(t, listed, "file", "write", "mode", goubus.ArgTypeNumber)
	assertArgType(t, listed, "file", "exec", "params", goubus.ArgTypeArray)
	assertArgType(t, listed, "file", "exec", "env", goubus.ArgTypeObject)
	assertArgType(t, listed, "luci", "getInitList", "name", goubus.ArgTypeString)
}

func assertArgType(
	t *testing.T,
	listed map[string]goubus.ObjectSignatures,
	objectName, methodName, argName string,
	want goubus.ArgType,
) {
	t.Helper()

	methodSig, ok := listed[objectName][methodName]
	if !ok {
		t.Fatalf("object %q missing method %q: %#v", objectName, methodName, listed[objectName])
	}
	if got := methodSig[argName]; got != want {
		t.Errorf("%s.%s(%s) = %q, want %q", objectName, methodName, argName, got, want)
	}
}

func (r *callRecorder) middleware(next goubus.Caller) goubus.Caller {
	return goubus.CallerFunc(func(ctx context.Context, service, method string, data any) (goubus.Result, error) {
		r.mu.Lock()
		r.methods = append(r.methods, service+"."+method)
		r.mu.Unlock()

		return next.Call(ctx, service, method, data)
	})
}

func (r *callRecorder) take() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := append([]string(nil), r.methods...)
	r.methods = nil

	return out
}

func TestLiveMiddlewareAndFlexDecoding(t *testing.T) {
	env := mustLoadLiveEnv(t)

	for _, tr := range liveTransports(env) {
		t.Run(tr.name, func(t *testing.T) {
			conn := tr.open(t)
			listed, err := goubus.List(t.Context(), conn, "luci")
			if err != nil {
				t.Fatalf("List(luci): %v", err)
			}

			rec := &callRecorder{}
			caller := goubus.ChainMiddleware(conn, rec.middleware)

			timeMethod := "getLocaltime"
			if _, ok := listed["luci"]["getUnixtime"]; ok {
				timeMethod = "getUnixtime"
			}
			timeResp, err := goubus.Call[luciTimeResp](t.Context(), caller, "luci", timeMethod, nil)
			if err != nil {
				t.Fatalf("luci.%s: %v", timeMethod, err)
			}
			if delta := time.Since(time.Unix(timeResp.Result, 0)).Abs(); delta > 10*time.Minute {
				t.Errorf("luci.%s = %d, drift %v exceeds 10m", timeMethod, timeResp.Result, delta)
			}

			procs, err := goubus.Call[luciProcessesResp](t.Context(), caller, "luci", "getProcessList", nil)
			if err != nil || len(procs.Result) == 0 || procs.Result[0].User == "" {
				t.Fatalf("luci.getProcessList = %+v, err %v", procs, err)
			}

			scripts, err := goubus.Call[map[string]rcScriptState](t.Context(), caller, "rc", "list", nil)
			if err != nil || !scripts["uhttpd"].Enabled {
				t.Fatalf("rc.list[uhttpd] = %+v, err %v", scripts["uhttpd"], err)
			}

			wantCalls := []string{"luci." + timeMethod, "luci.getProcessList", "rc.list"}
			if got := rec.take(); !reflect.DeepEqual(got, wantCalls) {
				t.Errorf("recorded calls = %v, want %v", got, wantCalls)
			}
		})
	}
}

func TestLiveUCIWireRoundTripAndRevert(t *testing.T) {
	env := mustLoadLiveEnv(t)

	for _, tr := range liveTransports(env) {
		t.Run(tr.name, func(t *testing.T) {
			conn := tr.open(t)
			t.Cleanup(func() {
				ctx, cancel := cleanupContext()
				defer cancel()
				_, _ = conn.Call(ctx, "uci", "revert", map[string]any{"config": "system"})
			})

			assertUCIScalarAndListRoundTrip(t, conn)
		})
	}
}

func assertUCIScalarAndListRoundTrip(t *testing.T, conn goubus.Caller) {
	t.Helper()

	secName := firstUCISystemSection(t, conn)

	_, err := conn.Call(t.Context(), "uci", "set", map[string]any{
		"config":  "system",
		"section": secName,
		"values": map[string]any{
			"goubus_live_opt":  "ephemeral-value",
			"goubus_live_list": []string{"alpha", "beta"},
		},
	})
	if err != nil {
		t.Fatalf("uci.set: %v", err)
	}

	optResp, err := goubus.Call[struct {
		Value string `json:"value"`
	}](t.Context(), conn, "uci", "get", map[string]any{
		"config":  "system",
		"section": secName,
		"option":  "goubus_live_opt",
	})
	if err != nil || optResp.Value != "ephemeral-value" {
		t.Fatalf("uci.get scalar = %+v, err %v", optResp, err)
	}

	listResp, err := goubus.Call[struct {
		Value []string `json:"value"`
	}](t.Context(), conn, "uci", "get", map[string]any{
		"config":  "system",
		"section": secName,
		"option":  "goubus_live_list",
	})
	if err != nil || !reflect.DeepEqual(listResp.Value, []string{"alpha", "beta"}) {
		t.Fatalf("uci.get list = %+v, err %v", listResp, err)
	}

	_, err = conn.Call(t.Context(), "uci", "revert", map[string]any{"config": "system"})
	if err != nil {
		t.Fatalf("uci.revert: %v", err)
	}

	_, err = goubus.Call[map[string]any](t.Context(), conn, "uci", "get", map[string]any{
		"config":  "system",
		"section": secName,
		"option":  "goubus_live_opt",
	})
	if !errors.Is(err, goubus.ErrNotFound) && !errors.Is(err, goubus.ErrNoData) {
		t.Fatalf("uci.get after revert error = %v, want ErrNotFound or ErrNoData", err)
	}
}

func firstUCISystemSection(t *testing.T, conn goubus.Caller) string {
	t.Helper()

	sections, err := goubus.Call[struct {
		Values map[string]map[string]any `json:"values"`
	}](t.Context(), conn, "uci", "get", map[string]any{
		"config": "system",
		"type":   "system",
	})
	if err != nil || len(sections.Values) == 0 {
		t.Fatalf("uci.get(system): %+v, err %v", sections, err)
	}
	for name := range sections.Values {
		return name
	}

	return ""
}

func TestLiveFileWireRoundTrip(t *testing.T) {
	env := mustLoadLiveEnv(t)

	for _, tr := range liveTransports(env) {
		t.Run(tr.name, func(t *testing.T) {
			conn := tr.open(t)
			targetPath := "/tmp/goubus-live-" + tr.name + ".txt"
			t.Cleanup(func() {
				ctx, cancel := cleanupContext()
				defer cancel()
				_, _ = conn.Call(ctx, "file", "remove", map[string]any{"path": targetPath})
			})

			assertFileWriteReadStatMD5(t, conn, targetPath)
			assertFileListExecRemove(t, conn, targetPath)
		})
	}
}

func assertFileWriteReadStatMD5(t *testing.T, conn goubus.Caller, targetPath string) {
	t.Helper()

	_, err := conn.Call(t.Context(), "file", "write", map[string]any{
		"path":   targetPath,
		"data":   liveFileContent,
		"append": false,
		"mode":   0o644,
	})
	if err != nil {
		t.Fatalf("file.write: %v", err)
	}

	st, err := goubus.Call[fileStatResp](t.Context(), conn, "file", "stat", map[string]any{"path": targetPath})
	if err != nil || st.Size != uint64(len(liveFileContent)) || st.Type != "file" {
		t.Fatalf("file.stat = %+v, err %v", st, err)
	}

	readRes, err := goubus.Call[fileReadResp](t.Context(), conn, "file", "read", map[string]any{"path": targetPath})
	if err != nil || readRes.Data != liveFileContent {
		t.Fatalf("file.read = %+v, err %v", readRes, err)
	}

	md5Res, err := goubus.Call[fileMD5Resp](t.Context(), conn, "file", "md5", map[string]any{"path": targetPath})
	if err != nil || md5Res.MD5 != liveFileMD5 {
		t.Fatalf("file.md5 = %+v, err %v", md5Res, err)
	}
}

func assertFileListExecRemove(t *testing.T, conn goubus.Caller, targetPath string) {
	t.Helper()

	listing, err := goubus.Call[fileListResp](t.Context(), conn, "file", "list", map[string]any{"path": "/tmp"})
	if err != nil {
		t.Fatalf("file.list(/tmp): %v", err)
	}
	wantBase := filepath.Base(targetPath)
	found := slices.ContainsFunc(listing.Entries, func(entry fileListEntry) bool {
		return entry.Name == wantBase
	})
	if !found {
		t.Errorf("file.list(/tmp) did not contain %q", wantBase)
	}

	execRes, err := goubus.Call[fileExecResp](t.Context(), conn, "file", "exec", map[string]any{
		"command": "/bin/echo",
		"params":  []string{"goubus-exec-ok"},
	})
	if err != nil || execRes.Code != 0 || execRes.Stdout != "goubus-exec-ok\n" {
		t.Fatalf("file.exec = %+v, err %v", execRes, err)
	}

	_, err = conn.Call(t.Context(), "file", "remove", map[string]any{"path": targetPath})
	if err != nil {
		t.Fatalf("file.remove: %v", err)
	}

	_, err = conn.Call(t.Context(), "file", "read", map[string]any{"path": targetPath})
	if !errors.Is(err, goubus.ErrNotFound) {
		t.Fatalf("file.read after remove error = %v, want ErrNotFound", err)
	}
}

func TestLiveExplicitSessionHoisting(t *testing.T) {
	env := mustLoadLiveEnv(t)

	for _, tr := range liveTransports(env) {
		t.Run(tr.name, func(t *testing.T) {
			conn := tr.open(t)
			assertLiveSessionHoisting(t, conn, env)
		})
	}
}

func assertLiveSessionHoisting(t *testing.T, conn goubus.Caller, env liveEnv) {
	t.Helper()

	loggedIn, err := goubus.Call[sessionCreateResp](t.Context(), conn, "session", "login", map[string]any{
		"username": env.user,
		"password": env.password,
		"timeout":  60,
	})
	if err != nil || loggedIn.UbusRPCSession == "" {
		t.Fatalf("session.login = %+v, err %v", loggedIn, err)
	}
	t.Cleanup(func() {
		ctx, cancel := cleanupContext()
		defer cancel()
		_, _ = conn.Call(ctx, "session", "destroy", map[string]any{"ubus_rpc_session": loggedIn.UbusRPCSession})
	})

	assertSessionAccessAndValues(t, conn, loggedIn.UbusRPCSession)
}

func assertSessionAccessAndValues(t *testing.T, conn goubus.Caller, sid string) {
	t.Helper()

	allowed, err := goubus.Call[sessionAccessResp](t.Context(), conn, "session", "access", map[string]any{
		"ubus_rpc_session": sid,
		"scope":            "ubus",
		"object":           "uci",
		"function":         "configs",
	})
	if err != nil || !allowed.Access {
		t.Fatalf("session.access(granted) = %+v, err %v", allowed, err)
	}

	_, err = conn.Call(t.Context(), "session", "set", map[string]any{
		"ubus_rpc_session": sid,
		"values":           map[string]any{"marker": "goubus-live"},
	})
	if err != nil {
		t.Fatalf("session.set: %v", err)
	}

	got, err := goubus.Call[sessionValuesResp](t.Context(), conn, "session", "get", map[string]any{
		"ubus_rpc_session": sid,
		"keys":             []string{"marker"},
	})
	if err != nil || got.Values["marker"] != "goubus-live" {
		t.Fatalf("session.get = %+v, err %v", got, err)
	}

	_, err = conn.Call(t.Context(), "session", "unset", map[string]any{
		"ubus_rpc_session": sid,
		"keys":             []string{"marker"},
	})
	if err != nil {
		t.Fatalf("session.unset: %v", err)
	}
}

func TestLiveSocketMultiplexConcurrency(t *testing.T) {
	env := mustLoadLiveEnv(t)
	sock, err := goubus.NewClient(t.Context(), env.socketPath)
	if err != nil {
		t.Fatalf("NewSocketClient: %v", err)
	}
	t.Cleanup(func() { _ = sock.Close() })

	const workers = 16
	var wg sync.WaitGroup
	errCh := make(chan error, workers)

	for i := range workers {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if idx%2 == 0 {
				resp, callErr := goubus.Call[uciConfigsResp](t.Context(), sock, "uci", "configs", nil)
				if callErr != nil || !slices.Contains(resp.Configs, "system") {
					errCh <- errConcurrentLiveCall
				}

				return
			}
			scripts, callErr := goubus.Call[map[string]rcScriptState](t.Context(), sock, "rc", "list", nil)
			if callErr != nil || !scripts["uhttpd"].Enabled {
				errCh <- errConcurrentLiveCall
			}
		}(i)
	}

	wg.Wait()
	close(errCh)
	for workerErr := range errCh {
		t.Error(workerErr)
	}
}

func TestLiveEventListenAndSend(t *testing.T) {
	env := mustLoadLiveEnv(t)

	listenerSock, err := goubus.NewClient(t.Context(), env.socketPath)
	if err != nil {
		t.Fatalf("listener NewSocketClient: %v", err)
	}
	t.Cleanup(func() { _ = listenerSock.Close() })

	senderSock, err := goubus.NewClient(t.Context(), env.socketPath)
	if err != nil {
		t.Fatalf("sender NewSocketClient: %v", err)
	}
	t.Cleanup(func() { _ = senderSock.Close() })

	events := make(chan goubus.Event, 1)
	sub, err := listenerSock.Listen(t.Context(), "goubus.live.*", func(ev goubus.Event) {
		events <- ev
	})
	if err != nil {
		t.Fatalf("Listen(): %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	err = senderSock.SendEvent(t.Context(), "goubus.live.ping", map[string]any{"seq": 99, "ok": true})
	if err != nil {
		t.Fatalf("SendEvent(): %v", err)
	}

	select {
	case ev := <-events:
		var data struct {
			Seq int  `json:"seq"`
			OK  bool `json:"ok"`
		}
		unmarshalErr := ev.Unmarshal(&data)
		if unmarshalErr != nil {
			t.Fatalf("Event.Unmarshal: %v", unmarshalErr)
		}
		if ev.Type != "goubus.live.ping" || data.Seq != 99 || !data.OK {
			t.Fatalf("received event = (%q, %+v)", ev.Type, data)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for live ubus event")
	}
}

func TestLiveObjectPublishSubscribeAndRemoteRPCCall(t *testing.T) {
	env := mustLoadLiveEnv(t)

	providerSock, err := goubus.NewClient(t.Context(), env.socketPath)
	if err != nil {
		t.Fatalf("provider NewSocketClient: %v", err)
	}
	t.Cleanup(func() { _ = providerSock.Close() })

	var pub *goubus.RegisteredObject
	pub, err = registerLiveDemoObject(t, providerSock, func(msg string) {
		_ = pub.Notify(context.Background(), "echoed", map[string]any{"msg": msg})
	})
	if err != nil {
		t.Fatalf("AddObject(): %v", err)
	}
	t.Cleanup(func() { _ = pub.Close() })

	assertLiveDemoInteractions(t, env, pub)
}

func registerLiveDemoObject(
	t *testing.T,
	providerSock *goubus.Client,
	onEcho func(msg string),
) (*goubus.RegisteredObject, error) {
	t.Helper()

	return providerSock.AddObject(t.Context(), goubus.ObjectDef{
		Name: "goubus.live.demo",
		Methods: map[string]goubus.Method{
			"echo": {
				Policy: goubus.MethodSignature{
					"msg": goubus.ArgTypeString,
					"n":   goubus.ArgTypeNumber,
				},
				Handler: func(_ context.Context, req goubus.CallRequest) (any, error) {
					var in struct {
						Msg string `json:"msg"`
						N   int    `json:"n"`
					}
					decodeErr := req.Unmarshal(&in)
					if decodeErr != nil {
						return nil, goubus.ErrInvalidParameter
					}
					onEcho(in.Msg)

					return map[string]any{"reply": in.Msg, "doubled": in.N * 2}, nil
				},
			},
		},
	})
}

func assertLiveDemoInteractions(t *testing.T, env liveEnv, pub *goubus.RegisteredObject) {
	t.Helper()

	subSock, err := goubus.NewClient(t.Context(), env.socketPath)
	if err != nil {
		t.Fatalf("subscriber NewSocketClient: %v", err)
	}
	t.Cleanup(func() { _ = subSock.Close() })

	notes := make(chan goubus.Notification, 1)
	sub, err := subSock.Subscribe(t.Context(), "goubus.live.demo", func(n goubus.Notification) {
		notes <- n
	})
	if err != nil {
		t.Fatalf("Subscribe(): %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	rpcCli, err := rpc.NewClient(t.Context(), env.httpURL, env.user, env.password)
	if err != nil {
		t.Fatalf("NewRPCClient: %v", err)
	}
	t.Cleanup(func() { _ = rpcCli.Close() })

	listed, err := goubus.List(t.Context(), rpcCli, "goubus.live.demo")
	if err != nil {
		t.Fatalf("List(goubus.live.demo) over HTTP RPC: %v", err)
	}
	assertArgType(t, listed, "goubus.live.demo", "echo", "msg", goubus.ArgTypeString)
	assertArgType(t, listed, "goubus.live.demo", "echo", "n", goubus.ArgTypeNumber)

	out, err := goubus.Call[struct {
		Reply   string `json:"reply"`
		Doubled int    `json:"doubled"`
	}](t.Context(), rpcCli, "goubus.live.demo", "echo", map[string]any{"msg": "from-http", "n": 21})
	if err != nil || out.Reply != "from-http" || out.Doubled != 42 {
		t.Fatalf("RPC call to Go object = %+v, err %v", out, err)
	}

	// Warm subSock's local objectID cache for goubus.live.demo before unregistering it.
	_, err = subSock.Call(t.Context(), "goubus.live.demo", "echo", map[string]any{"msg": "warm", "n": 1})
	if err != nil {
		t.Fatalf("socket Call to Go object: %v", err)
	}

	assertLiveNotificationAndUnregister(t, notes, pub, sub)
	assertLiveObjectRestartRecovery(t, env, subSock)
}

func assertLiveNotificationAndUnregister(
	t *testing.T,
	notes <-chan goubus.Notification,
	pub *goubus.RegisteredObject,
	sub *goubus.Subscription,
) {
	t.Helper()

	select {
	case note := <-notes:
		var payload struct {
			Msg string `json:"msg"`
		}
		unmarshalErr := note.Unmarshal(&payload)
		if unmarshalErr != nil || (payload.Msg != "from-http" && payload.Msg != "warm") {
			t.Fatalf("Notification = %+v (%+v), err %v", note, payload, unmarshalErr)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for object notification")
	}

	closeErr := pub.Close()
	if closeErr != nil {
		t.Fatalf("pub.Close(): %v", closeErr)
	}
	select {
	case <-sub.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("sub.Done() did not close after published object was removed")
	}
}

func assertLiveObjectRestartRecovery(t *testing.T, env liveEnv, existingCaller *goubus.Client) {
	t.Helper()

	pubConn, err := goubus.NewClient(t.Context(), env.socketPath)
	if err != nil {
		t.Fatalf("re-publish NewClient: %v", err)
	}
	t.Cleanup(func() { _ = pubConn.Close() })

	pub2, err := registerLiveDemoObject(t, pubConn, func(string) {})
	if err != nil {
		t.Fatalf("re-publish AddObject: %v", err)
	}
	t.Cleanup(func() { _ = pub2.Close() })

	// existingCaller still holds the old objid in objectCache; Call must refresh it seamlessly.
	out, err := goubus.Call[struct {
		Reply   string `json:"reply"`
		Doubled int    `json:"doubled"`
	}](t.Context(), existingCaller, "goubus.live.demo", "echo", map[string]any{"msg": "restarted", "n": 5})
	if err != nil || out.Reply != "restarted" || out.Doubled != 10 {
		t.Fatalf("Call after live object restart = %+v, err %v", out, err)
	}
}

func BenchmarkLiveTransports(b *testing.B) {
	socketPath := os.Getenv("UBUS_TEST_SOCKET")
	httpURL := os.Getenv("UBUS_TEST_HTTP_URL")
	httpsURL := os.Getenv("UBUS_TEST_HTTPS_URL")
	caFile := os.Getenv("UBUS_TEST_CA_FILE")
	user := os.Getenv("UBUS_TEST_USER")
	password := os.Getenv("UBUS_TEST_PASSWORD")
	if socketPath == "" || httpURL == "" || httpsURL == "" || caFile == "" || user == "" || password == "" {
		b.Skip("live OpenWrt environment variables are not set")
	}

	b.Run("socket", func(b *testing.B) {
		conn, err := goubus.NewClient(b.Context(), socketPath)
		if err != nil {
			b.Fatal(err)
		}
		b.Cleanup(func() { _ = conn.Close() })
		benchmarkLiveUCI(b, conn)
	})

	b.Run("rpc-http", func(b *testing.B) {
		conn, err := rpc.NewClient(b.Context(), httpURL, user, password)
		if err != nil {
			b.Fatal(err)
		}
		b.Cleanup(func() { _ = conn.Close() })
		benchmarkLiveUCI(b, conn)
	})
}

func benchmarkLiveUCI(b *testing.B, caller goubus.Caller) {
	b.Helper()
	b.ReportAllocs()

	for b.Loop() {
		_, err := goubus.Call[uciConfigsResp](b.Context(), caller, "uci", "configs", nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}
