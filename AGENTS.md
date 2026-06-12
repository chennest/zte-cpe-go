# PROJECT KNOWLEDGE BASE

**Generated:** 2026-06-12
**Commit:** a54f088
**Branch:** master

## OVERVIEW

Go CLI + library for ZTE CPE router management (G5TS 5G, MF289F LTE). Port of [zte-cpe-rs](https://github.com/1zun4/zte-cpe-rs). Cobra CLI + Prometheus exporter. GPL-3.0.

## STRUCTURE

```
zte-cpe-go/
├── main.go              # Entrypoint, calls cmd.Execute()
├── cmd/                 # CLI layer (cobra) — NOT multi-binary layout, just package cmd
│   ├── root.go          # RootCmd + 24 subcommand registration in init()
│   ├── commands.go      # 23 subcommands + flag wiring (973 lines)
│   └── serve.go         # "serve" subcommand (Prometheus exporter HTTP)
├── pkg/
│   ├── router/          # Shared interface + types
│   │   ├── router.go    # RouterClient interface (20+ methods), config types, ErrNotSupported
│   │   └── bands.go     # LteBand enum + bitmask conversion
│   ├── g5ts/            # ZTE G5TS client (ubus JSON-RPC 2.0 over HTTP)
│   │   ├── client.go    # G5tsClient struct, login/session/RPC logic (860 lines)
│   │   ├── commands.go  # UbusCommand interface + concrete command structs
│   │   ├── aes.go       # AES encryption for password hashing
│   │   └── commands/    # Additional command definitions
│   ├── mf289f/          # ZTE MF289F client (goform HTTP API)
│   │   ├── client.go    # Mf289fClient struct, LD/token/goform logic (448 lines)
│   │   ├── commands.go  # GoformCommand interface + concrete command structs
│   │   └── commands/    # Additional command definitions
│   └── exporter/        # Prometheus collector — scrapes RouterClient, emits metrics (643 lines)
└── Dockerfile           # Multi-stage build (golang:1.23-alpine → alpine:3.20), serves on :9101
```

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Add new CLI command | `cmd/commands.go` | Add `var xxCmd`, `init()` block wires flags, `root.go` `init()` registers it |
| Add new router operation | `pkg/router/router.go` | Add method to `RouterClient` interface, then implement in both `g5ts/` and `mf289f/` |
| Add G5TS-specific API | `pkg/g5ts/client.go` + `commands.go` | Add UbusCommand struct + client method |
| Add MF289F-specific API | `pkg/mf289f/client.go` + `commands.go` | Add GoformCommand struct + client method |
| Modify Prometheus metrics | `pkg/exporter/exporter.go` | All metric descriptors + collect logic |
| Change CLI flags/env vars | `cmd/commands.go` or `cmd/serve.go` | Flags in `init()`, env vars via `ZTE_*` prefix |
| Fix auth/login flow | `pkg/g5ts/client.go` (ubus) or `pkg/mf289f/client.go` (goform) | Different protocols per device |

## ARCHITECTURE

```
CLI (cmd/) ──uses──→ pkg/router.RouterClient (interface)
                        ├── pkg/g5ts.G5tsClient    (implements, ubus JSON-RPC)
                        └── pkg/mf289f.Mf289fClient (implements, goform HTTP)
Exporter (pkg/exporter/) ──uses──→ pkg/router.RouterClient
```

- **Device protocol split**: G5TS uses ubus JSON-RPC 2.0 (`/ubus`), MF289F uses goform HTTP POST (`/goform/goform_set_cmd_process`)
- **Command pattern**: Each device defines its own command interface (`UbusCommand` / `GoformCommand`) with concrete structs
- **Client factory**: `cmd/commands.go` has `getClient()` that maps `--type` flag → concrete client
- **Shared mutable state**: `routerType`, `routerURL`, `password` are package-level vars in `cmd/`

## CONVENTIONS

- **Context-first**: All router methods take `context.Context` as first param
- **JSON raw responses**: `GetStatus`, `GetNetworkInfo`, etc. return `json.RawMessage` — caller unmarshals
- **ErrNotSupported**: Devices return `router.ErrNotSupported` for unsupported operations (e.g., MF289F doesn't support `GetNetworkInfo`)
- **Password hashing**: G5TS uses SHA256+salt (AES), MF289F uses SHA256+LD token
- **Env vars**: `ZTE_TYPE`, `ZTE_URL`, `ZTE_PASSWORD`, `ZTE_LISTEN`, `ZTE_INTERVAL` (all optional, flags take precedence)
- **Module path discrepancy**: `go.mod` says `github.com/1zun4/zte-cpe-go`, README install uses `github.com/chennest/zte-cpe-go` (fork)
- **Bilingual docs**: `README.md` (EN) + `README_zh.md` (ZH)
- **Git commits**: Conventional commit style (`feat:`, `docs:`, `chore:`)

## COMMANDS

```bash
# Build
go build -o zte-cpe .
CGO_ENABLED=0 go build -ldflags="-s -w" -o zte-cpe .  # production

# Docker
docker build -t zte-cpe-go .

# Test (none exist yet)
go test ./...

# Run exporter
zte-cpe serve -t g5ts -u http://192.168.0.1 -p PASSWORD --listen :9101 --interval 30
```

## NOTES

- **No tests**: Zero `_test.go` files. `RouterClient` interface is mock-ready but no mocks exist
- **No CI/CD**: No `.github/workflows`, no Makefile, no linting config
- **No linter config**: No `.golangci.yml` — code is clean but unenforced
- **Large files**: `cmd/commands.go` (973 lines), `pkg/g5ts/client.go` (860 lines), `pkg/exporter/exporter.go` (643 lines)
- **Empty dir**: `cmd/zte-cpe/` exists but is empty (artifact)
- **Pre-built binary**: `zte-cpe` binary in root (gitignored but present locally)
