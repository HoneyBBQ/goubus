<h1 align="center">goubus</h1>

<p align="center">
  <strong>纯 Go 实现的 OpenWrt ubus 客户端，支持本地 Unix Socket 直连与远程 HTTP/HTTPS JSON-RPC。</strong><br>
  OpenWrt 24.10 &amp; 25.12 · 无 CGO · 零第三方依赖
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
  <a href="examples/">示例</a> ·
  <a href="CHANGELOG.md">变更日志</a> ·
  <a href="README.md">English</a>
</p>

## 核心特性

- **本地 Socket 与远程 RPC 共用同一套调用接口。** 程序直接跑在路由器上时，用 `NewSocketClient` 连 `/tmp/run/ubus/ubus.sock` 与 `ubusd` 通信，内置纯 Go 的 `blobmsg` 二进制编解码，不依赖 CGO 或第三方库，可直接交叉编译到 MIPS/ARM 路由固件；从外部机器远程管理路由器时，用 `NewRPCClient` 走 `uhttpd-mod-ubus` 的 HTTP/HTTPS JSON-RPC 接口。
- **15 个独立的类型化服务子包。** 每个 ubus 领域对应独立的 `services/<pkg>` 子包且互不依赖，编译进路由器 Flash 的小工具按需引入，未使用的服务不会打进二进制。
- **运行时抹平跨版本固件差异。** OpenWrt 24.10 的 `luci` 提供 `getLocaltime`，25.12 改为 `getUnixtime`；`luci.Client.Time` 会自动探测并缓存可用方法，也可通过 `luci.WithTimeMethod` 直接指定。`dhcp.Client.AddLease` 会查询 `ubus list` 签名来决定 `duid` 传单元素数组还是字符串。`rpcd` 与 `ubusd` 常见的 JSON 类型漂移（布尔值写成 `"1"`/`0`、单字符串与数组混用、数字带引号、时间戳混用秒数与 RFC3339）分别由 `Bool`、`StringOrSlice`、`StringInt`、`UnixTime` 处理。
- **未封装的第三方插件一行调用。** 遇到 `services/` 尚未内置的 OpenWrt 插件（如 `mwan3`、`banip` 或自定义 `rpcd` 脚本），直接在当前连接上调用 `goubus.Call[T]` 即可按结构体解码返回值。
- **方法签名查询与标准库错误处理。** `goubus.List` 在两种传输上返回结构一致的对象与参数类型表。只要 ubus 状态码非 0 或 JSON-RPC 报错，调用在解码响应体前就会返回 `*goubus.CallError`，可直接配合 `errors.Is` 与 `errors.AsType` 判别原因。

## 安装

```sh
go get github.com/honeybbq/goubus/v2
```

要求 Go 1.26 及以上版本。在 `v2.0.0` 正式打标前，不带版本号的 `go get` 默认跳过预发布版本，请从 [Releases](https://github.com/honeybbq/goubus/releases) 指定最新的预发布标签或直接拉取当前分支提交（注意 `v2.0.0-alpha2` 早于 `services/*` 重构）。从 `v2.0.0-alpha1` 或 `v2.0.0-alpha2` 升级可对照 [CHANGELOG.md 中的迁移表](CHANGELOG.md#migrating-from-v200-alpha1-and-v200-alpha2)。

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
	"github.com/honeybbq/goubus/v2/services/system"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var (
		conn goubus.Transport
		err  error
	)
	if host := os.Getenv("UBUS_URL"); host != "" {
		conn, err = goubus.NewRPCClient(ctx, host, os.Getenv("UBUS_USER"), os.Getenv("UBUS_PASSWORD"))
	} else {
		conn, err = goubus.NewSocketClient(ctx, "")
	}
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	sys := system.New(conn)
	board, err := sys.Board(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s (%s %s)\n", board.Model, board.Release.Version, board.Release.Revision)
}
```

## 常见用法

### 连接方式：路由器本地跑 vs 远程管路由器

程序运行在 OpenWrt 本机时，直连本地 Unix Socket。路径传空字符串会默认使用 `/tmp/run/ubus/ubus.sock`：

```go
conn, err := goubus.NewSocketClient(ctx, "")
if err != nil {
	return err
}
defer conn.Close()
```

程序运行在其他主机时，通过 HTTP 或 HTTPS 连接路由器的 `uhttpd-mod-ubus`。地址未写路径时默认请求 `/ubus`；`NewRPCClient` 会在初始化时调用 `session.login` 获取令牌，在 rpcd 空闲超时后重新登录，并在 `Close` 时调用 `session.destroy`。`Call` 遇到 `ErrPermissionDenied` 会直接返回，不会重发；需要强制重新登录时调用 `Authenticate(ctx)`。

```go
conn, err := goubus.NewRPCClient(ctx, "https://192.168.1.1", "root", os.Getenv("UBUS_PASSWORD"))
if err != nil {
	return err
}
defer conn.Close()
```

`*SocketClient` 与 `*RPCClient` 均实现 `goubus.Transport`（即 `Caller` 加 `Close`）。各业务客户端只接收 `goubus.Caller`，不接管也不关闭底层连接。

### 使用类型化服务包（`services/<pkg>`）

按实际访问的对象引入对应的 `services/<pkg>` 子包即可，未引用的服务不会参与编译，多个客户端可直接共用同一个 `Caller`：

```go
import (
	"github.com/honeybbq/goubus/v2/services/luci"
	"github.com/honeybbq/goubus/v2/services/network"
	"github.com/honeybbq/goubus/v2/services/system"
)

sys := system.New(conn)
netClient := network.New(conn)
luciClient := luci.New(conn)

info, err := sys.Info(ctx)
if err != nil {
	return err
}
wan, err := netClient.Interface("wan").Status(ctx)
```

针对同一台路由器连接，建议复用 `*luci.Client` 与 `*dhcp.Client` 实例，以保留其内部对 `luci` 时间方法及 `dhcp.add_lease` DUID 格式的运行期探测缓存。

### 错误处理

无论是本地 Socket 返回非 0 的 ubus 状态码（`1` 至 `10`），还是 HTTP 端返回 JSON-RPC 错误，都会在解码业务字段前返回 `*goubus.CallError`。用 `errors.Is` 判断根包哨兵错误，用 `errors.AsType` 读取出错的对象名、方法名与状态码：

```go
wan, err := netClient.Interface("wan6").Status(ctx)
switch {
case errors.Is(err, goubus.ErrNotFound):
	// ubus 对象或对应接口不存在
case errors.Is(err, goubus.ErrPermissionDenied):
	// 当前 rpcd 账号的 ACL 无权调用该方法
case err != nil:
	if callErr, ok := errors.AsType[*goubus.CallError](err); ok {
		log.Printf("%s.%s 调用失败，错误码 %d (%s)",
			callErr.Service, callErr.Method, callErr.Code, callErr.Message)
	}
	return err
}
_ = wan
```

### 调用未内置的第三方插件

对于 `mwan3`、`banip` 等第三方软件包或自研的 ubus 对象，无需等待库更新，直接用泛型函数 `goubus.Call[T]` 发起调用并解码：

```go
type Mwan3Interface struct {
	Status string      `json:"status"`
	Online int         `json:"online"`
	Up     goubus.Bool `json:"up"`
}

type Mwan3Status struct {
	Interfaces map[string]Mwan3Interface `json:"interfaces"`
}

mwan, err := goubus.Call[Mwan3Status](ctx, conn, "mwan3", "status", map[string]any{
	"section": "interfaces",
})
```

像 `system.reboot` 这类只返回状态码、没有响应体的操作，直接调 `_, err := conn.Call(ctx, "system", "reboot", nil)` 即可。

### 查询总线对象与方法签名

`goubus.List` 对应命令行 `ubus -v list`，在 Socket 与 HTTP RPC 下均将参数类型归一为 `string`、`number`、`boolean`、`array`、`object` 或 `unknown`：

```go
objects, err := goubus.List(ctx, conn, "dhcp")
if err != nil {
	return err
}
fmt.Println(objects["dhcp"]["add_lease"]) // map[duid:array ip:string ...]
```

注意：`List` 只反映当前总线上注册了哪些对象和方法；走 HTTP RPC 远程调用时，实际能否执行仍取决于当前 `rpcd` 会话的 ACL 授权。

## 包列表

| 导入路径 | 对应 ubus 对象 | 功能范围 |
| --- | --- | --- |
| `github.com/honeybbq/goubus/v2` | 任意对象 | `SocketClient`、`RPCClient`、`Caller`、`Call[T]`、`List`、`*CallError`、状态哨兵错误，以及 `Bool`、`StringOrSlice`、`StringInt`、`UnixTime` |
| `github.com/honeybbq/goubus/v2/services/container` | `container` | `procd` 容器配置、运行状态、特性查询与控制台操作 |
| `github.com/honeybbq/goubus/v2/services/dhcp` | `dhcp` | `odhcpd` / `dnsmasq` 的 IPv4/IPv6 租约、IPv6 RA 前缀与 `add_lease` 静态租约下发 |
| `github.com/honeybbq/goubus/v2/services/file` | `file` | 文件读写、目录列举、`stat`/`lstat`、MD5 计算与命令执行 |
| `github.com/honeybbq/goubus/v2/services/hostapd` | `hostapd`、`hostapd.*` | AP 状态、关联终端列表、踢除/封禁客户端与信道切换 |
| `github.com/honeybbq/goubus/v2/services/log` | `log` | 读取 `logd` 环形缓冲区日志与写入系统日志 |
| `github.com/honeybbq/goubus/v2/services/luci` | `luci`、`luci-rpc` | LuCI 版本、系统时间、进程、挂载点、块设备/USB、LED、`board.json`、网络/无线设备、主机提示与租约 |
| `github.com/honeybbq/goubus/v2/services/network` | `network`、`network.device`、`network.interface*`、`network.wireless` | `netifd` 逻辑接口、物理/桥接设备、主机路由、协议处理器与无线射频启停 |
| `github.com/honeybbq/goubus/v2/services/rc` | `rc` | `/etc/init.d` 脚本列表与 `start`、`stop`、`restart`、`enable`、`disable` 操作 |
| `github.com/honeybbq/goubus/v2/services/rpcsys` | `rpc-sys` | 已安装软件包查询、修改系统密码、恢复出厂设置与固件升级流程 |
| `github.com/honeybbq/goubus/v2/services/service` | `service` | `procd` 服务注册、实例列表、运行状态、信号发送与看门狗配置 |
| `github.com/honeybbq/goubus/v2/services/session` | `session` | `rpcd` 会话登录、创建、销毁、会话键值存储与 ACL 权限授予/回收 |
| `github.com/honeybbq/goubus/v2/services/system` | `system` | 设备型号与固件版本、运行时间/内存/存储占用、看门狗、进程信号、固件校验与 `sysupgrade` |
| `github.com/honeybbq/goubus/v2/services/uci` | `uci` | `/etc/config/*` 配置包、段、选项的读取、暂存修改、`apply` 回滚确认与 `revert` |
| `github.com/honeybbq/goubus/v2/services/wireless` | `iwinfo` | 无线网卡列表、射频参数、周边热点扫描、已关联终端速率/信号、频点表、功率表与信道勘测 |
| `github.com/honeybbq/goubus/v2/services/wpasupplicant` | `wpa_supplicant`、`wpa_supplicant.*` | STA 客户端接口状态、重载配置、WPS 配网与控制指令 |

## 延伸文档

完整的公开类型与函数签名见 [pkg.go.dev](https://pkg.go.dev/github.com/honeybbq/goubus/v2)。

- [架构说明](docs/architecture.md)：目录分层、包依赖方向，以及 `RPCClient` 和 `SocketClient` 的底层收发流程。
- [设计约定](docs/design.md)：连接所有权、中间件、`*CallError` 错误映射、`List` 类型归一、OpenWrt 24.10/25.12 兼容细节、凭据脱敏边界与 `api/*.txt` 快照校验。
- [示例程序](examples/)：`system`、`network`、`uci`、`file`、`service`、`session` 的可运行代码与传输层基准测试。
- [变更日志](CHANGELOG.md)：版本记录，以及从 `v2.0.0-alpha1` / `v2.0.0-alpha2` 迁移的对照表。

## 兼容性

| 目标环境 | 实测覆盖 |
| --- | --- |
| OpenWrt 24.10 | CI 中针对真实 OpenWrt `24.10` 容器（Unix Socket + HTTP/HTTPS RPC）持续集成测试，并包含 `x86/64` generic（`24.10.5`）实机采集的 JSON 与 `blobmsg` 二进制样本回归测试。 |
| OpenWrt 25.12 | CI 中针对真实 OpenWrt `25.12` 容器（Unix Socket + HTTP/HTTPS RPC）持续集成测试，并包含 CMCC RAX3000M（`25.12.0-rc1`）实机采集的 JSON 与 `blobmsg` 二进制样本回归测试。 |
| Go | 1.26 及以上（CI 覆盖 Go 1.26 / 1.27 测试及 `linux/mipsle`、`linux/arm`、`linux/386`、`linux/arm64` 交叉编译）；纯 Go 实现，零 CGO。 |

路由器上实际可调用的对象取决于固件安装了哪些软件包（如 `rpcd-mod-file`、`rpcd-mod-iwinfo`、`luci-base`、`odhcpd` 等）；通过 HTTP JSON-RPC 访问时，还需确保所用 `rpcd` 账号具备对应对象的读写 ACL。

## 参与贡献

本地开发环境与代码检查流程见 [CONTRIBUTING.md](.github/CONTRIBUTING.md)。如果你在其他型号或固件分支上遇到返回结构差异，欢迎参照[贡献实机测试数据](docs/CONTRIBUTING_DATA_CN.md)提交脱敏后的 `ubus call` 或 `ubus -v list` 样本。

## 许可证

[MIT](LICENSE)
