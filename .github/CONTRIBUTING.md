# Contributing to goubus

Bug reports, sanitized `ubus` fixtures, fixes, and documentation changes are welcome. For larger changes, such as adding a new `services/<pkg>` package or changing an exported signature, open an issue first so we can agree on the API shape before writing code.

By participating you agree to the [Code of Conduct](CODE_OF_CONDUCT.md). To report a security vulnerability, follow [SECURITY.md](SECURITY.md) rather than opening a public issue.

## Setup

You need Go 1.26 or newer and [golangci-lint](https://golangci-lint.run/) (CI currently runs `v2.15.0`).

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

Linter settings live in `.golangci.yml`. Every exported type, struct field, interface method, function, and method requires a GoDoc comment; `doc_coverage_test.go` fails `go test` when one is missing.

### Public API snapshots

The exported surface of every public package is recorded in `api/*.txt` and reviewed alongside code. `go run ./internal/tools/apidump -check` fails on any signature drift or if an exported declaration exposes an `internal/` or unexported type.

When an API change is intentional, regenerate the snapshots and commit them in the same commit:

```sh
go run ./internal/tools/apidump
```

Inspect `git diff api/` before committing so unintended exports never slip in.

### Fixtures and protocol corpus

Unit tests run offline without a live OpenWrt router:

- `internal/testdata/` holds sanitized JSON responses captured on OpenWrt 24.10.5 (`x86_generic`) and OpenWrt 25.12.0-rc1 (`rax3000m`). To capture new responses or method signatures from a router and turn them into regression tests, follow [docs/CONTRIBUTING_DATA.md](../docs/CONTRIBUTING_DATA.md).
- `internal/blobmsg/testdata/` holds captured binary `ubus` frames and the upstream OpenWrt `ubus` fuzz corpus (`fuzz_corpus/`), verified by `internal/blobmsg/golden_test.go` during `go test ./...` to ensure malformed binary frames return errors rather than panicking.

### Live OpenWrt container integration tests

With Docker installed locally, `scripts/openwrt-lab.sh` starts an ephemeral OpenWrt `24.10` (`24-10`) or `25.12` (`25-12`) container running `ubusd`, `rpcd`, and `uhttpd` over a Unix socket, HTTP JSON-RPC, and HTTPS JSON-RPC (with a generated CA):

```sh
./scripts/openwrt-lab.sh start 24-10 /tmp/openwrt-lab
. /tmp/openwrt-lab/env.sh
go test -v -race -count=1 -p 1 -tags integration ./...
./scripts/openwrt-lab.sh stop /tmp/openwrt-lab
```

Pass `25-12` instead of `24-10` to run the same suite against OpenWrt 25.12. CI runs both container targets on every push and pull request.

## Changes

- Include unit tests with every bug fix or feature. A bug fix should come with a test that fails without it.
- Wrap underlying errors with `%w` and root-package sentinels (`goubus.ErrNotFound`, `goubus.ErrInvalidResponse`, and others) so callers can match them with `errors.Is` and `errors.AsType`.
- Use `log/slog` where logging is needed, keep transports silent by default, and never log passwords or `ubus_rpc_session` tokens.
- User-visible changes belong under `## Unreleased` in [CHANGELOG.md](../CHANGELOG.md). GitHub release notes are extracted directly from `CHANGELOG.md` by `scripts/release-notes.sh`.
- Keep commits focused and follow [Conventional Commits](https://www.conventionalcommits.org/) (`feat`, `fix`, `docs`, `ci`, `test`, `refactor`, `chore`, with an optional scope such as `fix(uci): ...`).

## License

By contributing, you agree that your contribution is licensed under the [MIT License](../LICENSE).
