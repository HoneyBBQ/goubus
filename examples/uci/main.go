package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"strconv"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/profiles/x86_generic/uci"
)

type connectionConfig struct {
	Host     string
	Username string
	Password string
	Socket   string
}

func main() {
	ctx := context.Background()
	host := flag.String("host", os.Getenv("OPENWRT_HOST"), "OpenWrt router address")
	user := flag.String("user", os.Getenv("OPENWRT_USERNAME"), "ubus username")
	pass := flag.String("pass", os.Getenv("OPENWRT_PASSWORD"), "ubus password")
	socket := flag.String("socket", os.Getenv("UBUS_SOCKET_PATH"), "ubus socket path")
	newHostname := flag.String("set-hostname", "", "stage a new hostname (no automatic commit)")
	verbose := flag.Bool("v", false, "enable transport debug logs")

	flag.Parse()

	caller, label := initTransport(ctx, connectionConfig{
		Host:     *host,
		Username: *user,
		Password: *pass,
		Socket:   *socket,
	}, *verbose)

	defer func() {
		_ = caller.Close()
	}()

	slog.Info("Connected", "via", label)

	uciSvc := uci.New(caller)

	systemSection := resolveSystemSection(ctx, uciSvc)
	showSection(ctx, uciSvc, "system", systemSection)
	showPackageSections(ctx, uciSvc, "network")

	if *newHostname != "" {
		stageHostname(ctx, uciSvc, systemSection, *newHostname)
	} else {
		slog.Info("Use -set-hostname=<value> to stage a hostname change")
	}
}

func initTransport(ctx context.Context, cfg connectionConfig, verbose bool) (goubus.Transport, string) {
	if cfg.Host != "" && cfg.Username != "" && cfg.Password != "" {
		var opts []goubus.RpcOption

		if verbose {
			handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
			opts = append(opts, goubus.WithRpcLogger(slog.New(handler)))
		}

		rpcClient, err := goubus.NewRpcClient(ctx, cfg.Host, cfg.Username, cfg.Password, opts...)
		if err != nil {
			slog.Error("failed to create RPC client", "error", err)
			os.Exit(1)
		}

		return rpcClient, "JSON-RPC http://" + cfg.Host
	}

	if cfg.Socket == "" {
		cfg.Socket = "/var/run/ubus.sock"
	}

	var opts []goubus.SocketOption

	if verbose {
		handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
		opts = append(opts, goubus.WithSocketLogger(slog.New(handler)))
	}

	socketClient, err := goubus.NewSocketClient(ctx, cfg.Socket, opts...)
	if err != nil {
		slog.Error("failed to connect to ubus socket", "path", cfg.Socket, "error", err)
		os.Exit(1)
	}

	return socketClient, "ubus socket " + cfg.Socket
}

func showSection(ctx context.Context, svc *uci.Manager, pkg, section string) {
	slog.Info("Reading section", "package", pkg, "section", section)

	sec, err := svc.Package(pkg).Section(section).Get(ctx)
	if err != nil {
		slog.Error("unable to read section", "error", err)

		return
	}

	slog.Info("Section Metadata", "type", sec.Metadata.Type, "anonymous", bool(sec.Metadata.Anonymous))

	for option, values := range sec.Values.All() {
		if len(values) == 1 {
			slog.Info("Option", "name", option, "value", values[0])

			continue
		}

		slog.Info("Option", "name", option, "values", values)
	}
}

func showPackageSections(ctx context.Context, svc *uci.Manager, pkg string) {
	slog.Info("Listing package sections", "package", pkg)

	sections, err := svc.Package(pkg).GetAll(ctx)
	if err != nil {
		slog.Error("failed to load package", "error", err)

		return
	}

	for name, section := range sections {
		index := "n/a"
		if section.Metadata.Index != nil {
			index = strconv.Itoa(*section.Metadata.Index)
		}

		slog.Info("Section", "name", name, "type", section.Type, "index", index)
	}
}

func resolveSystemSection(ctx context.Context, svc *uci.Manager) string {
	_, err := svc.Package("system").Section("system").Get(ctx)
	if err == nil {
		return "system"
	}

	return "@system[0]"
}

func stageHostname(ctx context.Context, svc *uci.Manager, sectionName, hostname string) {
	slog.Info("Staging hostname change", "new_hostname", hostname)

	values := uci.NewSectionValues()
	values.Set("hostname", hostname)

	err := svc.Package("system").Section(sectionName).SetValues(ctx, values)
	if err != nil {
		slog.Error("failed to stage hostname", "error", err)

		return
	}

	err = svc.Package("system").Commit(ctx)
	if err != nil {
		slog.Error("failed to commit hostname", "error", err)

		return
	}

	slog.Info("Hostname staged. Run `/etc/init.d/system reload` on the device to apply it.")
}
