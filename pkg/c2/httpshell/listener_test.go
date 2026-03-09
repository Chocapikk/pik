package httpshell

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

const testPhrase = "n0litetebastardescarb0rund0rum"

func TestNew(t *testing.T) {
	l := New()
	if l == nil {
		t.Fatal("New() returned nil")
	}
	if l.bridges == nil {
		t.Fatal("bridges map should be initialized")
	}
	if len(l.bridges) != 0 {
		t.Errorf("bridges map should be empty, got %d entries", len(l.bridges))
	}
}

func TestName(t *testing.T) {
	l := New()
	if got := l.Name(); got != "httpshell" {
		t.Errorf("Name() = %q, want %q", got, "httpshell")
	}
}

func TestSetupAndShutdown(t *testing.T) {
	l := New()
	if err := l.Setup("127.0.0.1", 0); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if err := l.Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}

func TestSetupInvalidAddress(t *testing.T) {
	l := New()
	err := l.Setup("999.999.999.999", 0)
	if err == nil {
		t.Fatal("Setup with invalid address should fail")
	}
}

func TestGeneratePayloadKnownTypes(t *testing.T) {
	l := New()
	l.lhost = "10.0.0.1"
	l.lport = 4444

	tests := []struct {
		payloadType string
		mustContain string
	}{
		{"cmd/curl/reverse_http", "curl"},
		{"cmd/wget/reverse_http", "wget"},
		{"cmd/php/reverse_http", "php"},
		{"cmd/python/reverse_http", "python3"},
	}

	for _, tt := range tests {
		t.Run(tt.payloadType, func(t *testing.T) {
			out, err := l.GeneratePayload("linux", tt.payloadType)
			if err != nil {
				t.Fatalf("GeneratePayload(%q): %v", tt.payloadType, err)
			}
			if out == "" {
				t.Fatalf("GeneratePayload(%q) returned empty string", tt.payloadType)
			}
			if !strings.Contains(out, tt.mustContain) {
				t.Errorf("payload %q should contain %q, got: %s", tt.payloadType, tt.mustContain, out)
			}
			if !strings.Contains(out, "10.0.0.1") {
				t.Errorf("payload should contain lhost 10.0.0.1, got: %s", out)
			}
			if !strings.Contains(out, "4444") {
				t.Errorf("payload should contain lport 4444, got: %s", out)
			}
		})
	}
}

func TestGeneratePayloadFallback(t *testing.T) {
	l := New()
	l.lhost = "10.0.0.1"
	l.lport = 4444

	out, err := l.GeneratePayload("linux", "cmd/nonexistent/n0litetebastardescarb0rund0rum")
	if err != nil {
		t.Fatalf("GeneratePayload fallback: %v", err)
	}
	if out == "" {
		t.Fatal("fallback payload should not be empty")
	}
	// Fallback is CurlHTTP
	if !strings.Contains(out, "curl") {
		t.Errorf("fallback should use curl, got: %s", out)
	}
}

// --- bridgeAddr tests ---

func TestBridgeAddr(t *testing.T) {
	addr := bridgeAddr(testPhrase)
	if addr.Network() != "http" {
		t.Errorf("Network() = %q, want %q", addr.Network(), "http")
	}
	if addr.String() != testPhrase {
		t.Errorf("String() = %q, want %q", addr.String(), testPhrase)
	}
}

// --- httpBridge tests ---

func TestHTTPBridgeIsClosed(t *testing.T) {
	b := newHTTPBridge("127.0.0.1:9999")

	if b.isClosed() {
		t.Error("new bridge should not be closed")
	}

	close(b.closed)

	if !b.isClosed() {
		t.Error("bridge should be closed after closing the channel")
	}
}

func TestHTTPBridgeChannels(t *testing.T) {
	b := newHTTPBridge("127.0.0.1:9999")

	// Test cmd channel capacity
	msg := []byte(testPhrase)
	select {
	case b.cmd <- msg:
	default:
		t.Fatal("should be able to send to cmd channel")
	}

	select {
	case got := <-b.cmd:
		if !bytes.Equal(got, msg) {
			t.Errorf("cmd channel: got %q, want %q", got, msg)
		}
	default:
		t.Fatal("should be able to receive from cmd channel")
	}

	// Test out channel capacity
	select {
	case b.out <- msg:
	default:
		t.Fatal("should be able to send to out channel")
	}

	select {
	case got := <-b.out:
		if !bytes.Equal(got, msg) {
			t.Errorf("out channel: got %q, want %q", got, msg)
		}
	default:
		t.Fatal("should be able to receive from out channel")
	}
}

// --- bridgeConn tests ---

func TestBridgeConnWriteAndRead(t *testing.T) {
	b := newHTTPBridge("127.0.0.1:9999")
	c := &bridgeConn{bridge: b}

	// Write sends to cmd channel
	msg := []byte(testPhrase)
	n, err := c.Write(msg)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != len(msg) {
		t.Errorf("Write returned %d, want %d", n, len(msg))
	}

	got := <-b.cmd
	if !bytes.Equal(got, msg) {
		t.Errorf("cmd channel: got %q, want %q", got, msg)
	}

	// Read receives from out channel
	b.out <- []byte(testPhrase)

	buf := make([]byte, 128)
	n, err = c.Read(buf)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(buf[:n]) != testPhrase {
		t.Errorf("Read: got %q, want %q", string(buf[:n]), testPhrase)
	}
}

func TestBridgeConnReadBuffering(t *testing.T) {
	b := newHTTPBridge("127.0.0.1:9999")
	c := &bridgeConn{bridge: b}

	// Send a message larger than the read buffer
	data := []byte(testPhrase + "_extra_data_for_buffering")
	b.out <- data

	// Read with a small buffer to trigger buffering
	small := make([]byte, 5)
	n, err := c.Read(small)
	if err != nil {
		t.Fatalf("first Read: %v", err)
	}
	if n != 5 {
		t.Errorf("first Read: got %d bytes, want 5", n)
	}

	// Remaining data should come from the internal buffer
	rest := make([]byte, 256)
	n, err = c.Read(rest)
	if err != nil {
		t.Fatalf("second Read: %v", err)
	}
	combined := string(small[:5]) + string(rest[:n])
	if combined != string(data) {
		t.Errorf("combined reads: got %q, want %q", combined, string(data))
	}
}

func TestBridgeConnReadAfterClose(t *testing.T) {
	b := newHTTPBridge("127.0.0.1:9999")
	c := &bridgeConn{bridge: b}

	close(b.closed)

	buf := make([]byte, 128)
	_, err := c.Read(buf)
	if err != io.EOF {
		t.Errorf("Read after close: got %v, want io.EOF", err)
	}
}

func TestBridgeConnWriteAfterClose(t *testing.T) {
	b := newHTTPBridge("127.0.0.1:9999")
	c := &bridgeConn{bridge: b}

	// Fill the cmd channel to capacity so the send case cannot proceed
	for range cap(b.cmd) {
		b.cmd <- []byte("fill")
	}
	close(b.closed)

	_, err := c.Write([]byte(testPhrase))
	if err != io.ErrClosedPipe {
		t.Errorf("Write after close with full channel: got %v, want io.ErrClosedPipe", err)
	}
}

func TestBridgeConnClose(t *testing.T) {
	b := newHTTPBridge("127.0.0.1:9999")
	c := &bridgeConn{bridge: b}

	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !b.isClosed() {
		t.Error("bridge should be closed after Close()")
	}

	// Double close should not panic
	if err := c.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestBridgeConnAddresses(t *testing.T) {
	addr := testPhrase + ":1337"
	b := newHTTPBridge(addr)
	c := &bridgeConn{bridge: b}

	local := c.LocalAddr()
	remote := c.RemoteAddr()

	if local.Network() != "http" {
		t.Errorf("LocalAddr().Network() = %q, want %q", local.Network(), "http")
	}
	if local.String() != addr {
		t.Errorf("LocalAddr().String() = %q, want %q", local.String(), addr)
	}
	if remote.Network() != "http" {
		t.Errorf("RemoteAddr().Network() = %q, want %q", remote.Network(), "http")
	}
	if remote.String() != addr {
		t.Errorf("RemoteAddr().String() = %q, want %q", remote.String(), addr)
	}
}

func TestBridgeConnDeadlines(t *testing.T) {
	b := newHTTPBridge("127.0.0.1:9999")
	c := &bridgeConn{bridge: b}

	deadline := time.Now().Add(time.Hour)
	if err := c.SetDeadline(deadline); err != nil {
		t.Errorf("SetDeadline: %v", err)
	}
	if err := c.SetReadDeadline(deadline); err != nil {
		t.Errorf("SetReadDeadline: %v", err)
	}
	if err := c.SetWriteDeadline(deadline); err != nil {
		t.Errorf("SetWriteDeadline: %v", err)
	}
}

// --- chanListener tests ---

func TestChanListenerPushAndAccept(t *testing.T) {
	tcpAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	cl := newChanListener(tcpAddr)

	b := newHTTPBridge("127.0.0.1:5555")
	conn := b.conn()

	cl.push(conn)

	accepted, err := cl.Accept()
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}
	if accepted == nil {
		t.Fatal("Accept returned nil")
	}
}

func TestChanListenerAcceptAfterClose(t *testing.T) {
	tcpAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	cl := newChanListener(tcpAddr)
	cl.Close()

	_, err := cl.Accept()
	if err != net.ErrClosed {
		t.Errorf("Accept after Close: got %v, want net.ErrClosed", err)
	}
}

func TestChanListenerPushAfterClose(t *testing.T) {
	tcpAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	cl := newChanListener(tcpAddr)
	cl.Close()

	b := newHTTPBridge("127.0.0.1:5555")
	// Should not block or panic
	cl.push(b.conn())
}

func TestChanListenerCloseIdempotent(t *testing.T) {
	tcpAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	cl := newChanListener(tcpAddr)

	if err := cl.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := cl.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestChanListenerAddr(t *testing.T) {
	tcpAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:7777")
	cl := newChanListener(tcpAddr)

	addr := cl.Addr()
	if addr == nil {
		t.Fatal("Addr() returned nil")
	}
	if !strings.Contains(addr.String(), "7777") {
		t.Errorf("Addr() = %q, want address containing 7777", addr.String())
	}
}

// --- handleCmd / handleOut via HTTP ---

func setupListener(t *testing.T) (*Listener, string) {
	t.Helper()
	l := New()
	if err := l.Setup("127.0.0.1", 0); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	// Extract the actual port from the chanListener address
	addr := l.vln.Addr().String()
	return l, addr
}

func TestHandleCmdNoCommand(t *testing.T) {
	l, addr := setupListener(t)
	defer l.Shutdown()

	// GET /cmd with no command queued should return 200 with empty body
	resp, err := http.Get(fmt.Sprintf("http://%s/cmd", addr))
	if err != nil {
		t.Fatalf("GET /cmd: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /cmd status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHandleCmdWithCommand(t *testing.T) {
	l, addr := setupListener(t)
	defer l.Shutdown()

	// First request creates a bridge
	resp, err := http.Get(fmt.Sprintf("http://%s/cmd", addr))
	if err != nil {
		t.Fatalf("initial GET /cmd: %v", err)
	}
	resp.Body.Close()

	// Find the bridge and push a command
	l.mu.Lock()
	var bridge *httpBridge
	for _, b := range l.bridges {
		bridge = b
		break
	}
	l.mu.Unlock()

	if bridge == nil {
		t.Fatal("no bridge created after GET /cmd")
	}

	cmd := testPhrase
	bridge.cmd <- []byte(cmd)

	// Next GET /cmd should return the command
	resp, err = http.Get(fmt.Sprintf("http://%s/cmd", addr))
	if err != nil {
		t.Fatalf("GET /cmd: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != cmd {
		t.Errorf("GET /cmd body = %q, want %q", string(body), cmd)
	}
}

func TestHandleOut(t *testing.T) {
	l, addr := setupListener(t)
	defer l.Shutdown()

	// First hit /cmd to create the bridge
	resp, err := http.Get(fmt.Sprintf("http://%s/cmd", addr))
	if err != nil {
		t.Fatalf("GET /cmd: %v", err)
	}
	resp.Body.Close()

	// POST output to /out
	outData := testPhrase + "_output"
	resp, err = http.Post(
		fmt.Sprintf("http://%s/out", addr),
		"text/plain",
		strings.NewReader(outData),
	)
	if err != nil {
		t.Fatalf("POST /out: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("POST /out status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Verify the output was received on the bridge
	l.mu.Lock()
	var bridge *httpBridge
	for _, b := range l.bridges {
		bridge = b
		break
	}
	l.mu.Unlock()

	select {
	case got := <-bridge.out:
		if string(got) != outData {
			t.Errorf("bridge out = %q, want %q", string(got), outData)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for output on bridge")
	}
}

// --- getOrCreateBridge tests ---

func TestGetOrCreateBridgeReuse(t *testing.T) {
	l, addr := setupListener(t)
	defer l.Shutdown()

	// Two requests from same client should reuse the same bridge
	resp1, err := http.Get(fmt.Sprintf("http://%s/cmd", addr))
	if err != nil {
		t.Fatalf("first GET /cmd: %v", err)
	}
	resp1.Body.Close()

	resp2, err := http.Get(fmt.Sprintf("http://%s/cmd", addr))
	if err != nil {
		t.Fatalf("second GET /cmd: %v", err)
	}
	resp2.Body.Close()

	l.mu.Lock()
	count := len(l.bridges)
	l.mu.Unlock()

	if count != 1 {
		t.Errorf("expected 1 bridge (reused), got %d", count)
	}
}

func TestGetOrCreateBridgeAfterClose(t *testing.T) {
	l, addr := setupListener(t)
	defer l.Shutdown()

	// First request creates a bridge
	resp, err := http.Get(fmt.Sprintf("http://%s/cmd", addr))
	if err != nil {
		t.Fatalf("GET /cmd: %v", err)
	}
	resp.Body.Close()

	// Close the existing bridge
	l.mu.Lock()
	for _, b := range l.bridges {
		close(b.closed)
	}
	l.mu.Unlock()

	// Next request should create a new bridge
	resp, err = http.Get(fmt.Sprintf("http://%s/cmd", addr))
	if err != nil {
		t.Fatalf("GET /cmd after close: %v", err)
	}
	resp.Body.Close()

	l.mu.Lock()
	var openCount int
	for _, b := range l.bridges {
		if !b.isClosed() {
			openCount++
		}
	}
	l.mu.Unlock()

	if openCount != 1 {
		t.Errorf("expected 1 open bridge after reconnect, got %d", openCount)
	}
}

// --- Integration test ---

func TestIntegrationCmdOutLoop(t *testing.T) {
	l, addr := setupListener(t)
	defer l.Shutdown()

	client := &http.Client{Timeout: 5 * time.Second}

	// 1. Poll /cmd to establish the bridge (no command yet)
	resp, err := client.Get(fmt.Sprintf("http://%s/cmd", addr))
	if err != nil {
		t.Fatalf("poll /cmd: %v", err)
	}
	resp.Body.Close()

	// 2. Wait for the session manager to pick up the bridge conn
	sess, err := l.Manager.Accept(3 * time.Second)
	if err != nil {
		t.Fatalf("Accept session: %v", err)
	}
	if sess == nil {
		t.Fatal("no session created")
	}

	// 3. Write a command via the session's Conn (simulates user typing)
	cmd := "echo " + testPhrase
	_, err = sess.Conn.Write([]byte(cmd))
	if err != nil {
		t.Fatalf("write command to session: %v", err)
	}

	// 4. Client polls /cmd and should receive the command
	resp, err = client.Get(fmt.Sprintf("http://%s/cmd", addr))
	if err != nil {
		t.Fatalf("GET /cmd: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if string(body) != cmd {
		t.Errorf("polled command = %q, want %q", string(body), cmd)
	}

	// 5. Client sends output to /out
	output := testPhrase + "_result"
	resp, err = client.Post(
		fmt.Sprintf("http://%s/out", addr),
		"text/plain",
		strings.NewReader(output),
	)
	if err != nil {
		t.Fatalf("POST /out: %v", err)
	}
	resp.Body.Close()

	// 6. Read output from session's Conn
	buf := make([]byte, 256)
	n, err := sess.Conn.Read(buf)
	if err != nil {
		t.Fatalf("read from session: %v", err)
	}
	if string(buf[:n]) != output {
		t.Errorf("session output = %q, want %q", string(buf[:n]), output)
	}
}

func TestIntegrationSessionList(t *testing.T) {
	l, addr := setupListener(t)
	defer l.Shutdown()

	client := &http.Client{Timeout: 5 * time.Second}

	// Hit /cmd to create a session
	resp, err := client.Get(fmt.Sprintf("http://%s/cmd", addr))
	if err != nil {
		t.Fatalf("GET /cmd: %v", err)
	}
	resp.Body.Close()

	// Wait for session
	_, err = l.Manager.Accept(3 * time.Second)
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}

	sessions := l.Sessions()
	if len(sessions) != 1 {
		t.Fatalf("Sessions() = %d, want 1", len(sessions))
	}
	if !sessions[0].Alive() {
		t.Error("session should be alive")
	}
}

func TestShutdownIdempotent(t *testing.T) {
	l := New()
	if err := l.Setup("127.0.0.1", 0); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if err := l.Shutdown(); err != nil {
		t.Fatalf("first Shutdown: %v", err)
	}
	// Calling Shutdown again on a nil manager should not panic
	if err := l.Shutdown(); err != nil {
		t.Fatalf("second Shutdown: %v", err)
	}
}

func TestShutdownWithoutSetup(t *testing.T) {
	l := New()
	// Shutdown without Setup should not panic
	if err := l.Shutdown(); err != nil {
		t.Fatalf("Shutdown without Setup: %v", err)
	}
}

func TestWriteIsolatesInput(t *testing.T) {
	// Verify that Write copies the input slice to avoid aliasing
	b := newHTTPBridge("127.0.0.1:9999")
	c := &bridgeConn{bridge: b}

	original := []byte(testPhrase)
	backup := make([]byte, len(original))
	copy(backup, original)

	_, err := c.Write(original)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Mutate the original slice
	original[0] = 'X'

	got := <-b.cmd
	if !bytes.Equal(got, backup) {
		t.Errorf("Write did not copy: got %q, want %q", got, backup)
	}
}

func TestBridgeConnImplementsNetConn(t *testing.T) {
	b := newHTTPBridge("127.0.0.1:9999")
	var conn net.Conn = b.conn()
	if conn == nil {
		t.Fatal("conn() should return a valid net.Conn")
	}
}

func TestChanListenerImplementsNetListener(t *testing.T) {
	tcpAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	var ln net.Listener = newChanListener(tcpAddr)
	if ln == nil {
		t.Fatal("newChanListener should return a valid net.Listener")
	}
}
