# Blobmsg Test Data

This directory contains binary data used for Golden Files testing of the `blobmsg` package.

## Data Sources

### 1. Snapshots (`*.bin`)

#### RAX3000M (ARM64)
- **Source**: Captured from a real **CMCC RAX3000M** router.
- **Hardware**: MediaTek MT7981B (ARMv8 aarch64 / Little Endian).
- **Tool**: Collected using `dump_ubus`.
- **Date**: 2026-01-17
- **Samples**:
    - `rax3000m_system_board.bin`: Output of `ubus call system board`.
    - `rax3000m_system_info.bin`: Output of `ubus call system info`.

#### x86 Virtual Machine (x86_64)
- **Source**: Captured from an x86_64 virtual machine.
- **Hardware**: x86_64 / Little Endian.
- **Tool**: Collected using `dump_ubus`.
- **Date**: 2026-01-17
- **Samples**:
    - `x86_system_board.bin`: Output of `ubus call system board`.
    - `x86_system_info.bin`: Output of `ubus call system info`.

### 2. Fuzz Corpus (`fuzz_corpus/`)
- **Source**: Synchronized from the official OpenWrt [ubus repository](https://git.openwrt.org/?p=project/ubus.git;a=tree;f=tests/fuzz/corpus;hb=HEAD) (`tests/fuzz/corpus/`).
- **Purpose**: Robustness and defensive testing.
- **Goal**: Ensure the Go implementation handles malformed or malicious input gracefully by returning an error instead of triggering a `panic`.

## Usage
These files are automatically processed by `golden_test.go`. To run the tests:
```bash
go test -v ./internal/blobmsg/golden_test.go
```
