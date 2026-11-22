# 贡献实机测试数据

[English](CONTRIBUTING_DATA.md)

一份解码失败的响应，或一个随固件版本变化的方法签名，就可以作为贡献。请记录设备型号、OpenWrt 完整版本（含修订号）和相关软件包版本。

## 采集响应和签名

在路由器上运行能复现问题的只读命令。每条命令单独保存一个文件，保留 `ubus call` 输出的原始 JSON，不要另外包一层 RPC 响应。例如：

```sh
ubus call system board > /tmp/system_board.json
ubus call network.interface dump > /tmp/network_interface_dump.json
ubus call luci-rpc getDHCPLeases > /tmp/luci_rpc_getDHCPLeases.json
ubus -v list dhcp > /tmp/dhcp_list.txt
```

提交时请附上命令及参数。调用失败时记录标准错误和退出状态，不要提交空的 JSON 样本。`ubus -v list` 输出的是便于人阅读的签名文本，不是 JSON，请保存为 `.txt`。如果路由器已经提供 HTTPS `/ubus` 端点并允许 JSON-RPC `list` 方法，也可以采集 JSON 签名响应：

```sh
curl --fail --silent --show-error \
  -H 'Content-Type: application/json' \
  --data '{"jsonrpc":"2.0","id":1,"method":"list","params":["dhcp"]}' \
  https://router.example/ubus > dhcp_list.json
```

将示例主机名替换为路由器的主机名，并使用可信的 TLS 证书。测试 RPC 报文格式时保留完整响应；只测试签名时可以提取 `result`。如果只使用 Unix 套接字，附上 CLI 文本就足以报告签名差异。

## 添加回归测试

1. 分享前请脱敏密码、无线密钥、会话令牌、地址、主机名和其他隐私数据。保留字段名、JSON 类型和触发问题的数据结构。不要上传未经处理的配置转储。
2. 将脱敏样本放到 `internal/testdata/<firmware_or_device>/`，文件名如 `network_interface_dump.json`。如果只准备了样本，也可以连同采集信息一起提交 Issue。
3. 在受影响的客户端测试中添加表驱动用例，例如 `services/network/client_test.go`。对于调用响应，使用 `goubus.CallerFunc` 和 `goubus.NewJSONResult` 将样本交给类型化客户端；对于 JSON-RPC 签名，参考 `rpc_list_test.go`，让测试 HTTP handler 返回样本。断言有差异的解码字段、选用的方法或参数格式，不要只检查解码是否成功。
4. 在仓库根目录运行以下检查：

   ```sh
   golangci-lint run ./...
   go test -race ./...
   go run ./internal/tools/apidump -check
   ```
