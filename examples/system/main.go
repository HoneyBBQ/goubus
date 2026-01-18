package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/profiles/x86_generic/system"
)

const bytesPerMB = 1024 * 1024

func main() {
	ctx := context.Background()

	// 1. Initialize transport
	host := os.Getenv("OPENWRT_HOST")

	var (
		caller goubus.Transport
		err    error
	)

	if host != "" {
		user := os.Getenv("OPENWRT_USERNAME")
		pass := os.Getenv("OPENWRT_PASSWORD")
		caller, err = goubus.NewRpcClient(ctx, host, user, pass)
	} else {
		caller, err = goubus.NewSocketClient(ctx, "")
	}

	if err != nil {
		slog.Error("Failed to connect", "error", err)
		os.Exit(1)
	}

	defer func() {
		_ = caller.Close()
	}()

	// 2. Initialize System Manager
	sysSvc := system.New(caller)

	// 3. Get System Info
	info, err := sysSvc.Info(ctx)
	if err != nil {
		slog.Error("Failed to get system info", "error", err)

		return
	}

	slog.Info("System Info",
		"uptime", info.Uptime,
		"localtime", info.LocalTime,
		"memory_total_mb", info.Memory.Total/bytesPerMB,
		"memory_avail_mb", info.Memory.Available/bytesPerMB)

	// 4. Get Board Info
	board, err := sysSvc.Board(ctx)
	if err != nil {
		slog.Error("Failed to get board info", "error", err)

		return
	}

	slog.Info("Board Info",
		"model", board.Model,
		"hostname", board.Hostname,
		"distribution", board.Release.Distribution,
		"version", board.Release.Version)
}
