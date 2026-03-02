package session

import (
	"fmt"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/Chocapikk/pik/pkg/output"
)

// Manager handles multiple sessions from a single listener.
type Manager struct {
	listener net.Listener
	sessions map[int]*Session
	nextID   int
	mu       sync.Mutex
	incoming chan *Session
	closed   chan struct{}
}

// NewManager creates a session manager for the given listener.
func NewManager(listener net.Listener) *Manager {
	return &Manager{
		listener: listener,
		sessions: make(map[int]*Session),
		nextID:   1,
		incoming: make(chan *Session, 16),
		closed:   make(chan struct{}),
	}
}

// Start spawns the background accept loop.
func (m *Manager) Start() {
	go m.acceptLoop()
}

func (m *Manager) acceptLoop() {
	for {
		conn, err := m.listener.Accept()
		if err != nil {
			select {
			case <-m.closed:
			default:
			}
			return
		}

		m.mu.Lock()
		sess := newSession(m.nextID, conn)
		m.sessions[m.nextID] = sess
		m.nextID++
		m.mu.Unlock()

		output.Success("Session %d opened (%s)", sess.ID, sess.RemoteAddr)

		select {
		case m.incoming <- sess:
		case <-m.closed:
			return
		}
	}
}

// Accept blocks until a new session arrives or the timeout expires.
// A zero timeout blocks indefinitely.
func (m *Manager) Accept(timeout time.Duration) (*Session, error) {
	if timeout <= 0 {
		select {
		case sess := <-m.incoming:
			return sess, nil
		case <-m.closed:
			return nil, fmt.Errorf("manager closed")
		}
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case sess := <-m.incoming:
		return sess, nil
	case <-timer.C:
		return nil, fmt.Errorf("no session received within %s", timeout)
	case <-m.closed:
		return nil, fmt.Errorf("manager closed")
	}
}

// List returns all alive sessions sorted by ID.
func (m *Manager) List() []*Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []*Session
	for _, sess := range m.sessions {
		if sess.Alive() {
			result = append(result, sess)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result
}

// Get returns a session by ID.
func (m *Manager) Get(id int) (*Session, error) {
	return m.get(id)
}

// Kill terminates a session by ID.
func (m *Manager) Kill(id int) error {
	sess, err := m.get(id)
	if err != nil {
		return err
	}
	sess.Close()
	output.Status("Session %d killed", id)
	return nil
}

// Interact enters interactive mode for a session.
func (m *Manager) Interact(id int) error {
	sess, err := m.get(id)
	if err != nil {
		return err
	}
	if !sess.Alive() {
		return fmt.Errorf("session %d is dead", id)
	}
	sess.Interact()
	return nil
}

// Close shuts down the manager and all sessions.
func (m *Manager) Close() {
	select {
	case <-m.closed:
		return
	default:
	}
	close(m.closed)
	m.listener.Close()

	m.mu.Lock()
	defer m.mu.Unlock()
	for _, sess := range m.sessions {
		sess.Close()
	}
}

func (m *Manager) get(id int) (*Session, error) {
	m.mu.Lock()
	sess, ok := m.sessions[id]
	m.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("session %d not found", id)
	}
	return sess, nil
}
