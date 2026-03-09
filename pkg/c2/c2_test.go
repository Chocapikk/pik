package c2

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/Chocapikk/pik/pkg/c2/session"
)

// mockBackend is a minimal Backend for registry tests.
type mockBackend struct {
	name       string
	configured string
}

func (m *mockBackend) Name() string                                       { return m.name }
func (m *mockBackend) Setup(string, int) error                            { return nil }
func (m *mockBackend) GeneratePayload(string, string) (string, error)     { return "", nil }
func (m *mockBackend) WaitForSession(time.Duration) error                 { return nil }
func (m *mockBackend) Shutdown() error                                    { return nil }

// configurableBackend embeds mockBackend and implements Configurable.
type configurableBackend struct {
	mockBackend
	configPath string
}

func (c *configurableBackend) Configure(path string) {
	c.configPath = path
}

// saveRegistry snapshots the global registry so tests can restore it.
func saveRegistry() map[string]Backend {
	saved := make(map[string]Backend, len(registry))
	for k, v := range registry {
		saved[k] = v
	}
	return saved
}

func restoreRegistry(saved map[string]Backend) {
	for k := range registry {
		delete(registry, k)
	}
	for k, v := range saved {
		registry[k] = v
	}
}

// ---------------------------------------------------------------------------
// ResolvePayload
// ---------------------------------------------------------------------------

func TestResolvePayloadKnownType(t *testing.T) {
	payloads := PayloadMap{
		"n0litetebastardescarb0rund0rum": func(lhost string, lport int) string {
			return fmt.Sprintf("n0litetebastardescarb0rund0rum/%s:%d", lhost, lport)
		},
	}
	fallback := func(string, int) string { return "fallback" }

	got, err := ResolvePayload(payloads, "10.0.0.1", 4444, "n0litetebastardescarb0rund0rum", fallback)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "n0litetebastardescarb0rund0rum/10.0.0.1:4444"
	if got != want {
		t.Errorf("ResolvePayload = %q, want %q", got, want)
	}
}

func TestResolvePayloadUnknownTypeFallback(t *testing.T) {
	payloads := PayloadMap{
		"n0litetebastardescarb0rund0rum": func(string, int) string {
			return "should-not-be-called"
		},
	}
	fallback := func(lhost string, lport int) string {
		return fmt.Sprintf("fallback/%s:%d", lhost, lport)
	}

	got, err := ResolvePayload(payloads, "192.168.1.1", 9999, "unknown_type", fallback)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "fallback/192.168.1.1:9999"
	if got != want {
		t.Errorf("ResolvePayload = %q, want %q", got, want)
	}
}

func TestResolvePayloadEmptyMap(t *testing.T) {
	payloads := PayloadMap{}
	fallback := func(lhost string, lport int) string {
		return fmt.Sprintf("n0litetebastardescarb0rund0rum/%s:%d", lhost, lport)
	}

	got, err := ResolvePayload(payloads, "10.10.10.10", 1337, "anything", fallback)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "n0litetebastardescarb0rund0rum/10.10.10.10:1337"
	if got != want {
		t.Errorf("ResolvePayload = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// Register + Resolve
// ---------------------------------------------------------------------------

func TestRegisterAndResolve(t *testing.T) {
	saved := saveRegistry()
	defer restoreRegistry(saved)

	b := &mockBackend{name: "n0litetebastardescarb0rund0rum"}
	Register(b)

	got := Resolve("n0litetebastardescarb0rund0rum", "")
	if got == nil {
		t.Fatal("Resolve returned nil for registered backend")
	}
	if got.Name() != "n0litetebastardescarb0rund0rum" {
		t.Errorf("Resolve().Name() = %q, want %q", got.Name(), "n0litetebastardescarb0rund0rum")
	}
}

func TestResolveUnknownType(t *testing.T) {
	saved := saveRegistry()
	defer restoreRegistry(saved)

	// Clear registry to ensure no stale entries.
	for k := range registry {
		delete(registry, k)
	}

	got := Resolve("n0litetebastardescarb0rund0rum_missing", "")
	if got != nil {
		t.Errorf("Resolve should return nil for unknown type, got %v", got)
	}
}

func TestResolveConfigurableBackendWithConfigPath(t *testing.T) {
	saved := saveRegistry()
	defer restoreRegistry(saved)

	cb := &configurableBackend{
		mockBackend: mockBackend{name: "n0litetebastardescarb0rund0rum_configurable"},
	}
	Register(cb)

	got := Resolve("n0litetebastardescarb0rund0rum_configurable", "/etc/n0litetebastardescarb0rund0rum.conf")
	if got == nil {
		t.Fatal("Resolve returned nil for registered configurable backend")
	}
	if cb.configPath != "/etc/n0litetebastardescarb0rund0rum.conf" {
		t.Errorf("Configure was not called; configPath = %q, want %q", cb.configPath, "/etc/n0litetebastardescarb0rund0rum.conf")
	}
}

func TestResolveConfigurableBackendEmptyConfigPath(t *testing.T) {
	saved := saveRegistry()
	defer restoreRegistry(saved)

	cb := &configurableBackend{
		mockBackend: mockBackend{name: "n0litetebastardescarb0rund0rum_nopath"},
	}
	Register(cb)

	got := Resolve("n0litetebastardescarb0rund0rum_nopath", "")
	if got == nil {
		t.Fatal("Resolve returned nil for registered backend")
	}
	if cb.configPath != "" {
		t.Errorf("Configure should not be called with empty configPath, got %q", cb.configPath)
	}
}

func TestResolveNonConfigurableBackendWithConfigPath(t *testing.T) {
	saved := saveRegistry()
	defer restoreRegistry(saved)

	b := &mockBackend{name: "n0litetebastardescarb0rund0rum_plain"}
	Register(b)

	// Should not crash when configPath is provided to a non-Configurable backend.
	got := Resolve("n0litetebastardescarb0rund0rum_plain", "/some/path.conf")
	if got == nil {
		t.Fatal("Resolve returned nil for registered non-configurable backend")
	}
	if got.Name() != "n0litetebastardescarb0rund0rum_plain" {
		t.Errorf("Name() = %q, want %q", got.Name(), "n0litetebastardescarb0rund0rum_plain")
	}
}

func TestRegisterOverwrites(t *testing.T) {
	saved := saveRegistry()
	defer restoreRegistry(saved)

	b1 := &mockBackend{name: "n0litetebastardescarb0rund0rum_dup"}
	b2 := &mockBackend{name: "n0litetebastardescarb0rund0rum_dup"}
	Register(b1)
	Register(b2)

	got := Resolve("n0litetebastardescarb0rund0rum_dup", "")
	if got != b2 {
		t.Error("Register should overwrite previous backend with same name")
	}
}

// ---------------------------------------------------------------------------
// SessionBase (with real TCP listener + session.Manager)
// ---------------------------------------------------------------------------

func startSessionBase(t *testing.T) (*SessionBase, net.Listener) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	mgr := session.NewManager(ln)
	mgr.Start()
	return &SessionBase{Manager: mgr}, ln
}

func dial(t *testing.T, ln net.Listener) net.Conn {
	t.Helper()
	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	return conn
}

func TestSessionBaseWaitForSession(t *testing.T) {
	sb, ln := startSessionBase(t)
	defer sb.ShutdownManager()

	conn := dial(t, ln)
	defer conn.Close()

	err := sb.WaitForSession(2 * time.Second)
	if err != nil {
		t.Fatalf("WaitForSession: %v", err)
	}

	sessions := sb.Sessions()
	if len(sessions) != 1 {
		t.Fatalf("Sessions() returned %d, want 1", len(sessions))
	}
	if sessions[0].ID != 1 {
		t.Errorf("session ID = %d, want 1", sessions[0].ID)
	}
}

func TestSessionBaseWaitForSessionTimeout(t *testing.T) {
	sb, _ := startSessionBase(t)
	defer sb.ShutdownManager()

	err := sb.WaitForSession(50 * time.Millisecond)
	if err == nil {
		t.Fatal("WaitForSession should return error on timeout")
	}
}

func TestSessionBaseSessions(t *testing.T) {
	sb, ln := startSessionBase(t)
	defer sb.ShutdownManager()

	// No sessions initially.
	if sessions := sb.Sessions(); len(sessions) != 0 {
		t.Fatalf("expected 0 sessions, got %d", len(sessions))
	}

	// Create 3 sessions.
	for i := range 3 {
		conn := dial(t, ln)
		defer conn.Close()
		if err := sb.WaitForSession(2 * time.Second); err != nil {
			t.Fatalf("WaitForSession %d: %v", i+1, err)
		}
	}

	sessions := sb.Sessions()
	if len(sessions) != 3 {
		t.Fatalf("Sessions() returned %d, want 3", len(sessions))
	}
	for i, s := range sessions {
		if s.ID != i+1 {
			t.Errorf("Sessions()[%d].ID = %d, want %d", i, s.ID, i+1)
		}
	}
}

func TestSessionBaseKill(t *testing.T) {
	sb, ln := startSessionBase(t)
	defer sb.ShutdownManager()

	conn := dial(t, ln)
	defer conn.Close()

	if err := sb.WaitForSession(2 * time.Second); err != nil {
		t.Fatalf("WaitForSession: %v", err)
	}

	if err := sb.Kill(1); err != nil {
		t.Fatalf("Kill(1): %v", err)
	}

	sessions := sb.Sessions()
	if len(sessions) != 0 {
		t.Errorf("Sessions() after Kill = %d, want 0", len(sessions))
	}
}

func TestSessionBaseKillUnknown(t *testing.T) {
	sb, _ := startSessionBase(t)
	defer sb.ShutdownManager()

	err := sb.Kill(999)
	if err == nil {
		t.Error("Kill(999) should return error for unknown session")
	}
}

func TestSessionBaseInteract(t *testing.T) {
	sb, ln := startSessionBase(t)
	defer sb.ShutdownManager()

	conn := dial(t, ln)
	defer conn.Close()

	if err := sb.WaitForSession(2 * time.Second); err != nil {
		t.Fatalf("WaitForSession: %v", err)
	}

	// Kill it first so Interact returns an error (dead session) without
	// actually entering interactive mode (which would block on stdin).
	if err := sb.Kill(1); err != nil {
		t.Fatalf("Kill: %v", err)
	}

	err := sb.Interact(1)
	if err == nil {
		t.Error("Interact on dead session should return error")
	}
}

func TestSessionBaseInteractUnknown(t *testing.T) {
	sb, _ := startSessionBase(t)
	defer sb.ShutdownManager()

	err := sb.Interact(999)
	if err == nil {
		t.Error("Interact(999) should return error for unknown session")
	}
}

func TestSessionBaseShutdownManager(t *testing.T) {
	sb, ln := startSessionBase(t)

	conn := dial(t, ln)
	defer conn.Close()

	if err := sb.WaitForSession(2 * time.Second); err != nil {
		t.Fatalf("WaitForSession: %v", err)
	}

	err := sb.ShutdownManager()
	if err != nil {
		t.Fatalf("ShutdownManager: %v", err)
	}

	// After shutdown, sessions should be closed.
	sessions := sb.Sessions()
	if len(sessions) != 0 {
		t.Errorf("Sessions() after ShutdownManager = %d, want 0", len(sessions))
	}
}

func TestSessionBaseShutdownManagerNilManager(t *testing.T) {
	sb := &SessionBase{Manager: nil}
	err := sb.ShutdownManager()
	if err != nil {
		t.Fatalf("ShutdownManager with nil Manager should return nil, got %v", err)
	}
}

func TestSessionBaseShutdownManagerIdempotent(t *testing.T) {
	sb, _ := startSessionBase(t)

	if err := sb.ShutdownManager(); err != nil {
		t.Fatalf("first ShutdownManager: %v", err)
	}
	// Second call should not panic.
	if err := sb.ShutdownManager(); err != nil {
		t.Fatalf("second ShutdownManager: %v", err)
	}
}

func TestSessionBaseMultipleSessionsKillOne(t *testing.T) {
	sb, ln := startSessionBase(t)
	defer sb.ShutdownManager()

	conns := make([]net.Conn, 3)
	for i := range conns {
		conns[i] = dial(t, ln)
		defer conns[i].Close()
		if err := sb.WaitForSession(2 * time.Second); err != nil {
			t.Fatalf("WaitForSession %d: %v", i+1, err)
		}
	}

	// Kill the middle session.
	if err := sb.Kill(2); err != nil {
		t.Fatalf("Kill(2): %v", err)
	}

	sessions := sb.Sessions()
	if len(sessions) != 2 {
		t.Fatalf("Sessions() after Kill(2) = %d, want 2", len(sessions))
	}
	for _, s := range sessions {
		if s.ID == 2 {
			t.Error("killed session 2 should not appear in Sessions()")
		}
	}
}

// ---------------------------------------------------------------------------
// PayloadGen / PayloadMap types
// ---------------------------------------------------------------------------

func TestPayloadGenType(t *testing.T) {
	var gen PayloadGen = func(lhost string, lport int) string {
		return fmt.Sprintf("n0litetebastardescarb0rund0rum %s:%d", lhost, lport)
	}
	got := gen("127.0.0.1", 8080)
	want := "n0litetebastardescarb0rund0rum 127.0.0.1:8080"
	if got != want {
		t.Errorf("PayloadGen = %q, want %q", got, want)
	}
}

func TestPayloadMapMultipleEntries(t *testing.T) {
	payloads := PayloadMap{
		"alpha": func(lhost string, lport int) string {
			return fmt.Sprintf("alpha/%s:%d", lhost, lport)
		},
		"beta": func(lhost string, lport int) string {
			return fmt.Sprintf("beta/%s:%d", lhost, lport)
		},
	}
	fallback := func(string, int) string { return "n0litetebastardescarb0rund0rum_fallback" }

	for _, tt := range []struct {
		payloadType string
		wantPrefix  string
	}{
		{"alpha", "alpha/"},
		{"beta", "beta/"},
		{"gamma", "n0litetebastardescarb0rund0rum_fallback"},
	} {
		got, err := ResolvePayload(payloads, "10.0.0.1", 4444, tt.payloadType, fallback)
		if err != nil {
			t.Fatalf("ResolvePayload(%q): %v", tt.payloadType, err)
		}
		if tt.payloadType == "gamma" {
			if got != tt.wantPrefix {
				t.Errorf("ResolvePayload(%q) = %q, want %q", tt.payloadType, got, tt.wantPrefix)
			}
		} else {
			want := fmt.Sprintf("%s10.0.0.1:4444", tt.wantPrefix)
			if got != want {
				t.Errorf("ResolvePayload(%q) = %q, want %q", tt.payloadType, got, want)
			}
		}
	}
}
