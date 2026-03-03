// Package tcp provides a raw TCP client for exploit modules.
package tcp

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/sdk"
)

const (
	defaultTimeout = 10 * time.Second
	defaultBufSize = 4096
)

// Session holds a TCP connection to a target.
type Session struct {
	Conn    net.Conn
	Target  string
	Timeout time.Duration
	trace   bool
}

// Dial connects to target (host:port) with context and timeout.
func Dial(ctx context.Context, target string, timeout time.Duration) (*Session, error) {
	if timeout == 0 {
		timeout = defaultTimeout
	}
	output.Debug("TCP connecting to %s (timeout %s)", target, timeout)
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", target)
	if err != nil {
		return nil, fmt.Errorf("tcp connect %s: %w", target, err)
	}
	output.Debug("TCP connected to %s", target)
	return &Session{Conn: conn, Target: target, Timeout: timeout}, nil
}

// FromModule creates a TCP session from module params (mirrors http.FromModule).
func FromModule(params sdk.Params) (*Session, error) {
	target := params.Target()
	timeout := time.Duration(params.IntOr("TCP_TIMEOUT", 10)) * time.Second
	sess, err := Dial(params.Ctx, target, timeout)
	if err != nil {
		return nil, err
	}
	sess.trace = strings.EqualFold(params.Get("TCP_TRACE"), "true")
	return sess, nil
}

// Send writes data to the connection.
func (s *Session) Send(data []byte) error {
	if s.trace {
		debugSend(s.Target, data)
	}
	s.Conn.SetWriteDeadline(time.Now().Add(s.Timeout))
	_, err := s.Conn.Write(data)
	return err
}

// Recv reads up to n bytes from the connection.
func (s *Session) Recv(n int) ([]byte, error) {
	if n <= 0 {
		n = defaultBufSize
	}
	buf := make([]byte, n)
	s.Conn.SetReadDeadline(time.Now().Add(s.Timeout))
	read, err := s.Conn.Read(buf)
	data := buf[:read]
	if s.trace && len(data) > 0 {
		debugRecv(s.Target, data)
	}
	if err != nil {
		return data, err
	}
	return data, nil
}

// SendRecv sends data and reads the response.
func (s *Session) SendRecv(data []byte, recvSize int) ([]byte, error) {
	if err := s.Send(data); err != nil {
		return nil, err
	}
	return s.Recv(recvSize)
}

// Close closes the connection.
func (s *Session) Close() error {
	return s.Conn.Close()
}
