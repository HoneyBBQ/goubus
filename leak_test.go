package goubus_test

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	code := m.Run()
	if code != 0 {
		os.Exit(code)
	}

	// Give asynchronous close/cleanup goroutines a brief scheduling turn before GC leak analysis.
	time.Sleep(20 * time.Millisecond)
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
