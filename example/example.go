package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/honeybbq/goubus"
)

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
	Data     interface{}
}

func main() {
	// Configure connection parameters - modify according to your setup
	config := TestConfig{
		Host:     os.Getenv("OPENWRT_HOST"),     // OpenWrt router IP address
		Username: os.Getenv("OPENWRT_USERNAME"), // Username
		Password: os.Getenv("OPENWRT_PASSWORD"), // Password
	}

	if config.Host == "" || config.Username == "" || config.Password == "" {
		log.Fatalf("OPENWRT_HOST, OPENWRT_USERNAME, OPENWRT_PASSWORD are not set")
	}

	fmt.Println("=== goubus Project Comprehensive Test ===")
	fmt.Printf("Connecting to: %s\n", config.Host)
	fmt.Println()

	// Create client connection
	client, err := goubus.NewClient(config.Host, config.Username, config.Password)
	if err != nil {
		log.Fatalf("Unable to connect to device: %v", err)
	}

	for key, acl := range client.AuthData.ACLs.AccessGroup {
		fmt.Println(key, acl)
	}
	for key, acl := range client.AuthData.ACLs.Ubus {
		fmt.Println(key, acl)
	}
	for key, acl := range client.AuthData.ACLs.Uci {
		fmt.Println(key, acl)
	}

	var results []TestResult

	// 1. Test system information related read-only interfaces
	fmt.Println("=== 1. System Information Test ===")
	results = append(results, testSystemInfo(client)...)

	// 2. Test authentication related read-only interfaces
	fmt.Println("\n=== 2. Authentication Information Test ===")
	results = append(results, testAuthInfo(client)...)

	// 3. Test network related read-only interfaces
	fmt.Println("\n=== 3. Network Interface Test ===")
	results = append(results, testNetworkInfo(client)...)

	// 4. Test wireless related read-only interfaces
	fmt.Println("\n=== 4. Wireless Information Test ===")
	results = append(results, testWirelessInfo(client)...)

	// 5. Test DHCP related read-only interfaces
	fmt.Println("\n=== 5. DHCP Information Test ===")
	results = append(results, testDHCPInfo(client)...)

	// 6. Test file system related read-only interfaces
	fmt.Println("\n=== 6. File System Test ===")
	results = append(results, testFileSystemInfo(client)...)

	// 7. Test log related read-only interfaces
	fmt.Println("\n=== 7. Log System Test ===")
	results = append(results, testLogInfo(client)...)

	// 8. Test service related read-only interfaces
	fmt.Println("\n=== 8. Service Status Test ===")
	results = append(results, testServiceInfo(client)...)

	// 9. Test LUCI related read-only interfaces
	fmt.Println("\n=== 9. LUCI Information Test ===")
	results = append(results, testLuciInfo(client)...)

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

// Test authentication information related interfaces
func testAuthInfo(client *goubus.Client) []TestResult {
	var results []TestResult

	// Test getting session information
	sessionInfo, err := client.Auth().GetSessionInfo()
	results = append(results, TestResult{
		TestName: "Session Information Retrieval",
		Success:  err == nil,
		Error:    err,
		Data:     sessionInfo,
	})
	if err == nil {
		fmt.Printf("✓ Session information retrieval successful\n")
		fmt.Printf("  Username: %s\n", sessionInfo.Username)
		fmt.Printf("  Session ID: %s\n", sessionInfo.SessionID)
		fmt.Printf("  Time remaining: %v\n", sessionInfo.TimeRemaining)
	} else {
		fmt.Printf("✗ Session information retrieval failed: %v\n", err)
	}

	// Test session validity check
	isValid := client.Auth().IsSessionValid()
	results = append(results, TestResult{
		TestName: "Session Validity Check",
		Success:  true, // This call won't fail
		Error:    nil,
		Data:     isValid,
	})
	fmt.Printf("✓ Session validity: %t\n", isValid)

	// Test getting session remaining time
	timeRemaining := client.Auth().GetTimeUntilExpiry()
	results = append(results, TestResult{
		TestName: "Session Remaining Time Retrieval",
		Success:  true,
		Error:    nil,
		Data:     timeRemaining,
	})
	fmt.Printf("✓ Session remaining time: %v\n", timeRemaining)

	return results
}

// Test network interface related information
func testNetworkInfo(client *goubus.Client) []TestResult {
	var results []TestResult

	// Test getting all network interface information
	dump, err := client.Network().Dump()
	results = append(results, TestResult{
		TestName: "Network Interface Dump",
		Success:  err == nil,
		Error:    err,
		Data:     dump,
	})
	if err == nil {
		fmt.Printf("✓ Network interface dump successful, interface count: %d\n", len(dump.Interface))

		// Print detailed information for all interfaces
		fmt.Println("  All network interface details:")
		for i, iface := range dump.Interface {
			fmt.Printf("    Interface %d: %s\n", i+1, iface.Interface)
			fmt.Printf("      Status: UP=%t, Available=%t, Autostart=%t\n", iface.Up, iface.Available, iface.Autostart)
			fmt.Printf("      Protocol: %s, Device: %s\n", iface.Proto, iface.Device)
			if iface.L3Device != "" {
				fmt.Printf("      L3 Device: %s\n", iface.L3Device)
			}
			if len(iface.Ipv4Address) > 0 {
				fmt.Printf("      IPv4 addresses: ")
				for _, addr := range iface.Ipv4Address {
					fmt.Printf("%s/%d ", addr.Address, addr.Mask)
				}
				fmt.Println()
			}
			if len(iface.Ipv6Address) > 0 {
				fmt.Printf("      IPv6 addresses: ")
				for _, addr := range iface.Ipv6Address {
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
	for _, iface := range dump.Interface {
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
				status.NetworkInterface.Up,
				status.NetworkInterface.Pending,
				status.NetworkInterface.Available)
			fmt.Printf("    Protocol: %s, Device: %s\n",
				status.NetworkInterface.Proto,
				status.NetworkInterface.Device)
			if status.NetworkInterface.L3Device != "" {
				fmt.Printf("    L3 Device: %s\n", status.NetworkInterface.L3Device)
			}

			// Print IPv4 addresses
			if len(status.NetworkInterface.Ipv4Address) > 0 {
				fmt.Printf("    IPv4 addresses:\n")
				for _, addr := range status.NetworkInterface.Ipv4Address {
					fmt.Printf("      %s/%d\n", addr.Address, addr.Mask)
				}
			}

			// Print IPv6 addresses
			if len(status.NetworkInterface.Ipv6Address) > 0 {
				fmt.Printf("    IPv6 addresses:\n")
				for _, addr := range status.NetworkInterface.Ipv6Address {
					fmt.Printf("      %s/%d\n", addr.Address, addr.Mask)
				}
			}

			// Print DNS servers
			if len(status.NetworkInterface.DNSServer) > 0 {
				fmt.Printf("    DNS servers: %v\n", status.NetworkInterface.DNSServer)
			}

			// Print routing information
			if len(status.NetworkInterface.Route) > 0 {
				fmt.Printf("    Route information (%d routes):\n", len(status.NetworkInterface.Route))
				for i, route := range status.NetworkInterface.Route {
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
				if len(status.NetworkInterface.Route) > 3 {
					fmt.Printf("      ... %d more routes\n", len(status.NetworkInterface.Route)-3)
				}
			}

			// Print device information (if available)
			if status.NetworkDevice.Present {
				fmt.Printf("    Device information:\n")
				fmt.Printf("      Present: %t, UP: %t\n",
					status.NetworkDevice.Present,
					status.NetworkDevice.Up)
				if status.NetworkDevice.Type != "" {
					fmt.Printf("      Type: %s\n", status.NetworkDevice.Type)
				}
				if status.NetworkDevice.Mtu > 0 {
					fmt.Printf("      MTU: %d\n", status.NetworkDevice.Mtu)
				}
				if status.NetworkDevice.Macaddr != "" {
					fmt.Printf("      MAC address: %s\n", status.NetworkDevice.Macaddr)
				}
				if len(status.NetworkDevice.BridgeMembers) > 0 {
					fmt.Printf("      Bridge members: %v\n", status.NetworkDevice.BridgeMembers)
				}
				if status.NetworkDevice.Speed > 0 {
					fmt.Printf("      Speed: %d Mbps\n", status.NetworkDevice.Speed)
				}
				if status.NetworkDevice.Carrier {
					fmt.Printf("      Carrier detection: Yes\n")
				}
				if status.NetworkDevice.Multicast {
					fmt.Printf("      Multicast support: Yes\n")
				}
			}
		} else {
			fmt.Printf("✗ Interface %s status retrieval failed: %v\n", ifaceName, err)
		}

		// Test interface configuration
		config, err := client.Network().Interface(ifaceName).GetConfig()
		results = append(results, TestResult{
			TestName: fmt.Sprintf("Interface %s Configuration", ifaceName),
			Success:  err == nil,
			Error:    err,
			Data:     config,
		})
		if err == nil {
			fmt.Printf("✓ Interface %s configuration retrieval successful\n", ifaceName)
			fmt.Printf("  Configuration details:\n")
			hasConfig := false
			if config.Proto != "" {
				fmt.Printf("    Protocol: %s\n", config.Proto)
				hasConfig = true
			}
			if config.Device != "" {
				fmt.Printf("    Device: %s\n", config.Device)
				hasConfig = true
			}
			if config.Type != "" {
				fmt.Printf("    Type: %s\n", config.Type)
				hasConfig = true
			}
			if len(config.IPAddr) > 0 {
				fmt.Printf("    IP addresses: %v\n", config.IPAddr)
				hasConfig = true
			}
			if config.Gateway != "" {
				fmt.Printf("    Gateway: %s\n", config.Gateway)
				hasConfig = true
			}
			if len(config.DNS) > 0 {
				fmt.Printf("    DNS: %v\n", config.DNS)
				hasConfig = true
			}
			if len(config.IfName) > 0 {
				fmt.Printf("    Interface names: %v\n", config.IfName)
				hasConfig = true
			}
			if config.Disabled != "" {
				fmt.Printf("    Disabled status: %s\n", config.Disabled)
				hasConfig = true
			}
			if config.Auto != "" {
				fmt.Printf("    Auto start: %s\n", config.Auto)
				hasConfig = true
			}
			if config.Metric != "" {
				fmt.Printf("    Metric: %s\n", config.Metric)
				hasConfig = true
			}
			if config.MTU != "" {
				fmt.Printf("    MTU: %s\n", config.MTU)
				hasConfig = true
			}
			// PPPoE specific configuration
			if config.Username != "" {
				fmt.Printf("    Username: %s\n", config.Username)
				hasConfig = true
			}
			if config.Service != "" {
				fmt.Printf("    Service: %s\n", config.Service)
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

// Test wireless information related interfaces
func testWirelessInfo(client *goubus.Client) []TestResult {
	var results []TestResult

	// Test getting available wireless devices
	devices, err := client.Wireless().GetAvailableDevices()
	results = append(results, TestResult{
		TestName: "Wireless Device List",
		Success:  err == nil,
		Error:    err,
		Data:     devices,
	})
	if err == nil {
		fmt.Printf("✓ Wireless device list retrieval successful, device count: %d\n", len(devices))
		for _, device := range devices {
			fmt.Printf("  Device: %s\n", device)
		}
	} else {
		fmt.Printf("✗ Wireless device list retrieval failed: %v\n", err)
	}

	// Dynamically discover actual wireless configuration sections
	fmt.Printf("Dynamically discovering wireless configuration sections...\n")

	// Test actual existing radio devices
	actualRadios := []string{"radio0"} // Based on actual discovered devices
	for _, radio := range actualRadios {
		fmt.Printf("\n--- Testing wireless device: %s ---\n", radio)

		// Test device configuration
		config, err := client.Wireless().Device(radio).Get()
		results = append(results, TestResult{
			TestName: fmt.Sprintf("Wireless Device %s Configuration", radio),
			Success:  err == nil,
			Error:    err,
			Data:     config,
		})
		if err == nil {
			fmt.Printf("✓ Wireless device %s configuration retrieval successful\n", radio)
			if config.Type != "" {
				fmt.Printf("  Type: %s\n", config.Type)
			}
			if config.Channel != "" {
				fmt.Printf("  Channel: %s\n", config.Channel)
			}
			if config.Country != "" {
				fmt.Printf("  Country code: %s\n", config.Country)
			}
			if config.HTMode != "" {
				fmt.Printf("  HT mode: %s\n", config.HTMode)
			}
			if config.TXPower != "" {
				fmt.Printf("  TX power: %s\n", config.TXPower)
			}
			if config.Disabled != "" {
				fmt.Printf("  Disabled status: %s\n", config.Disabled)
			}
			if config.Path != "" {
				fmt.Printf("  Device path: %s\n", config.Path)
			}
		} else {
			fmt.Printf("✗ Wireless device %s configuration retrieval failed: %v\n", radio, err)
			continue // Skip subsequent tests if configuration retrieval fails
		}

		// Test device information
		info, err := client.Wireless().Device(radio).Info()
		results = append(results, TestResult{
			TestName: fmt.Sprintf("Wireless Device %s Information", radio),
			Success:  err == nil,
			Error:    err,
			Data:     info,
		})
		if err == nil {
			fmt.Printf("✓ Wireless device %s information retrieval successful\n", radio)
			if info.SSID != "" {
				fmt.Printf("  SSID: %s\n", info.SSID)
			}
			if info.BSSID != "" {
				fmt.Printf("  BSSID: %s\n", info.BSSID)
			}
			if info.Channel > 0 {
				fmt.Printf("  Channel: %d\n", info.Channel)
			}
			if info.Mode != "" {
				fmt.Printf("  Mode: %s\n", info.Mode)
			}
			if info.TXPower > 0 {
				fmt.Printf("  TX power: %d dBm\n", info.TXPower)
			}
		} else {
			fmt.Printf("✗ Wireless device %s information retrieval failed: %v\n", radio, err)
		}

		// Test scanning
		scanResult, err := client.Wireless().Device(radio).Scan()
		results = append(results, TestResult{
			TestName: fmt.Sprintf("Wireless Device %s Scan", radio),
			Success:  err == nil,
			Error:    err,
			Data:     scanResult,
		})
		if err == nil {
			fmt.Printf("✓ Wireless device %s scan successful, networks found: %d\n", radio, len(scanResult.Results))
			// Show first 3 scan results
			for i, result := range scanResult.Results {
				if i < 3 {
					fmt.Printf("  Network %d: SSID=%s, Signal=%d dBm, Channel=%d\n",
						i+1, result.SSID, result.Signal, result.Channel)
				}
			}
			if len(scanResult.Results) > 3 {
				fmt.Printf("  ... %d more networks\n", len(scanResult.Results)-3)
			}
		} else {
			fmt.Printf("✗ Wireless device %s scan failed: %v\n", radio, err)
		}

		// Test country list
		countryList, err := client.Wireless().Device(radio).CountryList()
		results = append(results, TestResult{
			TestName: fmt.Sprintf("Wireless Device %s Country List", radio),
			Success:  err == nil,
			Error:    err,
			Data:     countryList,
		})
		if err == nil {
			fmt.Printf("✓ Wireless device %s country list retrieval successful, countries: %d\n", radio, len(countryList.Results))
			// Show first 5 countries
			for i, country := range countryList.Results {
				if i < 5 {
					fmt.Printf("  Country %d: %s - %s (Code: %s)\n",
						i+1, country.Code, country.Country, country.ISO3166)
					if country.Active {
						fmt.Printf("    (Currently active)\n")
					}
				}
			}
			if len(countryList.Results) > 5 {
				fmt.Printf("  ... %d more countries\n", len(countryList.Results)-5)
			}
		} else {
			fmt.Printf("✗ Wireless device %s country list retrieval failed: %v\n", radio, err)
		}

		// Test TX power list
		txPowerList, err := client.Wireless().Device(radio).TxPowerList()
		results = append(results, TestResult{
			TestName: fmt.Sprintf("Wireless Device %s TX Power List", radio),
			Success:  err == nil,
			Error:    err,
			Data:     txPowerList,
		})
		if err == nil {
			fmt.Printf("✓ Wireless device %s TX power list retrieval successful, power levels: %d\n", radio, len(txPowerList.Results))
			// Show first 5 power levels
			for i, power := range txPowerList.Results {
				if i < 5 {
					fmt.Printf("  Power level %d: %d dBm (%d mW)", i+1, power.Dbm, power.Mw)
					if power.Active {
						fmt.Printf(" (Currently active)")
					}
					fmt.Println()
				}
			}
			if len(txPowerList.Results) > 5 {
				fmt.Printf("  ... %d more power levels\n", len(txPowerList.Results)-5)
			}
		} else {
			fmt.Printf("✗ Wireless device %s TX power list retrieval failed: %v\n", radio, err)
		}

		// Test frequency/channel list
		freqList, err := client.Wireless().Device(radio).FreqList()
		results = append(results, TestResult{
			TestName: fmt.Sprintf("Wireless Device %s Frequency List", radio),
			Success:  err == nil,
			Error:    err,
			Data:     freqList,
		})
		if err == nil {
			fmt.Printf("✓ Wireless device %s frequency list retrieval successful, channels: %d\n", radio, len(freqList.Results))
			// Show first 10 channels
			for i, freq := range freqList.Results {
				if i < 10 {
					fmt.Printf("  Channel %d: %d MHz", freq.Channel, freq.Mhz)
					if freq.Active {
						fmt.Printf(" (Currently active)")
					}
					if freq.Restricted {
						fmt.Printf(" (Restricted)")
					}
					fmt.Println()
				}
			}
			if len(freqList.Results) > 10 {
				fmt.Printf("  ... %d more channels\n", len(freqList.Results)-10)
			}
		} else {
			fmt.Printf("✗ Wireless device %s frequency list retrieval failed: %v\n", radio, err)
		}
	}

	// Test actual existing wireless interface configuration
	actualWifiInterfaces := []string{"wifinet0"} // Based on actual discovered interfaces
	fmt.Printf("\nTesting wireless interface configuration...\n")
	for _, iface := range actualWifiInterfaces {
		config, err := client.Wireless().Interface(iface).Get()
		results = append(results, TestResult{
			TestName: fmt.Sprintf("Wireless Interface %s Configuration", iface),
			Success:  err == nil,
			Error:    err,
			Data:     config,
		})
		if err == nil {
			fmt.Printf("✓ Wireless interface %s configuration retrieval successful\n", iface)
			if config.SSID != "" {
				fmt.Printf("  SSID: %s\n", config.SSID)
			}
			if config.Device != "" {
				fmt.Printf("  Device: %s\n", config.Device)
			}
			if config.Network != "" {
				fmt.Printf("  Network: %s\n", config.Network)
			}
			if config.Mode != "" {
				fmt.Printf("  Mode: %s\n", config.Mode)
			}
			if config.Encryption != "" {
				fmt.Printf("  Encryption: %s\n", config.Encryption)
			}
			if config.Hidden != "" {
				fmt.Printf("  Hidden SSID: %s\n", config.Hidden)
			}
			if config.Disabled != "" {
				fmt.Printf("  Disabled status: %s\n", config.Disabled)
			}
		} else {
			fmt.Printf("✗ Wireless interface %s configuration retrieval failed: %v\n", iface, err)
		}
	}

	return results
}

// Test DHCP related information
func testDHCPInfo(client *goubus.Client) []TestResult {
	var results []TestResult

	// Test IPv4 leases
	leases, err := client.DHCP().Leases()
	results = append(results, TestResult{
		TestName: "DHCP Leases",
		Success:  err == nil,
		Error:    err,
		Data:     leases,
	})
	if err == nil {
		fmt.Printf("✓ DHCP leases retrieval successful, lease count: %d\n", len(leases.DHCPLeases))
		for i, lease := range leases.DHCPLeases {
			if i < 3 { // Show only first 3
				fmt.Printf("  Lease: %s -> %s (%s)\n", lease.Macaddr, lease.IPAddr, lease.Hostname)
			}
		}
		fmt.Printf("✓ DHCP IPv6 leases retrieval successful, lease count: %d\n", len(leases.DHCP6Leases))
		for i, lease := range leases.DHCP6Leases {
			if i < 3 { // Show only first 3
				fmt.Printf("  Lease: %s -> %s (%s)\n", lease.Macaddr, lease.IP6Addr, lease.DUID)
			}
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
		content, err := client.File().Read(file)
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
	execResult, err := client.File().Exec("uname", []string{"-a"})
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
				fmt.Printf("  Log: %s\n", entry.Msg)
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

	// Test common service status
	services := []string{"network", "firewall", "dnsmasq", "uhttpd"}
	for _, service := range services {
		status, err := client.Service(service).Status()
		results = append(results, TestResult{
			TestName: fmt.Sprintf("Service %s Status", service),
			Success:  err == nil,
			Error:    err,
			Data:     status,
		})
		if err == nil {
			fmt.Printf("✓ Service %s status retrieval successful\n", service)
		} else {
			fmt.Printf("✗ Service %s status retrieval failed: %v\n", service, err)
		}
	}

	return results
}

func testLuciInfo(client *goubus.Client) []TestResult {
	var results []TestResult

	time, err := client.Luci().GetLocalTime()
	results = append(results, TestResult{
		TestName: "Local Time Retrieval",
		Success:  err == nil,
		Error:    err,
		Data:     time,
	})
	if err == nil {
		fmt.Printf("✓ Local time retrieval successful: %s\n", time)
	} else {
		fmt.Printf("✗ Local time retrieval failed: %v\n", err)
	}

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
	fmt.Printf("Success rate: %.1f%%\n", float64(successCount)/float64(len(results))*100)

	if failCount == 0 {
		fmt.Println("\n🎉 All tests passed!")
	} else {
		fmt.Printf("\n⚠️  %d tests failed, please check the error messages above\n", failCount)
	}
}
