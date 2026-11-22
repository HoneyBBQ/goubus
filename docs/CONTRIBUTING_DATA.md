# Contributing test data

[中文](CONTRIBUTING_DATA_CN.md)

A response that fails to decode, or a method signature that differs across
firmware versions, is enough for a contribution. Record the device model, exact
OpenWrt version (including the revision), and any relevant package version.

## Capture responses and signatures

On the router, run the read-only call that demonstrates the problem. Save one
command per file and keep the JSON returned by `ubus call` without adding an
RPC envelope. For example:

```sh
ubus call system board > /tmp/system_board.json
ubus call network.interface dump > /tmp/network_interface_dump.json
ubus call luci-rpc getDHCPLeases > /tmp/luci_rpc_getDHCPLeases.json
ubus -v list dhcp > /tmp/dhcp_list.txt
```

Keep the command and its arguments with the report. If the call fails, record
its stderr and exit status instead of submitting an empty JSON fixture.
`ubus -v list` prints a human-readable signature, not JSON; keep it as `.txt`.
When the router already exposes an HTTPS `/ubus` endpoint that allows the
JSON-RPC `list` method, capture the JSON signature response instead:

```sh
curl --fail --silent --show-error \
  -H 'Content-Type: application/json' \
  --data '{"jsonrpc":"2.0","id":1,"method":"list","params":["dhcp"]}' \
  https://router.example/ubus > dhcp_list.json
```

Replace the example hostname with the router's hostname and use a trusted TLS
certificate. Keep the full envelope for an RPC wire-format regression, or use
its `result` for a signature-only fixture. For a socket-only setup, the CLI
text is enough to report the difference.

## Add a regression test

1. Redact passwords, wireless keys, session tokens, addresses, hostnames, and
   other private data before sharing. Preserve field names, JSON types, and
   the shape that caused the problem. Do not upload raw configuration dumps.
2. Add the sanitized sample under `internal/testdata/<firmware_or_device>/`
   with a name such as `network_interface_dump.json`. If you only have a
   sample, open an issue with it and the capture details instead.
3. Add a table-driven case to the affected client test, for example
   `services/network/client_test.go`. For responses, use `goubus.CallerFunc`
   and `goubus.NewJSONResult` to send the fixture through the typed client.
   For JSON-RPC signatures, serve the fixture from the test HTTP handler as
   in `rpc_list_test.go`. Assert the decoded field or selected method or
   argument format that differed, not just that decoding succeeds.
4. Run these checks from the repository root:

   ```sh
   golangci-lint run ./...
   go test -race ./...
   go run ./internal/tools/apidump -check
   ```
