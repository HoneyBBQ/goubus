package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/profiles/x86_generic/service"
)

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

	// 2. Initialize Service Manager (Procd)
	srvSvc := service.New(caller)

	// 3. List all services
	services, err := srvSvc.List(ctx, "", false)
	if err != nil {
		slog.Error("Failed to list services", "error", err)

		return
	}

	slog.Info("Service Summary", "total", len(services))

	// 4. Check status of a common service (e.g., uhttpd)
	if srv, ok := services["uhttpd"]; ok {
		slog.Info("uhttpd service details")

		for name, inst := range srv.Instances {
			slog.Info("instance", "name", name, "running", inst.Running, "pid", inst.Pid)
		}
	}
}
