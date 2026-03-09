package session

import (
	"net"
	"testing"
	"time"
)

func startManager(t *testing.T) (*Manager, net.Listener) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	mgr := NewManager(ln)
	mgr.Start()
	return mgr, ln
}

func dial(t *testing.T, ln net.Listener) net.Conn {
	t.Helper()
	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	return conn
}

func TestAccept(t *testing.T) {
	mgr, ln := startManager(t)
	defer mgr.Close()

	conn := dial(t, ln)
	defer conn.Close()

	sess, err := mgr.Accept(2 * time.Second)
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}
	if sess.ID != 1 {
		t.Errorf("first session ID = %d, want 1", sess.ID)
	}
	if !sess.Alive() {
		t.Error("accepted session should be alive")
	}
}

func TestAcceptTimeout(t *testing.T) {
	mgr, _ := startManager(t)
	defer mgr.Close()

	_, err := mgr.Accept(50 * time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestMultipleSessions(t *testing.T) {
	mgr, ln := startManager(t)
	defer mgr.Close()

	conns := make([]net.Conn, 3)
	for i := range conns {
		conns[i] = dial(t, ln)
		defer conns[i].Close()

		sess, err := mgr.Accept(2 * time.Second)
		if err != nil {
			t.Fatalf("Accept session %d: %v", i+1, err)
		}
		if sess.ID != i+1 {
			t.Errorf("session ID = %d, want %d", sess.ID, i+1)
		}
	}

	sessions := mgr.List()
	if len(sessions) != 3 {
		t.Fatalf("List() returned %d sessions, want 3", len(sessions))
	}
	for i, sess := range sessions {
		if sess.ID != i+1 {
			t.Errorf("List()[%d].ID = %d, want %d", i, sess.ID, i+1)
		}
	}
}

func TestGet(t *testing.T) {
	mgr, ln := startManager(t)
	defer mgr.Close()

	conn := dial(t, ln)
	defer conn.Close()

	accepted, err := mgr.Accept(2 * time.Second)
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}

	got, err := mgr.Get(accepted.ID)
	if err != nil {
		t.Fatalf("Get(%d): %v", accepted.ID, err)
	}
	if got.ID != accepted.ID {
		t.Errorf("Get returned ID %d, want %d", got.ID, accepted.ID)
	}

	_, err = mgr.Get(999)
	if err == nil {
		t.Error("Get(999) should return error for unknown ID")
	}
}

func TestKill(t *testing.T) {
	mgr, ln := startManager(t)
	defer mgr.Close()

	conn := dial(t, ln)
	defer conn.Close()

	sess, err := mgr.Accept(2 * time.Second)
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}

	if err := mgr.Kill(sess.ID); err != nil {
		t.Fatalf("Kill: %v", err)
	}
	if sess.Alive() {
		t.Error("killed session should be dead")
	}

	if err := mgr.Kill(999); err == nil {
		t.Error("Kill(999) should return error for unknown ID")
	}
}

func TestListFiltersDeadSessions(t *testing.T) {
	mgr, ln := startManager(t)
	defer mgr.Close()

	for range 3 {
		conn := dial(t, ln)
		defer conn.Close()
		if _, err := mgr.Accept(2 * time.Second); err != nil {
			t.Fatalf("Accept: %v", err)
		}
	}

	mgr.Kill(2)

	sessions := mgr.List()
	if len(sessions) != 2 {
		t.Fatalf("List() returned %d sessions, want 2", len(sessions))
	}
	for _, sess := range sessions {
		if sess.ID == 2 {
			t.Error("List() should not include killed session 2")
		}
	}
}

func TestListSortedByID(t *testing.T) {
	mgr, ln := startManager(t)
	defer mgr.Close()

	for range 5 {
		conn := dial(t, ln)
		defer conn.Close()
		if _, err := mgr.Accept(2 * time.Second); err != nil {
			t.Fatalf("Accept: %v", err)
		}
	}

	sessions := mgr.List()
	for i := 1; i < len(sessions); i++ {
		if sessions[i].ID <= sessions[i-1].ID {
			t.Errorf("List() not sorted: ID %d after %d", sessions[i].ID, sessions[i-1].ID)
		}
	}
}

func TestInteractDeadSession(t *testing.T) {
	mgr, ln := startManager(t)
	defer mgr.Close()

	conn := dial(t, ln)
	defer conn.Close()

	sess, err := mgr.Accept(2 * time.Second)
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}

	sess.Close()

	if err := mgr.Interact(sess.ID); err == nil {
		t.Error("Interact on dead session should return error")
	}
}

func TestInteractUnknownSession(t *testing.T) {
	mgr, _ := startManager(t)
	defer mgr.Close()

	if err := mgr.Interact(999); err == nil {
		t.Error("Interact(999) should return error for unknown ID")
	}
}

func TestCloseIdempotent(t *testing.T) {
	mgr, _ := startManager(t)

	mgr.Close()
	mgr.Close() // should not panic
}

func TestAcceptZeroTimeout(t *testing.T) {
	mgr, ln := startManager(t)

	conn := dial(t, ln)
	defer conn.Close()

	sess, err := mgr.Accept(0)
	if err != nil {
		t.Fatalf("Accept(0): %v", err)
	}
	if sess == nil {
		t.Fatal("expected session")
	}
	mgr.Close()
}

func TestAcceptAfterClose(t *testing.T) {
	mgr, _ := startManager(t)
	mgr.Close()

	_, err := mgr.Accept(50 * time.Millisecond)
	if err == nil {
		t.Error("Accept after Close should return error")
	}
}

func TestAcceptZeroTimeoutAfterClose(t *testing.T) {
	mgr, _ := startManager(t)

	go func() {
		time.Sleep(50 * time.Millisecond)
		mgr.Close()
	}()

	_, err := mgr.Accept(0)
	if err == nil {
		t.Error("Accept(0) after Close should return error")
	}
}
