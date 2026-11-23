# 贡献实机测试数据

[English](CONTRIBUTING_DATA.md)

遇到字段解码报错、返回值漏解析，或者不同固件版本的方法签名不一致时，提交一份脱敏后的实机输出即可定位问题。请一并附上设备型号、`/etc/openwrt_release` 中的完整版本号（含 `DISTRIB_RELEASE` 与 `DISTRIB_REVISION`），以及对应软件包版本（`opkg list-installed` 或 25.12+ 的 `apk info`）。

## 采集响应与方法签名

在路由器上执行触发问题的只读命令，每条命令单独存成一个文件。直接保留 `ubus call` 标准输出的原始 JSON，不要手动套一层 JSON-RPC 外壳：

```sh
cat /etc/openwrt_release
ubus call system board > /tmp/system_board.json
ubus call network.interface dump > /tmp/network_interface_dump.json
ubus call luci-rpc getDHCPLeases > /tmp/luci_rpc_getDHCPLeases.json
ubus -v list dhcp > /tmp/dhcp_list.txt
```

提交时注明完整命令与参数；如果命令本身报错退出，请记录 stderr 与退出码，不要提交空的 JSON 文件。

`ubus -v list` 输出的是纯文本签名而非 JSON，保存为 `.txt` 即可。如果路由器已开启 HTTPS `/ubus` 端点且允许免会话的 JSON-RPC `list` 请求，也可以直接抓取原始签名报文：

```sh
curl --fail --silent --show-error \
  -H 'Content-Type: application/json' \
  --data '{"jsonrpc":"2.0","id":1,"method":"list","params":["dhcp"]}' \
  https://router.example/ubus > dhcp_list.json
```

排查 `RPCClient` 协议层解析问题时保留完整 JSON-RPC 报文；仅验证参数签名时提取其中的 `result` 字段即可。只走本地 Unix Socket 的环境直接贴 `ubus -v list` 文本就够用。

## 编写回归测试

1. 上传前务必清理敏感信息：管理员密码、Wi-Fi 密钥、PPPoE 账号密码、WireGuard 私钥、`ubus_rpc_session` 令牌、真实 MAC 地址、公网 IPv4/IPv6 地址与前缀以及设备主机名。脱敏时保留原有的键名、JSON 数据类型和嵌套层级，切勿直接上传未处理的 `/etc/config` 配置备份。
2. 将脱敏样本放入 `internal/testdata/<固件目标或机型>/` 目录（例如 `internal/testdata/x86_generic/network_interface_dump.json`）。如果只方便提供样本、不打算改 Go 代码，直接连同固件版本信息发在 Issue 里即可。
3. 在对应服务包的测试文件（如 `services/network/client_test.go`）中用 `goubus.CallerFunc` 搭配 `goubus.NewJSONResult` 喂入样本；若是 JSON-RPC `list` 样本，参考 `rpc_list_test.go` 用 `httptest.Server` 返回。断言时请直接校验存在差异的具体字段值、探测选中的方法名或请求参数格式，不要只断言 `err == nil`。
4. 在仓库根目录执行与 CI 一致的检查：

```sh
golangci-lint run ./...
go test -race ./...
go run ./internal/tools/apidump -check
```
