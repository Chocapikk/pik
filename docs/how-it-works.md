# How Pik Works

A deep dive into what Pik solves, why it exists, and the design decisions behind it.

## The Problem

A security researcher finds a CVE. To prove it, they need to:

1. **Write a PoC script** - 200+ lines of boilerplate just to send HTTP requests, handle SSL, parse responses, and demonstrate the bug. Most of this code has nothing to do with the vulnerability itself.

2. **Set up a test environment** - Manually pull Docker images, configure containers, wire up networking, seed databases, wait for services to start. Every CVE needs a different setup.

3. **Write a scanner** - If the vulnerability affects more than one target, the PoC needs threading, input parsing, output formatting, proxy support, and rate limiting. That's another 300 lines.

4. **Build a portable binary** - The PoC is a Python script with dependencies, or a Go program that assumes your entire toolchain is installed. To share it with a colleague, you need to package it somehow.

5. **Rewrite for Metasploit** - If the researcher wants wide distribution, they rewrite the whole thing in Ruby with Metasploit's conventions, mixins, and module structure.

Each step is throwaway code, rebuilt from scratch for every CVE. The actual vulnerability logic - the part that matters - is maybe 20 lines buried inside hundreds of lines of infrastructure code.

This is the workflow for *one* vulnerability. Multiply it by every CVE a researcher works on in a year.

## What Pik Changes

Pik separates **what you want to do** (the exploit logic) from **how to do it** (networking, C2, lab setup, scanning, binary compilation).

The researcher writes a module - typically 30-50 lines of pure vulnerability logic. No networking boilerplate, no Docker configuration, no CLI argument parsing. Just the exploit.

From that single module, the framework automatically provides:

- **Check** - Is this target vulnerable?
- **Exploit** - Trigger the vulnerability and get a shell
- **Scanner** - Test thousands of targets with threading, proxies, and output formats
- **Lab** - One-command Docker environment with the vulnerable application
- **Standalone binary** - Self-contained executable with everything built in
- **Source export** - Generated Go source code, ready to compile anywhere

The module author writes the logic once. The framework handles everything else.

## The Two Verbs

All network communication in a Pik module reduces to two operations.

### `run.Send(request)` - Structured request/response

`Send()` is a single entry point for all structured protocols. Today it handles HTTP. Tomorrow it could handle gRPC, GraphQL, SOAP, or any protocol that follows a request/response pattern - without changing a single module.

The idea: the module describes *what* it wants to send by choosing a request type. The type itself selects the protocol. `HTTPRequest` means HTTP. A future `GRPCRequest` would mean gRPC. The module never picks a protocol by name - the type *is* the protocol selector.

```go
// HTTP today
resp, err := run.Send(sdk.HTTPRequest{
    Method: "POST",
    Path:   "api/v1/exec",
    Body:   payload,
})

// gRPC tomorrow - same run.Send(), different request type
// resp, err := run.Send(sdk.GRPCRequest{
//     Method: "UserService/GetUser",
//     Body:   protobuf,
// })
```

Internally, `Send()` does a type switch on the request. Each case dispatches to a protocol handler that was registered at startup. Adding a protocol means adding a case and a handler - the module-facing API doesn't change. This is what makes modules portable: a module written for HTTP works with any HTTP transport the framework provides (direct, proxied, pooled), and the same dispatch pattern extends to any future protocol.

Today `Send()` returns `*sdk.HTTPResponse` because HTTP is the only structured protocol. When a second protocol arrives, the return type will generalize (likely to an interface or generic), and existing modules will keep working.

The response comes back with helper methods:

```go
if resp.Contains("root:") {
    return sdk.Vulnerable("/etc/passwd leaked")
}

// Other response helpers
resp.ContainsAny("root:", "admin:")  // match any substring
resp.Header("X-Custom")             // case-insensitive header lookup
body, _ := resp.BodyBytes()          // raw response bytes (cached)
resp.JSON(&result)                   // unmarshal JSON body into struct
```

For blind exploits where you don't need the response, set `FireAndForget`:

```go
run.Send(sdk.HTTPRequest{
    Method:        "POST",
    Path:          "trigger",
    Body:          payload,
    FireAndForget: true,  // send and move on, ignore response and errors
})
```

### `run.Dial()` - Raw connections

For protocols that don't fit a request/response model - or when you need to speak a custom binary protocol - you open a raw connection and send bytes directly.

```go
conn, err := run.Dial()
defer conn.Close()

// Send a crafted packet and read the response
resp, err := conn.SendRecv(craftedPacket, 4096)
```

This is for things like SSH handshakes, custom TCP protocols, or any situation where you need byte-level control. The framework still handles connection setup, timeouts, and tracing - you just work with the data.

### Why only two verbs?

Every network interaction is either "send something, get something back" or "open a pipe and talk". There is no third pattern. By reducing the API surface to these two operations, Pik makes modules trivially portable across protocols and transports.

## Protocol Dispatch

When you write `run.Send(sdk.HTTPRequest{...})`, here's what happens under the hood:

```
Module calls run.Send(sdk.HTTPRequest{...})
         |
         v
    Send() does a type switch on the request
         |
         v
    case HTTPRequest:
      looks up "http" in c.senders map
         |
         v
    The sender was registered by pkg/protocol/http
    during init() via sdk.RegisterSenderFactory("http", ...)
         |
         v
    The sender closure converts HTTPRequest into a real HTTP request
    Applies SSL, proxy, user-agent, URI normalization
         |
         v
    Response comes back as *sdk.HTTPResponse
    Module gets it from run.Send()
```

The actual dispatch code in `sdk/context.go` is a type switch:

```go
func (c *Context) Send(req Sendable) (*HTTPResponse, error) {
    switch r := req.(type) {
    case HTTPRequest:
        fn, ok := c.senders["http"]
        if !ok {
            return nil, fmt.Errorf("no HTTP client configured")
        }
        return fn.(func(HTTPRequest) (*HTTPResponse, error))(r)
    default:
        return nil, fmt.Errorf("unsupported protocol: %s", req.protocol())
    }
}
```

Each request type implements the `Sendable` interface - a single unexported method `protocol() string`. It's unexported because module authors never call it directly; it's internal to the dispatch mechanism.

The module sees none of this machinery. It sees `run.Send(request) -> response`. The entire protocol stack is invisible.

Adding a new protocol means adding a new case to this switch. For example, adding gRPC support:

1. Define `GRPCRequest` and `GRPCResponse` types in `sdk/types.go`
2. Make `GRPCRequest` implement `Sendable` (`func (GRPCRequest) protocol() string { return "grpc" }`)
3. Add the `case GRPCRequest:` branch in `Send()`
4. Write the handler in `pkg/protocol/grpc/`
5. Register it via `RegisterSenderFactory("grpc", ...)`

Every existing module that uses `run.Send()` keeps working. New modules can use gRPC with the same `run.Send()` call. Zero changes to the runner, the CLI, the TUI, the scanner, or the standalone builder.

## Late Binding and Dead Code Elimination

Protocols register themselves via Go's `init()` mechanism. When you import a package, its `init()` function runs automatically. When you don't import it, the entire package is excluded from compilation.

```go
// pkg/protocol/http/option.go
func init() {
    sdk.SetPoolFactory(WithPool)
    sdk.RegisterSenderFactory("http", func(params sdk.Params) any {
        run := FromModule(params)  // creates HTTP session from module options
        return func(req sdk.HTTPRequest) (*sdk.HTTPResponse, error) {
            resp, err := run.Send(Request{
                Method:  req.Method,
                Path:    req.Path,
                Query:   url.Values(req.Query),
                Form:    url.Values(req.Form),
                Body:    req.BodyReader(),
                Headers: req.Headers,
                // ...
            })
            // convert internal response to sdk.HTTPResponse
            return &sdk.HTTPResponse{StatusCode: resp.StatusCode, Body: resp.Body, Headers: headers}, nil
        }
    })
}
```

The factory receives the module's parameters (TARGET, SSL, TARGETURI, PROXIES, etc.) and creates an HTTP session preconfigured with those values. It returns a closure that translates `sdk.HTTPRequest` to the internal HTTP client's format and back. The module never touches any of this - it just calls `run.Send()`.

This means:

- A **framework build** (`pik`) imports all protocols. It can run any module.
- A **standalone HTTP exploit** imports only `pkg/protocol/http`. No TCP code is compiled.
- A **standalone TCP exploit** imports only `pkg/protocol/tcp`. No HTTP code is compiled.

The Go compiler's dead code elimination takes care of the rest. A standalone TCP exploit binary is around 8 MB and contains zero HTTP client code, zero TLS negotiation, zero cookie handling. It only contains what it uses.

This is the same mechanism used for C2 backends, payload generators, lab support, and the CLI itself. Everything is opt-in at compile time through imports.

```go
// Standalone exploit - only what you need
import (
    "github.com/Chocapikk/pik/sdk"
    _ "github.com/Chocapikk/pik/pkg/cli"            // CLI runner
    _ "github.com/Chocapikk/pik/pkg/protocol/http"   // HTTP protocol
)
```

Add `_ "pkg/protocol/tcp"` if you need TCP. Add `_ "pkg/lab"` if you want lab support. Each import pulls in exactly that capability and nothing more.

## The Module Lifecycle

Here's the full journey from module registration to a root shell:

```
1. Module author writes 40 lines, calls sdk.Register()
   The module is now in the global registry.
                    |
                    v
2. User runs: pik run my_exploit -t target -s LHOST=10.0.0.1
   The CLI parses options, finds the module in the registry.
                    |
                    v
3. The runner creates an sdk.Context
   - Looks up registered protocol factories (Send, Dial)
   - Wires C2 backend based on PAYLOAD/C2 options
   - Injects enriched options (SSL, USER_AGENT, TARGETURI, etc.)
   - The Context is a fully wired execution environment.
                    |
                    v
4. Check(run *sdk.Context) is called
   - The module uses run.Send() or run.Dial()
   - Framework handles the actual network I/O
   - Module returns a CheckResult: Vulnerable, Safe, Detected, or Unknown
                    |
                    v
5. If vulnerable, Exploit(run *sdk.Context) is called
   - The module sends the payload
   - Framework has already started the C2 listener
   - When the target connects back, a session is created
                    |
                    v
6. Shell opened
   - The user drops into an interactive shell
   - Ctrl+Z backgrounds the session
   - Multiple sessions can run concurrently
```

The module author only writes steps 1 and 4-5. Steps 2, 3, and 6 are the framework.

## Module Registration

Modules self-register with a single call in `init()`:

```go
package http

import "github.com/Chocapikk/pik/sdk"

type Langflow struct{ sdk.Pik }

func init() { sdk.Register(&Langflow{}) }
```

The module never provides its own name. `sdk.Register()` uses `runtime.Caller()` to inspect the call stack, extracts the file path relative to `modules/`, and derives the name automatically. A module at `modules/exploit/linux/http/langflow_validate_code_rce.go` becomes `exploit/linux/http/langflow_validate_code_rce`. The path also determines which enrichers apply - if it contains `/http/`, HTTP options are injected; if `/tcp/`, TCP options.

This means moving a module to a different directory changes its name and its enriched options. The file path is the identity.

## Architecture Layers

```
+---------------------------------------------------+
|  Module                                           |  The researcher writes this.
|  30-50 lines of vulnerability logic.              |  run.Send(), run.Dial(),
|  No boilerplate.                                  |  run.Payload(), run.Base64Bash()
+---------------------------------------------------+
|  SDK                                              |  The contract.
|  Types, interfaces, constants.                    |  HTTPRequest, Conn, Sendable,
|  Source of truth for the entire framework.         |  Context, Info, CheckResult
+---------------------------------------------------+
|  Runner                                           |  The wiring layer.
|  Creates Context, resolves factories,             |  BuildContext, enrichers,
|  dispatches Check/Exploit, manages C2.            |  target resolution, C2 lifecycle
+---------------------------------------------------+
|  Protocol Factories                               |  Network implementations.
|  pkg/protocol/http - full HTTP client             |  Auto-HTTPS, proxy, raw headers,
|  pkg/protocol/tcp  - raw TCP connections          |  tracing, connection pooling
+---------------------------------------------------+
|  Infrastructure                                   |  Everything else.
|  C2 backends (shell, sslshell, httpshell, sliver) |  CLI, TUI, lab management,
|  Docker labs, standalone builder, scanner engine   |  payload generation, encoding
+---------------------------------------------------+
```

A module only sees the SDK layer. Everything below is invisible. A module author never imports a protocol package, never touches the runner, never interacts with C2 code. They write exploit logic against the SDK interface and the framework handles the rest.

## The Context API

When `Check()` or `Exploit()` is called, the module receives an `*sdk.Context` - a fully wired execution environment. Here's what it provides:

**Network:**
- `Send(Sendable) (*HTTPResponse, error)` - polymorphic structured request
- `Dial() (Conn, error)` - raw TCP connection

**Parameters:**
- `Get(key string) string` - read any option value
- `Payload() string` - the generated reverse shell command

**Payload helpers:**
- `Base64Bash(cmd string) string` - wraps command in `echo <b64> | base64 -d | bash`
- `CommentTrail(cmd string) string` - appends `# <junk>` to prevent command-end artifacts in injections
- `RandText(n int) string` - random alphanumeric string (useful for canary markers in checks)

**Logging:**
- `Status(format, args...)` - informational message
- `Success(format, args...)` - green success message
- `Error(format, args...)` - red error message
- `Warning(format, args...)` - yellow warning

**Timing:**
- `Elapsed(start bool) float64` - call with `true` to start timer, `false` to read elapsed seconds

**CmdStager (dropper targets only):**
- `Commands() []string` - the chunked commands to execute
- `Target() Target` - the selected target metadata

## Check Results

Check functions return a `CheckResult` with a severity code and a reason. There are five levels, not just "vulnerable" or "safe":

```go
sdk.Vulnerable("RCE confirmed: marker found in response")     // Confirmed exploitable
sdk.Detected("version 4.2.1 is in the vulnerable range")      // Looks vulnerable but not confirmed
sdk.Safe("patched version 4.3.0 detected")                    // Confirmed not vulnerable
sdk.Unknown(err)                                                // Check failed, can't determine
```

`Appears` exists as well (between Detected and Vulnerable). The scanner considers both `Appears` and `Vulnerable` as positive results via `IsVulnerable()`.

The `Vulnerable` constructor accepts optional key-value details:

```go
sdk.Vulnerable("RCE confirmed", "version", "4.2.1", "os", "Ubuntu 22.04")
```

These details are included in JSON scan output for post-processing.

## What One Module Produces

A single module gives you six capabilities with zero additional code:

| Command | What it does |
|---|---|
| `pik check my_exploit -t target` | Check if one target is vulnerable |
| `pik run my_exploit -t target -s LHOST=ip` | Exploit + interactive shell |
| `pik check my_exploit -f targets.txt --threads 50` | Scan thousands of targets in parallel |
| `pik lab run my_exploit` | Start Docker lab + exploit, zero configuration |
| `pik build my_exploit` | Compile a standalone binary (~8 MB) |
| `pik generate my_exploit` | Export standalone source code |

The standalone binary has the same capabilities:

```bash
./my_exploit -t target -s LHOST=ip                        # Exploit
./my_exploit -t target --check                             # Check only
./my_exploit -f targets.txt --threads 50 -o vulns.txt      # Mass scan
./my_exploit lab run                                       # Lab + exploit
```

One module. Six outputs. No code duplication.

## Option Enrichment

Modules don't declare network options. The framework injects them automatically based on which protocols the module uses.

Enrichers are functions with the signature `func(mod Exploit, opts []Option) []Option`. They're chained - each receives the previous enricher's output. The registration order is:

1. **enrichBase** - `RPORT`, `LHOST`, `LPORT`, `PAYLOAD`, `PROXIES`, `AUTOCHECK`
2. **enrichC2** - `C2`, `C2CONFIG`, `SRVHOST`, `SRVPORT`, `TUNNEL`, `REMOTE_PATH`, `WAITSESSION`, `ARCH`, `FETCH_COMMAND`
3. **enrichCmdStager** (only if module implements `CmdStager`) - `CMDSTAGER`, `CMDSTAGER_LINEMAX`
4. **enrichScan** - `THREADS`
5. **enrichHTTP** (only if module path contains `/http/`) - `TARGETURI`, `SSL`, `USER_AGENT`, `HTTP_TIMEOUT`, `FOLLOW_REDIRECTS`, `KEEP_COOKIES`, `HTTP_TRACE`
6. **enrichTCP** (only if module path contains `/tcp/`) - `TCP_TIMEOUT`, `TCP_TRACE`

Most of these are marked as "advanced" - they don't clutter `show options`. Only `TARGET`, `LHOST`, `LPORT`, and `PAYLOAD` are visible by default. Power users can see everything with `show advanced`.

Setting `TARGET` with a port (e.g., `http://target:8080`) automatically syncs `RPORT` to `8080`. Setting `RPORT` updates `TARGET`. This bidirectional sync means you never set the same value twice.

## Connection Pooling

When scanning thousands of targets, creating a new HTTP transport per request would be wasteful. The framework handles this automatically.

The scanner calls `sdk.WithPool(ctx, threads, proxy)` before spawning goroutines. This creates a shared `http.Transport` with connection limits matching the thread count and stores it in the context. When the HTTP factory creates a session for each target, it checks for this pooled transport and reuses it instead of creating a new one.

```
Scanner.Run()
    |
    sdk.WithPool(ctx, 50, proxy)  // shared transport: 50 max conns
    |
    for each target (goroutine):
        params.Clone()            // clone params per target
        params.Set("TARGET", t)   // override target
        BuildContext(params, "")   // HTTP factory picks up pooled transport
        mod.Check(run)            // reuses shared connections
```

The module author writes the same `run.Send()` call whether it's checking one target or fifty thousand. The pooling is invisible.

## Lab Environments

Modules can declare Docker lab environments inline:

```go
Lab: sdk.Lab{
    Services: []sdk.Service{
        sdk.NewLabService("db", "mysql:5.7").
            WithEnv("MYSQL_ROOT_PASSWORD", sdk.Rand("db_pass")).
            WithHealthcheck("mysqladmin ping -h localhost"),
        sdk.NewLabService("web", "vulnerable-app:latest", "8080").
            WithEnv("DB_PASSWORD", sdk.Rand("db_pass")),
    },
},
```

`sdk.Rand("db_pass")` generates a random value at lab start. The same label across services resolves to the same value - so `db` and `web` share the same database password without hardcoding it.

`pik lab run my_exploit` does everything:

1. Pull images if needed
2. Create containers with randomized host ports
3. Wait for TCP ports to accept connections
4. Probe with `Check()` until the application responds (not just the port)
5. Auto-detect `LHOST` from the Docker gateway
6. Run the exploit
7. Drop into a shell

Labs bind to `127.0.0.1` only. They're never exposed to the network.

## C2 Backends

The same exploit works with any C2 backend. The module doesn't know or care which one is active.

```bash
# TCP reverse shell (default)
pik run my_exploit -t target -s LHOST=ip

# TLS encrypted reverse shell
pik run my_exploit -t target -s LHOST=ip -s C2=sslshell

# HTTP polling (traverses firewalls)
pik run my_exploit -t target -s LHOST=ip -s C2=httpshell

# Sliver C2 (full implant with persistence, pivoting, etc.)
pik run my_exploit -t target -s LHOST=ip -s C2=sliver -s C2CONFIG=~/.sliver/configs/operator.cfg
```

C2 backends register themselves the same way protocols do - via `init()` and a factory registry. Adding a new C2 backend doesn't touch any existing code.

The runner handles the C2 lifecycle: start listener before exploit, wait for incoming session after exploit, auto-interact when the shell connects back. The timeout is configurable via `WAITSESSION` (default 30 seconds).

## CmdStager - Chunked Delivery

Some targets can execute commands but have size limits per command. CmdStager solves this by splitting a payload into chunks and reassembling it on the target.

A module that supports chunked delivery implements both `Exploit` and the `CmdStager` interface:

```go
// CmdStager interface - marks the module as supporting chunked delivery
type CmdStager interface {
    ExecuteCommand(run *Context, cmd string) error
}
```

The module's `Exploit()` method reads the pre-generated chunks from `run.Commands()` and sends each one via its `ExecuteCommand()` method:

```go
func (m *MyExploit) Exploit(run *sdk.Context) error {
    for _, cmd := range run.Commands() {
        if err := m.ExecuteCommand(run, cmd); err != nil {
            return err
        }
    }
    return nil
}

func (m *MyExploit) ExecuteCommand(run *sdk.Context, cmd string) error {
    _, err := run.Send(sdk.HTTPRequest{
        Method: "POST",
        Path:   "execute",
        Form:   sdk.Values{"cmd": {cmd}},
    })
    return err
}
```

The runner orchestrates this: when the target type is `"dropper"`, it generates the chunked commands via `cmdstager.Generate()`, calls `run.SetCommands(commands)`, then calls `mod.Exploit(run)`. The module just iterates. The chunking strategy (printf, bourne shell), line length limits (default 2047), and temporary file path are all handled by the framework.

## Standalone Builds

When you run `pik build my_exploit`, the framework generates a `main.go` that imports only what the module needs:

```go
package main

import (
    "github.com/Chocapikk/pik/sdk"
    _ "github.com/Chocapikk/pik/pkg/cli"
    _ "github.com/Chocapikk/pik/pkg/lab"
    _ "github.com/Chocapikk/pik/pkg/protocol/tcp"   // auto-detected from module path
    _ "modules/exploit/linux/tcp"                    // only this module package
)

func main() {
    mods := sdk.List()
    if len(mods) == 0 {
        panic("no module registered")
    }
    sdk.Run(mods[0], sdk.WithConsole(), sdk.WithLab())
}
```

The protocol is auto-detected from the module path: if the path contains `/http/`, import `pkg/protocol/http`; if `/tcp/`, import `pkg/protocol/tcp`. The binary only contains the code it actually uses.

## Design Philosophy

Pik's architecture follows a few core principles:

**Write once, use everywhere.** A module is a pure function: vulnerability in, shell out. Every deployment mode (check, exploit, scan, lab, standalone) is a different way to call that same function.

**Late binding over configuration.** Protocols, C2 backends, payload generators, and even the CLI runner are all registered at init-time via Go's `init()` mechanism. Nothing is hardcoded. The binary contains only what it imports. This breaks import cycles and enables dead code elimination - a TCP standalone doesn't compile HTTP code, and vice versa.

**Types are the API.** `HTTPRequest` isn't just a struct - it's a protocol selector. The type system does the dispatch via a type switch in `Send()`. No string matching, no protocol constants, no configuration files. You send a typed request, the framework routes it.

**The module author is the user.** Every design decision optimizes for the person writing the module. The framework can be complex internally, as long as the module-facing API stays simple. Two verbs (`Send` and `Dial`), typed requests, helper methods, and automatic option enrichment - that's the entire surface area a module author needs to learn.
