package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/profiles/x86_generic/session"
)

const defaultSessionTimeout = 60

func main() {
	ctx := context.Background()

	// 1. Initialize transport
	var caller goubus.Transport

	host := os.Getenv("OPENWRT_HOST")

	var err error

	if host != "" {
		caller, err = goubus.NewRpcClient(ctx, host, os.Getenv("OPENWRT_USERNAME"), os.Getenv("OPENWRT_PASSWORD"))
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

	// 2. Initialize Session Manager
	sessSvc := session.New(caller)

	// 3. Create a temporary session (timeout: 60 seconds)
	data, err := sessSvc.Create(ctx, defaultSessionTimeout)
	if err != nil {
		slog.Error("Failed to create session", "error", err)

		return
	}

	slog.Info("Temporary Session Created",
		"id", data.UbusRPCSession,
		"expires", data.ExpireTime)

	// 4. List all active sessions (Requires high permissions)
	sessions, errList := sessSvc.List(ctx)
	if errList == nil {
		slog.Info("Active Sessions", "count", len(sessions))
	} else {
		slog.Warn("Could not list sessions (permission issue)", "error", errList)
	}
}
