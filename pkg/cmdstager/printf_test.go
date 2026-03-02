package cmdstager

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func TestPrintfChunking(t *testing.T) {
	binary := make([]byte, 256)
	for i := range binary {
		binary[i] = byte(i)
	}

	opts := Options{TempPath: "/tmp/.test", LineMax: 100}
	commands := Printf(binary, opts)

	// Must have at least 1 chunk + chmod + exec + rm = 4 commands minimum
	if len(commands) < 4 {
		t.Fatalf("expected at least 4 commands, got %d", len(commands))
	}

	// All chunk commands must respect LineMax
	for i, cmd := range commands {
		if strings.HasPrefix(cmd, "printf ") && len(cmd) > opts.LineMax {
			t.Errorf("chunk %d exceeds LineMax: %d > %d", i, len(cmd), opts.LineMax)
		}
	}

	// Last 3 commands: chmod, exec, rm
	n := len(commands)
	if commands[n-3] != "chmod +x /tmp/.test" {
		t.Errorf("expected chmod, got %q", commands[n-3])
	}
	if commands[n-2] != "/tmp/.test &" {
		t.Errorf("expected exec, got %q", commands[n-2])
	}
	if commands[n-1] != "rm -f /tmp/.test" {
		t.Errorf("expected rm, got %q", commands[n-1])
	}
}

func TestPrintfRoundTrip(t *testing.T) {
	binary := []byte{0x7f, 'E', 'L', 'F', 0x02, 0x01, 0x00, 0xff}

	opts := Options{TempPath: "/tmp/.test", LineMax: 2047}
	commands := Printf(binary, opts)

	// Decode the octal sequences from all printf chunks
	var decoded []byte
	for _, cmd := range commands {
		if !strings.HasPrefix(cmd, "printf '") {
			continue
		}
		// Extract between printf ' and '>>/tmp/.test
		start := len("printf '")
		end := strings.Index(cmd, "'>>")
		if end < 0 {
			t.Fatalf("malformed printf command: %s", cmd)
		}
		octal := cmd[start:end]
		// Parse \NNN sequences
		for i := 0; i < len(octal); i += 4 {
			if octal[i] != '\\' {
				t.Fatalf("expected backslash at pos %d, got %c", i, octal[i])
			}
			val, err := strconv.ParseUint(octal[i+1:i+4], 8, 8)
			if err != nil {
				t.Fatalf("invalid octal at pos %d: %v", i, err)
			}
			decoded = append(decoded, byte(val))
		}
	}

	if string(decoded) != string(binary) {
		t.Errorf("round-trip failed: got %v, want %v", decoded, binary)
	}
}

func TestPrintfSmallLineMax(t *testing.T) {
	binary := []byte{0x41, 0x42, 0x43, 0x44}
	// Very small LineMax forces one byte per chunk
	opts := Options{TempPath: "/tmp/.x", LineMax: 30}
	commands := Printf(binary, opts)

	chunks := 0
	for _, cmd := range commands {
		if strings.HasPrefix(cmd, "printf ") {
			chunks++
		}
	}
	if chunks < 2 {
		t.Errorf("expected multiple chunks with small LineMax, got %d", chunks)
	}
}

func TestPrintfAllBytes(t *testing.T) {
	// Every possible byte value
	binary := make([]byte, 256)
	for i := range binary {
		binary[i] = byte(i)
	}

	opts := Options{TempPath: "/tmp/.test", LineMax: 2047}
	commands := Printf(binary, opts)

	// Verify all octal sequences are valid 3-digit octal
	for _, cmd := range commands {
		if !strings.HasPrefix(cmd, "printf '") {
			continue
		}
		start := len("printf '")
		end := strings.Index(cmd, "'>>")
		octal := cmd[start:end]
		if len(octal)%4 != 0 {
			t.Fatalf("octal data length %d not divisible by 4", len(octal))
		}
		for i := 0; i < len(octal); i += 4 {
			seq := octal[i : i+4]
			if seq[0] != '\\' {
				t.Errorf("expected backslash at %d, got %q", i, seq)
			}
			val, err := strconv.ParseUint(seq[1:4], 8, 8)
			if err != nil {
				t.Errorf("invalid octal %q: %v", seq, err)
			}
			if val > 255 {
				t.Errorf("octal value %d out of byte range", val)
			}
		}
	}
}

func TestPrintfEmpty(t *testing.T) {
	commands := Printf(nil, Options{TempPath: "/tmp/.test"})
	// Should just have chmod + exec + rm, no printf chunks
	if len(commands) != 3 {
		t.Errorf("expected 3 commands for empty binary, got %d: %v", len(commands), commands)
	}
}

func TestEncodeOctal(t *testing.T) {
	got := encodeOctal([]byte{0, 127, 255})
	want := `\000\177\377`
	if got != want {
		t.Errorf("encodeOctal: got %q, want %q", got, want)
	}
}

func BenchmarkPrintf(b *testing.B) {
	// 1MB binary
	binary := make([]byte, 1<<20)
	for i := range binary {
		binary[i] = byte(i)
	}
	opts := Options{TempPath: "/tmp/.bench", LineMax: 2047}
	b.ResetTimer()
	for b.Loop() {
		Printf(binary, opts)
	}
}

func ExamplePrintf() {
	binary := []byte{0x7f, 'E', 'L', 'F'}
	commands := Printf(binary, Options{TempPath: "/tmp/.implant", LineMax: 2047})
	for _, cmd := range commands {
		fmt.Println(cmd)
	}
	// Output:
	// printf '\177\105\114\106'>>/tmp/.implant
	// chmod +x /tmp/.implant
	// /tmp/.implant &
	// rm -f /tmp/.implant
}
