# Contributing test data

[中文](CONTRIBUTING_DATA_CN.md)

If a ubus response fails to decode, drops fields, or uses a method signature that differs on your firmware, a sanitized capture is enough to reproduce and fix it. Include the router model, `/etc/openwrt_release` (`DISTRIB_RELEASE` and `DISTRIB_REVISION`), and the relevant package version (`opkg list-installed` or `apk info` on 25.12+).

## Capture responses and signatures

Run the read-only command on the router and write one call per file. Keep the raw JSON printed by `ubus call` without wrapping it in a JSON-RPC envelope:

```sh
cat /etc/openwrt_release
ubus call system board > /tmp/system_board.json
ubus call network.interface dump > /tmp/network_interface_dump.json
ubus call luci-rpc getDHCPLeases > /tmp/luci_rpc_getDHCPLeases.json
ubus -v list dhcp > /tmp/dhcp_list.txt
```

Record the exact command and arguments alongside the file. If a call exits non-zero, record stderr and the exit code instead of submitting an empty file.

`ubus -v list` outputs human-readable text rather than JSON; keep it as `.txt`. When the router exposes `/ubus` over HTTPS and permits the sessionless JSON-RPC `list` method, you can also capture the wire signature JSON directly:

```sh
curl --fail --silent --show-error \
  -H 'Content-Type: application/json' \
  --data '{"jsonrpc":"2.0","id":1,"method":"list","params":["dhcp"]}' \
  https://router.example/ubus > dhcp_list.json
```

Keep the full JSON-RPC envelope when reporting an `RPCClient` parser bug, or extract `.result` for a signature-only fixture. When testing over the local socket only, `ubus -v list` text is sufficient.

## Add a regression test

1. Scrub secrets and identifiers before sharing: passwords, Wi-Fi keys, PPPoE credentials, WireGuard keys, `ubus_rpc_session` tokens, MAC addresses, public IPv4/IPv6 addresses, and hostnames. Keep every JSON key name, value type, and nesting level intact so the sample still reproduces the issue. Never upload raw `/etc/config` archives.
2. Place the sanitized JSON file under `internal/testdata/<target_or_device>/`, for example `internal/testdata/x86_generic/network_interface_dump.json`. If you only want to report the sample without writing Go code, attach it to an issue with the release details above.
3. Add a table-driven test case in the corresponding service package (for example `services/network/client_test.go`) using `goubus.CallerFunc` and `goubus.NewJSONResult`, or serve the JSON-RPC `list` payload from an `httptest.Server` as in `rpc_list_test.go`. Assert the specific decoded field, chosen method, or argument shape that differed, not only `err == nil`.
4. Run the CI checks from the repository root:

```sh
golangci-lint run ./...
go test -race ./...
go run ./internal/tools/apidump -check
```
