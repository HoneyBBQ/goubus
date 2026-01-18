# Guide: Contributing Real-World Test Data

Due to the vast diversity of OpenWrt hardware, we rely on real `ubus` data to ensure `goubus` parsing logic is accurate.

You can help us improve by following these steps:

## 1. Export Raw Data

Log in to your router (SSH) and run the following commands to export ubus object data into JSON files. You can choose the modules you are interested in:

```bash
# System Info
ubus call system board > system_board.json
ubus call system info > system_info.json

# Network Interfaces
ubus call network.interface dump > network_interface_dump.json
ubus call network.device status > network_device_status.json

# Wireless Status (iwinfo)
ubus call iwinfo devices > iwinfo_devices.json
# Assuming a device named phy0-ap0
ubus call iwinfo info '{"device": "phy0-ap0"}' > iwinfo_info_phy0.json

# DHCP Leases (Critical area for differences)
ubus call dhcp ipv4leases > dhcp_ipv4leases.json
ubus call luci-rpc getDHCPLeases > luci_rpc_getDHCPLeases.json

# UCI Configs
ubus call uci configs > uci_configs.json
```

## 2. How to Contribute

### A. Submit an Issue
Paste JSON contents into an Issue with your device model and firmware version.

### B. Create a Profile (Recommended)
1. Copy `profiles/x86_generic` to a new folder (e.g., `profiles/tplink_ax6000`).
2. Place JSONs in `internal/testdata/<your_device>/`.
3. Update `manager_test.go` in the new Profile to point to your JSONs.
4. Run `go test`. If it fails, implement a new `Dialect` to handle the differences.
5. Submit a PR.

## Privacy
Data may contain MAC addresses or hostnames. Feel free to replace them with generic values (e.g., `AA:BB:CC:DD:EE:FF`) before submitting.
