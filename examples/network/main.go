package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/profiles/x86_generic/network"
)

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

	// 2. Initialize Network Manager
	netSvc := network.New(caller)

	// 3. Dump all interfaces
	ifaces, err := netSvc.Dump(ctx)
	if err != nil {
		slog.Error("Failed to dump interfaces", "error", err)

		return
	}

	slog.Info("Network Summary", "count", len(ifaces))

	for _, iface := range ifaces {
		slog.Info("Interface",
			"name", iface.Interface,
			"proto", iface.Proto,
			"device", iface.Device,
			"up", iface.Up)
	}

	// 4. Get specific interface status (lan)
	if len(ifaces) > 0 {
		lanStatus, errStatus := netSvc.Interface("lan").Status(ctx)
		if errStatus == nil {
			slog.Info("LAN Status", "uptime", lanStatus.Uptime)

			if len(lanStatus.IPv4Address) > 0 {
				slog.Info("LAN Address",
					"ip", lanStatus.IPv4Address[0].Address,
					"mask", lanStatus.IPv4Address[0].Mask)
			}
		}
	}
}
