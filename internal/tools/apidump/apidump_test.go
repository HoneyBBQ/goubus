package main

import (
	"errors"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden file")

const samplePath = "example.com/sample"

// TestSampleReport pins how each kind of declaration is rendered, on a
// package that has one of everything: a change to the format of a report
// would otherwise only show up as noise in the reports of the real packages.
func TestSampleReport(t *testing.T) {
	api, err := newDumper().dump(filepath.Join("testdata", "sample"), samplePath)
	if err != nil {
		t.Fatal(err)
	}
	got := api.text() + "\n# leaks\n" + strings.Join(api.leaks, "\n") + "\n"

	testdata, err := os.OpenRoot("testdata")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testdata.Close() }()
	if *update {
		err = testdata.WriteFile("sample.golden", []byte(got), 0o600)
		if err != nil {
			t.Fatal(err)
		}
	}
	want, err := testdata.ReadFile("sample.golden")
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != got {
		t.Errorf("the report of testdata/sample changed (run go test -update to accept):\n%s",
			unifiedDiff("sample.golden", string(want), got))
	}
}

func TestDiffShowsChangedLinesWithContext(t *testing.T) {
	old := "a\nb\nc\nd\ne\nf\ng\nh\ni\nj\nk\n"
	updated := "a\nb\nc\nD\ne\nf\ng\nh\ni\nj\nk\nl\n"
	got := unifiedDiff("x.txt", old, updated)
	want := "--- x.txt (committed)\n+++ x.txt (from the code)\n" +
		"  b\n  c\n- d\n+ D\n  e\n  f\n...\n  j\n  k\n+ l\n"
	if got != want {
		t.Errorf("diff:\n%s\nwant:\n%s", got, want)
	}
}

// tempRoot is an empty module root that is removed with the test.
func tempRoot(t *testing.T) *os.Root {
	t.Helper()
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })

	return root
}

func TestCheckReportsCatchesWhatChanged(t *testing.T) {
	root := tempRoot(t)
	reports := map[string]string{"a.txt": "# h\nconst A int = 1\n"}
	err := writeReports(root, reports)
	if err != nil {
		t.Fatal(err)
	}
	diff, err := checkReports(root, reports)
	if err != nil {
		t.Fatalf("fresh reports reported as stale: %v\n%s", err, diff)
	}

	changed := map[string]string{"a.txt": "# h\nconst A int = 2\n", "b.txt": "# h\n"}
	diff, err = checkReports(root, changed)
	if !errors.Is(err, errStale) {
		t.Fatalf("err = %v, want errStale", err)
	}
	for _, want := range []string{"- const A int = 1", "+ const A int = 2", "api/b.txt is missing"} {
		if !strings.Contains(diff, want) {
			t.Errorf("output lacks %q:\n%s", want, diff)
		}
	}
}

func TestStaleReportIsCaughtAndRemoved(t *testing.T) {
	root := tempRoot(t)
	reports := map[string]string{"a.txt": "# h\n"}
	err := writeReports(root, reports)
	if err != nil {
		t.Fatal(err)
	}
	err = root.WriteFile("api/gone.txt", []byte("x\n"), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	diff, err := checkReports(root, reports)
	if !errors.Is(err, errStale) || !strings.Contains(diff, "api/gone.txt is stale") {
		t.Errorf("a report no package writes was not caught: %v\n%s", err, diff)
	}
	err = writeReports(root, reports)
	if err != nil {
		t.Fatal(err)
	}
	_, err = root.Stat("api/gone.txt")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("writeReports left the stale report: %v", err)
	}
}
