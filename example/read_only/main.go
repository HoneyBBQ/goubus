package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/honeybbq/goubus"
	"github.com/honeybbq/goubus/errdefs"
	"github.com/honeybbq/goubus/transport"
	"github.com/honeybbq/goubus/types"
	"github.com/honeybbq/goubus/uci/config"
)

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestConfig test configuration
type TestConfig struct {
	Host     string
	Username string
	Password string
}

// TestResult test result
type TestResult struct {
	TestName string
	Success  bool
	Error    error
	Data     any
}

func suppressStdout() (func(), error) {
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return nil, err
	}
	original := os.Stdout
	os.Stdout = devNull
	return func() {
		os.Stdout = original
		_ = devNull.Close()
	}, nil
}

func main() {
	verbose := flag.Bool("v", false, "enable verbose transport logging")
	flag.Parse()

	var restoreStdout func()
	if !*verbose {
		if restore, err := suppressStdout(); err == nil {
			restoreStdout = restore
		}
	}

	// Configure connection parameters - modify according to your setup
	testCfg := TestConfig{
		Host:     os.Getenv("OPENWRT_HOST"),     // OpenWrt router IP address
		Username: os.Getenv("OPENWRT_USERNAME"), // Username
		Password: os.Getenv("OPENWRT_PASSWORD"), // Password
	}

	fmt.Println("=== goubus Project Comprehensive Test ===")

	var (
		caller         types.Transport
		transportLabel string
		err            error
	)

	if testCfg.Host != "" && testCfg.Username != "" && testCfg.Password != "" {
		fmt.Println("使用环境变量配置，尝试通过 JSON-RPC 连接设备...")
		rpcClient, rpcErr := transport.NewRpcClient(testCfg.Host, testCfg.Username, testCfg.Password)
		if rpcErr != nil {
			log.Fatalf("Unable to connect via JSON-RPC: %v", rpcErr)
		}
		rpcClient.SetDebug(*verbose)
		caller = rpcClient
		transportLabel = fmt.Sprintf("JSON-RPC http://%s", testCfg.Host)
	} else {
		socketPath := os.Getenv("UBUS_SOCKET_PATH")
		fmt.Println("Remote environment variables not detected, falling back to ubus Unix socket ...")
		socketClient, err := transport.NewSocketClient(socketPath)
		if err != nil {
			log.Fatalf("Unable to connect via ubus socket /tmp/run/ubus/ubus.sock: %v", err)
		}
		socketClient.SetDebug(*verbose)
		caller = socketClient
		transportLabel = "unix socket"
	}

	fmt.Printf("Active transport: %s\n\n", transportLabel)

	// Create client connection
	client := goubus.NewClient(caller)

	var results []TestResult

	// 1. Test system information related read-only interfaces
	fmt.Println("=== 1. System Information Test ===")
	results = append(results, testSystemInfo(client)...)

	// 2. Test network related read-only interfaces
	fmt.Println("\n=== 2. Network Interface Test ===")
	results = append(results, testNetworkInfo(client)...)

	fmt.Printf("\n=== Index Field Demonstration ===\n")
	fmt.Printf("Showing difference between single section query vs. full config query:\n\n")

	// Single section query (no index)
	var singleConfig config.NetworkInterfaceConfig
	err = client.Uci().Package("network").Section("lan").Get(&singleConfig)
	if err == nil {
		indexStr := "nil (single section query)"
		if singleConfig.Metadata().Index != nil {
			indexStr = fmt.Sprintf("%d", *singleConfig.Metadata().Index)
		}
		fmt.Printf("Single section query - lan interface:\n")
		fmt.Printf("  .index = %s\n", indexStr)
	}

	// Full config query (with index)
	allSections, err := client.Uci().Package("network").GetAll()
	if err == nil {
		fmt.Printf("\nFull config query - network package (first 3 sections with .index):\n")
		count := 0
		for sectionName, sectionData := range allSections {
			if count >= 3 {
				break
			}
			if indexVal, ok := sectionData[".index"]; ok {
				fmt.Printf("  Section '%s': .index = %v\n", sectionName, indexVal)
			}
			count++
		}
		fmt.Printf("  (Index field is available when querying entire config)\n")
	}

	// 3. Test network related read-only device
	fmt.Println("\n=== 3. Network Device Test ===")
	results = append(results, testNetworkDevice(client)...)

	// 4. Test wireless related read-only interfaces
	fmt.Println("\n=== 4. Wireless Information Test ===")
	results = append(results, testWirelessInfo(client)...)

	// 5. Test wireless related read-only interfaces
	fmt.Println("\n=== 4. Wireless Information Test ===")
	results = append(results, testWirelessInfo(client)...)

	// 6. Test DHCP related read-only interfaces
	fmt.Println("\n=== 6. DHCP Information Test ===")
	results = append(results, testDHCPInfo(client)...)

	// 7. Test file system related read-only interfaces
	fmt.Println("\n=== 7. File System Test ===")
	results = append(results, testFileSystemInfo(client)...)

	// 8. Test log related read-only interfaces
	fmt.Println("\n=== 8. Log System Test ===")
	results = append(results, testLogInfo(client)...)

	// 9. Test service related read-only interfaces
	fmt.Println("\n=== 9. Service Status Test ===")
	results = append(results, testServiceInfo(client)...)

	// 10. Test LUCI related read-only interfaces
	fmt.Println("\n=== 10. LUCI Information Test ===")
	results = append(results, testLuciInfo(client)...)

	// 11. Test enhanced UCI configuration structures with new serialization features
	fmt.Println("\n=== 11. Enhanced UCI Configuration Structures Test ===")
	results = append(results, testEnhancedConfigStructures(client)...)

	// Revert UCI changes to ensure clean state for next run
	if err := client.Uci().Package("network").Revert(); err != nil {
		fmt.Printf("Warning: failed to revert network config: %v\n", err)
	}

	// 12. Test close client
	fmt.Println("\n=== 12. Close Client Test ===")
	results = append(results, testClose(client)...)

	if restoreStdout != nil {
		restoreStdout()
	}
	// Print test summary
	printTestSummary(results)
}

// Test system information related interfaces
func testSystemInfo(client *goubus.Client) []TestResult {
	var results []TestResult

	// Test getting system information
	systemInfo, err := client.System().Info()
	results = append(results, TestResult{
		TestName: "System Information Retrieval",
		Success:  err == nil,
		Error:    err,
		Data:     systemInfo,
	})
	if err == nil {
		fmt.Printf("✓ System information retrieval successful\n")
		fmt.Printf("  Uptime: %d seconds\n", systemInfo.Uptime)
		if len(systemInfo.Load) > 0 {
			fmt.Printf("  System load: %v\n", systemInfo.Load)
		}
		if systemInfo.Memory.Total > 0 {
			fmt.Printf("  Memory: %d MB total, %d MB available\n",
				systemInfo.Memory.Total/1024/1024,
				systemInfo.Memory.Available/1024/1024)
		}
		if systemInfo.Root.Total > 0 {
			fmt.Printf("  Root partition: %d KB total, %d KB available\n",
				systemInfo.Root.Total, systemInfo.Root.Avail)
		}
	} else {
		fmt.Printf("✗ System information retrieval failed: %v\n", err)
	}

	// Test getting hardware information
	boardInfo, err := client.System().Board()
	results = append(results, TestResult{
		TestName: "Hardware Information Retrieval",
		Success:  err == nil,
		Error:    err,
		Data:     boardInfo,
	})
	if err == nil {
		fmt.Printf("✓ Hardware information retrieval successful\n")
		if boardInfo.Model != "" {
			fmt.Printf("  Model: %s\n", boardInfo.Model)
		}
		if boardInfo.Hostname != "" {
			fmt.Printf("  Hostname: %s\n", boardInfo.Hostname)
		}
		if boardInfo.System != "" {
			fmt.Printf("  System: %s\n", boardInfo.System)
		}
		if boardInfo.Kernel != "" {
			fmt.Printf("  Kernel: %s\n", boardInfo.Kernel)
		}
		if boardInfo.BoardName != "" {
			fmt.Printf("  Board: %s\n", boardInfo.BoardName)
		}
		if boardInfo.Release.Distribution != "" {
			fmt.Printf("  Distribution: %s %s\n", boardInfo.Release.Distribution, boardInfo.Release.Version)
		}
	} else {
		fmt.Printf("✗ Hardware information retrieval failed: %v\n", err)
	}

	return results
}

// Test network interface related information
func testNetworkInfo(client *goubus.Client) []TestResult {
	var results []TestResult

	// Test getting all network interface information
	dump, err := client.Network().Interface("").Dump()
	results = append(results, TestResult{
		TestName: "Network Interface Dump",
		Success:  err == nil,
		Error:    err,
		Data:     dump,
	})
	if err == nil {
		fmt.Printf("✓ Network interface dump successful, interface count: %d\n", len(dump))

		// Print detailed information for all interfaces
		fmt.Println("  All network interface details:")
		for i, iface := range dump {
			fmt.Printf("    Interface %d: %s\n", i+1, iface.Interface)
			fmt.Printf("      Status: UP=%t, Available=%t, Autostart=%t\n", iface.Up, iface.Available, iface.Autostart)
			fmt.Printf("      Protocol: %s, Device: %s\n", iface.Proto, iface.Device)
			if iface.L3Device != "" {
				fmt.Printf("      L3 Device: %s\n", iface.L3Device)
			}
			if len(iface.IPv4Address) > 0 {
				fmt.Printf("      IPv4 addresses: ")
				for _, addr := range iface.IPv4Address {
					fmt.Printf("%s/%d ", addr.Address, addr.Mask)
				}
				fmt.Println()
			}
			if len(iface.IPv6Address) > 0 {
				fmt.Printf("      IPv6 addresses: ")
				for _, addr := range iface.IPv6Address {
					fmt.Printf("%s/%d ", addr.Address, addr.Mask)
				}
				fmt.Println()
			}
			if len(iface.DNSServer) > 0 {
				fmt.Printf("      DNS servers: %v\n", iface.DNSServer)
			}
			if len(iface.Route) > 0 {
				fmt.Printf("      Route count: %d\n", len(iface.Route))
			}
			fmt.Printf("      Uptime: %d seconds, Metric: %d\n", iface.Uptime, iface.Metric)
			fmt.Println()
		}
	} else {
		fmt.Printf("✗ Network interface dump failed: %v\n", err)
		return results
	}

	// Dynamically get all interface names and test each one
	var interfaceNames []string
	for _, iface := range dump {
		interfaceNames = append(interfaceNames, iface.Interface)
	}

	fmt.Printf("Found %d network interfaces, starting individual tests...\n", len(interfaceNames))

	for _, ifaceName := range interfaceNames {
		fmt.Printf("\n--- Testing interface: %s ---\n", ifaceName)

		// Test interface status
		status, err := client.Network().Interface(ifaceName).Status()
		results = append(results, TestResult{
			TestName: fmt.Sprintf("Interface %s Status", ifaceName),
			Success:  err == nil,
			Error:    err,
			Data:     status,
		})
		if err == nil {
			fmt.Printf("✓ Interface %s status retrieval successful\n", ifaceName)
			fmt.Printf("  Status details:\n")
			fmt.Printf("    UP: %t, Pending: %t, Available: %t\n",
				status.Up,
				status.Pending,
				status.Available)
			fmt.Printf("    Protocol: %s, Device: %s\n",
				status.Proto,
				status.Device)
			if status.L3Device != "" {
				fmt.Printf("    L3 Device: %s\n", status.L3Device)
			}

			// Print IPv4 addresses
			if len(status.IPv4Address) > 0 {
				fmt.Printf("    IPv4 addresses:\n")
				for _, addr := range status.IPv4Address {
					fmt.Printf("      %s/%d\n", addr.Address, addr.Mask)
				}
			}

			// Print IPv6 addresses
			if len(status.IPv6Address) > 0 {
				fmt.Printf("    IPv6 addresses:\n")
				for _, addr := range status.IPv6Address {
					fmt.Printf("      %s/%d\n", addr.Address, addr.Mask)
				}
			}

			// Print DNS servers
			if len(status.DNSServer) > 0 {
				fmt.Printf("    DNS servers: %v\n", status.DNSServer)
			}

			// Print routing information
			if len(status.Route) > 0 {
				fmt.Printf("    Route information (%d routes):\n", len(status.Route))
				for i, route := range status.Route {
					if i < 3 { // Show only first 3 routes
						fmt.Printf("      Target: %s/%d", route.Target, route.Mask)
						if route.Nexthop != "" {
							fmt.Printf(", Next hop: %s", route.Nexthop)
						}
						if route.Source != "" {
							fmt.Printf(", Source: %s", route.Source)
						}
						fmt.Println()
					}
				}
				if len(status.Route) > 3 {
					fmt.Printf("      ... %d more routes\n", len(status.Route)-3)
				}
			}

		} else {
			fmt.Printf("✗ Interface %s status retrieval failed: %v\n", ifaceName, err)
		}

		// Test interface configuration
		var configModel config.NetworkInterfaceConfig
		err = client.Uci().Package("network").Section(ifaceName).Get(&configModel)
		results = append(results, TestResult{
			TestName: fmt.Sprintf("Interface %s Configuration", ifaceName),
			Success:  err == nil,
			Error:    err,
			Data:     configModel,
		})
		if err == nil {
			fmt.Printf("✓ Interface %s configuration retrieval successful\n", ifaceName)
			indexStr := "nil (single section query)"
			if configModel.Metadata().Index != nil {
				indexStr = fmt.Sprintf("%d", *configModel.Metadata().Index)
			}
			fmt.Printf("  Metadata: .anonymous=%t, .type=%s, .name=%s, .index=%s\n",
				configModel.Metadata().Anonymous, configModel.Metadata().Type, configModel.Metadata().Name, indexStr)
			fmt.Printf("  Configuration details:\n")
			hasConfig := false
			if configModel.Proto != "" {
				fmt.Printf("    Protocol: %s\n", configModel.Proto)
				hasConfig = true
			}
			if configModel.Device != "" {
				fmt.Printf("    Device: %s\n", configModel.Device)
				hasConfig = true
			}
			if configModel.Type != "" {
				fmt.Printf("    Type: %s\n", configModel.Type)
				hasConfig = true
			}
			// Static IP configuration
			if configModel.StaticConfig != nil {
				fmt.Printf("    Static configuration:\n")
				fmt.Printf("      IP addresses: %v\n", configModel.StaticConfig.IPAddr)
				if configModel.StaticConfig.Gateway != nil {
					fmt.Printf("      Gateway: %s\n", *configModel.StaticConfig.Gateway)
				}
				if len(configModel.StaticConfig.DNS) > 0 {
					fmt.Printf("      DNS: %v\n", configModel.StaticConfig.DNS)
				}
				if configModel.StaticConfig.Netmask != nil {
					fmt.Printf("      Netmask: %s\n", *configModel.StaticConfig.Netmask)
				}
				if configModel.StaticConfig.Broadcast != nil {
					fmt.Printf("      Broadcast: %s\n", *configModel.StaticConfig.Broadcast)
				}
				hasConfig = true
			}

			// IPv6 configuration
			if (configModel.IP6Assign != nil && *configModel.IP6Assign != 0) || len(configModel.IP6Addr) > 0 || len(configModel.IP6Prefix) > 0 {
				fmt.Printf("    IPv6 configuration:\n")
				if configModel.IP6Assign != nil && *configModel.IP6Assign != 0 {
					fmt.Printf("      IP6Assign: %d\n", *configModel.IP6Assign)
				}
				if len(configModel.IP6Addr) > 0 {
					fmt.Printf("      IP6Addr: %v\n", configModel.IP6Addr)
				}
				if configModel.IP6GW != nil && *configModel.IP6GW != "" {
					fmt.Printf("      IP6Gateway: %s\n", *configModel.IP6GW)
				}
				if len(configModel.IP6Prefix) > 0 {
					fmt.Printf("      IP6Prefix: %v\n", configModel.IP6Prefix)
				}
				if len(configModel.IP6Class) > 0 {
					fmt.Printf("      IP6Class: %v\n", configModel.IP6Class)
				}
				if configModel.IP6Hint != nil && *configModel.IP6Hint != "" {
					fmt.Printf("      IP6Hint: %s\n", *configModel.IP6Hint)
				}
				if configModel.SourceFilter != nil {
					fmt.Printf("      SourceFilter: %t\n", *configModel.SourceFilter)
				}
				hasConfig = true
			}

			// DHCP configuration
			if configModel.DHCPConfig != nil {
				fmt.Printf("    DHCP configuration:\n")
				if configModel.DHCPConfig.Hostname != nil && *configModel.DHCPConfig.Hostname != "" {
					fmt.Printf("      Hostname: %s\n", *configModel.DHCPConfig.Hostname)
				}
				if configModel.DHCPConfig.ClientID != nil && *configModel.DHCPConfig.ClientID != "" {
					fmt.Printf("      ClientID: %s\n", *configModel.DHCPConfig.ClientID)
				}
				if configModel.DHCPConfig.VendorClass != nil && *configModel.DHCPConfig.VendorClass != "" {
					fmt.Printf("      VendorClass: %s\n", *configModel.DHCPConfig.VendorClass)
				}
				if configModel.DHCPConfig.DefaultRoute != nil {
					fmt.Printf("      DefaultRoute: %t\n", *configModel.DHCPConfig.DefaultRoute)
				}
				hasConfig = true
			}

			// PPP/PPPoE configuration
			if configModel.PPPConfig != nil {
				fmt.Printf("    PPPoE configuration:\n")
				if configModel.PPPConfig.Username != nil {
					fmt.Printf("      Username: %s\n", *configModel.PPPConfig.Username)
				}
				if configModel.PPPConfig.Password != nil {
					fmt.Printf("      Password: %s\n", *configModel.PPPConfig.Password)
				}
				if configModel.PPPConfig.Service != nil {
					fmt.Printf("      Service: %s\n", *configModel.PPPConfig.Service)
				}
				if configModel.PPPConfig.Server != nil && *configModel.PPPConfig.Server != "" {
					fmt.Printf("      Server: %s\n", *configModel.PPPConfig.Server)
				}
				if configModel.PPPConfig.Keepalive != nil && *configModel.PPPConfig.Keepalive != 0 {
					fmt.Printf("      Keepalive: %d seconds\n", *configModel.PPPConfig.Keepalive)
				}
				hasConfig = true
			}

			// Tunnel protocols (6in4, 6rd, 6to4, dslite, l2tp)
			if configModel.TunnelConfig != nil && ((configModel.TunnelConfig.PeerAddr != nil && *configModel.TunnelConfig.PeerAddr != "") ||
				(configModel.TunnelConfig.IP6Addr != nil && *configModel.TunnelConfig.IP6Addr != "") ||
				(configModel.TunnelConfig.TunnelID != nil && *configModel.TunnelConfig.TunnelID != "")) {
				fmt.Printf("    Tunnel configuration:\n")
				if configModel.TunnelConfig.PeerAddr != nil && *configModel.TunnelConfig.PeerAddr != "" {
					fmt.Printf("      PeerAddr: %s\n", *configModel.TunnelConfig.PeerAddr)
				}
				if configModel.TunnelConfig.IP6Addr != nil && *configModel.TunnelConfig.IP6Addr != "" {
					fmt.Printf("      IP6Addr: %s\n", *configModel.TunnelConfig.IP6Addr)
				}
				if configModel.TunnelConfig.IP6Prefix != nil && *configModel.TunnelConfig.IP6Prefix != "" {
					fmt.Printf("      IP6Prefix: %s\n", *configModel.TunnelConfig.IP6Prefix)
				}
				if configModel.TunnelConfig.TunnelID != nil && *configModel.TunnelConfig.TunnelID != "" {
					fmt.Printf("      TunnelID (HE.net): %s\n", *configModel.TunnelConfig.TunnelID)
				}
				if configModel.TunnelConfig.Username != nil && *configModel.TunnelConfig.Username != "" {
					fmt.Printf("      Username (HE.net): %s\n", *configModel.TunnelConfig.Username)
				}
				if configModel.TunnelConfig.Server != nil && *configModel.TunnelConfig.Server != "" {
					fmt.Printf("      Server (L2TP): %s\n", *configModel.TunnelConfig.Server)
				}
				if configModel.TunnelConfig.IP6PrefixLen != nil && *configModel.TunnelConfig.IP6PrefixLen != 0 {
					fmt.Printf("      IP6PrefixLen (6rd): %d\n", *configModel.TunnelConfig.IP6PrefixLen)
				}
				if configModel.TunnelConfig.TTL != nil && *configModel.TunnelConfig.TTL != 0 {
					fmt.Printf("      TTL: %d\n", *configModel.TunnelConfig.TTL)
				}
				hasConfig = true
			}

			// WireGuard configuration
			if configModel.WireGuardConfig != nil && ((configModel.WireGuardConfig.PrivateKey != nil && *configModel.WireGuardConfig.PrivateKey != "") ||
				(configModel.WireGuardConfig.PublicKey != nil && *configModel.WireGuardConfig.PublicKey != "") ||
				(configModel.WireGuardConfig.Endpoint != nil && *configModel.WireGuardConfig.Endpoint != "")) {
				fmt.Printf("    WireGuard configuration:\n")
				if configModel.WireGuardConfig.PrivateKey != nil && *configModel.WireGuardConfig.PrivateKey != "" {
					key := *configModel.WireGuardConfig.PrivateKey
					fmt.Printf("      PrivateKey: %s...\n", key[:min(16, len(key))])
				}
				if configModel.WireGuardConfig.PublicKey != nil && *configModel.WireGuardConfig.PublicKey != "" {
					key := *configModel.WireGuardConfig.PublicKey
					fmt.Printf("      PublicKey: %s...\n", key[:min(16, len(key))])
				}
				if configModel.WireGuardConfig.ListenPort != nil && *configModel.WireGuardConfig.ListenPort != 0 {
					fmt.Printf("      ListenPort: %d\n", *configModel.WireGuardConfig.ListenPort)
				}
				if len(configModel.WireGuardConfig.Addresses) > 0 {
					fmt.Printf("      Addresses: %v\n", configModel.WireGuardConfig.Addresses)
				}
				if configModel.WireGuardConfig.Endpoint != nil && *configModel.WireGuardConfig.Endpoint != "" {
					fmt.Printf("      Endpoint: %s\n", *configModel.WireGuardConfig.Endpoint)
				}
				if len(configModel.WireGuardConfig.AllowedIPs) > 0 {
					fmt.Printf("      AllowedIPs: %v\n", configModel.WireGuardConfig.AllowedIPs)
				}
				hasConfig = true
			}

			// Mobile network configuration (3G, QMI, NCM, WWAN)
			if configModel.MobileConfig != nil {
				fmt.Printf("    Mobile network configuration:\n")
				if configModel.MobileConfig.Device != nil && *configModel.MobileConfig.Device != "" {
					fmt.Printf("      Device: %s\n", *configModel.MobileConfig.Device)
				}
				if configModel.MobileConfig.APN != nil {
					fmt.Printf("      APN: %s\n", *configModel.MobileConfig.APN)
				}
				if configModel.MobileConfig.PINCode != nil {
					fmt.Printf("      PINCode: ****\n") // 隐藏PIN码
				}
				if configModel.MobileConfig.Username != nil {
					fmt.Printf("      Username: %s\n", *configModel.MobileConfig.Username)
				}
				if configModel.MobileConfig.Auth != nil {
					fmt.Printf("      Auth: %s\n", *configModel.MobileConfig.Auth)
				}
				if configModel.MobileConfig.Mode != nil {
					fmt.Printf("      Mode: %s\n", *configModel.MobileConfig.Mode)
				}
				if configModel.MobileConfig.Profile != nil {
					fmt.Printf("      Profile (QMI): %d\n", *configModel.MobileConfig.Profile)
				}
				hasConfig = true
			}

			// Virtual/Advanced protocols (GRE, VTI, VXLAN)
			if configModel.VirtualConfig != nil {
				fmt.Printf("    Virtual/Advanced configuration:\n")
				if configModel.VirtualConfig.RemoteIP != nil {
					fmt.Printf("      RemoteIP (GRE): %s\n", *configModel.VirtualConfig.RemoteIP)
				}
				if configModel.VirtualConfig.LocalIP != nil {
					fmt.Printf("      LocalIP (GRE): %s\n", *configModel.VirtualConfig.LocalIP)
				}
				if configModel.VirtualConfig.Key != nil {
					fmt.Printf("      Key (GRE): %d\n", *configModel.VirtualConfig.Key)
				}
				if configModel.VirtualConfig.VNI != nil {
					fmt.Printf("      VNI (VXLAN): %d\n", *configModel.VirtualConfig.VNI)
				}
				if configModel.VirtualConfig.Port != nil {
					fmt.Printf("      Port (VXLAN): %d\n", *configModel.VirtualConfig.Port)
				}
				if configModel.VirtualConfig.Group != nil {
					fmt.Printf("      Group (VXLAN): %s\n", *configModel.VirtualConfig.Group)
				}
				if configModel.VirtualConfig.TunSrc != nil {
					fmt.Printf("      TunSrc (VTI): %s\n", *configModel.VirtualConfig.TunSrc)
				}
				if configModel.VirtualConfig.TunDst != nil {
					fmt.Printf("      TunDst (VTI): %s\n", *configModel.VirtualConfig.TunDst)
				}
				hasConfig = true
			}

			// Bridge configuration
			if configModel.BridgeConfig != nil {
				fmt.Printf("    Bridge configuration:\n")
				if configModel.BridgeConfig.STP != nil {
					fmt.Printf("      STP: %v\n", *configModel.BridgeConfig.STP)
				}
				if configModel.BridgeConfig.ForwardDelay != nil {
					fmt.Printf("      ForwardDelay: %d\n", *configModel.BridgeConfig.ForwardDelay)
				}
				if configModel.BridgeConfig.IGMPSnooping != nil {
					fmt.Printf("      IGMPSnooping: %v\n", *configModel.BridgeConfig.IGMPSnooping)
				}
				if configModel.BridgeConfig.Priority != nil {
					fmt.Printf("      Priority: %d\n", *configModel.BridgeConfig.Priority)
				}
				hasConfig = true
			}

			// Advanced network options
			if configModel.Zone != nil || len(configModel.ReqOpts) > 0 || len(configModel.SendOpts) > 0 {
				fmt.Printf("    Advanced options:\n")
				if configModel.Zone != nil {
					fmt.Printf("      Firewall Zone: %v\n", *configModel.Zone)
				}
				if len(configModel.ReqOpts) > 0 {
					fmt.Printf("      ReqOpts: %v\n", configModel.ReqOpts)
				}
				if len(configModel.SendOpts) > 0 {
					fmt.Printf("      SendOpts: %v\n", configModel.SendOpts)
				}
				if configModel.DNSMetric != nil {
					fmt.Printf("      DNSMetric: %d\n", *configModel.DNSMetric)
				}
				hasConfig = true
			}

			if len(configModel.IfName) > 0 {
				fmt.Printf("    Interface names: %v\n", configModel.IfName)
				hasConfig = true
			}
			if configModel.Disabled != nil {
				fmt.Printf("    Disabled status: %v\n", *configModel.Disabled)
				hasConfig = true
			}
			if configModel.Auto != nil {
				fmt.Printf("    Auto start: %v\n", *configModel.Auto)
				hasConfig = true
			}
			if configModel.Metric != nil {
				fmt.Printf("    Metric: %d\n", *configModel.Metric)
				hasConfig = true
			}
			if configModel.MTU != nil {
				fmt.Printf("    MTU: %d\n", *configModel.MTU)
				hasConfig = true
			}
			if !hasConfig {
				fmt.Printf("    (No configuration information or empty configuration)\n")
			}
		} else {
			fmt.Printf("✗ Interface %s configuration retrieval failed: %v\n", ifaceName, err)
		}
	}

	return results
}

// Test network device related information
func testNetworkDevice(client *goubus.Client) []TestResult {
	var results []TestResult

	// Test getting all network device information
	dump, err := client.Network().Device().Status("")
	results = append(results, TestResult{
		TestName: "Network Device Dump",
		Success:  err == nil,
		Error:    err,
		Data:     dump,
	})
	if err != nil {
		fmt.Printf("✗ Network device status retrieval failed: %v. Skipping further network device tests.\n", err)
		return results
	}

	for name, device := range dump {
		fmt.Printf("Network device: %s\n", name)
		fmt.Printf("  Type: %s\n", device.Type)
		fmt.Printf("  Up: %t\n", device.Up)
	}
	return results
}

// Test wireless information related interfaces
func testWirelessInfo(client *goubus.Client) []TestResult {
	var results []TestResult

	// Get the overall wireless status first to discover devices and interfaces
	status, err := client.Network().Wireless().Status()
	results = append(results, TestResult{
		TestName: "Wireless Status Retrieval",
		Success:  err == nil,
		Error:    err,
		Data:     status,
	})
	if err != nil {
		fmt.Printf("✗ Wireless status retrieval failed: %v. Skipping further wireless tests.\n", err)
		return results
	}
	fmt.Printf("✓ Wireless status retrieval successful\n")

	// Test getting available wireless devices from iwinfo
	devices, err := client.IwInfo().Devices()
	results = append(results, TestResult{
		TestName: "Wireless Device List (iwinfo)",
		Success:  err == nil,
		Error:    err,
		Data:     devices,
	})
	if err == nil {
		fmt.Printf("✓ Wireless device list (iwinfo) retrieval successful, device count: %d\n", len(devices))
		for _, device := range devices {
			fmt.Printf("  Device: %s\n", device)
		}
	} else {
		fmt.Printf("✗ Wireless device list (iwinfo) retrieval failed: %v\n", err)
	}

	// Test each radio found in the status
	for radioName := range status {
		fmt.Printf("\n--- Testing wireless device: %s ---\n", radioName)

		// Test device configuration (UCI)
		var configModel config.WifiDeviceConfig
		err := client.Uci().Package("wireless").Section(radioName).Get(&configModel)
		results = append(results, TestResult{
			TestName: fmt.Sprintf("Wireless Device %s Configuration", radioName),
			Success:  err == nil,
			Error:    err,
			Data:     configModel,
		})
		if err == nil {
			fmt.Printf("✓ Wireless device %s configuration retrieval successful\n", radioName)
			// Print some config details
			if configModel.Type != "" && configModel.Channel != nil && configModel.Country != nil && configModel.HTMode != nil {
				fmt.Printf("  Type: %s, Channel: %d, Country: %s, HTMode: %s\n", configModel.Type, *configModel.Channel, *configModel.Country, *configModel.HTMode)
			}
		} else {
			fmt.Printf("✗ Wireless device %s configuration retrieval failed: %v\n", radioName, err)
		}

		// Test device-specific iwinfo calls
		info, err := client.IwInfo().Info(radioName)
		results = append(results, TestResult{
			TestName: fmt.Sprintf("Wireless Device %s Info", radioName),
			Success:  err == nil,
			Error:    err,
			Data:     info,
		})
		if err == nil && info != nil {
			fmt.Printf("✓ Wireless device %s info retrieval successful. Country: %s\n", radioName, info.Country)
		} else {
			fmt.Printf("✗ Wireless device %s info retrieval failed: %v\n", radioName, err)
		}

		countryList, err := client.IwInfo().CountryList(radioName)
		results = append(results, TestResult{
			TestName: fmt.Sprintf("Wireless Device %s Country List", radioName),
			Success:  err == nil,
			Error:    err,
			Data:     countryList,
		})
		if err == nil {
			fmt.Printf("✓ Wireless device %s country list retrieval successful, countries: %d\n", radioName, len(countryList))
		} else {
			fmt.Printf("✗ Wireless device %s country list retrieval failed: %v\n", radioName, err)
		}

		txPowerList, err := client.IwInfo().TxPowerList(radioName)
		results = append(results, TestResult{
			TestName: fmt.Sprintf("Wireless Device %s TX Power List", radioName),
			Success:  err == nil,
			Error:    err,
			Data:     txPowerList,
		})
		if err == nil {
			fmt.Printf("✓ Wireless device %s TX power list retrieval successful, power levels: %d\n", radioName, len(txPowerList))
		} else {
			fmt.Printf("✗ Wireless device %s TX power list retrieval failed: %v\n", radioName, err)
		}

		freqList, err := client.IwInfo().FreqList(radioName)
		results = append(results, TestResult{
			TestName: fmt.Sprintf("Wireless Device %s Frequency List", radioName),
			Success:  err == nil,
			Error:    err,
			Data:     freqList,
		})
		if err == nil {
			fmt.Printf("✓ Wireless device %s frequency list retrieval successful, channels: %d\n", radioName, len(freqList))
		} else {
			fmt.Printf("✗ Wireless device %s frequency list retrieval failed: %v\n", radioName, err)
		}

		// Test interfaces associated with this radio
		radioStatus := status[radioName]
		for _, iface := range radioStatus.Interfaces {
			ifaceName := iface.Section
			fmt.Printf("\n--- Testing wireless interface: %s (on %s) ---\n", ifaceName, radioName)

			// Test interface configuration (UCI)
			var ifaceConfig config.WifiIfaceConfig
			err := client.Uci().Package("wireless").Section(ifaceName).Get(&ifaceConfig)
			results = append(results, TestResult{
				TestName: fmt.Sprintf("Wireless Interface %s Configuration", ifaceName),
				Success:  err == nil,
				Error:    err,
				Data:     ifaceConfig,
			})
			if err != nil {
				fmt.Printf("✗ Wireless interface %s configuration retrieval failed: %v\n", ifaceName, err)
			}

			// Check if the interface is up before running info/scan tests
			if iface.Ifname == "" {
				fmt.Printf("✗ Wireless interface %s is down, skipping info and scan tests.\n", ifaceName)
				// Add skipped tests as successful to not fail the whole run
				results = append(results, TestResult{TestName: fmt.Sprintf("Wireless Interface %s Info", ifaceName), Success: true})
				results = append(results, TestResult{TestName: fmt.Sprintf("Wireless Interface %s Scan", ifaceName), Success: true})
				results = append(results, TestResult{TestName: fmt.Sprintf("Wireless Interface %s AssocList", ifaceName), Success: true})
				results = append(results, TestResult{TestName: fmt.Sprintf("Wireless Interface %s Survey", ifaceName), Success: true})
				results = append(results, TestResult{TestName: fmt.Sprintf("Wireless Interface %s PhyName", ifaceName), Success: true})
				continue
			}

			// Test interface information (iwinfo)
			info, err := client.IwInfo().Info(iface.Ifname)
			results = append(results, TestResult{
				TestName: fmt.Sprintf("Wireless Interface %s Info", ifaceName),
				Success:  err == nil,
				Error:    err,
				Data:     info,
			})
			if err == nil {
				fmt.Printf("✓ Wireless interface %s info retrieval successful\n", ifaceName)
				if info != nil {
					channelInfo := fmt.Sprintf("%d", info.Channel)
					if info.Channel == 0 {
						// 从wireless status获取频道配置信息来提供更好的说明
						radioStatus := status[radioName]
						if radioStatus.Config.Channel == "auto" || radioStatus.Config.Channel == "" {
							channelInfo = "0 (auto-selected, configured as 'auto')"
						} else {
							channelInfo = fmt.Sprintf("0 (configured as '%v' but not available)", radioStatus.Config.Channel)
						}
					}
					fmt.Printf("  BSSID: %s, Channel: %s, TX Power: %d dBm\n",
						info.BSSID, channelInfo, info.TXPower)
				}
			} else {
				fmt.Printf("✗ Wireless interface %s info retrieval failed: %v\n", ifaceName, err)
			}

			// Test scanning (iwinfo)
			scanResult, err := client.IwInfo().Scan(iface.Ifname)
			results = append(results, TestResult{
				TestName: fmt.Sprintf("Wireless Interface %s Scan", ifaceName),
				Success:  err == nil,
				Error:    err,
				Data:     scanResult,
			})
			if err == nil {
				fmt.Printf("✓ Wireless interface %s scan successful, networks found: %d\n", ifaceName, len(scanResult))
			} else {
				fmt.Printf("✗ Wireless interface %s scan failed: %v\n", ifaceName, err)
			}

			// Test AssocList (iwinfo)
			assocList, err := client.IwInfo().AssocList(iface.Ifname)
			results = append(results, TestResult{
				TestName: fmt.Sprintf("Wireless Interface %s AssocList", ifaceName),
				Success:  err == nil,
				Error:    err,
				Data:     assocList,
			})
			if err == nil {
				fmt.Printf("✓ Wireless interface %s assoclist successful, associated stations: %d\n", ifaceName, len(assocList))
			} else {
				fmt.Printf("✗ Wireless interface %s assoclist failed: %v\n", ifaceName, err)
			}

			// Test Survey (iwinfo)
			survey, err := client.IwInfo().Survey(iface.Ifname)
			results = append(results, TestResult{
				TestName: fmt.Sprintf("Wireless Interface %s Survey", ifaceName),
				Success:  err == nil,
				Error:    err,
				Data:     survey,
			})
			if err == nil {
				fmt.Printf("✓ Wireless interface %s survey successful, results: %d\n", ifaceName, len(survey))
			} else {
				fmt.Printf("✗ Wireless interface %s survey failed: %v\n", ifaceName, err)
			}

			// Test PhyName (iwinfo)
			phyName, err := client.IwInfo().PhyName(ifaceName)
			results = append(results, TestResult{
				TestName: fmt.Sprintf("Wireless Interface %s PhyName", ifaceName),
				Success:  err == nil,
				Error:    err,
				Data:     phyName,
			})
			if err == nil && phyName != nil {
				fmt.Printf("✓ Wireless interface %s phyname successful: %s\n", ifaceName, *phyName)
			} else if err == nil && phyName == nil {
				fmt.Printf("✓ Wireless interface %s phyname successful: (no phyname)\n", ifaceName)
			} else {
				fmt.Printf("✗ Wireless interface %s phyname failed: %v\n", ifaceName, err)
			}
		}
	}
	return results
}

// Test DHCP related information
func testDHCPInfo(client *goubus.Client) []TestResult {
	var results []TestResult

	// Test IPv4 leases
	leases, err := client.Luci().GetDHCPLeases()
	results = append(results, TestResult{
		TestName: "DHCP Leases",
		Success:  err == nil,
		Error:    err,
		Data:     leases,
	})
	if err == nil {
		fmt.Printf("✓ DHCP leases retrieval successful, lease count: %d\n", len(leases.IPv4Leases))
		for i, lease := range leases.IPv4Leases {
			if i < 3 { // Show only first 3
				fmt.Printf("  Lease: %s -> %s (%s)\n", lease.Macaddr, lease.IPAddr, lease.Hostname)
			}
		}
		if len(leases.IPv4Leases) > 3 {
			fmt.Printf("  ... and %d more IPv4 leases\n", len(leases.IPv4Leases)-3)
		}
		fmt.Printf("✓ DHCP IPv6 leases retrieval successful, lease count: %d\n", len(leases.IPv6Leases))
		for i, lease := range leases.IPv6Leases {
			if i < 3 { // Show only first 3
				fmt.Printf("  Lease: %s -> %s (%s)\n", lease.Macaddr, lease.IPAddr, lease.DUID)
			}
		}
		if len(leases.IPv6Leases) > 3 {
			fmt.Printf("  ... and %d more IPv6 leases\n", len(leases.IPv6Leases)-3)
		}
	} else {
		fmt.Printf("✗ DHCP leases retrieval failed: %v\n", err)
	}
	return results
}

// Test file system related interfaces
func testFileSystemInfo(client *goubus.Client) []TestResult {
	var results []TestResult

	// Test reading system files
	testFiles := []string{"/proc/stat", "/etc/passwd", "/proc/filesystems"}
	for _, file := range testFiles {
		content, err := client.File().Read(file, false)
		results = append(results, TestResult{
			TestName: fmt.Sprintf("Read File %s", file),
			Success:  err == nil,
			Error:    err,
			Data:     content,
		})
		if err == nil {
			fmt.Printf("✓ Reading file %s successful\n", file)
			if len(content.Data) > 100 {
				fmt.Printf("  Content: %s...\n", content.Data[:100])
			} else {
				fmt.Printf("  Content: %s\n", content.Data)
			}
		} else {
			fmt.Printf("✗ Reading file %s failed: %v\n", file, err)
		}
	}

	// Test listing directories
	testDirs := []string{"/etc", "/tmp", "/proc"}
	for _, dir := range testDirs {
		list, err := client.File().List(dir)
		results = append(results, TestResult{
			TestName: fmt.Sprintf("List Directory %s", dir),
			Success:  err == nil,
			Error:    err,
			Data:     list,
		})
		if err == nil {
			fmt.Printf("✓ Listing directory %s successful, entry count: %d\n", dir, len(list.Entries))
		} else {
			fmt.Printf("✗ Listing directory %s failed: %v\n", dir, err)
		}
	}

	// Test file status
	statResult, err := client.File().Stat("/etc")
	results = append(results, TestResult{
		TestName: "File Status Query",
		Success:  err == nil,
		Error:    err,
		Data:     statResult,
	})
	if err == nil {
		fmt.Printf("✓ File status query successful, type: %s\n", statResult.Type)
	} else {
		fmt.Printf("✗ File status query failed: %v\n", err)
	}

	// Test executing commands
	execResult, err := client.File().Exec("uname", []string{"-a"}, nil)
	results = append(results, TestResult{
		TestName: "Execute System Command",
		Success:  err == nil,
		Error:    err,
		Data:     execResult,
	})
	if err == nil {
		fmt.Printf("✓ System command execution successful\n")
		fmt.Printf("  Output: %s\n", execResult.Stdout)
	} else {
		fmt.Printf("✗ System command execution failed: %v\n", err)
	}

	return results
}

// Test log related interfaces
func testLogInfo(client *goubus.Client) []TestResult {
	var results []TestResult

	// Test reading system logs
	logData, err := client.Log().Read(10, false, true)
	results = append(results, TestResult{
		TestName: "System Log Reading",
		Success:  err == nil,
		Error:    err,
		Data:     logData,
	})
	if err == nil {
		fmt.Printf("✓ System log reading successful, entry count: %d\n", len(logData.Log))
		for i, entry := range logData.Log {
			if i < 3 { // Show only first 3
				fmt.Printf("  Log: %s\n", entry.Text)
			}
		}
	} else {
		fmt.Printf("✗ System log reading failed: %v\n", err)
	}

	return results
}

// Test service related interfaces
func testServiceInfo(client *goubus.Client) []TestResult {
	var results []TestResult

	// Test listing all services
	services, err := client.Service().List("", false)
	results = append(results, TestResult{
		TestName: "List All Services",
		Success:  err == nil,
		Error:    err,
		Data:     services,
	})

	if err == nil {
		fmt.Printf("✓ Service list retrieval successful, service count: %d\n", len(services))
		// Test a few common services from the list
		commonServices := []string{"network", "firewall", "dnsmasq", "uhttpd"}
		for _, srvName := range commonServices {
			if srv, ok := services[srvName]; ok {
				fmt.Printf("  Service %s found:\n", srvName)
				for instName, inst := range srv.Instances {
					fmt.Printf("    Instance '%s': running=%t, pid=%d\n", instName, inst.Running, inst.Pid)
				}
			} else {
				fmt.Printf("  Service %s not found in list.\n", srvName)
			}
		}
	} else {
		fmt.Printf("✗ Service list retrieval failed: %v\n", err)
	}

	return results
}

func testLuciInfo(client *goubus.Client) []TestResult {
	var results []TestResult

	lucitime, err := client.Luci().GetLocaltime()
	results = append(results, TestResult{
		TestName: "Local Time Retrieval",
		Success:  err == nil,
		Error:    err,
		Data:     lucitime,
	})
	if err == nil {
		fmt.Printf("✓ Local time retrieval successful: %s\n", lucitime)
	} else {
		fmt.Printf("✗ Local time retrieval failed: %v\n", err)
	}

	return results
}

// Test enhanced UCI configuration structures with new serialization features
func testEnhancedConfigStructures(client *goubus.Client) []TestResult {
	var results []TestResult

	// Test getting all UCI packages
	packages, err := client.Uci().Configs()
	results = append(results, TestResult{
		TestName: "Get All UCI Packages",
		Success:  err == nil,
		Error:    err,
		Data:     packages,
	})
	if err == nil {
		fmt.Printf("✓ Retrieved all UCI packages: %d\n", len(packages))
		for _, pkgName := range packages {
			fmt.Printf("  Package: %s\n", pkgName)
		}
	} else {
		fmt.Printf("✗ Failed to retrieve UCI packages: %v\n", err)
	}

	// Test getting a specific package
	networkPkg, err := client.Uci().Package("network").GetAll()
	results = append(results, TestResult{
		TestName: "Get Network Package",
		Success:  err == nil,
		Error:    err,
		Data:     networkPkg,
	})
	if err == nil {
		fmt.Printf("✓ Retrieved network package, section count: %d\n", len(networkPkg))
		for sectionName, sectionData := range networkPkg {
			fmt.Printf("  Section '%s':\n", sectionName)
			for key, value := range sectionData {
				fmt.Printf("    Key: %s, Value: %v\n", key, value)
			}
		}
	} else {
		fmt.Printf("✗ Failed to retrieve network package: %v\n", err)
	}

	// Test getting a specific section
	var lanSection config.NetworkInterfaceConfig
	err = client.Uci().Package("network").Section("lan").Get(&lanSection)
	results = append(results, TestResult{
		TestName: "Get LAN Section",
		Success:  err == nil,
		Error:    err,
		Data:     lanSection,
	})
	if err == nil {
		fmt.Printf("✓ Retrieved LAN section\n")
		indexStr := "nil (single section query)"
		if lanSection.Metadata().Index != nil {
			indexStr = fmt.Sprintf("%d", *lanSection.Metadata().Index)
		}
		fmt.Printf("  Metadata: .anonymous=%t, .type=%s, .name=%s, .index=%s\n",
			lanSection.Metadata().Anonymous, lanSection.Metadata().Type, lanSection.Metadata().Name, indexStr)
		fmt.Printf("  Configuration summary:\n")
		if lanSection.Proto != "" {
			fmt.Printf("    Protocol: %s\n", lanSection.Proto)
		}
		if lanSection.Device != "" {
			fmt.Printf("    Device: %s\n", lanSection.Device)
		}
		if len(lanSection.IfName) > 0 {
			fmt.Printf("    Interface names: %v\n", lanSection.IfName)
		}
		if lanSection.Disabled != nil {
			fmt.Printf("    Disabled: %v\n", *lanSection.Disabled)
		}
		if lanSection.Auto != nil {
			fmt.Printf("    Auto start: %v\n", *lanSection.Auto)
		}
		if lanSection.IP6Assign != nil && *lanSection.IP6Assign != 0 {
			fmt.Printf("    IPv6 assign: %d\n", *lanSection.IP6Assign)
		}
	} else {
		fmt.Printf("✗ Failed to retrieve LAN section: %v\n", err)
	}

	// Test getting a specific option
	lanProto, err := client.Uci().Package("network").Section("lan").Option("proto").Get()
	results = append(results, TestResult{
		TestName: "Get LAN Proto Option",
		Success:  err == nil,
		Error:    err,
		Data:     lanProto,
	})
	if err == nil {
		fmt.Printf("✓ Retrieved LAN proto option: %v\n", lanProto)
	} else {
		fmt.Printf("✗ Failed to retrieve LAN proto option: %v\n", err)
	}

	// Test setting an option
	err = client.Uci().Package("network").Section("lan").Option("proto").Set("static")
	results = append(results, TestResult{
		TestName: "Set LAN Proto Option",
		Success:  err == nil,
		Error:    err,
		Data:     nil,
	})
	if err == nil {
		fmt.Printf("✓ Set LAN proto option to 'static'\n")
	} else {
		fmt.Printf("✗ Failed to set LAN proto option: %v\n", err)
	}

	// Test getting the option again to confirm change
	lanProtoAfterSet, err := client.Uci().Package("network").Section("lan").Option("proto").Get()
	results = append(results, TestResult{
		TestName: "Get LAN Proto Option After Set",
		Success:  err == nil,
		Error:    err,
		Data:     lanProtoAfterSet,
	})
	if err == nil {
		fmt.Printf("✓ Retrieved LAN proto option after set: %v\n", lanProtoAfterSet)
	} else {
		fmt.Printf("✗ Failed to retrieve LAN proto option after set: %v\n", err)
	}

	// Test deleting an option
	err = client.Uci().Package("network").Section("lan").Option("proto").Delete()
	results = append(results, TestResult{
		TestName: "Delete LAN Proto Option",
		Success:  err == nil,
		Error:    err,
		Data:     nil,
	})
	if err == nil {
		fmt.Printf("✓ Deleted LAN proto option\n")
	} else {
		fmt.Printf("✗ Failed to delete LAN proto option: %v\n", err)
	}

	// Test getting the option again to confirm deletion (should fail with "not found")
	lanProtoAfterDelete, err := client.Uci().Package("network").Section("lan").Option("proto").Get()
	// Check if the error is the expected "section not found" error
	isExpectedError := err != nil && errdefs.IsNoData(err)
	results = append(results, TestResult{
		TestName: "Get LAN Proto Option After Delete",
		Success:  isExpectedError,
		Error:    err,
		Data:     lanProtoAfterDelete,
	})
	if isExpectedError {
		fmt.Printf("✓ Confirmed LAN proto option was deleted (expected error): %v\n", err)
	} else {
		fmt.Printf("✗ LAN proto option was not properly deleted: %s, err: %v\n", lanProtoAfterDelete, err)
	}

	// Test adding a new section
	err = client.Uci().Package("network").Add("interface", "new_section", &config.NetworkInterfaceConfig{})
	results = append(results, TestResult{
		TestName: "Add New Section",
		Success:  err == nil,
		Error:    err,
		Data:     nil,
	})
	if err == nil {
		fmt.Printf("✓ Added new section 'new_section'\n")
	} else {
		fmt.Printf("✗ Failed to add new section: %v\n", err)
	}

	// Test getting the new section
	var newSection config.NetworkInterfaceConfig
	err = client.Uci().Package("network").Section("new_section").Get(&newSection)
	results = append(results, TestResult{
		TestName: "Get New Section",
		Success:  err == nil,
		Error:    err,
		Data:     newSection,
	})
	if err == nil {
		fmt.Printf("✓ Retrieved new section\n")
		indexStr := "nil (single section query)"
		if newSection.Metadata().Index != nil {
			indexStr = fmt.Sprintf("%d", *newSection.Metadata().Index)
		}
		fmt.Printf("  Metadata: .anonymous=%t, .type=%s, .name=%s, .index=%s\n",
			newSection.Metadata().Anonymous, newSection.Metadata().Type, newSection.Metadata().Name, indexStr)
		fmt.Printf("  Configuration summary:\n")
		if newSection.Proto != "" {
			fmt.Printf("    Protocol: %s\n", newSection.Proto)
		}
		if newSection.Device != "" {
			fmt.Printf("    Device: %s\n", newSection.Device)
		}
		if len(newSection.IfName) > 0 {
			fmt.Printf("    Interface names: %v\n", newSection.IfName)
		}
		if newSection.Disabled != nil {
			fmt.Printf("    Disabled: %v\n", *newSection.Disabled)
		}
		if newSection.Auto != nil {
			fmt.Printf("    Auto start: %v\n", *newSection.Auto)
		}
	} else {
		fmt.Printf("✗ Failed to retrieve new section: %v\n", err)
	}

	// Test deleting the new section
	err = client.Uci().Package("network").Section("new_section").Delete()
	results = append(results, TestResult{
		TestName: "Delete New Section",
		Success:  err == nil,
		Error:    err,
		Data:     nil,
	})
	if err == nil {
		fmt.Printf("✓ Deleted new section 'new_section'\n")
	} else {
		fmt.Printf("✗ Failed to delete new section: %v\n", err)
	}

	return results
}

func testClose(client *goubus.Client) []TestResult {
	var results []TestResult

	err := client.Close()
	results = append(results, TestResult{
		TestName: "Close Client",
		Success:  err == nil,
		Error:    err,
		Data:     nil,
	})

	return results
}

// Print test summary
func printTestSummary(results []TestResult) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Test Summary")
	fmt.Println(strings.Repeat("=", 60))

	successCount := 0
	failCount := 0

	for _, result := range results {
		if result.Success {
			successCount++
			fmt.Printf("✓ %s\n", result.TestName)
		} else {
			failCount++
			fmt.Printf("✗ %s: %v\n", result.TestName, result.Error)
		}
	}

	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("Total tests: %d\n", len(results))
	fmt.Printf("Successful: %d\n", successCount)
	fmt.Printf("Failed: %d\n", failCount)
	if len(results) > 0 {
		fmt.Printf("Success rate: %.1f%%\n", float64(successCount)/float64(len(results))*100)
	}

	if failCount == 0 {
		fmt.Println("\n🎉 All tests passed!")
	} else {
		fmt.Printf("\n⚠️  %d tests failed, please check the error messages above\n", failCount)
	}
}
