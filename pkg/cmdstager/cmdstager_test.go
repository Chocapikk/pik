package cmdstager

import (
	"strings"
	"testing"
)

func TestGeneratePrintf(t *testing.T) {
	binary := []byte{0x7f, 'E', 'L', 'F'}
	opts := Options{TempPath: "/tmp/.test", LineMax: 2047}
	cmds, err := Generate(binary, FlavorPrintf, opts)
	if err != nil {
		t.Fatalf("Generate(FlavorPrintf) error: %v", err)
	}
	if cmds == nil {
		t.Fatal("Generate(FlavorPrintf) returned nil")
	}
	if len(cmds) == 0 {
		t.Fatal("Generate(FlavorPrintf) returned empty slice")
	}
	if !strings.HasPrefix(cmds[0], "printf ") {
		t.Errorf("expected printf command, got %q", cmds[0])
	}
}

func TestGenerateBourne(t *testing.T) {
	binary := []byte{0x7f, 'E', 'L', 'F'}
	opts := Options{TempPath: "/tmp/.test", LineMax: 2047}
	cmds, err := Generate(binary, FlavorBourne, opts)
	if err != nil {
		t.Fatalf("Generate(FlavorBourne) error: %v", err)
	}
	if cmds == nil {
		t.Fatal("Generate(FlavorBourne) returned nil")
	}
	if len(cmds) == 0 {
		t.Fatal("Generate(FlavorBourne) returned empty slice")
	}
	if !strings.HasPrefix(cmds[0], "echo -n ") {
		t.Errorf("expected echo command, got %q", cmds[0])
	}
}

func TestGenerateEmptyFlavorDefaultsPrintf(t *testing.T) {
	binary := []byte{0x41, 0x42}
	opts := Options{TempPath: "/tmp/.test", LineMax: 2047}
	cmds, err := Generate(binary, "", opts)
	if err != nil {
		t.Fatalf("Generate('') error: %v", err)
	}
	if cmds == nil {
		t.Fatal("Generate('') returned nil")
	}
	// Empty flavor should default to printf
	if !strings.HasPrefix(cmds[0], "printf ") {
		t.Errorf("empty flavor should default to printf, got %q", cmds[0])
	}
}

func TestGenerateUnknownFlavor(t *testing.T) {
	binary := []byte{0x41}
	opts := Options{TempPath: "/tmp/.test"}
	cmds, err := Generate(binary, "nonexistent", opts)
	if err == nil {
		t.Fatal("Generate(unknown) should return error")
	}
	if cmds != nil {
		t.Errorf("Generate(unknown) should return nil commands, got %v", cmds)
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention the unknown flavor, got: %v", err)
	}
}
