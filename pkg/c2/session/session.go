package session

import (
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/pkg/sigutil"
)

// Session wraps a single reverse shell connection.
type Session struct {
	ID         int
	Conn       net.Conn
	RemoteAddr string
	CreatedAt  time.Time

	mu    sync.Mutex
	alive bool
}

func newSession(id int, conn net.Conn) *Session {
	return &Session{
		ID:         id,
		Conn:       conn,
		RemoteAddr: conn.RemoteAddr().String(),
		CreatedAt:  time.Now(),
		alive:      true,
	}
}

// Alive reports whether the session is still open.
func (s *Session) Alive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.alive
}

// SetAlive sets the alive state. Used by backends that don't use net.Conn.
func (s *Session) SetAlive(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alive = v
}

// Close terminates the session.
func (s *Session) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.alive {
		return
	}
	s.alive = false
	s.Conn.Close()
}

// Interact takes over stdin/stdout for interactive shell access.
// Ctrl+Z (SIGTSTP) backgrounds the session and returns.
// InteractTUI is like Interact but clears the screen first (used when resuming from TUI).
func (s *Session) InteractTUI() {
	fmt.Fprint(os.Stderr, "\033[2J\033[H")
	output.Status("Session %d (%s) - Ctrl+Z to return to TUI", s.ID, s.RemoteAddr)
	fmt.Fprintln(os.Stderr)
	s.interact()
}

// Interact takes over stdin/stdout for interactive shell access.
// Ctrl+Z (SIGTSTP) backgrounds the session and returns.
func (s *Session) Interact() {
	output.Status("Interacting with session %d (%s)", s.ID, s.RemoteAddr)
	output.Status("Press Ctrl+Z to background session")
	s.interact()
}

func (s *Session) interact() {
	done := make(chan struct{})
	bg := make(chan os.Signal, 1)
	sigutil.NotifySuspend(bg)
	defer sigutil.StopSuspend(bg)

	// Use a pipe so we can stop the stdin copy goroutine
	pr, pw := io.Pipe()

	go func() {
		io.Copy(os.Stdout, s.Conn)
		close(done)
	}()
	go func() {
		io.Copy(s.Conn, pr)
	}()
	go func() {
		io.Copy(pw, os.Stdin)
	}()

	select {
	case <-done:
		pw.Close()
		s.Close()
		output.Warning("Session %d closed", s.ID)
	case <-bg:
		pw.Close()
		output.Println()
		output.Status("Session %d backgrounded", s.ID)
	}
}
