# Pik

Go exploit framework. Interactive console with multi-session, multi-C2, CmdStager, and standalone SDK.

## Architecture

```
sdk/                Types, interfaces, constants (source of truth)
  exploit.go        Exploit, Checker, CmdStager, CommandExecutor interfaces
  context.go        Execution context with Commands(), Target(), SetCommands(), SetTarget()
  info.go           Info, Target, Lab, Service, Author, CheckResult, Reliability, Feature
  fake.go           Faker interface + late binding via SetFaker()
  timing.go         SleepCheck() helper for sleep-based timing vuln checks
  lab.go            LabManager interface + late binding via SetLabManager()
  run.go            sdk.Run() with late binding via SetRunner()

pkg/cli/            CLI + standalone runner
  standalone.go     RunStandaloneWith() + init() registers runner via sdk.SetRunner()
  build.go          `pik build` + `pik generate` (standalone binary/source generation)
  cmd_lab.go        `pik lab start|stop|status|run` subcommands (framework)
  cmd_lab_standalone.go  Standalone lab subcommands (no module arg needed)

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
  context.go        BuildContext() shared between console and runner (wires bridges via sdk factories)
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

pkg/protocol/http/  HTTP client (registers SendFactory + PoolFactory via init())
  client.go         Session, Run, AutoScheme (HTTPS probe), WithProxy, NormalizeURI
  debug.go          HTTP request/response tracing (HTTP_TRACE option)
  option.go         init(): registers sdk.SetSendFactory + sdk.SetPoolFactory

pkg/protocol/tcp/   Raw TCP client (registers DialFactory via init())
  client.go         Session, Dial, FromModule, Send, Recv, SendRecv
  debug.go          TCP hex dump tracing (TCP_TRACE option)
  option.go         init(): registers sdk.SetDialFactory

pkg/enricher/       Protocol option enrichers (imported by runner)
  enricher.go       Init: registers HTTP + TCP option enrichers
  http.go           HTTP enricher (SSL, USER_AGENT, TARGETURI, HTTP_TRACE - all advanced)
  tcp.go            TCP enricher (TCP_TIMEOUT, TCP_TRACE - all advanced)

pkg/fake/           Faker integration (gofakeit, lazy-loaded via init())
  fake.go           init() registers sdk.SetFaker(), wraps gofakeit

pkg/encode/         Encoding helpers
  encode.go         Base64, Hex, URL, UTF16LE, XOR, ROT13
  binary.go         Buffer (fluent binary packing: Byte, Uint16/32/64, String, NameList, Zeroes)

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
- Standalone binaries need: `import "sdk"` + `import _ "pkg/cli"` + `import _ "pkg/protocol/<proto>"`. Add `_ "pkg/lab"` + `sdk.WithLab()` for lab support. Features (`pkg/fake`, `pkg/xmlutil`) are auto-added by scaffold based on `Info().Features`.
- Option enrichers (`sdk.RegisterEnricher`) auto-inject LHOST, LPORT, C2, HTTP/TCP options etc. Enrichers in `pkg/enricher/`, protocol factories in `pkg/protocol/*/option.go`.
- Protocol late binding: `sdk.RegisterSenderFactory` (structured protos), `sdk.SetDialFactory` (raw connections), `sdk.SetPoolFactory` (pooling). Only imported protocols are compiled into the binary.
- `pik build <module>`: compiles standalone binary. `pik generate <module>`: emits source code. Both auto-detect protocol from module path.
- Advanced options via `sdk.OptAdvanced()`. Only TARGET, LHOST, LPORT, PAYLOAD in `show options`.
- Module request paths are relative (`"install.php"` not `"/install.php"`). NormalizeURI handles TARGETURI.
- Target types are module-defined strings (e.g. `"cmd"`, `"dropper"`). Runner dispatches on type, module switches in Exploit().
- CmdStager: runner sets commands on Context via `SetCommands()`, module reads `run.Commands()` and loops.
- Check results: `sdk.Vulnerable("reason")`, `sdk.Safe("reason")`, `sdk.Unknown(err)`.
- Timing checks: `sdk.SleepCheck(run, func(delay int) error { ... })` - 3 rounds, random delay 2-4s, returns Vulnerable if 2+ match.
- Fake data: `sdk.Fake().DomainName()`, `.Email()`, `.URL()`, `.IPv4Address()`, etc. Late binding via `pkg/fake` init(). Module declares `Features: []sdk.Feature{sdk.FakeData}`.
- Console commands are a dynamic map (`registerCommands()`). Adding a command = one line.
- Console `use <id>` selects module + target by global index from `list`.
- C2 backends self-register via `c2.RegisterFactory()` in `init()`.
- Constants: `cmdstager.DefaultLineMax` (2047), `cmdstager.DefaultFlavor` (printf).
- `run.Send()` is polymorphic: dispatches by request type. `run.Send(sdk.HTTPRequest{...})` for HTTP. Future protos add new request types.
- `run.Dial()` -> `sdk.Conn` for raw persistent connections (TCP). Send/Recv/SendRecv/Close.
- `sdk.HTTPRequest` / `sdk.HTTPResponse` - explicitly named, implements `sdk.Sendable` interface.
- Binary packing: `sdk.NewBuffer().Byte(0x14).Uint32(val).String("data").NameList("a","b").Build()` - generic fluent builder for crafting protocol messages.
- Protocol tracing: `HTTP_TRACE=true` or `TCP_TRACE=true` as advanced options (not --debug).
- HTTP client preserves raw header casing (no canonicalization). Some servers are case-sensitive.
- HTTP sender factory drains response body before returning (ensures accurate Elapsed() timing for sleep-based checks).
- Features (standalone build deps): `sdk.Feature` declared in `Info().Features`. `sdk.XML` (xmlutil), `sdk.FakeData` (faker). Scaffold template auto-includes matching `_ "pkg/..."` imports. Infrastructure deps (HTTPServerModule) use interface markers instead.
- Author: `sdk.Author{Name, Handle, Email, Company}`. Email must use `<user[at]domain>` format, Register() panics on raw @.
- References: `sdk.CVE("2025-XXXXX")`, `sdk.GHSA("xxxx-yyyy-zzzz")` (global) or `sdk.GHSA("xxxx-yyyy-zzzz", "owner/repo")` (repo-scoped).
- String helpers: `sdk.Contains(s, sub)`, `sdk.ContainsI(s, sub)` (case-insensitive).
- Lab: `sdk.NewLabService(name, image, ports...).WithEnv(k, v).WithHealthcheck(cmd)`. `sdk.Rand("label")` for random shared creds.
- Lab internals: late binding via `sdk.SetLabManager()`, Docker SDK isolated in `pkg/lab`, random host ports, 127.0.0.1 only, DNS aliases by service name, labels for tracking.
- `pik lab run <module>`: start lab, TCP wait, probe Check() until app ready, auto LHOST (docker gateway), exploit.
- `pik` or `pik console`: readline REPL (default). `pik tui`: bubbletea TUI dashboard.
- TUI uses `MsgSender` interface (not `*tea.Program` directly) to avoid pulling bubbletea into standalone builds.
- Shared types in `pkg/types/` break the import cycle: `console` -> `types`, `tui` -> `types` + `console`.
- `lab start` auto-sets TARGET (from port bindings) and LHOST (from Docker gateway).
- Setting TARGET with a port auto-syncs RPORT, and vice versa.

## Adding a new structured protocol (e.g. gRPC, GraphQL)

1. Define request/response types in `sdk/types.go`:
```go
type GRPCRequest struct { Method string; Body string }
type GRPCResponse struct { Body []byte; Error string }
func (GRPCRequest) protocol() string { return "grpc" }
```

2. Add the case in `sdk/context.go` Send():
```go
case GRPCRequest:
    fn, ok := c.senders["grpc"]
    if !ok { return nil, fmt.Errorf("no gRPC client configured") }
    return fn.(func(GRPCRequest) (*GRPCResponse, error))(r)
```

3. Create `pkg/protocol/grpc/` with factory registration in init():
```go
sdk.RegisterSenderFactory("grpc", func(params sdk.Params) any {
    // return func(sdk.GRPCRequest) (*sdk.GRPCResponse, error)
})
```

4. Add enricher options in `pkg/enricher/grpc.go` if needed.
5. Add `protoFromPath` case in `pkg/cli/helpers.go` for standalone builds.

Module author just writes: `run.Send(sdk.GRPCRequest{Method: "GetUser"})`.

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
pik lab run langflow_validate_code_rce          # HTTP module: cold start to shell
pik lab run erlang_ssh_rce                      # TCP module: cold start to shell
pik lab start erlang_ssh_rce                    # just start the lab
pik lab status                                  # list running labs
pik lab stop erlang_ssh_rce                     # tear down
```

Generate standalone exploits:
```bash
pik build erlang_ssh_rce                        # compile standalone binary
pik build erlang_ssh_rce -o ssh_exploit         # custom output name
pik generate erlang_ssh_rce                     # emit source code (no compile)
pik generate erlang_ssh_rce -o ./my_exploit     # custom output directory
```

## Commit style

`Feat:`, `Fix:`, `Refactor:`, `Docs:`, `Test:`, `Perf:`. No Co-Authored-By.
