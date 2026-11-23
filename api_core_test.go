package goubus_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/honeybbq/goubus/v2"
)

func TestCallErrorUnwrapTree(t *testing.T) {
	callErr := &goubus.CallError{
		Err: goubus.ErrNotFound, Service: "system", Method: "info", Message: "missing", Code: goubus.StatusNotFound,
	}
	if got, want := callErr.Error(), "ubus call system.info failed (code 4: missing): not found"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	if !errors.Is(callErr, goubus.ErrCall) || !errors.Is(callErr, goubus.ErrNotFound) {
		t.Errorf("Unwrap() error tree does not contain both sentinels: %v", callErr)
	}
	got := callErr.Unwrap()
	if len(got) != 2 || !errors.Is(got[0], goubus.ErrCall) || !errors.Is(got[1], goubus.ErrNotFound) {
		t.Errorf("Unwrap() = %#v, want ErrCall and ErrNotFound", got)
	}
	got = (&goubus.CallError{Code: 99}).Unwrap()
	if len(got) != 1 || !errors.Is(got[0], goubus.ErrCall) {
		t.Errorf("Unwrap() without cause = %#v, want ErrCall", got)
	}
	if got := (&goubus.CallError{Method: "list", Code: 1}).Error(); got != "ubus list failed (code 1): <nil>" {
		t.Errorf("Error() without service = %q", got)
	}
}

func TestCallHelpers(t *testing.T) {
	var order []string
	base := goubus.CallerFunc(func(context.Context, string, string, any) (goubus.Result, error) {
		order = append(order, "call")

		return goubus.NewJSONResult(json.RawMessage(`{"ok":true}`)), nil
	})
	wrap := func(name string) goubus.CallMiddleware {
		return func(next goubus.Caller) goubus.Caller {
			return goubus.CallerFunc(func(ctx context.Context, service, method string, data any) (goubus.Result, error) {
				order = append(order, "before-"+name)
				result, err := next.Call(ctx, service, method, data)
				order = append(order, "after-"+name)

				return result, err
			})
		}
	}

	got, err := goubus.Call[struct {
		OK bool `json:"ok"`
	}](context.Background(), goubus.ChainMiddleware(base, wrap("one"), wrap("two")), "svc", "method", nil)
	if err != nil || !got.OK {
		t.Fatalf("Call() = %#v, %v", got, err)
	}
	wantOrder := []string{"before-one", "before-two", "call", "after-two", "after-one"}
	if !reflect.DeepEqual(order, wantOrder) {
		t.Errorf("middleware order = %#v, want %#v", order, wantOrder)
	}
}

func TestNewJSONResultNoDataAndInvalid(t *testing.T) {
	for _, raw := range []json.RawMessage{nil, []byte(" "), []byte("null")} {
		err := goubus.NewJSONResult(raw).Unmarshal(new(any))
		if !errors.Is(err, goubus.ErrNoData) {
			t.Errorf("Unmarshal(%q) error = %v, want ErrNoData", raw, err)
		}
	}
	err := goubus.NewJSONResult([]byte("{")).Unmarshal(new(any))
	if !errors.Is(err, goubus.ErrInvalidResponse) {
		t.Errorf("invalid JSON result error = %v, want ErrInvalidResponse", err)
	}
}

func TestVersionConstant(t *testing.T) {
	semver := regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`)
	if !semver.MatchString(goubus.Version) {
		t.Fatalf("Version = %q, want valid semver without leading v", goubus.Version)
	}
	raw, err := os.ReadFile("version.go")
	if err != nil {
		t.Fatalf("read version.go: %v", err)
	}
	wantLine := `const Version = "` + goubus.Version + `"`
	if !strings.Contains(string(raw), wantLine) {
		t.Fatalf("version.go missing exact line %q", wantLine)
	}
}

func TestReleaseNotesScriptLinkRewriting(t *testing.T) {
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not available")
	}
	scriptPath := prepareReleaseNotesRepo(t)
	cmd := exec.CommandContext(t.Context(), bashPath, scriptPath, "v2.0.0-beta.1") //nolint:gosec // Temp test script.
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("release-notes.sh failed: %v\n%s", err, out)
	}
	got := string(out)
	for _, want := range []string{
		"go get github.com/honeybbq/goubus/v2@v2.0.0-beta.1",
		"`goubus.Call[T](ctx, caller, ...)`",
		"`errors.AsType[*goubus.CallError](err)`",
		"`` `nested[T](arg)` ``",
		"[`docs/design.md`](https://github.com/HoneyBBQ/goubus/blob/v2.0.0-beta.1/docs/design.md)",
		"[migration](https://github.com/HoneyBBQ/goubus/blob/v2.0.0-beta.1/CHANGELOG.md#migrating)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("release notes output missing %q\ngot:\n%s", want, got)
		}
	}
}

func prepareReleaseNotesRepo(t *testing.T) string {
	t.Helper()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not available")
	}
	scriptBytes, err := os.ReadFile(filepath.Join("scripts", "release-notes.sh"))
	if err != nil {
		t.Fatalf("read release-notes.sh: %v", err)
	}
	root := t.TempDir()
	initCmd := exec.CommandContext(t.Context(), gitPath, "init", "-q", root) //nolint:gosec // Temp test repo.
	initOut, err := initCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git init: %v\n%s", err, initOut)
	}
	scriptPath := filepath.Join(root, "scripts", "release-notes.sh")
	err = os.MkdirAll(filepath.Dir(scriptPath), 0o750)
	if err != nil {
		t.Fatalf("mkdir scripts: %v", err)
	}
	err = os.WriteFile(scriptPath, scriptBytes, 0o750) //nolint:gosec // Temp executable script.
	if err != nil {
		t.Fatalf("write release-notes.sh: %v", err)
	}
	changelog := strings.Join([]string{
		"# Changelog", "", "## v2.0.0-beta.1 (2026-11-23)", "",
		"- Generic call `goubus.Call[T](ctx, caller, ...)` and `errors.AsType[*goubus.CallError](err)`.",
		"- Nested span `` `nested[T](arg)` `` stays literal.",
		"- Docs at [`docs/design.md`](docs/design.md) and [migration](#migrating).", "",
	}, "\n")
	err = os.WriteFile(filepath.Join(root, "CHANGELOG.md"), []byte(changelog), 0o600)
	if err != nil {
		t.Fatalf("write CHANGELOG.md: %v", err)
	}

	return scriptPath
}
