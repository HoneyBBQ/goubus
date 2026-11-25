<h1 align="center">goubus</h1>

<p align="center">
  <strong>纯 Go 实现的 OpenWrt <code>ubus</code> 协议库 —— 根包提供全双工原生 Unix Socket 总线协议，<code>rpc</code> 子包提供远程 HTTP/HTTPS JSON-RPC 网关客户端。</strong><br>
  无 CGO · 零第三方依赖 · 支持 MIPS / ARM / x86 全路由架构
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/honeybbq/goubus/v2"><img alt="Go Reference" src="https://pkg.go.dev/badge/github.com/honeybbq/goubus/v2.svg"></a>
  <a href="https://github.com/honeybbq/goubus/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/honeybbq/goubus/actions/workflows/ci.yml/badge.svg"></a>
  <a href="https://codecov.io/gh/honeybbq/goubus"><img alt="Codecov" src="https://codecov.io/gh/honeybbq/goubus/branch/main/graph/badge.svg"></a>
  <a href="https://scorecard.dev/viewer/?uri=github.com/HoneyBBQ/goubus"><img alt="OpenSSF Scorecard" src="https://api.scorecard.dev/projects/github.com/HoneyBBQ/goubus/badge"></a>
  <a href="https://github.com/honeybbq/goubus/releases"><img alt="Release" src="https://img.shields.io/github/v/release/honeybbq/goubus?include_prereleases&amp;sort=semver"></a>
  <a href="go.mod"><img alt="Go version" src="https://img.shields.io/github/go-mod/go-version/honeybbq/goubus"></a>
  <a href="LICENSE"><img alt="License: MIT" src="https://img.shields.io/badge/License-MIT-blue.svg"></a>
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/honeybbq/goubus/v2">API 文档</a> ·
  <a href="docs/architecture.md">架构</a> ·
  <a href="docs/design.md">设计</a> ·
  <a href="CHANGELOG.md">变更日志</a> ·
  <a href="README.md">English</a>
</p>

## 架构定位

在 OpenWrt 中，`ubusd` 原生只支持本机 Unix Domain Socket（UDS）与二进制 `blobmsg` 帧协议；而远程 HTTP/HTTPS JSON-RPC 接口是由 `uhttpd-mod-ubus` 网页插件配合 `rpcd` 会话服务提供的外部网关。`goubus` 按真实协议边界划分为两个包：

- **全双工多路复用原生 `ubus`（根包 `goubus`）。** 通过 `goubus.NewClient` 直连 `/tmp/run/ubus/ubus.sock`（或通过 `NewClientFromConn` 传入 SSH 转发的 `net.Conn`），单条连接按序列号并发复用 `Call` 与 `List`；路由器本地编译的程序只引根包时完全不链接 `net/http` 与 `crypto/tls`，显著减小固件二进制体积。
- **事件广播、对象订阅与纯 Go 注册守护对象（根包 `goubus`）。** 原生支持系统事件收发（`Listen`、`SendEvent`）、对象级通知订阅（`Subscribe`）以及将 Go 方法直接发布注册为 `ubusd` 总线对象（`AddObject`、`RegisteredObject.Notify`），无需 CGO 或 `libubus`。
- **远程 HTTP/HTTPS JSON-RPC 网关（子包 `goubus/v2/rpc`）。** 从外部主机管理路由器时引入 `rpc.NewClient`，内置 `rpcd` 登录、空闲超时顺延、强制重认证与凭据脱敏，并完整实现根包的 `goubus.Transport` 与 `goubus.Lister`，与原生客户端无缝共享上层代码。
- **统一错误体系与签名自省。** 非 0 状态码（`1`..`10`）与 JSON-RPC 报错在解码前即返回 `*goubus.CallError`；`goubus.List` 将两套传输的参数策略归一为 `ArgType`。

## 包划分

| 导入路径 | 职责范围 |
| --- | --- |
| `github.com/honeybbq/goubus/v2` | 原生 UDS `Client`（并发 `Call`、`List`、`Ping`、`Listen`、`SendEvent`、`Subscribe`、`AddObject`），`Caller` / `Transport` / `Call[T]`，以及 `*CallError` 与状态哨兵错误 |
| `github.com/honeybbq/goubus/v2/rpc` | 面向 `uhttpd-mod-ubus` + `rpcd` 的远程 HTTP/HTTPS JSON-RPC 2.0 `Client`（实现 `Transport` 与 `Lister`） |

## 安装

```sh
go get github.com/honeybbq/goubus/v2@latest
```

要求 Go 1.26 及以上版本。

## 快速上手

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/rpc"
)

type boardInfo struct {
	Model   string `json:"model"`
	Release struct {
		Version  string `json:"version"`
		Revision string `json:"revision"`
	} `json:"release"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var (
		conn goubus.Transport
		err  error
	)
	if host := os.Getenv("UBUS_URL"); host != "" {
		conn, err = rpc.NewClient(ctx, host, os.Getenv("UBUS_USER"), os.Getenv("UBUS_PASSWORD"))
	} else {
		conn, err = goubus.NewClient(ctx, "")
	}
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	board, err := goubus.Call[boardInfo](ctx, conn, "system", "board", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s (%s %s)\n", board.Model, board.Release.Version, board.Release.Revision)
}
```

## 常见用法

### 方法调用与总线签名自省

```go
type wanStatus struct {
	Proto  string   `json:"proto"`
	DNS    []string `json:"dns-server"`
	Up     bool     `json:"up"`
	Uptime int64    `json:"uptime"`
}

wan, err := goubus.Call[wanStatus](ctx, conn, "network.interface.wan", "status", nil)

// 仅校验状态码、无响应体的调用：
_, err = conn.Call(ctx, "uci", "commit", map[string]any{"config": "network"})

// 查询对象与入参类型签名（对应 ubus -v list）：
sigs, err := goubus.List(ctx, conn, "uci")
```

### 事件广播、对象订阅与注册纯 Go 服务对象（原生 UDS）

```go
sock, err := goubus.NewClient(ctx, "")
if err != nil {
	return err
}
defer sock.Close()

// 监听系统事件（ubus listen）并广播自定义事件（ubus send）：
evSub, err := sock.Listen(ctx, "network.interface", func(ev goubus.Event) {
	log.Printf("event %s: %s", ev.Type, ev.Data)
})
if err != nil {
	return err
}
defer evSub.Close()

_ = sock.SendEvent(ctx, "custom.ready", map[string]any{"ok": true})

// 在 ubusd 上注册纯 Go 服务对象供其他进程或远程 LuCI/RPC 调用：
obj, err := sock.AddObject(ctx, goubus.ObjectDef{
	Name: "myapp.greeter",
	Methods: map[string]goubus.Method{
		"hello": {
			Policy: goubus.MethodSignature{"name": goubus.ArgTypeString},
			Handler: func(ctx context.Context, req goubus.CallRequest) (any, error) {
				var in struct {
					Name string `json:"name"`
				}
				if err := req.Unmarshal(&in); err != nil {
					return nil, goubus.ErrInvalidParameter
				}
				return map[string]any{"greeting": "Hello, " + in.Name}, nil
			},
		},
	},
})
if err != nil {
	return err
}
defer obj.Close()
```

### 错误处理

```go
_, err := conn.Call(ctx, "network.interface.wan6", "status", nil)
switch {
case errors.Is(err, goubus.ErrNotFound):
	// ubus 对象或目标资源不存在
case errors.Is(err, goubus.ErrPermissionDenied):
	// 被 ubusd 或 rpcd ACL 拒绝
case err != nil:
	if callErr, ok := errors.AsType[*goubus.CallError](err); ok {
		log.Printf("%s.%s 调用失败: code=%d msg=%q",
			callErr.Service, callErr.Method, callErr.Code, callErr.Message)
	}
	return err
}
```

## 延伸文档

完整公开签名见 [pkg.go.dev/github.com/honeybbq/goubus/v2](https://pkg.go.dev/github.com/honeybbq/goubus/v2)。

- [架构说明](docs/architecture.md)：原生 UDS 与 `uhttpd-mod-ubus` 网关边界、目录分层与双向报文流转。
- [设计约定](docs/design.md)：连接所有权、多路复用、事件与对象发布、`*CallError` 映射、网关会话与 `api/*.txt` 快照校验。
- [变更日志](CHANGELOG.md)：版本演进记录。

## 兼容性

| 目标环境 | 实测覆盖 |
| --- | --- |
| OpenWrt 21.02 – 25.12 | CI 并行覆盖 `21.02`、`22.03`、`23.05`、`24.10`、`25.12` 五大存量主线真实容器集成测试（多路复用 Socket、`Listen`/`SendEvent`、`AddObject`/`Subscribe`、HTTP RPC、HTTPS RPC）及二进制 `blobmsg` 语料回归。 |
| Go | 1.26 及以上（CI 覆盖 Go 1.26 / 1.27 测试及 `linux/mipsle`、`linux/arm`、`linux/386`、`linux/arm64` 交叉编译）；纯 Go 实现，零 CGO。 |

## 参与贡献

本地开发环境、格式化（`golangci-lint fmt`）、API 快照检查与容器集成测试流程见 [CONTRIBUTING.md](.github/CONTRIBUTING.md)。

## 许可证

[MIT](LICENSE)
