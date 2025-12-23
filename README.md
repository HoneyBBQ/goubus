# goubus: Go Client Library for OpenWrt ubus

[![Go Version](https://img.shields.io/badge/go-1.24-blue)](https://golang.org/)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/honeybbq/goubus)](https://goreportcard.com/report/github.com/honeybbq/goubus)

[Read this document in Chinese (中文文档)](README_CN.md)

A Go client library for OpenWrt's ubus (micro bus) system. Supports both HTTP JSON-RPC and native Unix socket transports with type-safe APIs for system management, network configuration, and device control.

## Table of Contents

- [goubus: Go Client Library for OpenWrt ubus](#goubus-go-client-library-for-openwrt-ubus)
  - [Table of Contents](#table-of-contents)
  - [Features](#features)
  - [Architecture](#architecture)
  - [Installation](#installation)
  - [Quick Start](#quick-start)
    - [Remote Access (HTTP JSON-RPC)](#remote-access-http-json-rpc)
    - [Local Access (Unix Socket)](#local-access-unix-socket)
  - [API Usage Examples](#api-usage-examples)
    - [**1. System Management**](#1-system-management)
    - [**2. Network Status \& Control**](#2-network-status--control)
    - [**3. UCI Configuration Management**](#3-uci-configuration-management)
      - [Chained API](#chained-api)
      - [Configuration Models](#configuration-models)
      - [Example: Modifying Network Configuration](#example-modifying-network-configuration)
    - [**4. Wireless (IwInfo \& Network.Wireless)**](#4-wireless-iwinfo--networkwireless)
    - [**5. DHCP Service**](#5-dhcp-service)
    - [**6. Files \& Commands**](#6-files--commands)
    - [**7. Service Management (RC \& Service)**](#7-service-management-rc--service)
    - [**8. Logging System**](#8-logging-system)
    - [**9. Sessions \& Permissions**](#9-sessions--permissions)
    - [**10. LuCI Extension Interface**](#10-luci-extension-interface)
  - [Troubleshooting](#troubleshooting)
    - [Permission Issues](#permission-issues)
      - [**Example 1: Complete Network Management Access**](#example-1-complete-network-management-access)
      - [**Example 2: Comprehensive System Administrator Access**](#example-2-comprehensive-system-administrator-access)
      - [**Assign ACL Roles to Users**](#assign-acl-roles-to-users)
      - [**Apply Changes**](#apply-changes)
  - [License](#license)
  - [Acknowledgments](#acknowledgments)
  - [Related Resources](#related-resources)

## Features

- **Dual Transport**: HTTP JSON-RPC for remote access, Unix socket for local operations
- **Type-Safe API**: Structured types for all ubus operations, no `map[string]interface{}`
- **UCI Configuration**: Type-safe models for OpenWrt configuration management
- **Module Coverage**: System, network, wireless, DHCP, services, files, and logging
- **Session Management**: Automatic authentication for HTTP transport
- **Error Handling**: Typed errors matching ubus status codes
- **Concurrency Safe**: Thread-safe for multi-goroutine usage

## Architecture

- **`goubus`**: User-facing API with manager pattern (`client.System()`, `client.Network()`, etc.)
- **`api`**: ubus call construction and response parsing
- **`transport`**: HTTP JSON-RPC or Unix socket communication
- **`types`**: Request/response structures for type safety
- **`errdefs`**: Error types matching ubus status codes

## Installation

```bash
go get github.com/honeybbq/goubus
```

## Quick Start

`goubus` supports two transport modes - choose the one that fits your scenario:

### Remote Access (HTTP JSON-RPC)

For remote management over network:

```go
package main

import (
    "fmt"
    "log"
    "github.com/honeybbq/goubus"
    "github.com/honeybbq/goubus/transport"
)

func main() {
    // Create an HTTP client with authentication credentials
    rpcClient, err := transport.NewRpcClient("192.168.1.1", "root", "password")
    if err != nil {
        log.Fatalf("Failed to connect to device: %v", err)
    }
    client := goubus.NewClient(rpcClient)

    // Fetch system runtime information
    systemInfo, err := client.System().Info()
    if err != nil {
        log.Fatalf("Failed to get system info: %v", err)
    }

    fmt.Printf("System Uptime: %d seconds\n", systemInfo.Uptime)
    fmt.Printf("Memory Usage: %d MB / %d MB\n",
        (systemInfo.Memory.Total-systemInfo.Memory.Free)/1024/1024,
        systemInfo.Memory.Total/1024/1024)

    // Fetch board hardware information
    boardInfo, err := client.System().Board()
    if err != nil {
        log.Fatalf("Failed to get board info: %v", err)
    }
    fmt.Printf("Device Model: %s\n", boardInfo.Release.BoardName)
}
```

### Local Access (Unix Socket)

For on-device applications with direct socket access (no authentication required):

```go
package main

import (
    "fmt"
    "log"
    "github.com/honeybbq/goubus"
    "github.com/honeybbq/goubus/transport"
)

func main() {
    // Create a Unix socket client
    // Empty string uses default path: /tmp/run/ubus/ubus.sock
    socketClient, err := transport.NewSocketClient("")
    if err != nil {
        log.Fatalf("Unable to connect to ubus socket: %v", err)
    }
    client := goubus.NewClient(socketClient)

    // Same API as HTTP transport
    systemInfo, err := client.System().Info()
    if err != nil {
        log.Fatalf("Failed to get system info: %v", err)
    }

    fmt.Printf("System Uptime: %d seconds\n", systemInfo.Uptime)
    
    boardInfo, err := client.System().Board()
    if err != nil {
        log.Fatalf("Failed to get board info: %v", err)
    }
    fmt.Printf("Device Model: %s\n", boardInfo.Release.BoardName)
}
```

**Transport Comparison:**

| Feature | HTTP (JSON-RPC) | Unix Socket |
|---------|----------------|-------------|
| **Use Case** | Remote management | On-device applications |
| **Authentication** | Required (username/password) | Not required |
| **Network** | Requires network access | Direct local access |
| **Performance** | Network overhead | Zero overhead |
| **Default Path** | `http://host/ubus` | `/tmp/run/ubus/ubus.sock` |

**Performance Difference:**

Based on benchmark testing, the Unix Socket transport significantly outperforms HTTP JSON-RPC for local operations:

- **Connection Time**: ~50x faster (sub-millisecond vs ~30ms)
- **Single Call Latency**: ~60x faster (~800µs vs ~50ms)  
- **Throughput**: ~1000 ops/s vs ~20 ops/s

For high-frequency operations or real-time requirements, Unix Socket is strongly recommended when available. You can run your own benchmarks using:

```bash
cd example/benchmark
go run . -n 100  # Test both transports with 100 iterations
```

## API Usage Examples

`goubus` provides a dedicated "manager" for each ubus module, accessible via methods on the client, such as `client.System()`, `client.Network()`, and `client.Uci()`.

### **1. System Management**

Use `client.System()` to get the `SystemManager`.

```go
// Get hardware information
boardInfo, err := client.System().Board()

// Reboot the system
err = client.System().Reboot()
```

### **2. Network Status & Control**

Use `client.Network()` to get the `NetworkManager`. The API design mimics the hierarchical structure of `ubus`.

```go
// Get a summary of all network interfaces
dump, err := client.Network().Interface("").Dump()
for _, iface := range dump {
    fmt.Printf("Interface: %s, Protocol: %s, Up: %t\n", iface.Interface, iface.Proto, iface.Up)
}

// Get the detailed status of the 'lan' interface.
// .Interface("lan") returns an InterfaceManager.
lanStatus, err := client.Network().Interface("lan").Status()
if err == nil && len(lanStatus.IPv4Address) > 0 {
    fmt.Printf("LAN IP Address: %s\n", lanStatus.IPv4Address[0].Address)
}

// Control interface state
err = client.Network().Interface("wan").Down()
// ...
err = client.Network().Interface("wan").Up()

// Reload the network service
err = client.Network().Reload()
```

### **3. UCI Configuration Management**

UCI packages are now exposed as lightweight key/value structures:

- `client.Uci().Package("network").Section("lan").Get()` returns a `*goubus.Section`.
- `Section.Values` stores `map[string][]string`, keeping list semantics intact.
- Use `goubus.SectionValues` helpers to stage updates.

```go
// Inspect a section
sec, err := client.Uci().Package("network").Section("wan").Get()
if err != nil {
    log.Fatalf("Failed to read WAN section: %v", err)
}
proto, _ := sec.Values.First("proto")
fmt.Printf("Original WAN protocol: %s\n", proto)

// Stage a change
values := goubus.NewSectionValues()
values.Set("proto", "static")
values.Set("ipaddr", "192.168.100.2")
values.Set("netmask", "255.255.255.0")
values.Set("gateway", "192.168.100.1")
values.Set("dns", "8.8.8.8", "1.1.1.1")

err = client.Uci().Package("network").Section("wan").SetValues(values)
if err != nil {
    log.Fatalf("Failed to stage WAN config: %v", err)
}

// Commit and reload if desired
_ = client.Uci().Package("network").Commit()
_ = client.Network().Reload()
```

### **4. Wireless (IwInfo & Network.Wireless)**

Wireless operations are split into two parts:

- **`client.IwInfo()`**: For real-time wireless status, like scans and associated client lists. It corresponds to the `iwinfo` command.
- **`client.Uci().Package("wireless")`**: For reading and writing the `/etc/config/wireless` configuration file.

```go
// Get all physical wireless devices (radio0, radio1, ...)
devices, err := client.IwInfo().Devices()
if err != nil || len(devices) == 0 {
    log.Fatal("No wireless devices found")
}

// Scan using the first wireless device
scanResults, err := client.IwInfo().Scan(devices[0])
if err == nil {
    fmt.Printf("Found %d networks on %s:\n", len(scanResults), devices[0])
    for _, net := range scanResults {
        fmt.Printf("  SSID: %s, Signal: %d dBm\n", net.SSID, net.Signal)
    }
}

// Get the list of associated clients
assocList, err := client.IwInfo().AssocList(devices[0])
```

### **5. DHCP Service**

Use `client.DHCP()` to get the `DHCPManager`.

```go
// goubus currently provides an interface for adding static leases.
// Fetching the lease list is typically done via the LuCI interface or by parsing the lease file.
err := client.DHCP().AddLease(types.AddLeaseRequest{
    Mac:  "00:11:22:33:44:55",
    Ip:   "192.168.1.100",
    Name: "my-device",
})
```

### **6. Files & Commands**

Use `client.File()` to get the `FileManager` for file operations and command execution on the device.

```go
// Execute a command
output, err := client.File().Exec("uname", []string{"-a"}, nil)

// Read file contents (returns a base64 encoded string)
fileContent, err := client.File().Read("/etc/os-release", true)

// Write to a file
err = client.File().Write("/tmp/greeting.txt", "SGVsbG8sIGdvdWJ1cyE=", true, 0644, true)

// Get file stats
stats, err := client.File().Stat("/etc/config/network")

// List a directory
files, err := client.File().List("/etc/config")
```

### **7. Service Management (RC & Service)**

- **`client.RC()`**: Corresponds to `/etc/init.d/` scripts for starting, stopping, and restarting services.
- **`client.Service()`**: `ubus`'s more powerful, built-in service manager.

```go
// Restart the network service using rc
err = client.RC().Restart("network")

// Get the status of all services
services, err := client.Service().List("", false)
for name, service := range services {
    running := false
    if len(service.Instances) > 0 {
        // Simplified check; in practice, you should iterate through instances
        running = service.Instances["instance1"].Running
    }
    fmt.Printf("Service: %-15s, Running: %t\n", name, running)
}
```

### **8. Logging System**

Use `client.Log()` to get the `LogManager` for reading from and writing to the system log (`logd`).

```go
// Read the last 50 system log entries
logs, err := client.Log().Read(50, false, true)
for _, entry := range logs.Log {
    t := time.Unix(int64(entry.Time), 0)
    fmt.Printf("[%s] Source:%d Priority:%d %s\n", 
        t.Format("2006-01-02 15:04:05"), 
        entry.Source, 
        entry.Priority,
        entry.Text)
}
```

### **9. Sessions & Permissions**

Use `client.Session()` to get the `SessionManager` for managing ubus session ACLs.

```go
// Create a session with a 300-second timeout
sessionData, err := client.Session().Create(300)

// Grant the session full access to network and uci
err = client.Session().Grant(sessionData.UbusRpcSession, "ubus", []string{"network.*", "uci.*"})
```

### **10. LuCI Extension Interface**

`client.Luci()` provides access to the LuCI RPC interface, which often returns richer, more UI-friendly data than standard `ubus` calls.

```go
// Get more detailed device information than network.interface.dump provides
devices, err := client.Luci().GetNetworkDevices()

// Get DHCP lease information
leases, err := client.Luci().GetDHCPLeases()
if err == nil {
    for _, lease := range leases.IPv4Leases {
        fmt.Printf("Client %s (%s) -> %s\n", lease.Hostname, lease.Macaddr, lease.IPAddr)
    }
}
```

## Troubleshooting

### Permission Issues

Using `ubus` via the SSH command line typically grants full permissions. However, `goubus` accesses ubus via HTTP RPC, which is subject to OpenWrt's ACLs (Access Control Lists). If you encounter "permission denied" errors, you must configure the appropriate permissions for the user you are logging in with.

To resolve permission issues, create or modify ACL configuration files in the `/usr/share/rpcd/acl.d/` directory on your OpenWrt device.

**Note**: The default `root` user typically has full (`*`) permissions, so you can often skip this step if you are connecting as `root`.

#### **Example 1: Complete Network Management Access**

Create `/usr/share/rpcd/acl.d/network-full.json`:

```json
{
    "network-manager": {
        "description": "Full network management access",
        "read": {
            "ubus": {
                "network": ["*"],
                "network.device": ["*"],
                "network.interface": ["*"],
                "network.interface.*": ["*"],
                "network.wireless": ["*"],
                "iwinfo": ["*"]
            },
            "uci": ["*"]
        },
        "write": {
            "ubus": {
                "network": ["*"],
                "network.device": ["*"],
                "network.interface": ["*"],
                "network.interface.*": ["*"],
                "network.wireless": ["*"]
            },
            "uci": ["*"]
        }
    }
}
```

#### **Example 2: Comprehensive System Administrator Access**

Create `/usr/share/rpcd/acl.d/system-admin.json`:

```json
{
    "system-admin": {
        "description": "System administration access",
        "read": {
            "ubus": {
                "system": ["*"],
                "service": ["*"],
                "file": ["*"],
                "network": ["*"],
                "network.device": ["*"],
                "network.interface": ["*"],
                "network.interface.*": ["*"],
                "network.wireless": ["*"],
                "iwinfo": ["*"],
                "dhcp": ["*"],
                "luci-rpc": ["*"]
            },
            "uci": ["*"]
        },
        "write": {
            "ubus": {
                "system": ["*"],
                "service": ["*"],
                "file": ["*"],
                "network": ["*"],
                "network.device": ["*"],
                "network.interface": ["*"],
                "network.interface.*": ["*"],
                "rc": ["*"]
            },
            "uci": ["*"]
        }
    }
}
```

#### **Assign ACL Roles to Users**

After creating the ACL files, assign the role to a user in `/etc/config/rpcd`:

```ini
config login
    option username 'admin'
    option password '$p$admin'
    list read 'system-admin'
    list write 'system-admin'
```

#### **Apply Changes**

After modifying the configuration, restart the `rpcd` service to apply the changes:

```bash
# Restart rpcd service to apply changes
/etc/init.d/rpcd restart
```

**📖 For more details, see the [OpenWrt ubus ACLs documentation](https://openwrt.org/docs/techref/ubus#acls)**

## License

Apache License 2.0 - See [LICENSE](LICENSE) file for details.

## Acknowledgments

Inspired by [Kubernetes client-go](https://github.com/kubernetes/client-go), [moby/moby](https://github.com/moby/moby), and [cdavid14/goubus](https://github.com/cdavid14/goubus).

## Related Resources

- [OpenWrt](https://openwrt.org/) - Linux distribution for embedded devices
- [ubus](https://git.openwrt.org/project/ubus.git) - OpenWrt's micro bus system
- [libubus](https://git.openwrt.org/project/libubus.git) - C library for ubus
