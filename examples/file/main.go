package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/profiles/x86_generic/file"
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

	// 2. Initialize File Manager
	fileSvc := file.New(caller)

	// 3. Read a file
	res, err := fileSvc.Read(ctx, "/etc/os-release", false)
	if err != nil {
		slog.Error("Failed to read file", "error", err)

		return
	}

	slog.Info("File Content", "path", "/etc/os-release", "data", res.Data)

	// 4. List a directory
	list, err := fileSvc.List(ctx, "/etc/config")
	if err != nil {
		slog.Error("Failed to list directory", "error", err)

		return
	}

	slog.Info("Directory Listing", "path", "/etc/config")

	for _, entry := range list.Entries {
		slog.Info("Entry", "name", entry.Name, "size_bytes", entry.Size)
	}
}
