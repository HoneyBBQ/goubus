// Command testlab provides TLS generation, Unix/TCP socket bridging, and
// readiness probing for the live OpenWrt integration laboratory.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
)

var (
	errUsage        = errors.New("usage: testlab <rand|tls|bridge|wait> [flags]")
	errMissingFlag  = errors.New("missing required flag")
	errInvalidCA    = errors.New("failed to append CA certificate")
	errWaitTimedOut = errors.New("lab readiness check timed out")
	errMissingObj   = errors.New("expected ubus object not registered yet")
)

func main() {
	err := run(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "testlab:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errUsage
	}

	switch args[0] {
	case "rand":
		return runRand()
	case "tls":
		return runTLS(args[1:])
	case "bridge":
		return runBridge(args[1:])
	case "wait":
		return runWait(args[1:])
	default:
		return fmt.Errorf("%w: unknown subcommand %q", errUsage, args[0])
	}
}

func runRand() error {
	buf := make([]byte, 16)
	_, err := rand.Read(buf)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, hex.EncodeToString(buf))

	return err
}
