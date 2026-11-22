package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type bridgeConfig struct {
	listenNetwork string
	listenAddr    string
	targetNetwork string
	targetAddr    string
	readyFile     string
}

func runBridge(args []string) error {
	cfg, err := parseBridgeFlags(args)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if cfg.listenNetwork == "unix" {
		removeUnixFile(cfg.listenAddr)
		defer removeUnixFile(cfg.listenAddr)
	}

	var lc net.ListenConfig
	listener, err := lc.Listen(ctx, cfg.listenNetwork, cfg.listenAddr)
	if err != nil {
		return err
	}
	defer func() {
		_ = listener.Close()
	}()

	if cfg.readyFile != "" {
		err = writeSmallFile(cfg.readyFile, []byte("ready\n"))
		if err != nil {
			return err
		}
	}

	return serveBridge(ctx, listener, cfg.targetNetwork, cfg.targetAddr)
}

func parseBridgeFlags(args []string) (bridgeConfig, error) {
	fs := flag.NewFlagSet("bridge", flag.ContinueOnError)
	var cfg bridgeConfig
	fs.StringVar(&cfg.listenNetwork, "listen-network", "", "network to listen on (unix or tcp)")
	fs.StringVar(&cfg.listenAddr, "listen-addr", "", "address to listen on")
	fs.StringVar(&cfg.targetNetwork, "target-network", "", "network to dial (unix or tcp)")
	fs.StringVar(&cfg.targetAddr, "target-addr", "", "address to dial")
	fs.StringVar(&cfg.readyFile, "ready-file", "", "optional file written once listener is bound")
	err := fs.Parse(args)
	if err != nil {
		return bridgeConfig{}, err
	}
	if cfg.listenNetwork == "" || cfg.listenAddr == "" || cfg.targetNetwork == "" || cfg.targetAddr == "" {
		return bridgeConfig{}, fmt.Errorf(
			"%w: -listen-network, -listen-addr, -target-network, -target-addr",
			errMissingFlag,
		)
	}

	return cfg, nil
}

func serveBridge(ctx context.Context, listener net.Listener, targetNet, targetAddr string) error {
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	var (
		wg      sync.WaitGroup
		loopErr error
	)
	for {
		src, acceptErr := listener.Accept()
		if acceptErr != nil {
			if ctx.Err() == nil && !errors.Is(acceptErr, net.ErrClosed) {
				loopErr = acceptErr
			}

			break
		}
		wg.Go(func() {
			relayConn(ctx, src, targetNet, targetAddr)
		})
	}
	wg.Wait()

	return loopErr
}

func relayConn(ctx context.Context, src net.Conn, targetNet, targetAddr string) {
	dialer := net.Dialer{Timeout: 5 * time.Second}
	dst, err := dialer.DialContext(ctx, targetNet, targetAddr)
	if err != nil {
		_ = src.Close()

		return
	}

	var pair sync.WaitGroup
	pair.Go(func() {
		_, _ = io.Copy(dst, src)
		closeWriteHalf(dst)
	})
	pair.Go(func() {
		_, _ = io.Copy(src, dst)
		closeWriteHalf(src)
	})
	pair.Wait()
	_ = src.Close()
	_ = dst.Close()
}

func closeWriteHalf(conn net.Conn) {
	type writeCloser interface {
		CloseWrite() error
	}
	if wc, ok := conn.(writeCloser); ok {
		_ = wc.CloseWrite()

		return
	}
	_ = conn.Close()
}

func removeUnixFile(filePath string) {
	root, err := os.OpenRoot(filepath.Dir(filePath))
	if err != nil {
		return
	}
	_ = root.Remove(filepath.Base(filePath))
	_ = root.Close()
}

func writeSmallFile(filePath string, content []byte) error {
	dir := filepath.Dir(filePath)
	err := os.MkdirAll(dir, 0o750)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}

	return errors.Join(root.WriteFile(filepath.Base(filePath), content, 0o600), root.Close())
}
