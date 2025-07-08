# goubus: OpenWrt ubus Client Library

[![Go Version](https://img.shields.io/badge/go-1.24-blue)](https://golang.org/)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/honeybbq/goubus)](https://goreportcard.com/report/github.com/honeybbq/goubus)

goubus is a comprehensive Go client library for OpenWrt's ubus (micro bus) system. It provides a type-safe, idiomatic Go interface for interacting with OpenWrt routers and devices, enabling seamless integration of network management, system monitoring, and wireless configuration into Go applications.

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [API Reference](#api-reference)
- [Troubleshooting](#troubleshooting)
- [Examples](#examples)
- [Contributing](#contributing)
- [License](#license)
- [Acknowledgments](#acknowledgments)

## Features

- **Complete API Coverage**: Full support for OpenWrt ubus services including:
  - Authentication and session management
  - Network interface configuration and monitoring
  - Wireless network management
  - DHCP configuration
  - System information and control
  - File system operations
  - Service management
  - Event handling
  - UCI configuration system

- **Type-Safe**: Strongly typed Go structs for all API responses
- **Session Management**: Automatic session handling with refresh and expiry detection
- **Error Handling**: Comprehensive error handling with ubus-specific error codes
- **Concurrent-Safe**: Thread-safe operations for concurrent usage
- **Extensible**: Modular design for easy extension and customization

## Installation

```bash
go get github.com/honeybbq/goubus
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "goubus"
)

func main() {
    // Create a new client
    client, err := goubus.NewClient("192.168.1.1", "root", "password")
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer client.Auth().Logout()

    // Get system information
    systemInfo, err := client.System().Info()
    if err != nil {
        log.Fatalf("Failed to get system info: %v", err)
    }

    fmt.Printf("System uptime: %d seconds\n", systemInfo.Uptime)
    fmt.Printf("Memory usage: %d MB / %d MB\n", 
        (systemInfo.Memory.Total-systemInfo.Memory.Available)/1024/1024,
        systemInfo.Memory.Total/1024/1024)
}
```

## API Reference

### Authentication

```go
// Create a new client (automatically logs in)
client, err := goubus.NewClient("192.168.1.1", "root", "password")

// Get session information
sessionInfo, err := client.Auth().GetSessionInfo()

// Check if session is valid
isValid := client.Auth().IsSessionValid()

// Refresh session
err = client.Auth().Refresh()

// Logout
err = client.Auth().Logout()
```

### System Management

```go
// Get system information
systemInfo, err := client.System().Info()

// Get board information
boardInfo, err := client.System().Board()

// Reboot system
err = client.System().Reboot()
```

### Network Management

```go
// Get all network interfaces
dump, err := client.Network().Dump()

// Get specific interface status
status, err := client.Network().Interface("lan").Status()

// Get interface configuration
config, err := client.Network().Interface("lan").GetConfig()

// Set interface configuration
err = client.Network().Interface("lan").SetConfig(config)
```

### Wireless Management

```go
// Get wireless devices
devices, err := client.Wireless().GetAvailableDevices()

// Get wireless device configuration
config, err := client.Wireless().Device("radio0").Get()

// Set wireless device configuration
err = client.Wireless().Device("radio0").Set(config)

// Get wireless information
info, err := client.Wireless().Device("radio0").Info()

// Scan for networks
scanResults, err := client.Wireless().Device("radio0").Scan()
```

### DHCP Management

```go
// Get DHCP leases
leases, err := client.DHCP().GetLeases()

// Get DHCP IPv4 leases
ipv4Leases, err := client.DHCP().GetIPv4Leases()

// Get DHCP IPv6 leases
ipv6Leases, err := client.DHCP().GetIPv6Leases()
```

### File System Operations

```go
// Execute command
result, err := client.File().Exec("ls", []string{"-la", "/tmp"})

// Read file
content, err := client.File().Read("/etc/config/network")

// Write file
err = client.File().Write("/tmp/test.txt", "Hello World", false, 0644, false)

// Get file stats
stats, err := client.File().Stat("/etc/config/network")

// List directory
files, err := client.File().List("/etc/config")
```

### Service Management

```go
// Get service list
services, err := client.Service("").List()

// Start service
err = client.Service("network").Start()

// Stop service
err = client.Service("network").Stop()

// Restart service
err = client.Service("network").Restart()
```

### Event Handling

```go
// Subscribe to events
handler := func(event goubus.Event) {
    fmt.Printf("Received event: %s\n", event.Type)
}

err = client.Events().Subscribe([]string{"network.interface"}, handler)

// Publish event
err = client.Events().Publish("custom.event", map[string]interface{}{
    "message": "Hello from Go",
})
```

### Logging

```go
// Read system logs
logs, err := client.Log().Read(100, false, true)

// Write to system log
err = client.Log().Write("Application started")
```

## Troubleshooting

### Permission Issues

If you encounter permission issues when accessing certain ubus services, please refer to the official OpenWrt ubus documentation ACLs (Access Control Lists) section:

**📖 [OpenWrt ubus ACLs Documentation](https://openwrt.org/docs/techref/ubus#acls)**

The ACL system controls which users and processes can access specific ubus objects and methods. You may need to configure appropriate ACL rules for your use case.

## Examples

### Complete Network Status Check

```go
package main

import (
    "fmt"
    "log"
    "goubus"
)

func main() {
    client, err := goubus.NewClient("192.168.1.1", "root", "password")
    if err != nil {
        log.Fatalf("Connection failed: %v", err)
    }
    defer client.Auth().Logout()

    // Get system information
    systemInfo, err := client.System().Info()
    if err != nil {
        log.Printf("Failed to get system info: %v", err)
        return
    }

    fmt.Printf("=== System Information ===\n")
    fmt.Printf("Uptime: %d seconds\n", systemInfo.Uptime)
    fmt.Printf("Load: %v\n", systemInfo.Load)
    fmt.Printf("Memory: %d MB total, %d MB available\n",
        systemInfo.Memory.Total/1024/1024,
        systemInfo.Memory.Available/1024/1024)

    // Get network interfaces
    dump, err := client.Network().Dump()
    if err != nil {
        log.Printf("Failed to get network dump: %v", err)
        return
    }

    fmt.Printf("\n=== Network Interfaces ===\n")
    for _, iface := range dump.Interface {
        fmt.Printf("Interface: %s\n", iface.Interface)
        fmt.Printf("  Status: UP=%t, Available=%t\n", iface.Up, iface.Available)
        fmt.Printf("  Protocol: %s\n", iface.Proto)
        
        if len(iface.Ipv4Address) > 0 {
            fmt.Printf("  IPv4: %s/%d\n", 
                iface.Ipv4Address[0].Address,
                iface.Ipv4Address[0].Mask)
        }
        
        if len(iface.DNSServer) > 0 {
            fmt.Printf("  DNS: %v\n", iface.DNSServer)
        }
        fmt.Println()
    }

    // Get DHCP leases
    leases, err := client.DHCP().GetLeases()
    if err != nil {
        log.Printf("Failed to get DHCP leases: %v", err)
        return
    }

    fmt.Printf("=== DHCP Leases ===\n")
    for _, lease := range leases.DHCPLeases {
        fmt.Printf("Device: %s (%s) - %s\n", 
            lease.Hostname, lease.Macaddr, lease.IPAddr)
    }
}
```

### Wireless Network Scanner

```go
package main

import (
    "fmt"
    "log"
    "goubus"
)

func main() {
    client, err := goubus.NewClient("192.168.1.1", "root", "password")
    if err != nil {
        log.Fatalf("Connection failed: %v", err)
    }
    defer client.Auth().Logout()

    // Get available wireless devices
    devices, err := client.Wireless().GetAvailableDevices()
    if err != nil {
        log.Fatalf("Failed to get wireless devices: %v", err)
    }

    for _, device := range devices {
        fmt.Printf("Scanning with device: %s\n", device)
        
        // Scan for networks
        scanResults, err := client.Wireless().Device(device).Scan()
        if err != nil {
            log.Printf("Failed to scan with %s: %v", device, err)
            continue
        }

        fmt.Printf("Found %d networks:\n", len(scanResults.Results))
        for _, result := range scanResults.Results {
            fmt.Printf("  SSID: %s\n", result.SSID)
            fmt.Printf("  Signal: %d dBm\n", result.Signal)
            fmt.Printf("  Channel: %d\n", result.Channel)
            fmt.Printf("  Security: %s\n", result.Encryption.Description)
            fmt.Println()
        }
    }
}
```

### Service Monitor

```go
package main

import (
    "fmt"
    "log"
    "time"
    "goubus"
)

func main() {
    client, err := goubus.NewClient("192.168.1.1", "root", "password")
    if err != nil {
        log.Fatalf("Connection failed: %v", err)
    }
    defer client.Auth().Logout()

    // Monitor critical services
    services := []string{"network", "wireless", "dhcp", "firewall"}
    
    for {
        fmt.Printf("=== Service Status at %s ===\n", time.Now().Format("15:04:05"))
        
        for _, serviceName := range services {
            // Note: This would require implementing service status checking
            // For now, we'll just show the concept
            fmt.Printf("Service %s: Monitoring...\n", serviceName)
        }
        
        fmt.Println()
        time.Sleep(30 * time.Second)
    }
}
```

## Contributing

We welcome contributions to goubus! Please see our [Contributing Guide](CONTRIBUTING.md) for details on how to get started.

### Development Setup

1. Fork the repository
2. Clone your fork: `git clone https://github.com/honeybbq/goubus.git`
3. Create a feature branch: `git checkout -b feature/your-feature`
4. Make your changes and add tests
5. Run tests: `go test ./...`
6. Submit a pull request

### Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Add comprehensive tests for new features
- Document public APIs with clear comments

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

### Inspiration Sources

This project was inspired by:
- **[Kubernetes SDK](https://github.com/kubernetes/client-go)**: For its clean API design and comprehensive approach to client libraries
- **[cdavid14/goubus](https://github.com/cdavid14/goubus)**: For the foundational ubus integration concepts and initial implementation ideas

### Special Thanks

- The OpenWrt development team for creating the ubus system
- The Go community for excellent tooling and libraries
- Contributors who helped improve this library

## Related Projects

- [OpenWrt](https://openwrt.org/) - The Linux distribution for embedded devices
- [ubus](https://git.openwrt.org/project/ubus.git) - OpenWrt micro bus architecture
- [libubus](https://git.openwrt.org/project/libubus.git) - C library for ubus

---

For more information, please open an [issue](https://github.com/honeybbq/goubus/issues) if you need help. 