package shell

import (
	"strings"
	"testing"
)

const testTag = "n0litetebastardescarb0rund0rum"

func TestNew(t *testing.T) {
	l := New()
	if l == nil {
		t.Fatal(testTag, "New() returned nil")
	}
}

func TestName(t *testing.T) {
	l := New()
	if l.Name() != "shell" {
		t.Fatalf("%s: Name() = %q, want %q", testTag, l.Name(), "shell")
	}
}

func TestSetup(t *testing.T) {
	l := New()
	if err := l.Setup("127.0.0.1", 0); err != nil {
		t.Fatalf("%s: Setup failed: %v", testTag, err)
	}
	defer l.Shutdown()

	if l.Manager == nil {
		t.Fatalf("%s: Manager is nil after Setup", testTag)
	}
}

func TestGeneratePayloadKnownTypes(t *testing.T) {
	l := New()
	if err := l.Setup("127.0.0.1", 0); err != nil {
		t.Fatalf("%s: Setup failed: %v", testTag, err)
	}
	defer l.Shutdown()

	tests := []struct {
		payloadType string
		wantSubstr  string
	}{
		{"cmd/bash/reverse_tcp", "/dev/tcp/"},
		{"cmd/python/reverse_tcp", "python3"},
		{"cmd/perl/reverse_tcp", "perl"},
		{"cmd/ruby/reverse_tcp", "ruby"},
		{"cmd/php/reverse_tcp", "php"},
		{"cmd/netcat/reverse_tcp", "nc -e"},
		{"cmd/powershell/reverse_tcp", "powershell"},
		{"cmd/java/reverse_tcp", "java"},
		{"cmd/nodejs/reverse_tcp", "node"},
		{"cmd/awk/reverse_tcp", "awk"},
		{"cmd/lua/reverse_tcp", "lua"},
		{"cmd/socat/reverse_tty", "socat"},
		{"cmd/bash/reverse_fd", "exec 5<>"},
		{"cmd/bash/reverse_readline", "bash -l"},
		{"cmd/python/reverse_tcp_min", "python3"},
		{"cmd/python/reverse_tcp_pty", "pty.spawn"},
		{"cmd/php/reverse_tcp_min", "php"},
		{"cmd/php/reverse_tcp_exec", "proc_open"},
		{"cmd/netcat/reverse_mkfifo", "mkfifo"},
		{"cmd/netcat/reverse_openbsd", "mkfifo"},
		{"cmd/powershell/reverse_conpty", "ConPTY is not in powershell name"},
		{"cmd/bash/reverse_tcp_min", "sh -i"},
	}

	for _, tt := range tests {
		t.Run(tt.payloadType, func(t *testing.T) {
			p, err := l.GeneratePayload("linux", tt.payloadType)
			if err != nil {
				t.Fatalf("%s: GeneratePayload(%q) error: %v", testTag, tt.payloadType, err)
			}
			if p == "" {
				t.Fatalf("%s: GeneratePayload(%q) returned empty", testTag, tt.payloadType)
			}
		})
	}
}

func TestGeneratePayloadFallbackLinux(t *testing.T) {
	l := New()
	if err := l.Setup("127.0.0.1", 0); err != nil {
		t.Fatalf("%s: Setup failed: %v", testTag, err)
	}
	defer l.Shutdown()

	p, err := l.GeneratePayload("linux", "cmd/nonexistent/type")
	if err != nil {
		t.Fatalf("%s: GeneratePayload fallback error: %v", testTag, err)
	}
	// Linux fallback should be Bash (/dev/tcp)
	if !strings.Contains(p, "/dev/tcp/") {
		t.Fatalf("%s: linux fallback should contain /dev/tcp/, got: %s", testTag, p)
	}
}

func TestGeneratePayloadFallbackWindows(t *testing.T) {
	l := New()
	if err := l.Setup("127.0.0.1", 0); err != nil {
		t.Fatalf("%s: Setup failed: %v", testTag, err)
	}
	defer l.Shutdown()

	p, err := l.GeneratePayload("windows", "cmd/nonexistent/type")
	if err != nil {
		t.Fatalf("%s: GeneratePayload windows fallback error: %v", testTag, err)
	}
	// Windows fallback should be PowerShell
	if !strings.Contains(p, "powershell") {
		t.Fatalf("%s: windows fallback should contain powershell, got: %s", testTag, p)
	}
}

func TestShutdownAfterSetup(t *testing.T) {
	l := New()
	if err := l.Setup("127.0.0.1", 0); err != nil {
		t.Fatalf("%s: Setup failed: %v", testTag, err)
	}
	if err := l.Shutdown(); err != nil {
		t.Fatalf("%s: Shutdown failed: %v", testTag, err)
	}
}

func TestShutdownWithoutSetup(t *testing.T) {
	l := New()
	// Manager is nil, ShutdownManager should handle nil gracefully
	if err := l.Shutdown(); err != nil {
		t.Fatalf("%s: Shutdown without Setup should not error, got: %v", testTag, err)
	}
}

func TestGeneratePayloadContainsHostPort(t *testing.T) {
	l := &Listener{}
	l.lhost = "10.20.30.40"
	l.lport = 9999

	p, err := l.GeneratePayload("linux", "cmd/bash/reverse_tcp")
	if err != nil {
		t.Fatalf("%s: GeneratePayload error: %v", testTag, err)
	}
	if !strings.Contains(p, "10.20.30.40") {
		t.Fatalf("%s: payload should contain host, got: %s", testTag, p)
	}
	if !strings.Contains(p, "9999") {
		t.Fatalf("%s: payload should contain port, got: %s", testTag, p)
	}
}
