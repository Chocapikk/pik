package tcp

import (
	"bytes"
	"context"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/Chocapikk/pik/sdk"
)

// ---------------------------------------------------------------------------
// Dial
// ---------------------------------------------------------------------------

func TestDialSuccess(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	sess, err := Dial(context.Background(), ln.Addr().String(), 5*time.Second)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer sess.Close()

	if sess.Conn == nil {
		t.Error("Conn should not be nil")
	}
	if sess.Target != ln.Addr().String() {
		t.Errorf("Target = %q, want %q", sess.Target, ln.Addr().String())
	}
	if sess.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want 5s", sess.Timeout)
	}
}

func TestDialDefaultTimeout(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	sess, err := Dial(context.Background(), ln.Addr().String(), 0)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer sess.Close()

	if sess.Timeout != defaultTimeout {
		t.Errorf("Timeout = %v, want %v (defaultTimeout)", sess.Timeout, defaultTimeout)
	}
}

func TestDialClosedPort(t *testing.T) {
	// Listen and immediately close to get a known-closed port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()

	_, err = Dial(context.Background(), addr, 1*time.Second)
	if err == nil {
		t.Fatal("expected error dialing closed port")
	}
	if !strings.Contains(err.Error(), "tcp connect") {
		t.Errorf("error = %q, want wrapped with 'tcp connect'", err.Error())
	}
}

func TestDialCancelledContext(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err = Dial(ctx, ln.Addr().String(), 5*time.Second)
	if err == nil {
		t.Fatal("expected error with cancelled context")
	}
}

// ---------------------------------------------------------------------------
// Send / Recv / SendRecv / Close (via net.Pipe)
// ---------------------------------------------------------------------------

func newPipeSession(timeout time.Duration, trace bool) (*Session, net.Conn) {
	client, server := net.Pipe()
	sess := &Session{
		Conn:    client,
		Target:  "pipe://test",
		Timeout: timeout,
		trace:   trace,
	}
	return sess, server
}

func TestSend(t *testing.T) {
	sess, server := newPipeSession(5*time.Second, false)
	defer sess.Close()
	defer server.Close()

	payload := []byte("n0litetebastardescarb0rund0rum")

	done := make(chan []byte, 1)
	go func() {
		buf := make([]byte, 256)
		n, _ := server.Read(buf)
		done <- buf[:n]
	}()

	if err := sess.Send(payload); err != nil {
		t.Fatalf("Send: %v", err)
	}

	got := <-done
	if !bytes.Equal(got, payload) {
		t.Errorf("server received %q, want %q", got, payload)
	}
}

func TestRecv(t *testing.T) {
	sess, server := newPipeSession(5*time.Second, false)
	defer sess.Close()
	defer server.Close()

	payload := []byte("n0litetebastardescarb0rund0rum")

	go func() {
		server.Write(payload)
	}()

	got, err := sess.Recv(256)
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("Recv = %q, want %q", got, payload)
	}
}

func TestRecvDefaultBufferSize(t *testing.T) {
	sess, server := newPipeSession(5*time.Second, false)
	defer sess.Close()
	defer server.Close()

	payload := []byte("n0litetebastardescarb0rund0rum")

	go func() {
		server.Write(payload)
	}()

	// n=0 should use defaultBufSize
	got, err := sess.Recv(0)
	if err != nil {
		t.Fatalf("Recv(0): %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("Recv(0) = %q, want %q", got, payload)
	}
}

func TestRecvNegativeN(t *testing.T) {
	sess, server := newPipeSession(5*time.Second, false)
	defer sess.Close()
	defer server.Close()

	payload := []byte("n0litetebastardescarb0rund0rum")

	go func() {
		server.Write(payload)
	}()

	// n<0 should also use defaultBufSize
	got, err := sess.Recv(-1)
	if err != nil {
		t.Fatalf("Recv(-1): %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("Recv(-1) = %q, want %q", got, payload)
	}
}

func TestSendRecv(t *testing.T) {
	sess, server := newPipeSession(5*time.Second, false)
	defer sess.Close()
	defer server.Close()

	request := []byte("n0litetebastardescarb0rund0rum:request")
	response := []byte("n0litetebastardescarb0rund0rum:response")

	go func() {
		buf := make([]byte, 256)
		n, _ := server.Read(buf)
		if !bytes.Equal(buf[:n], request) {
			return
		}
		server.Write(response)
	}()

	got, err := sess.SendRecv(request, 256)
	if err != nil {
		t.Fatalf("SendRecv: %v", err)
	}
	if !bytes.Equal(got, response) {
		t.Errorf("SendRecv = %q, want %q", got, response)
	}
}

func TestSendRecvSendFails(t *testing.T) {
	sess, server := newPipeSession(5*time.Second, false)
	server.Close() // close server side first
	defer sess.Close()

	_, err := sess.SendRecv([]byte("n0litetebastardescarb0rund0rum"), 256)
	if err == nil {
		t.Fatal("expected error when sending to closed connection")
	}
}

func TestClose(t *testing.T) {
	sess, server := newPipeSession(5*time.Second, false)
	defer server.Close()

	if err := sess.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Writing to closed session should fail
	if err := sess.Send([]byte("n0litetebastardescarb0rund0rum")); err == nil {
		t.Error("expected error sending to closed session")
	}
}

func TestRecvAfterRemoteClose(t *testing.T) {
	sess, server := newPipeSession(5*time.Second, false)
	defer sess.Close()

	server.Close()

	data, err := sess.Recv(256)
	// Should get io.EOF or similar error
	if err == nil {
		t.Fatal("expected error reading from closed connection")
	}
	// Data should be empty
	if len(data) != 0 {
		t.Errorf("data = %q, want empty", data)
	}
}

// ---------------------------------------------------------------------------
// FromModule
// ---------------------------------------------------------------------------

func TestFromModule(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	params := sdk.NewParams(context.Background(), map[string]string{
		"TARGET":      ln.Addr().String(),
		"TCP_TIMEOUT": "3",
		"TCP_TRACE":   "true",
	})

	sess, err := FromModule(params)
	if err != nil {
		t.Fatalf("FromModule: %v", err)
	}
	defer sess.Close()

	if sess.Timeout != 3*time.Second {
		t.Errorf("Timeout = %v, want 3s", sess.Timeout)
	}
	if !sess.trace {
		t.Error("trace should be true")
	}
}

func TestFromModuleDefaultTimeout(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	params := sdk.NewParams(context.Background(), map[string]string{
		"TARGET": ln.Addr().String(),
	})

	sess, err := FromModule(params)
	if err != nil {
		t.Fatalf("FromModule: %v", err)
	}
	defer sess.Close()

	// Default TCP_TIMEOUT is 10, converted to 10*time.Second, then Dial
	// sees it as non-zero so uses it as-is.
	if sess.Timeout != 10*time.Second {
		t.Errorf("Timeout = %v, want 10s", sess.Timeout)
	}
	if sess.trace {
		t.Error("trace should be false by default")
	}
}

func TestFromModuleTraceCaseInsensitive(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	params := sdk.NewParams(context.Background(), map[string]string{
		"TARGET":    ln.Addr().String(),
		"TCP_TRACE": "True",
	})

	sess, err := FromModule(params)
	if err != nil {
		t.Fatalf("FromModule: %v", err)
	}
	defer sess.Close()

	if !sess.trace {
		t.Error("trace should be true for 'True' (case-insensitive)")
	}
}

func TestFromModuleError(t *testing.T) {
	// Listen and close to get a dead port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()

	params := sdk.NewParams(context.Background(), map[string]string{
		"TARGET":      addr,
		"TCP_TIMEOUT": "1",
	})

	_, err = FromModule(params)
	if err == nil {
		t.Fatal("expected error from FromModule with closed port")
	}
}

// ---------------------------------------------------------------------------
// Trace mode (debug functions should not panic)
// ---------------------------------------------------------------------------

func TestSendWithTrace(t *testing.T) {
	sess, server := newPipeSession(5*time.Second, true)
	defer sess.Close()
	defer server.Close()

	payload := []byte("n0litetebastardescarb0rund0rum")

	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 256)
		server.Read(buf)
	}()

	// Should not panic even though debugSend writes to stderr
	if err := sess.Send(payload); err != nil {
		t.Fatalf("Send with trace: %v", err)
	}
	<-done
}

func TestRecvWithTrace(t *testing.T) {
	sess, server := newPipeSession(5*time.Second, true)
	defer sess.Close()
	defer server.Close()

	payload := []byte("n0litetebastardescarb0rund0rum")

	go func() {
		server.Write(payload)
	}()

	// Should not panic even though debugRecv writes to stderr
	got, err := sess.Recv(256)
	if err != nil {
		t.Fatalf("Recv with trace: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("Recv with trace = %q, want %q", got, payload)
	}
}

func TestRecvWithTraceEmptyData(t *testing.T) {
	sess, server := newPipeSession(5*time.Second, true)
	defer sess.Close()

	// Close server so Recv returns io.EOF with no data - debugRecv should NOT be called
	server.Close()

	data, err := sess.Recv(256)
	if err == nil {
		t.Fatal("expected error")
	}
	// Empty data with trace should not trigger debugRecv, so no panic
	if len(data) != 0 {
		t.Errorf("data = %q, want empty", data)
	}
}

func TestSendRecvWithTrace(t *testing.T) {
	sess, server := newPipeSession(5*time.Second, true)
	defer sess.Close()
	defer server.Close()

	request := []byte("n0litetebastardescarb0rund0rum:trace-request")
	response := []byte("n0litetebastardescarb0rund0rum:trace-response")

	go func() {
		buf := make([]byte, 256)
		server.Read(buf)
		server.Write(response)
	}()

	got, err := sess.SendRecv(request, 256)
	if err != nil {
		t.Fatalf("SendRecv with trace: %v", err)
	}
	if !bytes.Equal(got, response) {
		t.Errorf("SendRecv with trace = %q, want %q", got, response)
	}
}

// ---------------------------------------------------------------------------
// printHexDump
// ---------------------------------------------------------------------------

func TestPrintHexDumpSmallData(t *testing.T) {
	// Should not panic on small data
	printHexDump([]byte("n0litetebastardescarb0rund0rum"))
}

func TestPrintHexDumpEmptyData(t *testing.T) {
	// Should not panic on empty data
	printHexDump([]byte{})
}

func TestPrintHexDumpTruncation(t *testing.T) {
	// Create data larger than maxDumpBytes (512) to cover truncation branch
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	// Should not panic and should print truncation message
	printHexDump(data)
}

func TestPrintHexDumpExactMaxBytes(t *testing.T) {
	// Exactly maxDumpBytes - should NOT trigger truncation
	data := make([]byte, maxDumpBytes)
	for i := range data {
		data[i] = byte(i % 256)
	}
	printHexDump(data)
}

func TestPrintHexDumpOneBeyondMax(t *testing.T) {
	// One byte over maxDumpBytes - should trigger truncation
	data := make([]byte, maxDumpBytes+1)
	for i := range data {
		data[i] = byte(i % 256)
	}
	printHexDump(data)
}

// ---------------------------------------------------------------------------
// debugSend / debugRecv (should not panic)
// ---------------------------------------------------------------------------

func TestDebugSend(t *testing.T) {
	debugSend("127.0.0.1:2222", []byte("n0litetebastardescarb0rund0rum"))
}

func TestDebugRecv(t *testing.T) {
	debugRecv("127.0.0.1:2222", []byte("n0litetebastardescarb0rund0rum"))
}

func TestDebugSendLargeData(t *testing.T) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	debugSend("127.0.0.1:2222", data)
}

func TestDebugRecvLargeData(t *testing.T) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	debugRecv("127.0.0.1:2222", data)
}

// ---------------------------------------------------------------------------
// DialFactory (init registration)
// ---------------------------------------------------------------------------

func TestDialFactoryRegistered(t *testing.T) {
	// The init() in option.go calls sdk.SetDialFactory.
	// Verify we can call sdk.DialWith with a real listener.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	params := sdk.NewParams(context.Background(), map[string]string{
		"TARGET":      ln.Addr().String(),
		"TCP_TIMEOUT": "2",
	})

	conn, err := sdk.DialWith(params)
	if err != nil {
		t.Fatalf("DialWith: %v", err)
	}
	defer conn.Close()
}

func TestDialFactoryReturnsSdkConn(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			defer conn.Close()
			buf := make([]byte, 256)
			n, _ := conn.Read(buf)
			conn.Write(buf[:n])
		}
	}()

	params := sdk.NewParams(context.Background(), map[string]string{
		"TARGET":      ln.Addr().String(),
		"TCP_TIMEOUT": "2",
	})

	conn, err := sdk.DialWith(params)
	if err != nil {
		t.Fatalf("DialWith: %v", err)
	}
	defer conn.Close()

	// Verify the returned conn implements sdk.Conn interface properly
	payload := []byte("n0litetebastardescarb0rund0rum")
	resp, err := conn.SendRecv(payload, 256)
	if err != nil {
		t.Fatalf("SendRecv via sdk.Conn: %v", err)
	}
	if !bytes.Equal(resp, payload) {
		t.Errorf("response = %q, want %q", resp, payload)
	}
}

// ---------------------------------------------------------------------------
// Session implements sdk.Conn
// ---------------------------------------------------------------------------

func TestSessionImplementsConn(t *testing.T) {
	sess, server := newPipeSession(5*time.Second, false)
	defer sess.Close()
	defer server.Close()

	// Compile-time check: *Session satisfies sdk.Conn
	var _ sdk.Conn = sess
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestSendToClosedConn(t *testing.T) {
	sess, server := newPipeSession(5*time.Second, false)
	server.Close()
	defer sess.Close()

	err := sess.Send([]byte("n0litetebastardescarb0rund0rum"))
	if err == nil {
		t.Fatal("expected error sending to closed pipe")
	}
}

func TestRecvTimeout(t *testing.T) {
	sess, server := newPipeSession(100*time.Millisecond, false)
	defer sess.Close()
	defer server.Close()

	// Don't write anything to server - should timeout
	_, err := sess.Recv(256)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestSendEmptyData(t *testing.T) {
	sess, server := newPipeSession(5*time.Second, false)
	defer sess.Close()
	defer server.Close()

	done := make(chan int, 1)
	go func() {
		buf := make([]byte, 256)
		n, _ := server.Read(buf)
		done <- n
	}()

	if err := sess.Send([]byte{}); err != nil {
		t.Fatalf("Send empty: %v", err)
	}

	// net.Pipe may deliver empty writes differently, but Send should not error
}

func TestRecvExactSize(t *testing.T) {
	sess, server := newPipeSession(5*time.Second, false)
	defer sess.Close()
	defer server.Close()

	payload := []byte("n0litetebastardescarb0rund0rum")

	go func() {
		server.Write(payload)
	}()

	// Request exact size of payload
	got, err := sess.Recv(len(payload))
	if err != nil {
		t.Fatalf("Recv exact: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("Recv exact = %q, want %q", got, payload)
	}
}

func TestRecvSmallBuffer(t *testing.T) {
	sess, server := newPipeSession(5*time.Second, false)
	defer sess.Close()
	defer server.Close()

	payload := []byte("n0litetebastardescarb0rund0rum")

	go func() {
		server.Write(payload)
	}()

	// Request smaller buffer than payload - should get partial data
	got, err := sess.Recv(5)
	if err != nil {
		t.Fatalf("Recv small: %v", err)
	}
	if len(got) > 5 {
		t.Errorf("Recv(5) returned %d bytes, want <= 5", len(got))
	}
	if !bytes.Equal(got, payload[:len(got)]) {
		t.Errorf("Recv small = %q, want prefix of %q", got, payload)
	}
}

func TestMultipleSendRecv(t *testing.T) {
	sess, server := newPipeSession(5*time.Second, false)
	defer sess.Close()
	defer server.Close()

	go func() {
		buf := make([]byte, 256)
		for {
			n, err := server.Read(buf)
			if err != nil {
				return
			}
			// Echo back uppercase-ish prefix
			server.Write(append([]byte("RE:"), buf[:n]...))
		}
	}()

	for i := 0; i < 3; i++ {
		msg := []byte("n0litetebastardescarb0rund0rum")
		resp, err := sess.SendRecv(msg, 256)
		if err != nil {
			t.Fatalf("round %d: SendRecv: %v", i, err)
		}
		expected := append([]byte("RE:"), msg...)
		if !bytes.Equal(resp, expected) {
			t.Errorf("round %d: got %q, want %q", i, resp, expected)
		}
	}
}

func TestCloseIdempotent(t *testing.T) {
	sess, server := newPipeSession(5*time.Second, false)
	defer server.Close()

	if err := sess.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	// Second close should not panic (error behavior depends on net.Conn impl)
	sess.Close()
}

func TestCloseRealConnIdempotent(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	sess, err := Dial(context.Background(), ln.Addr().String(), 5*time.Second)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}

	if err := sess.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	// Real TCP conn returns error on double close
	if err := sess.Close(); err == nil {
		t.Error("expected error on second Close of real TCP conn")
	}
}

// ---------------------------------------------------------------------------
// Recv returns partial data on read error
// ---------------------------------------------------------------------------

func TestRecvPartialDataOnError(t *testing.T) {
	// Use a real TCP connection so we can test partial read + EOF
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		conn.Write([]byte("n0litetebastardescarb0rund0rum"))
		conn.Close() // close after writing -> client gets data + EOF
	}()

	sess, err := Dial(context.Background(), ln.Addr().String(), 5*time.Second)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer sess.Close()

	// First read gets the data
	data, err := sess.Recv(256)
	if len(data) == 0 {
		t.Error("expected some data before EOF")
	}
	if !bytes.Equal(data, []byte("n0litetebastardescarb0rund0rum")) {
		t.Errorf("data = %q", data)
	}

	// Second read should get EOF
	data2, err2 := sess.Recv(256)
	if err2 != io.EOF {
		t.Errorf("expected io.EOF, got %v", err2)
	}
	if len(data2) != 0 {
		t.Errorf("expected empty data on EOF, got %q", data2)
	}
}

// ---------------------------------------------------------------------------
// Trace on Send to closed conn should not panic
// ---------------------------------------------------------------------------

func TestSendTraceClosedConn(t *testing.T) {
	sess, server := newPipeSession(5*time.Second, true)
	server.Close()
	defer sess.Close()

	// debugSend runs, then Write fails - should not panic
	err := sess.Send([]byte("n0litetebastardescarb0rund0rum"))
	if err == nil {
		t.Fatal("expected error")
	}
}
