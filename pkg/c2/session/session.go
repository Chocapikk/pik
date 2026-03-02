package session

import (
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Chocapikk/pik/pkg/output"
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
func (s *Session) Interact() {
	output.Status("Interacting with session %d (%s)", s.ID, s.RemoteAddr)
	output.Status("Press Ctrl+Z to background session")

	done := make(chan struct{})
	bg := make(chan os.Signal, 1)
	signal.Notify(bg, syscall.SIGTSTP)
	defer signal.Stop(bg)

	go func() {
		io.Copy(os.Stdout, s.Conn)
		close(done)
	}()
	go io.Copy(s.Conn, os.Stdin)

	select {
	case <-done:
		s.Close()
		output.Warning("Session %d closed", s.ID)
	case <-bg:
		output.Println()
		output.Status("Session %d backgrounded", s.ID)
	}
}
