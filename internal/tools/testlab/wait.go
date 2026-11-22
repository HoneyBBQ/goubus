package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/honeybbq/goubus/v2"
)

type waitConfig struct {
	socketPath string
	httpURL    string
	httpsURL   string
	caFile     string
	user       string
	password   string
	timeout    time.Duration
}

func runWait(args []string) error {
	cfg, err := parseWaitFlags(args)
	if err != nil {
		return err
	}

	tlsClient, err := buildCAHTTPClient(cfg.caFile)
	if err != nil {
		return err
	}

	deadline := time.Now().Add(cfg.timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		lastErr = probeAllTransports(cfg, tlsClient)
		if lastErr == nil {
			return nil
		}
		time.Sleep(250 * time.Millisecond)
	}

	return fmt.Errorf("%w: %w", errWaitTimedOut, lastErr)
}

func parseWaitFlags(args []string) (waitConfig, error) {
	fs := flag.NewFlagSet("wait", flag.ContinueOnError)
	var cfg waitConfig
	fs.StringVar(&cfg.socketPath, "socket", "", "Unix socket path")
	fs.StringVar(&cfg.httpURL, "http", "", "HTTP base URL")
	fs.StringVar(&cfg.httpsURL, "https", "", "HTTPS base URL")
	fs.StringVar(&cfg.caFile, "ca", "", "CA PEM file")
	fs.StringVar(&cfg.user, "user", "root", "RPC username")
	fs.StringVar(&cfg.password, "password", "", "RPC password")
	fs.DurationVar(&cfg.timeout, "timeout", 45*time.Second, "overall readiness timeout")
	err := fs.Parse(args)
	if err != nil {
		return waitConfig{}, err
	}
	if cfg.socketPath == "" || cfg.httpURL == "" || cfg.httpsURL == "" || cfg.caFile == "" || cfg.password == "" {
		return waitConfig{}, fmt.Errorf("%w: -socket, -http, -https, -ca, -password", errMissingFlag)
	}

	return cfg, nil
}

func buildCAHTTPClient(caFile string) (*http.Client, error) {
	root, err := os.OpenRoot(filepath.Dir(caFile))
	if err != nil {
		return nil, err
	}
	caPEM, readErr := root.ReadFile(filepath.Base(caFile))
	closeErr := root.Close()
	err = errors.Join(readErr, closeErr)
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, errInvalidCA
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:    pool,
			MinVersion: tls.VersionTLS12,
		},
	}

	return &http.Client{Transport: transport, Timeout: 5 * time.Second}, nil
}

func probeAllTransports(cfg waitConfig, tlsHTTPClient *http.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	err := probeSocket(ctx, cfg.socketPath)
	if err != nil {
		return fmt.Errorf("socket: %w", err)
	}

	httpCli, err := goubus.NewRpcClient(ctx, cfg.httpURL, cfg.user, cfg.password)
	if err != nil {
		return fmt.Errorf("http rpc: %w", err)
	}
	_ = httpCli.Close()

	httpsCli, err := goubus.NewRpcClient(
		ctx,
		cfg.httpsURL,
		cfg.user,
		cfg.password,
		goubus.WithHTTPClient(tlsHTTPClient),
	)
	if err != nil {
		return fmt.Errorf("https rpc: %w", err)
	}
	_ = httpsCli.Close()

	return nil
}

func probeSocket(ctx context.Context, socketPath string) error {
	sock, err := goubus.NewSocketClient(ctx, socketPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = sock.Close()
	}()

	required := []string{"session", "uci", "file", "luci", "rc"}
	listed, err := goubus.List(ctx, sock, required...)
	if err != nil {
		return err
	}
	for _, name := range required {
		if _, ok := listed[name]; !ok {
			return fmt.Errorf("%w: %s", errMissingObj, name)
		}
	}

	return nil
}
