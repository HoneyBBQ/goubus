# Contributing to goubus

Bug reports, binary wire captures, fixes, and documentation changes are welcome. For larger changes, such as changing an exported signature or transport behavior, open an issue first so we can agree on the API shape before writing code.

By participating you agree to the [Code of Conduct](CODE_OF_CONDUCT.md). To report a security vulnerability, follow [SECURITY.md](SECURITY.md) rather than opening a public issue.

## Setup

You need Go 1.27 or newer and [golangci-lint](https://golangci-lint.run/) (CI currently runs `v2.15.0`).

```sh
git clone https://github.com/HoneyBBQ/goubus
cd goubus
go build ./...
```

## Checks

CI runs all three commands below on every push and pull request:

```sh
golangci-lint run ./...
go test -race ./...
go run ./internal/tools/apidump -check
```

Linter settings live in `.golangci.yml`. Every exported type, struct field, interface method, function, and method requires a GoDoc comment; `internal/tools/apidump` fails `go test ./...` when one is missing.

### Public API snapshots

The exported surface of the module is recorded in `api/goubus.txt` and reviewed alongside code. `go run ./internal/tools/apidump -check` fails on any signature drift or if an exported declaration exposes an `internal/` or unexported type.

When an API change is intentional, regenerate the snapshot and commit it in the same commit:

```sh
go run ./internal/tools/apidump
```

Inspect `git diff api/` before committing so unintended exports never slip in.

### Protocol corpus

Unit tests run offline without a live OpenWrt router:

- `internal/blobmsg/testdata/` holds captured binary `ubus` frames and the upstream OpenWrt `ubus` fuzz corpus (`fuzz_corpus/`), verified by `internal/blobmsg/golden_test.go` during `go test ./...` to ensure malformed binary frames return errors rather than panicking.

### Live OpenWrt container integration tests

With Docker installed locally, `scripts/openwrt-lab.sh` starts an ephemeral OpenWrt container (`21-02`, `22-03`, `23-05`, `24-10`, or `25-12`) running `ubusd`, `rpcd`, and `uhttpd` over a Unix socket, HTTP JSON-RPC, and HTTPS JSON-RPC (with a generated CA):

```sh
./scripts/openwrt-lab.sh start 24-10 /tmp/openwrt-lab
. /tmp/openwrt-lab/env.sh
go test -v -race -count=1 -p 1 -tags integration ./...
./scripts/openwrt-lab.sh stop /tmp/openwrt-lab
```

Pass any of `21-02`, `22-03`, `23-05`, `24-10`, or `25-12` to run the suite against that OpenWrt release series. CI runs all five container targets in parallel on every push and pull request.

## Changes

- Include unit tests with every bug fix or feature. A bug fix should come with a test that fails without it.
- Wrap underlying errors with `%w` and root-package sentinels (`goubus.ErrNotFound`, `goubus.ErrInvalidResponse`, and others) so callers can match them with `errors.Is` and `errors.AsType`.
- Use `log/slog` where logging is needed, keep transports silent by default, and never log passwords or `ubus_rpc_session` tokens.
- User-visible changes belong under `## Unreleased` in [CHANGELOG.md](../CHANGELOG.md). GitHub release notes are extracted directly from `CHANGELOG.md` by `scripts/release-notes.sh`.
- Keep commits focused and follow [Conventional Commits](https://www.conventionalcommits.org/) (`feat`, `fix`, `docs`, `ci`, `test`, `refactor`, `chore`, with an optional scope such as `fix(socket): ...`).

## License

By contributing, you agree that your contribution is licensed under the [MIT License](../LICENSE).
