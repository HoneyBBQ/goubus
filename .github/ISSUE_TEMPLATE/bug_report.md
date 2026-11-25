---
name: Bug Report
about: Report a protocol, transport, or decoding bug in goubus
title: '[BUG] '
labels: bug
assignees: ''
---

## Summary

Describe what happened and what you expected instead.

## Minimal reproduction

```go
// Paste a minimal Go snippet or ubus call / wire capture that reproduces the issue.
```

## Error or debug log

Paste any returned error (`*goubus.CallError` fields, sentinel error) or redacted transport debug output.

## Environment

- `goubus` version: (for example `v2.0.0-beta.1` or commit SHA)
- Go version: (for example `go1.26.1`)
- Transport: (`SocketClient` / `RPCClient` HTTP / `RPCClient` HTTPS)
- OpenWrt release & target: (for example `24.10.5 x86/64` or `25.12.6 mediatek/filogic`)
