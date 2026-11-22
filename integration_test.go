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
	"github.com/honeybbq/goubus/v2/services/file"
	"github.com/honeybbq/goubus/v2/services/luci"
	"github.com/honeybbq/goubus/v2/services/rc"
	"github.com/honeybbq/goubus/v2/services/session"
	"github.com/honeybbq/goubus/v2/services/uci"
)

const (
	liveFileContent = "hello goubus live integration\n"
	liveFileMD5     = "d45da1d1728750d3d1ec27b24c00d06b"
	release2410     = "24.10"
	release2512     = "25.12"
)

type liveEnv struct {
	tlsHTTPClient *http.Client
	socketPath    string
	httpURL       string
	httpsURL      string
	caFile        string
	user          string
	password      string
	release       string
}

type liveTransport struct {
	open func(t *testing.T) goubus.Transport
	name string
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
		release:    mustGetenv(t, "UBUS_TEST_RELEASE"),
	}
	if env.release != release2410 && env.release != release2512 {
		t.Fatalf("unexpected UBUS_TEST_RELEASE=%q, want %q or %q", env.release, release2410, release2512)
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
				"./scripts/openwrt-lab.sh start <24-10|25-12> <work_dir> && source <work_dir>/env.sh",
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
				cli, err := goubus.NewSocketClient(t.Context(), env.socketPath)
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
				cli, err := goubus.NewRpcClient(t.Context(), env.httpURL, env.user, env.password)
				if err != nil {
					t.Fatalf("NewRpcClient(HTTP %q): %v", env.httpURL, err)
				}
				t.Cleanup(func() { _ = cli.Close() })

				return cli
			},
		},
		{
			name: "rpc-https",
			open: func(t *testing.T) goubus.Transport {
				t.Helper()
				cli, err := goubus.NewRpcClient(
					t.Context(),
					env.httpsURL,
					env.user,
					env.password,
					goubus.WithHTTPClient(env.tlsHTTPClient),
				)
				if err != nil {
					t.Fatalf("NewRpcClient(HTTPS %q): %v", env.httpsURL, err)
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

	cli, err := goubus.NewSocketClientFromConn(ctx, rawConn)
	if err != nil {
		t.Fatalf("NewSocketClientFromConn: %v", err)
	}
	if cli.PeerID() == 0 {
		t.Errorf("PeerID() = 0, want non-zero ubus peer ID")
	}

	configs, err := uci.New(cli).Configs(ctx)
	if err != nil {
		t.Fatalf("uci.Configs over FromConn: %v", err)
	}
	if !slices.Contains(configs, "system") {
		t.Errorf("uci.Configs() = %v, missing system", configs)
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
		_, err := goubus.NewRpcClient(t.Context(), env.httpURL, env.user, env.password+"-wrong")
		if !errors.Is(err, goubus.ErrPermissionDenied) {
			t.Fatalf("HTTP wrong password error = %v, want ErrPermissionDenied", err)
		}

		_, err = goubus.NewRpcClient(
			t.Context(),
			env.httpsURL,
			env.user,
			env.password+"-wrong",
			goubus.WithHTTPClient(env.tlsHTTPClient),
		)
		if !errors.Is(err, goubus.ErrPermissionDenied) {
			t.Fatalf("HTTPS wrong password error = %v, want ErrPermissionDenied", err)
		}
	})

	t.Run("https-untrusted-ca-rejected", func(t *testing.T) {
		_, err := goubus.NewRpcClient(t.Context(), env.httpsURL, env.user, env.password)
		if !errors.Is(err, goubus.ErrConnectionFailed) {
			t.Fatalf("HTTPS without custom RootCAs error = %v, want ErrConnectionFailed", err)
		}
	})

	for _, tr := range liveTransports(env) {
		t.Run(tr.name, func(t *testing.T) {
			conn := tr.open(t)
			assertMissingObjectAndMethod(t, conn)
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

type callRecorder struct {
	methods []string
	mu      sync.Mutex
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

func TestLiveLuCIDialectAndOperations(t *testing.T) {
	env := mustLoadLiveEnv(t)

	for _, tr := range liveTransports(env) {
		t.Run(tr.name, func(t *testing.T) {
			conn := tr.open(t)
			assertLuCITimeDialect(t, conn, env.release)
			assertLuCISystemQueries(t, conn, env.release)
		})
	}
}

func assertLuCITimeDialect(t *testing.T, conn goubus.Caller, release string) {
	t.Helper()

	listed, err := goubus.List(t.Context(), conn, "luci")
	if err != nil {
		t.Fatalf("List(luci): %v", err)
	}
	luciMethods := listed["luci"]

	rec := &callRecorder{}
	cli := luci.New(goubus.ChainMiddleware(conn, rec.middleware))

	firstTime, err := cli.Time(t.Context())
	if err != nil {
		t.Fatalf("first luci.Time(): %v", err)
	}
	if delta := time.Since(firstTime).Abs(); delta > 10*time.Minute {
		t.Errorf("first luci.Time() = %v, drift %v exceeds 10m", firstTime, delta)
	}
	firstCalls := rec.take()

	secondTime, err := cli.Time(t.Context())
	if err != nil {
		t.Fatalf("second luci.Time(): %v", err)
	}
	if delta := time.Since(secondTime).Abs(); delta > 10*time.Minute {
		t.Errorf("second luci.Time() = %v, drift %v exceeds 10m", secondTime, delta)
	}
	secondCalls := rec.take()

	assertDialectCallTrace(t, release, luciMethods, firstCalls, secondCalls)
}

func assertDialectCallTrace(
	t *testing.T,
	release string,
	luciMethods goubus.ObjectSignatures,
	firstCalls, secondCalls []string,
) {
	t.Helper()

	if release == release2410 {
		if _, hasUnix := luciMethods[luci.TimeMethodUnixtime]; hasUnix {
			t.Errorf("OpenWrt 24.10 unexpectedly exposed %s", luci.TimeMethodUnixtime)
		}
		wantFirst := []string{"luci.getUnixtime", "luci.getLocaltime"}
		wantSecond := []string{"luci.getLocaltime"}
		if !reflect.DeepEqual(firstCalls, wantFirst) || !reflect.DeepEqual(secondCalls, wantSecond) {
			t.Errorf("24.10 Time() calls = %v then %v, want %v then %v", firstCalls, secondCalls, wantFirst, wantSecond)
		}

		return
	}

	if _, hasUnix := luciMethods[luci.TimeMethodUnixtime]; !hasUnix {
		t.Errorf("OpenWrt 25.12 missing %s", luci.TimeMethodUnixtime)
	}
	want := []string{"luci.getUnixtime"}
	if !reflect.DeepEqual(firstCalls, want) || !reflect.DeepEqual(secondCalls, want) {
		t.Errorf("25.12 Time() calls = %v then %v, want %v twice", firstCalls, secondCalls, want)
	}
}

func assertLuCISystemQueries(t *testing.T, conn goubus.Caller, release string) {
	t.Helper()

	cli := luci.New(conn)

	ver, err := cli.Version(t.Context())
	if err != nil {
		t.Fatalf("luci.Version(): %v", err)
	}
	if ver.Revision == "" || ver.Branch == "" {
		t.Errorf("luci.Version() = %+v, want non-empty revision and branch", ver)
	}

	features, err := cli.Features(t.Context())
	if err != nil {
		t.Fatalf("luci.Features(): %v", err)
	}
	wantAPK := release == release2512
	if bool(features["apk"].Enabled) != wantAPK {
		t.Errorf("features[apk].Enabled = %v on %s, want %v", features["apk"].Enabled, release, wantAPK)
	}
	if len(features["dnsmasq"].Subfeatures) == 0 {
		t.Errorf("features[dnsmasq].Subfeatures is empty: %+v", features["dnsmasq"])
	}

	procs, err := cli.Processes(t.Context())
	if err != nil || len(procs) == 0 {
		t.Fatalf("luci.Processes() = len %d, err %v", len(procs), err)
	}

	mounts, err := cli.MountPoints(t.Context())
	if err != nil || len(mounts) == 0 {
		t.Fatalf("luci.MountPoints() = len %d, err %v", len(mounts), err)
	}

	inits, err := cli.InitList(t.Context(), "uhttpd")
	if err != nil || inits["uhttpd"] == nil {
		t.Fatalf("luci.InitList(uhttpd) = %#v, err %v", inits, err)
	}
}

func TestLiveUCIRoundTripAndRevert(t *testing.T) {
	env := mustLoadLiveEnv(t)

	for _, tr := range liveTransports(env) {
		t.Run(tr.name, func(t *testing.T) {
			conn := tr.open(t)
			cli := uci.New(conn)
			pkg := cli.Package("system")
			t.Cleanup(func() {
				ctx, cancel := cleanupContext()
				defer cancel()
				_ = pkg.Revert(ctx)
			})

			secName := assertUCIReadSystem(t, cli, pkg)
			sec := pkg.Section(secName)
			assertUCIScalarOptionRevert(t, pkg, sec)
			assertUCIListOptionRevert(t, pkg, sec)
		})
	}
}

func assertUCIReadSystem(t *testing.T, cli *uci.Client, pkg *uci.PackageContext) string {
	t.Helper()

	configs, err := cli.Configs(t.Context())
	if err != nil {
		t.Fatalf("uci.Configs(): %v", err)
	}
	if !slices.Contains(configs, "system") {
		t.Fatalf("uci.Configs() = %v, want system", configs)
	}

	sections, err := pkg.Sections(t.Context())
	if err != nil || len(sections) == 0 {
		t.Fatalf("pkg.Sections() = %v, err %v", sections, err)
	}

	allSections, err := pkg.All(t.Context())
	if err != nil || len(allSections) == 0 {
		t.Fatalf("pkg.All() = len %d, err %v", len(allSections), err)
	}

	sysSections, err := pkg.SectionsOfType(t.Context(), "system")
	if err != nil || len(sysSections) == 0 {
		t.Fatalf("pkg.SectionsOfType(system) = %v, err %v", sysSections, err)
	}

	return sysSections[0]
}

func assertUCIScalarOptionRevert(t *testing.T, pkg *uci.PackageContext, sec *uci.SectionContext) {
	t.Helper()

	opt := sec.Option("goubus_live_opt")
	err := opt.Set(t.Context(), "ephemeral-value")
	if err != nil {
		t.Fatalf("opt.Set(): %v", err)
	}

	got, err := opt.Get(t.Context())
	if err != nil || got != "ephemeral-value" {
		t.Fatalf("opt.Get() = %q, err %v; want %q", got, err, "ephemeral-value")
	}

	changes, err := pkg.Changes(t.Context())
	if err != nil || len(changes.Changes) == 0 {
		t.Fatalf("pkg.Changes() = %+v, err %v; want staged changes", changes, err)
	}

	err = pkg.Revert(t.Context())
	if err != nil {
		t.Fatalf("pkg.Revert(): %v", err)
	}

	_, err = opt.Get(t.Context())
	if !errors.Is(err, goubus.ErrNotFound) && !errors.Is(err, goubus.ErrNoData) {
		t.Fatalf("opt.Get() after Revert error = %v, want ErrNotFound or ErrNoData", err)
	}

	afterRevert, err := pkg.Changes(t.Context())
	if err != nil || len(afterRevert.Changes) != 0 {
		t.Fatalf("pkg.Changes() after Revert = %+v, err %v; want empty", afterRevert, err)
	}
}

func assertUCIListOptionRevert(t *testing.T, pkg *uci.PackageContext, sec *uci.SectionContext) {
	t.Helper()

	listOpt := sec.Option("goubus_live_list")
	err := listOpt.AddToList(t.Context(), "alpha")
	if err != nil {
		t.Fatalf("AddToList(alpha): %v", err)
	}
	err = listOpt.AddToList(t.Context(), "beta")
	if err != nil {
		t.Fatalf("AddToList(beta): %v", err)
	}

	vals, err := listOpt.Values(t.Context())
	if err != nil || !reflect.DeepEqual(vals, []string{"alpha", "beta"}) {
		t.Fatalf("listOpt.Values() = %v, err %v; want [alpha beta]", vals, err)
	}

	err = listOpt.DeleteFromList(t.Context(), "alpha")
	if err != nil {
		t.Fatalf("DeleteFromList(alpha): %v", err)
	}

	vals, err = listOpt.Values(t.Context())
	if err != nil || !reflect.DeepEqual(vals, []string{"beta"}) {
		t.Fatalf("listOpt.Values() after delete = %v, err %v; want [beta]", vals, err)
	}

	err = pkg.Revert(t.Context())
	if err != nil {
		t.Fatalf("pkg.Revert(): %v", err)
	}

	_, err = listOpt.Values(t.Context())
	if !errors.Is(err, goubus.ErrNotFound) && !errors.Is(err, goubus.ErrNoData) {
		t.Fatalf("listOpt.Values() after Revert error = %v, want ErrNotFound or ErrNoData", err)
	}
}

func TestLiveFileLifecycle(t *testing.T) {
	env := mustLoadLiveEnv(t)

	for _, tr := range liveTransports(env) {
		t.Run(tr.name, func(t *testing.T) {
			conn := tr.open(t)
			cli := file.New(conn)
			targetPath := "/tmp/goubus-live-" + tr.name + ".txt"
			t.Cleanup(func() {
				ctx, cancel := cleanupContext()
				defer cancel()
				_ = cli.Remove(ctx, targetPath)
			})

			assertFileWriteReadStatMD5(t, cli, targetPath)
			assertFileListExecRemove(t, cli, targetPath)
		})
	}
}

func assertFileWriteReadStatMD5(t *testing.T, cli *file.Client, targetPath string) {
	t.Helper()

	err := cli.Write(t.Context(), targetPath, liveFileContent, false, 0o644, false)
	if err != nil {
		t.Fatalf("file.Write(): %v", err)
	}

	st, err := cli.Stat(t.Context(), targetPath)
	if err != nil {
		t.Fatalf("file.Stat(): %v", err)
	}
	if st.Size != uint64(len(liveFileContent)) {
		t.Errorf("file.Stat().Size = %d, want %d", st.Size, len(liveFileContent))
	}

	readRes, err := cli.Read(t.Context(), targetPath, false)
	if err != nil {
		t.Fatalf("file.Read(): %v", err)
	}
	if readRes.Data != liveFileContent {
		t.Errorf("file.Read().Data = %q, want %q", readRes.Data, liveFileContent)
	}

	digest, err := cli.MD5(t.Context(), targetPath)
	if err != nil {
		t.Fatalf("file.MD5(): %v", err)
	}
	if digest != liveFileMD5 {
		t.Errorf("file.MD5() = %q, want %q", digest, liveFileMD5)
	}
}

func assertFileListExecRemove(t *testing.T, cli *file.Client, targetPath string) {
	t.Helper()

	listing, err := cli.List(t.Context(), "/tmp")
	if err != nil {
		t.Fatalf("file.List(/tmp): %v", err)
	}
	wantBase := filepath.Base(targetPath)
	found := false
	for _, entry := range listing.Entries {
		if entry.Name == wantBase {
			found = true

			break
		}
	}
	if !found {
		t.Errorf("file.List(/tmp) did not contain %q", wantBase)
	}

	execRes, err := cli.Exec(t.Context(), "/bin/echo", []string{"goubus-exec-ok"}, nil)
	if err != nil {
		t.Fatalf("file.Exec(/bin/echo): %v", err)
	}
	if execRes.Code != 0 || execRes.Stdout != "goubus-exec-ok\n" {
		t.Errorf("file.Exec() = %+v, want code 0 stdout %q", execRes, "goubus-exec-ok\n")
	}

	err = cli.Remove(t.Context(), targetPath)
	if err != nil {
		t.Fatalf("file.Remove(): %v", err)
	}

	_, err = cli.Read(t.Context(), targetPath, false)
	if !errors.Is(err, goubus.ErrNotFound) {
		t.Fatalf("file.Read() after Remove error = %v, want ErrNotFound", err)
	}
}

func TestLiveSessionAndRC(t *testing.T) {
	env := mustLoadLiveEnv(t)

	for _, tr := range liveTransports(env) {
		t.Run(tr.name, func(t *testing.T) {
			conn := tr.open(t)
			assertLiveSessionOps(t, conn, env)
			assertLiveRCList(t, conn)
		})
	}
}

func assertLiveSessionOps(t *testing.T, conn goubus.Caller, env liveEnv) {
	t.Helper()

	cli := session.New(conn)

	listed, err := cli.List(t.Context())
	if err != nil || len(listed) == 0 {
		t.Fatalf("session.List() = %#v, err %v", listed, err)
	}

	loggedIn, err := cli.Login(t.Context(), session.LoginRequest{
		Username: env.user,
		Password: env.password,
		Timeout:  60,
	})
	if err != nil {
		t.Fatalf("session.Login(): %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := cleanupContext()
		defer cancel()
		_ = cli.Destroy(ctx, loggedIn.UbusRPCSession)
	})

	unpriv, err := cli.Create(t.Context(), 60)
	if err != nil {
		t.Fatalf("session.Create(): %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := cleanupContext()
		defer cancel()
		_ = cli.Destroy(ctx, unpriv.UbusRPCSession)
	})

	assertSessionAccessAndKV(t, cli, loggedIn.UbusRPCSession)
}

func assertSessionAccessAndKV(t *testing.T, cli *session.Client, privSID string) {
	t.Helper()

	allowed, err := cli.Access(t.Context(), session.AccessRequest{
		Session:  privSID,
		Scope:    "ubus",
		Object:   "uci",
		Function: "configs",
	})
	if err != nil || !allowed {
		t.Fatalf("session.Access(granted) = %v, err %v; want true", allowed, err)
	}

	denied, err := cli.Access(t.Context(), session.AccessRequest{
		Session:  privSID,
		Scope:    "custom-ungranted-scope",
		Object:   "uci",
		Function: "configs",
	})
	if err != nil || denied {
		t.Fatalf("session.Access(ungranted) = %v, err %v; want false", denied, err)
	}

	err = cli.Set(t.Context(), privSID, map[string]any{"marker": "goubus-live"})
	if err != nil {
		t.Fatalf("session.Set(): %v", err)
	}

	got, err := cli.Get(t.Context(), privSID, []string{"marker"})
	if err != nil {
		t.Fatalf("session.Get(): %v", err)
	}
	valuesMap, _ := got["values"].(map[string]any)
	if valuesMap["marker"] != "goubus-live" {
		t.Fatalf("session.Get() = %#v, want values.marker=goubus-live", got)
	}

	err = cli.Unset(t.Context(), privSID, []string{"marker"})
	if err != nil {
		t.Fatalf("session.Unset(): %v", err)
	}

	after, err := cli.Get(t.Context(), privSID, []string{"marker"})
	if err != nil {
		t.Fatalf("session.Get() after Unset: %v", err)
	}
	afterValues, _ := after["values"].(map[string]any)
	if _, exists := afterValues["marker"]; exists {
		t.Fatalf("session.Get() after Unset still contains marker: %#v", after)
	}
}

func assertLiveRCList(t *testing.T, conn goubus.Caller) {
	t.Helper()

	scripts, err := rc.New(conn).List(t.Context(), "", false)
	if err != nil {
		t.Fatalf("rc.List(): %v", err)
	}
	if len(scripts) == 0 {
		t.Fatalf("rc.List() returned empty map")
	}
	if _, ok := scripts["uhttpd"]; !ok {
		t.Errorf("rc.List() missing uhttpd init script: %#v", scripts)
	}
}
