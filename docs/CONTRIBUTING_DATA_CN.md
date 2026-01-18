# 获取实机测试数据指南

由于 OpenWrt 硬件差异巨大，我们需要真实的 `ubus` 数据来确保 `goubus` 的解析逻辑正确。

你可以通过以下步骤帮助我们改进：

## 1. 导出原始数据

登录你的路由器（SSH），执行以下命令将 ubus 对象的数据导出为 JSON 文件。你可以根据你感兴趣的模块选择执行：

```bash
# 系统信息
ubus call system board > system_board.json
ubus call system info > system_info.json

# 网络接口
ubus call network.interface dump > network_interface_dump.json
ubus call network.device status > network_device_status.json

# 无线状态 (iwinfo)
ubus call iwinfo devices > iwinfo_devices.json
# 假设有一个设备叫 phy0-ap0
ubus call iwinfo info '{"device": "phy0-ap0"}' > iwinfo_info_phy0.json

# DHCP 租约 (关键差异点)
ubus call dhcp ipv4leases > dhcp_ipv4leases.json
ubus call luci-rpc getDHCPLeases > luci_rpc_getDHCPLeases.json

# UCI 配置
ubus call uci configs > uci_configs.json
```

## 2. 贡献方式

### A. 提交 Issue
将导出的 JSON 内容贴到 Issue 中，注明设备型号和固件版本。

### B. 创建 Profile (推荐)
1. 复制 `profiles/x86_generic` 到新目录（如 `profiles/tplink_ax6000`）。
2. 将 JSON 放入 `internal/testdata/<设备名>/`。
3. 修改新 Profile 下的 `manager_test.go`，将路径指向你的 JSON。
4. 运行 `go test`，如果报错，说明有结构差异。实现一个新的 `Dialect` 进行适配。
5. 提交 PR。

## 隐私
JSON 中可能包含 MAC 地址或主机名，提交前可手动将其替换为通用值（如 `AA:BB:CC:DD:EE:FF`）。
