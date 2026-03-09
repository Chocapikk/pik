package session

import (
	"net"
	"testing"
)

func TestNewSession(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	sess := newSession(42, client)

	if sess.ID != 42 {
		t.Errorf("ID = %d, want 42", sess.ID)
	}
	if sess.RemoteAddr != client.RemoteAddr().String() {
		t.Errorf("RemoteAddr = %q, want %q", sess.RemoteAddr, client.RemoteAddr().String())
	}
	if !sess.Alive() {
		t.Error("new session should be alive")
	}
	if sess.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestSetAlive(t *testing.T) {
	_, client := net.Pipe()
	defer client.Close()
	sess := newSession(1, client)
	sess.SetAlive(false)
	if sess.Alive() {
		t.Error("should be dead after SetAlive(false)")
	}
	sess.SetAlive(true)
	if !sess.Alive() {
		t.Error("should be alive after SetAlive(true)")
	}
}

func TestClose(t *testing.T) {
	_, client := net.Pipe()

	sess := newSession(1, client)
	sess.Close()

	if sess.Alive() {
		t.Error("session should be dead after Close")
	}

	// Double close should not panic.
	sess.Close()
}
