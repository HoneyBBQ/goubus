package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/honeybbq/goubus"
	"github.com/honeybbq/goubus/transport"
)

type connectionConfig struct {
	Host     string
	Username string
	Password string
	Socket   string
}

func main() {
	host := flag.String("host", os.Getenv("OPENWRT_HOST"), "OpenWrt router address")
	user := flag.String("user", os.Getenv("OPENWRT_USERNAME"), "ubus username")
	pass := flag.String("pass", os.Getenv("OPENWRT_PASSWORD"), "ubus password")
	socket := flag.String("socket", os.Getenv("UBUS_SOCKET_PATH"), "ubus socket path")
	newHostname := flag.String("set-hostname", "", "stage a new hostname (no automatic commit)")
	verbose := flag.Bool("v", false, "enable transport debug logs")
	flag.Parse()

	client, label := initClient(connectionConfig{
		Host:     *host,
		Username: *user,
		Password: *pass,
		Socket:   *socket,
	}, *verbose)
	defer client.Close() // safe for both transports

	fmt.Printf("Connected via %s\n\n", label)

	systemSection := resolveSystemSection(client)
	showSection(client, "system", systemSection)
	showPackageSections(client, "network")

	if *newHostname != "" {
		stageHostname(client, systemSection, *newHostname)
	} else {
		fmt.Println("\nUse -set-hostname=<value> to stage a hostname change.")
	}
}

func initClient(cfg connectionConfig, verbose bool) (*goubus.Client, string) {
	if cfg.Host != "" && cfg.Username != "" && cfg.Password != "" {
		rpcClient, err := transport.NewRpcClient(cfg.Host, cfg.Username, cfg.Password)
		if err != nil {
			log.Fatalf("failed to create RPC client: %v", err)
		}
		rpcClient.SetDebug(verbose)
		return goubus.NewClient(rpcClient), fmt.Sprintf("JSON-RPC http://%s", cfg.Host)
	}

	if cfg.Socket == "" {
		// OpenWrt default
		cfg.Socket = "/var/run/ubus.sock"
	}
	socketClient, err := transport.NewSocketClient(cfg.Socket)
	if err != nil {
		log.Fatalf("failed to connect to ubus socket %s: %v", cfg.Socket, err)
	}
	socketClient.SetDebug(verbose)
	return goubus.NewClient(socketClient), fmt.Sprintf("ubus socket %s", cfg.Socket)
}

func showSection(client *goubus.Client, pkg, section string) {
	fmt.Printf("=== %s.%s ===\n", pkg, section)
	sec, err := client.Uci().Package(pkg).Section(section).Get()
	if err != nil {
		fmt.Printf("unable to read section: %v\n", err)
		return
	}

	fmt.Printf("type=%s anonymous=%t\n", sec.Metadata.Type, bool(sec.Metadata.Anonymous))
	for option, values := range sec.Values {
		if len(values) == 1 {
			fmt.Printf("  %s = %s\n", option, values[0])
			continue
		}
		fmt.Printf("  %s = %v\n", option, values)
	}
}

func showPackageSections(client *goubus.Client, pkg string) {
	fmt.Printf("\n=== %s package ===\n", pkg)
	sections, err := client.Uci().Package(pkg).GetAll()
	if err != nil {
		fmt.Printf("failed to load package: %v\n", err)
		return
	}
	for name, section := range sections {
		index := "n/a"
		if section.Metadata.Index != nil {
			index = fmt.Sprintf("%d", *section.Metadata.Index)
		}
		fmt.Printf("- %s (type=%s, index=%s)\n", name, section.Type, index)
	}
}

func resolveSystemSection(client *goubus.Client) string {
	if _, err := client.Uci().Package("system").Section("system").Get(); err == nil {
		return "system"
	}
	return "@system[0]"
}

func stageHostname(client *goubus.Client, sectionName, hostname string) {
	fmt.Printf("\nStaging hostname change to %q ...\n", hostname)
	values := goubus.NewSectionValues()
	values.Set("hostname", hostname)

	if err := client.Uci().Package("system").Section(sectionName).SetValues(values); err != nil {
		fmt.Printf("failed to stage hostname: %v\n", err)
		return
	}

	if err := client.Uci().Package("system").Commit(); err != nil {
		fmt.Printf("failed to commit hostname: %v\n", err)
		return
	}

	fmt.Println("Hostname staged. Run `/etc/init.d/system reload` on the device to apply it.")
}
