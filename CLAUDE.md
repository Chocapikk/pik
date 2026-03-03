# Pik

Go exploit framework. Interactive console with multi-session, multi-C2, CmdStager, and standalone SDK.

## Architecture

```
sdk/                Types, interfaces, constants (source of truth)
  exploit.go        Exploit, Checker, CmdStager, CommandExecutor interfaces
  context.go        Execution context with Commands(), Target(), SetCommands(), SetTarget()
  info.go           Info, Target, Lab, Service, Author, CheckResult, Reliability
  lab.go            LabManager interface + late binding via SetLabManager()
  run.go            sdk.Run() with late binding via SetRunner()

pkg/cli/            CLI + standalone runner
  standalone.go     RunStandaloneWith() + init() registers runner via sdk.SetRunner()
  cmd_lab.go        `pik lab start|stop|status|run` subcommands

pkg/console/        Business logic + readline REPL
  console.go        Console struct, command registry, exec dispatcher
  readline.go       Readline-based Run/RunWith, tab completion, history
  api.go            Exported methods for TUI (SetProgram, Exec, UseByName, etc.)
  options.go        Option type (= types.Option), init, get/set/build params, TARGET<->RPORT sync
  commands.go       User commands (use, set, show, info, target, resource, lab, clear, etc.)
  exploit.go        Check, exploit, cmdstager, C2 wiring, BuildContext bridge
  sessions.go       Session list, interact, kill, backend lifecycle
  display.go        Table rendering with ANSI-aware log.Pad(), styling helpers

pkg/tui/            Bubbletea TUI dashboard (optional, `pik tui`)
  run.go            TUI entry point Run/RunWith, tea.Program setup
  model.go          Bubbletea model (tabs, viewport, input, focus zones)
  tabs.go           Browse/Config/Sessions tab rendering + key handling
  fuzzy.go          Fuzzy picker overlay (module/payload selection)
  tuiwriter.go      io.Writer adapter that forwards output to bubbletea viewport
  completion.go     Tab completion for TUI input
  history.go        Command history persistence (~/.pik_history)

pkg/types/          Shared types between console and tui (breaks import cycle)
  types.go          Option, FuzzyItem, FuzzySelectMsg, SessionInteractMsg, ClearOutputMsg

pkg/runner/         Execution engine
  single.go         RunSingle: resolveTarget -> deliverPayload or deliverCmdStager
  scanner.go        Multi-target scanning with thread pool
  context.go        BuildContext() shared between console and runner
  options.go        Enrichers: enrichBase, enrichC2, enrichCmdStager, enrichScan

pkg/lab/            Docker lab management (Docker Engine SDK, no shell-out)
  lab.go            Start/Stop/Status/IsRunning, WaitReady, WaitProbe, Target, DockerGateway

pkg/c2/             C2 backends
  c2.go             Backend interface + SessionHandler + factory registry
  shell/            TCP reverse shell (session.Manager)
  sslshell/         TLS reverse shell (self-signed cert, session.Manager)
  httpshell/        HTTP polling C2 (curl/wget/php/python payloads)
  sliver/           Sliver gRPC integration (mTLS, staging, implant gen)
  session/          Session + Manager (accept loop, registry, Ctrl+Z background)

pkg/http/           HTTP client
  client.go         Session, Run, AutoScheme (HTTPS probe), WithProxy, NormalizeURI
  debug.go          HTTP request/response tracing (--debug flag)
  option.go         HTTP enricher (SSL, USER_AGENT, TARGETURI - all advanced)

pkg/payload/        Payload generators
  reverse.go        TCP, TLS, HTTP reverse shell one-liners
  registry.go       Payload metadata + registration
  stager.go         Download-and-execute wrappers (curl, wget, certutil, etc.)
  transform.go      Encoding chain (base64, hex, url, gzip, comment trail)

pkg/cmdstager/      Chunked command delivery (printf, bourne flavors)
pkg/stager/         TCP stager shellcode (memfd_create, XOR, fileless)
```

## Key patterns

- Types in `sdk/`, all internal packages import `sdk`. No re-export, no codegen.
- `sdk.Run()` uses late binding: `pkg/cli.init()` calls `sdk.SetRunner(RunStandaloneWith)`.
- Standalone binaries need: `import "sdk"` + `import _ "pkg/cli"`. Add `_ "pkg/lab"` + `sdk.WithLab()` for lab support.
- Option enrichers (`sdk.RegisterEnricher`) auto-inject LHOST, LPORT, C2, HTTP options etc.
- Advanced options via `sdk.OptAdvanced()`. Only TARGET, LHOST, LPORT, PAYLOAD in `show options`.
- Module request paths are relative (`"install.php"` not `"/install.php"`). NormalizeURI handles TARGETURI.
- Target types are module-defined strings (e.g. `"cmd"`, `"dropper"`). Runner dispatches on type, module switches in Exploit().
- CmdStager: runner sets commands on Context via `SetCommands()`, module reads `run.Commands()` and loops.
- Check results: `sdk.Vulnerable("reason")`, `sdk.Safe("reason")`, `sdk.Unknown(err)`.
- Console commands are a dynamic map (`registerCommands()`). Adding a command = one line.
- Console `use <id>` selects module + target by global index from `list`.
- C2 backends self-register via `c2.RegisterFactory()` in `init()`.
- Constants: `cmdstager.DefaultLineMax` (2047), `cmdstager.DefaultFlavor` (printf).
- HTTP client preserves raw header casing (no canonicalization). Some servers are case-sensitive.
- Author: `sdk.Author{Name, Handle, Email}`. Email must use `<user[at]domain>` format, Register() panics on raw @.
- Lab: `sdk.NewLabService(name, image, ports...).WithEnv(k, v).WithHealthcheck(cmd)`. `sdk.Rand("label")` for random shared creds.
- Lab internals: late binding via `sdk.SetLabManager()`, Docker SDK isolated in `pkg/lab`, random host ports, 127.0.0.1 only, DNS aliases by service name, labels for tracking.
- `pik lab run <module>`: start lab, TCP wait, probe Check() until app ready, auto LHOST (docker gateway), exploit.
- `pik` or `pik console`: readline REPL (default). `pik tui`: bubbletea TUI dashboard.
- TUI uses `MsgSender` interface (not `*tea.Program` directly) to avoid pulling bubbletea into standalone builds.
- Shared types in `pkg/types/` break the import cycle: `console` -> `types`, `tui` -> `types` + `console`.
- `lab start` auto-sets TARGET (from port bindings) and LHOST (from Docker gateway).
- Setting TARGET with a port auto-syncs RPORT, and vice versa.

## Go conventions

- Receivers: single letter (`(s *Session)`, `(m *Manager)`)
- Loop vars: `i`, `j`, `k`, `v` in `for`/`range`
- Everything else: descriptive (`mod`, `params`, `backend`, `entry`, `scan`)

## Build and test

```bash
make build                    # dev build
make build VERSION=1.0.0      # versioned build (-ldflags -X pkg/cli.Version=...)
make test                     # go test ./...
make vet                      # go vet ./...
```

Test exploits against lab containers:
```bash
pik lab run langflow_validate_code_rce          # cold start to shell, zero config
pik lab start langflow_validate_code_rce        # just start the lab
pik lab status                                  # list running labs
pik lab stop langflow_validate_code_rce         # tear down
go run ./cmd/pik run opendcim_sqli_rce -t 127.0.0.1:18091 -s LHOST=<ip> -s LPORT=4444
```

## Commit style

`Feat:`, `Fix:`, `Refactor:`, `Docs:`, `Test:`, `Perf:`. No Co-Authored-By.
