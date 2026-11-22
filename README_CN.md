# goubus

OpenWrt ubus 的 Go 客户端，支持 HTTP JSON-RPC 和 Unix 套接字。

[![Go Reference](https://pkg.go.dev/badge/github.com/honeybbq/goubus/v2.svg)](https://pkg.go.dev/github.com/honeybbq/goubus/v2) [![CI](https://github.com/honeybbq/goubus/actions/workflows/ci.yml/badge.svg)](https://github.com/honeybbq/goubus/actions/workflows/ci.yml) [![Codecov](https://codecov.io/gh/honeybbq/goubus/branch/main/graph/badge.svg)](https://codecov.io/gh/honeybbq/goubus) [![Release](https://img.shields.io/github/v/release/honeybbq/goubus?include_prereleases&sort=semver)](https://github.com/honeybbq/goubus/releases) [![Go version](https://img.shields.io/github/go-mod/go-version/honeybbq/goubus)](go.mod) [![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[API 文档](https://pkg.go.dev/github.com/honeybbq/goubus/v2) · [架构](docs/architecture.md) · [设计](docs/design.md) · [示例](examples/) · [变更日志](CHANGELOG.md) · [English](README.md)

## 为什么使用 goubus

- **一个 `Caller`，两种传输。** 远程通过 `uhttpd-mod-ubus` 使用 HTTP 或 HTTPS JSON-RPC；本地通过 Unix 套接字直连 `ubusd`。二进制 `blobmsg` 编解码采用纯 Go，无需 CGO 或第三方依赖。
- **类型化服务。** 15 个 `services/<pkg>` 子包可以独立导入，也可以用 `services.Client` 让所有服务共享同一个调用方。
- **运行时内省。** `goubus.List` 提供与 `ubus list` 对应的对象和方法签名，在两种传输上返回相同的结构。
- **固件差异。** 客户端在运行时适配方法和参数差异。`Bool`、`StringOrSlice`、`StringInt` 和 `UnixTime` 可解码 JSON 表示不同的字段。
- **中间件与日志。** `CallMiddleware` 包装与传输方式无关的调用。`log/slog` 默认不输出；RPC 客户端的格式化输出及含凭据的 RPC 流量会脱敏，具体范围见[设计文档](docs/design.md#logging-and-credentials)。

## 安装

```sh
go get github.com/honeybbq/goubus/v2
```

`v2.0.0` 正式发布前，请显式指定预发布版本，例如 `go get github.com/honeybbq/goubus/v2@v2.0.0-alpha2`，也可以从[发布列表](https://github.com/honeybbq/goubus/releases)选择更新的预发布标签。下方示例使用重构后的 API，`alpha2` 尚未包含这些接口；请使用本仓库源码或包含此次重构的标签构建示例。

## 快速开始

在路由器上运行此程序时，默认连接 `/tmp/run/ubus/ubus.sock`。远程调用时，将 `UBUS_URL` 设为路由器的 HTTPS URL，并通过 `UBUS_USERNAME` 和 `UBUS_PASSWORD` 指定有权调用目标方法的 `rpcd` 账号。URL 未指定路径时，RPC 默认使用 `/ubus`。

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/services"
	"github.com/honeybbq/goubus/v2/services/system"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var conn goubus.Transport
	var err error
	if url := os.Getenv("UBUS_URL"); url != "" {
		conn, err = goubus.NewRpcClient(ctx, url, os.Getenv("UBUS_USERNAME"), os.Getenv("UBUS_PASSWORD"))
	} else {
		conn, err = goubus.NewSocketClient(ctx, "")
	}
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	board, err := system.New(conn).Board(ctx)
	if err != nil {
		return err
	}
	fmt.Println(board.Model)

	info, err := services.New(conn).System().Info(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Uptime: %d s\n", info.Uptime)

	objects, err := goubus.List(ctx, conn)
	if err != nil {
		return err
	}
	fmt.Printf("Objects: %d\n", len(objects))

	if _, ok := objects["example.stats"]["read"]; ok {
		type stats struct {
			Requests int `json:"requests"`
		}
		result, err := goubus.Call[stats](ctx, conn, "example.stats", "read", nil)
		if err != nil {
			return err
		}
		fmt.Printf("Requests: %d\n", result.Requests)
	}
	return nil
}
```

只使用系统服务时，调用 `system.New(conn)`；需要管理多种服务时，`services.New(conn)` 为每种服务保留一个客户端实例。`List` 返回当前传输可见的对象，不代表调用方有权调用每个方法。最后一个调用演示如何解码自定义对象；仅当路由器提供 `example.stats` 时才会执行。

## 包

| 导入路径 | 用途 |
| --- | --- |
| `github.com/honeybbq/goubus/v2` | 传输实现、调用及内省接口、状态错误和弹性 JSON 类型。 |
| `github.com/honeybbq/goubus/v2/services` | 聚合所有类型化服务的客户端。 |
| `github.com/honeybbq/goubus/v2/services/container` | 容器配置、状态和控制台操作。 |
| `github.com/honeybbq/goubus/v2/services/dhcp` | DHCP 租约、静态租约创建和 IPv6 路由器通告。 |
| `github.com/honeybbq/goubus/v2/services/file` | 通过 `file` 对象访问文件、读取元数据和执行命令。 |
| `github.com/honeybbq/goubus/v2/services/hostapd` | 接入点状态、已连接客户端和信道控制。 |
| `github.com/honeybbq/goubus/v2/services/log` | 读取和写入系统日志。 |
| `github.com/honeybbq/goubus/v2/services/luci` | LuCI 系统信息，以及 `luci-rpc` 提供的设备、网络和租约数据。 |
| `github.com/honeybbq/goubus/v2/services/network` | 网络接口、设备、路由，以及无线射频配置和状态。 |
| `github.com/honeybbq/goubus/v2/services/rc` | 列出初始化脚本并执行脚本操作。 |
| `github.com/honeybbq/goubus/v2/services/rpcsys` | 通过 `rpc-sys` 查询已安装的软件包并执行系统维护操作。 |
| `github.com/honeybbq/goubus/v2/services/service` | 服务实例、配置和生命周期。 |
| `github.com/honeybbq/goubus/v2/services/session` | rpcd 会话、会话数据和访问控制。 |
| `github.com/honeybbq/goubus/v2/services/system` | 设备信息、运行状态和系统操作。 |
| `github.com/honeybbq/goubus/v2/services/uci` | 读取和修改 UCI 配置包、段和选项。 |
| `github.com/honeybbq/goubus/v2/services/wireless` | 通过 `iwinfo` 查询设备状态、扫描结果和站点信息。 |
| `github.com/honeybbq/goubus/v2/services/wpasupplicant` | `wpa_supplicant` 站点状态和控制。 |

## 文档

- [API 参考](https://pkg.go.dev/github.com/honeybbq/goubus/v2)：公开类型、选项和方法。
- [架构](docs/architecture.md)：包布局和依赖关系。
- [设计](docs/design.md)：调用、错误、内省、固件差异和 API 稳定性。
- [示例](examples/)：各服务的完整程序。
- [变更日志](CHANGELOG.md)：发布历史和旧硬件 Profile 的迁移说明。

## 兼容性

| 目标 | 覆盖范围 |
| --- | --- |
| OpenWrt 24.10 | x86 generic（24.10.5）的实测响应样本。 |
| OpenWrt 25.12 | CMCC RAX3000M（25.12.0-rc1）的实测响应样本。 |
| Go | 1.25 及以上；纯 Go，无需 CGO。 |

可用的对象取决于 OpenWrt 已安装的软件包和配置。远程调用还受 rpcd ACL 限制。

## 贡献

参阅[贡献实机测试数据](docs/CONTRIBUTING_DATA_CN.md)，采集响应和签名并添加回归测试。

## 许可证

[MIT](LICENSE)
