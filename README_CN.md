# goubus: OpenWrt ubus Go 客户端库

[![Go Version](https://img.shields.io/badge/go-1.24-blue)](https://golang.org/)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/honeybbq/goubus)](https://goreportcard.com/report/github.com/honeybbq/goubus)

OpenWrt ubus 系统的 Go 客户端库。支持 HTTP JSON-RPC 和原生 Unix socket 两种传输方式，提供类型安全的 API 用于系统管理、网络配置和设备控制。

## 目录

- [goubus: OpenWrt ubus Go 客户端库](#goubus-openwrt-ubus-go-客户端库)
  - [目录](#目录)
  - [特性](#特性)
  - [架构](#架构)
  - [安装](#安装)
  - [快速开始](#快速开始)
    - [远程访问（HTTP JSON-RPC）](#远程访问http-json-rpc)
    - [本地访问（Unix Socket）](#本地访问unix-socket)
  - [API 使用示例](#api-使用示例)
    - [**1. 系统管理 (System)**](#1-系统管理-system)
    - [**2. 网络状态与控制 (Network)**](#2-网络状态与控制-network)
    - [**3. UCI 配置管理**](#3-uci-配置管理)
      - [链式 API](#链式-api)
      - [配置模型](#配置模型)
      - [示例：修改网络配置](#示例修改网络配置)
    - [**4. 无线网络 (IwInfo \& Network.Wireless)**](#4-无线网络-iwinfo--networkwireless)
    - [**5. DHCP 服务**](#5-dhcp-服务)
    - [**6. 文件与命令 (File)**](#6-文件与命令-file)
    - [**7. 服务管理 (RC \& Service)**](#7-服务管理-rc--service)
    - [**8. 日志系统 (Log)**](#8-日志系统-log)
    - [**9. 会话与权限 (Session)**](#9-会话与权限-session)
    - [**10. LuCI 扩展接口**](#10-luci-扩展接口)
  - [问题排查](#问题排查)
    - [权限问题](#权限问题)
      - [**示例 1: 完整的网络管理权限**](#示例-1-完整的网络管理权限)
      - [**示例 2: 综合的系统管理员权限**](#示例-2-综合的系统管理员权限)
      - [**为用户分配 ACL 角色**](#为用户分配-acl-角色)
      - [**应用变更**](#应用变更)
  - [许可](#许可)
  - [致谢](#致谢)
  - [相关资源](#相关资源)

## 特性

- **双传输支持**：HTTP JSON-RPC 用于远程访问，Unix socket 用于本地操作
- **类型安全**：所有 ubus 操作都有结构化类型，无需 `map[string]interface{}`
- **UCI 配置管理**：类型安全的 OpenWrt 配置模型
- **模块覆盖**：系统、网络、无线、DHCP、服务、文件和日志
- **会话管理**：HTTP 传输自动处理认证
- **错误处理**：类型化错误对应 ubus 状态码
- **并发安全**：支持多 Goroutine 使用

## 架构

- **`goubus`**：用户 API，管理器模式（`client.System()`, `client.Network()` 等）
- **`api`**：ubus 调用构造和响应解析
- **`transport`**：HTTP JSON-RPC 或 Unix socket 通信
- **`types`**：请求/响应结构，类型安全核心
- **`errdefs`**：错误类型对应 ubus 状态码

## 安装

```bash
go get github.com/honeybbq/goubus
```

## 快速开始

`goubus` 支持两种传输模式，根据使用场景选择：

### 远程访问（HTTP JSON-RPC）

通过网络远程管理：

```go
package main

import (
    "fmt"
    "log"
    "github.com/honeybbq/goubus"
    "github.com/honeybbq/goubus/transport"
)

func main() {
    // 创建 HTTP 客户端，需要认证凭据
    rpcClient, err := transport.NewRpcClient("192.168.1.1", "root", "password")
    if err != nil {
        log.Fatalf("无法连接到设备: %v", err)
    }
    client := goubus.NewClient(rpcClient)

    // 获取系统运行时信息
    systemInfo, err := client.System().Info()
    if err != nil {
        log.Fatalf("无法获取系统信息: %v", err)
    }

    fmt.Printf("系统运行时间: %d 秒\n", systemInfo.Uptime)
    fmt.Printf("内存使用: %d MB / %d MB\n",
        (systemInfo.Memory.Total-systemInfo.Memory.Free)/1024/1024,
        systemInfo.Memory.Total/1024/1024)

    // 获取硬件板信息
    boardInfo, err := client.System().Board()
    if err != nil {
        log.Fatalf("无法获取板信息: %v", err)
    }
    fmt.Printf("设备型号: %s\n", boardInfo.Release.BoardName)
}
```

### 本地访问（Unix Socket）

设备上直接通过 socket 访问（无需认证）：

```go
package main

import (
    "fmt"
    "log"
    "github.com/honeybbq/goubus"
    "github.com/honeybbq/goubus/transport"
)

func main() {
    // 创建 Unix socket 客户端
    // 空字符串使用默认路径: /tmp/run/ubus/ubus.sock
    socketClient, err := transport.NewSocketClient("")
    if err != nil {
        log.Fatalf("无法连接到 ubus socket: %v", err)
    }
    client := goubus.NewClient(socketClient)

    // API 与 HTTP 传输完全相同
    systemInfo, err := client.System().Info()
    if err != nil {
        log.Fatalf("无法获取系统信息: %v", err)
    }

    fmt.Printf("系统运行时间: %d 秒\n", systemInfo.Uptime)
    
    boardInfo, err := client.System().Board()
    if err != nil {
        log.Fatalf("无法获取板信息: %v", err)
    }
    fmt.Printf("设备型号: %s\n", boardInfo.Release.BoardName)
}
```

**传输方式对比：**

| 特性 | HTTP (JSON-RPC) | Unix Socket |
|------|----------------|-------------|
| **使用场景** | 远程管理 | 设备本地应用 |
| **认证** | 需要（用户名/密码） | 不需要 |
| **网络** | 需要网络访问 | 直接本地访问 |
| **性能** | 有网络开销 | 零开销 |
| **默认路径** | `http://host/ubus` | `/tmp/run/ubus/ubus.sock` |

**性能差异：**

根据基准测试，Unix Socket 传输在本地操作中显著优于 HTTP JSON-RPC：

- **连接时间**：快约 50 倍（亚毫秒级 vs 约 30ms）
- **单次调用延迟**：快约 60 倍（约 800µs vs 约 50ms）
- **吞吐量**：约 1000 次操作/秒 vs 约 20 次操作/秒

对于高频操作或实时性要求高的场景，强烈推荐优先使用 Unix Socket（如果可用）。你可以使用以下命令运行性能测试：

```bash
cd example/benchmark
go run . -n 100  # 测试两种传输方式，每种 100 次迭代
```

## API 使用示例

`goubus` 为每个 ubus 模块提供了一个专属的“管理器”，通过 `client` 的方法进行访问，例如 `client.System()`、`client.Network()`、`client.Uci()`。

### **1. 系统管理 (System)**

使用 `client.System()` 获取 `SystemManager`。

```go
// 获取硬件信息
boardInfo, err := client.System().Board()

// 重启系统
err = client.System().Reboot()
```

### **2. 网络状态与控制 (Network)**

使用 `client.Network()` 获取 `NetworkManager`。API 设计模仿了 `ubus` 的层级结构。

```go
// 获取所有网络接口的摘要信息
dump, err := client.Network().Interface("").Dump()
for _, iface := range dump {
    fmt.Printf("接口: %s, 协议: %s, 状态: %t\n", iface.Interface, iface.Proto, iface.Up)
}

// 获取 'lan' 接口的详细状态
// .Interface("lan") 返回一个 InterfaceManager
lanStatus, err := client.Network().Interface("lan").Status()
if err == nil && len(lanStatus.IPv4Address) > 0 {
    fmt.Printf("LAN IP 地址: %s\n", lanStatus.IPv4Address[0].Address)
}

// 控制接口状态
err = client.Network().Interface("wan").Down()
// ...
err = client.Network().Interface("wan").Up()

// 重新加载网络服务
err = client.Network().Reload()
```

### **3. UCI 配置管理**

UCI 配置现在以轻量的 KV 形式呈现：

- `Section.Values` 是 `map[string][]string`，原生保留 list 语义。
- 通过 `goubus.NewSectionValues()` 构造更新数据。

```go
// 读取 wan 配置
sec, err := client.Uci().Package("network").Section("wan").Get()
if err != nil {
    log.Fatalf("读取 WAN 配置失败: %v", err)
}
proto, _ := sec.Values.First("proto")
fmt.Printf("当前 WAN 协议: %s\n", proto)

// 构造待写入的 KV
values := goubus.NewSectionValues()
values.Set("proto", "static")
values.Set("ipaddr", "192.168.100.2")
values.Set("netmask", "255.255.255.0")
values.Set("gateway", "192.168.100.1")
values.Set("dns", "8.8.8.8", "1.1.1.1")

if err := client.Uci().Package("network").Section("wan").SetValues(values); err != nil {
    log.Fatalf("设置 WAN 配置失败: %v", err)
}

// 可选：提交并重载
_ = client.Uci().Package("network").Commit()
_ = client.Network().Reload()
```

### **4. 无线网络 (IwInfo & Network.Wireless)**

无线相关的操作分为两部分：

- **`client.IwInfo()`**：用于获取实时的无线状态，如扫描、关联客户端列表等。它对应 `iwinfo` 命令。
- **`client.Uci().Package("wireless")`**: 用于读写 `/etc/config/wireless` 配置文件。

```go
// 获取所有无线物理设备 (radio0, radio1, ...)
devices, err := client.IwInfo().Devices()
if err != nil || len(devices) == 0 {
    log.Fatal("未找到无线设备")
}

// 使用第一个无线设备进行扫描
scanResults, err := client.IwInfo().Scan(devices[0])
if err == nil {
    fmt.Printf("在 %s 上扫描到 %d 个网络:\n", devices[0], len(scanResults))
    for _, net := range scanResults {
        fmt.Printf("  SSID: %s, 信号: %d dBm\n", net.SSID, net.Signal)
    }
}

// 获取关联的客户端列表
assocList, err := client.IwInfo().AssocList(devices[0])
```

### **5. DHCP 服务**

使用 `client.DHCP()` 获取 `DHCPManager`。

```go
// 目前 goubus 提供了添加静态租约的接口
// 获取租约列表通常通过 luci 接口或解析租约文件
err := client.DHCP().AddLease(types.AddLeaseRequest{
    Mac:  "00:11:22:33:44:55",
    Ip:   "192.168.1.100",
    Name: "my-device",
})
```

### **6. 文件与命令 (File)**

使用 `client.File()` 获取 `FileManager`，可以在设备上进行文件操作和命令执行。

```go
// 执行命令
output, err := client.File().Exec("uname", []string{"-a"}, nil)

// 读取文件内容 (返回 base64 编码的字符串)
fileContent, err := client.File().Read("/etc/os-release", true)

// 写文件
err = client.File().Write("/tmp/greeting.txt", "SGVsbG8sIGdvdWJ1cyE=", true, 0644, true)

// 获取文件状态
stats, err := client.File().Stat("/etc/config/network")

// 列出目录
files, err := client.File().List("/etc/config")
```

### **7. 服务管理 (RC & Service)**

- **`client.RC()`**: 对应 `/etc/init.d/` 脚本，用于启动、停止、重启服务。
- **`client.Service()`**: `ubus` 内置的服务管理器，功能更强大。

```go
// 使用 rc 重启网络服务
err = client.RC().Restart("network")

// 获取所有服务的状态
services, err := client.Service().List("", false)
for name, service := range services {
    running := false
    if len(service.Instances) > 0 {
        // 简化判断，实际应遍历 instances
        running = service.Instances["instance1"].Running
    }
    fmt.Printf("服务: %-15s, 运行中: %t\n", name, running)
}
```

### **8. 日志系统 (Log)**

使用 `client.Log()` 获取 `LogManager` 来读写系统日志 (`logd`)。

```go
// 读取最近 50 条系统日志
logs, err := client.Log().Read(50, false, true)
for _, entry := range logs.Log {
    t := time.Unix(int64(entry.Time), 0)
    fmt.Printf("[%s] 源:%d 优先级:%d %s\n", 
        t.Format("2006-01-02 15:04:05"), 
        entry.Source, 
        entry.Priority,
        entry.Text)
}
```

### **9. 会话与权限 (Session)**

使用 `client.Session()` 获取 `SessionManager`，可以管理 ubus 会话的 ACL 权限。

```go
// 创建一个有效期为 300 秒的会话
sessionData, err := client.Session().Create(300)

// 为该会话授予对 network 和 uci 的完全访问权限
err = client.Session().Grant(sessionData.UbusRpcSession, "ubus", []string{"network.*", "uci.*"})
```

### **10. LuCI 扩展接口**

`client.Luci()` 提供了对 LuCI RPC 接口的访问，这些接口通常返回比标准 `ubus` 更丰富、更适合 UI 展示的数据。

```go
// 获取比 network.interface.dump 更详细的设备信息
devices, err := client.Luci().GetNetworkDevices()

// 获取 DHCP 租约信息
leases, err := client.Luci().GetDHCPLeases()
if err == nil {
    for _, lease := range leases.IPv4Leases {
        fmt.Printf("客户端 %s (%s) -> %s\n", lease.Hostname, lease.Macaddr, lease.IPAddr)
    }
}
```

## 问题排查

### 权限问题

通过 SSH 命令行使用 `ubus` 通常拥有完全权限，但 `goubus` 通过 HTTP RPC 访问，会受到 OpenWrt 的 ACL（访问控制列表）限制。如果遇到“permission denied” (权限被拒绝) 的错误，您必须为登录的用户配置相应的访问权限。

要解决权限问题，请在您的 OpenWrt 设备上创建或修改位于 `/usr/share/rpcd/acl.d/` 目录下的 ACL 配置文件。

**请注意**：默认的 `root` 用户通常拥有完全 (`*`) 权限，因此如果您使用 `root` 用户连接，通常可以跳过此步骤。

#### **示例 1: 完整的网络管理权限**

创建 `/usr/share/rpcd/acl.d/network-full.json`:

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

#### **示例 2: 综合的系统管理员权限**

创建 `/usr/share/rpcd/acl.d/system-admin.json`:

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

#### **为用户分配 ACL 角色**

创建 ACL 文件后，在 `/etc/config/rpcd` 文件中为用户分配相应的角色：

```ini
config login
    option username 'admin'
    option password '$p$admin'
    list read 'system-admin'
    list write 'system-admin'
```

#### **应用变更**

修改配置后，重启 `rpcd` 服务以应用更改：

```bash
# 重启 rpcd 服务以应用变更
/etc/init.d/rpcd restart
```

**📖 更多详情，请参阅 [OpenWrt ubus ACLs 官方文档](https://openwrt.org/docs/techref/ubus#acls)**

## 许可

Apache License 2.0 - 详见 [LICENSE](LICENSE) 文件。

## 致谢

灵感来源于 [Kubernetes client-go](https://github.com/kubernetes/client-go)、[moby/moby](https://github.com/moby/moby) 和 [cdavid14/goubus](https://github.com/cdavid14/goubus)。

## 相关资源

- [OpenWrt](https://openwrt.org/) - 嵌入式设备的 Linux 发行版
- [ubus](https://git.openwrt.org/project/ubus.git) - OpenWrt 微型总线系统
- [libubus](https://git.openwrt.org/project/libubus.git) - ubus 的 C 语言库
