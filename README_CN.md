<div align="center">
  <h1>goubus</h1>
  <p><strong>OpenWrt ubus Go 客户端</strong></p>

  <p>
    <a href="https://github.com/honeybbq/goubus/actions/workflows/ci.yml"><img src="https://github.com/honeybbq/goubus/actions/workflows/ci.yml/badge.svg" alt="CI Status"></a>
    <a href="https://codecov.io/gh/honeybbq/goubus"><img src="https://codecov.io/gh/honeybbq/goubus/branch/main/graph/badge.svg" alt="Coverage"></a>
    <a href="https://goreportcard.com/report/github.com/honeybbq/goubus"><img src="https://goreportcard.com/badge/github.com/honeybbq/goubus" alt="Go Report Card"></a>
    <a href="https://pkg.go.dev/github.com/honeybbq/goubus"><img src="https://pkg.go.dev/badge/github.com/honeybbq/goubus.svg" alt="Go Reference"></a>
  </p>

  <p>
    <a href="https://github.com/honeybbq/goubus/releases"><img src="https://img.shields.io/github/v/release/honeybbq/goubus?display_name=tag" alt="Release"></a>
    <a href="https://golang.org/"><img src="https://img.shields.io/badge/go-1.25-blue" alt="Go Version"></a>
    <a href="LICENSE"><img src="https://img.shields.io/github/license/honeybbq/goubus" alt="License"></a>
    <a href="https://github.com/honeybbq/goubus/graphs/contributors"><img src="https://img.shields.io/github/contributors/honeybbq/goubus" alt="Contributors"></a>
  </p>

  <p>
    <a href="https://github.com/honeybbq/goubus/issues"><img src="https://img.shields.io/github/issues/honeybbq/goubus" alt="Issues"></a>
    <a href="https://github.com/honeybbq/goubus/pulls"><img src="https://img.shields.io/github/issues-pr/honeybbq/goubus" alt="Pull Requests"></a>
    <a href="https://github.com/honeybbq/goubus/stargazers"><img src="https://img.shields.io/github/stars/honeybbq/goubus" alt="Stars"></a>
    <img src="https://img.shields.io/github/repo-size/honeybbq/goubus" alt="Repo Size">
  </p>

  <p>
    <code>goubus</code> 用于与 OpenWrt 的 <strong>ubus</strong> (微型总线) 系统交互，通过 <strong>Profile (配置预设)</strong> 系统适配不同硬件的差异（如移动版 RAX3000M），并提供一致的 API 体验。
  </p>

  <p>
    <a href="README.md">English Version</a> | 
    <a href="https://github.com/honeybbq/goubus/tree/main/examples">示例代码</a> | 
    <a href="https://pkg.go.dev/github.com/honeybbq/goubus/v2">接口文档</a>
  </p>
</div>

---

## 核心特性

- **硬件 Profile 系统**：原生支持特定硬件的方言适配（如 **CMCC RAX3000M**, **X86 Generic**）。
- **按需引入**：仅引入所需的包，避免冗余。
- **双传输支持**：同时支持 **HTTP JSON-RPC**（远程访问）和 **Unix Socket**（本地访问）。
- **全类型安全 API**：强类型请求与响应。
- **Context 原生支持**：支持超时控制、取消请求等 `Context` 特性。

## 安装

```bash
go get github.com/honeybbq/goubus/v2
```

## 快速开始

### 1. 初始化传输层 (Transport)

```go
import (
    "context"
    "github.com/honeybbq/goubus/v2"
)

ctx := context.Background()

// 远程访问 (RPC) - 需要在设备上配置 ACL 权限
caller, _ := goubus.NewRpcClient(ctx, "192.168.1.1", "root", "password")

// 本地访问 (Unix Socket) - 在设备本地运行
caller, _ := goubus.NewSocketClient(ctx, "/var/run/ubus/ubus.sock")
```

### 2. 使用特定硬件的管理器

推荐引入与你硬件匹配的 Profile：

```go
import (
    "github.com/honeybbq/goubus/v2/profiles/cmcc_rax3000m/system"
    "github.com/honeybbq/goubus/v2/profiles/cmcc_rax3000m/network"
)

// 初始化管理器
sysSvc := system.New(caller)
netSvc := network.New(caller)

// 获取系统硬件信息
board, _ := sysSvc.Board(ctx)
fmt.Printf("型号: %s, 内核: %s\n", board.Model, board.Kernel)

// 获取所有网络接口状态
ifaces, _ := netSvc.Dump(ctx)
```

### 3. 调试与日志

`goubus` 原生支持 `log/slog`。你可以注入自定义日志器来观察原始的 ubus 交互（请求与响应详情）：

```go
import "log/slog"

// 创建一个调试级别的日志器
logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

// 注入到传输层客户端中
caller.SetLogger(logger)
```

## 目前已支持的对象

| 对象          | 说明                                                     |
| :------------ | :------------------------------------------------------- |
| **System**    | 硬件信息、运行状态、重启、看门狗、信号控制、固件升级     |
| **Network**   | 接口生命周期控制、物理设备状态、路由管理、网络命名空间   |
| **Wireless**  | IWInfo 无线扫描、关联列表查询、无线网卡底层控制          |
| **UCI**       | 完整的 CRUD 操作、Commit/Rollback 事务管理、运行状态跟踪 |
| **DHCP**      | IPv4/v6 租约查询、IPv6 RA 信息、静态租约管理             |
| **Service**   | 服务生命周期管理、配置校验、自定义数据操作               |
| **Session**   | 会话登录、ACL 权限检查、授权与撤销                       |
| **Container** | LxC 容器管理、控制台接入                                 |
| **Hostapd**   | 底层 AP 管理（踢除客户端、动态信道切换）                 |
| **RPC-SYS**   | 软件包管理、恢复出厂设置、固件校验                       |

## 项目架构

`goubus` 采用了多层架构设计，旨在实现代码复用：

- **核心层 (`goubus/`)**：包含两种传输层实现（HTTP RPC 和 Unix Socket）、原始 ubus 消息处理 (`blobmsg`)、结果解析逻辑以及通用的错误定义。
- **基础实现层 (`goubus/internal/base/`)**：提供标准 ubus 对象的通用、可复用实现（如 system, network, uci 等）。这一层封装了适用于大多数 OpenWrt 设备的共有逻辑。
- **Profile 层 (`goubus/profiles/`)**：公共 API 入口。Profile（如 `cmcc_rax3000m`, `generic`）通过 **Dialects (方言)** 机制处理不同硬件间的差异（如参数类型差异、特有方法名等），同时向外暴露一致的高级接口。
- **示例与测试数据 (`examples/`, `internal/testdata/`)**：包含基于实机数据的全量集成测试套件以及各模块的使用示例。

## 传输方式对比

| 特性         | HTTP (JSON-RPC)    | Unix Socket  |
| ------------ | ------------------ | ------------ |
| **使用场景** | 远程管理           | 设备本地应用 |
| **认证**     | 需要 (用户名/密码) | 不需要       |
| **性能**     | 有网络开销         | 无网络开销   |

## 贡献与硬件支持

仓库包含 RAX3000M 和 x86 的 Profile。其中 x86 Profile 基于虚拟机环境开发，受限于硬件及驱动的多样性，相关实现仍需进一步完善。

欢迎通过贡献 **Profile** 或 **实机测试数据 (testdata)** 扩展更多设备支持。

### 快速开发建议
1. **复制 (Copy)**：复制 `profiles/x86_generic` 文件夹。
2. **测试 (Test)**：使用实机数据运行测试，通过报错找出结构差异。
3. **适配 (Fix)**：编写 **Dialect** 修正差异。

详细的 **测试数据获取指南** 请参阅：[获取实机测试数据文档](docs/CONTRIBUTING_DATA_CN.md)。

## 许可协议

MIT License - 详见 [LICENSE](LICENSE) 文件。

## 致谢

灵感来源于 [Kubernetes client-go](https://github.com/kubernetes/client-go) 和 [moby/moby](https://github.com/moby/moby)。
