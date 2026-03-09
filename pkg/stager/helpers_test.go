package stager

import (
	"encoding/binary"
	"testing"
)

func TestRandSplitZero(t *testing.T) {
	a, b := randSplit(0)
	if a != 0 || b != 0 {
		t.Errorf("randSplit(0) = (%d, %d), want (0, 0)", a, b)
	}
}

func TestRandSplitOne(t *testing.T) {
	a, b := randSplit(1)
	if a != 1 || b != 0 {
		t.Errorf("randSplit(1) = (%d, %d), want (1, 0)", a, b)
	}
}

func TestRandSplitSumsCorrectly(t *testing.T) {
	for _, n := range []uint32{2, 3, 10, 100, 1000, 0xFFFF} {
		a, b := randSplit(n)
		if a+b != n {
			t.Errorf("randSplit(%d) = (%d, %d), sum = %d", n, a, b, a+b)
		}
		if a == 0 {
			t.Errorf("randSplit(%d): a should be > 0, got 0", n)
		}
	}
}

func TestRandSplitRandomness(t *testing.T) {
	// With n=1000, consecutive calls should produce different splits most of the time.
	seen := make(map[uint32]bool)
	for range 20 {
		a, _ := randSplit(1000)
		seen[a] = true
	}
	if len(seen) < 2 {
		t.Error("randSplit(1000) returned the same base 20 times - not random")
	}
}

func TestPatchRel32(t *testing.T) {
	buf := make([]byte, 8)
	// offset at position 2, target at position 10
	// rel = 10 - (2 + 4) = 4
	patchRel32(buf, 2, 10)
	got := int32(binary.LittleEndian.Uint32(buf[2:6]))
	if got != 4 {
		t.Errorf("patchRel32(buf, 2, 10) wrote %d, want 4", got)
	}
}

func TestPatchRel32Negative(t *testing.T) {
	buf := make([]byte, 8)
	// offset at position 4, target at position 0
	// rel = 0 - (4 + 4) = -8
	patchRel32(buf, 4, 0)
	got := int32(binary.LittleEndian.Uint32(buf[4:8]))
	if got != -8 {
		t.Errorf("patchRel32(buf, 4, 0) wrote %d, want -8", got)
	}
}

func TestAppendU64(t *testing.T) {
	buf := appendU64(nil, 0x0102030405060708)
	if len(buf) != 8 {
		t.Fatalf("appendU64 returned %d bytes, want 8", len(buf))
	}
	got := binary.LittleEndian.Uint64(buf)
	if got != 0x0102030405060708 {
		t.Errorf("appendU64 = %#x, want %#x", got, uint64(0x0102030405060708))
	}
}

func TestAppendU32(t *testing.T) {
	buf := appendU32(nil, 0xDEADBEEF)
	if len(buf) != 4 {
		t.Fatalf("appendU32 returned %d bytes, want 4", len(buf))
	}
	got := binary.LittleEndian.Uint32(buf)
	if got != 0xDEADBEEF {
		t.Errorf("appendU32 = %#x, want %#x", got, uint32(0xDEADBEEF))
	}
}

func TestUint16BE(t *testing.T) {
	tests := []struct {
		in, want uint16
	}{
		{0x0102, 0x0201},
		{0x0000, 0x0000},
		{0xFF00, 0x00FF},
		{0x00FF, 0xFF00},
	}
	for _, tt := range tests {
		got := uint16BE(tt.in)
		if got != tt.want {
			t.Errorf("uint16BE(%#04x) = %#04x, want %#04x", tt.in, got, tt.want)
		}
	}
}

func TestX64EmitSyscall(t *testing.T) {
	buf := x64EmitSyscall(nil, 59) // execve
	// Expected format: 0xb8 <4 bytes base> 0x05 <4 bytes offset>
	if len(buf) != 10 {
		t.Fatalf("x64EmitSyscall produced %d bytes, want 10", len(buf))
	}
	if buf[0] != 0xb8 {
		t.Errorf("first opcode = %#x, want 0xb8 (mov eax)", buf[0])
	}
	if buf[5] != 0x05 {
		t.Errorf("second opcode = %#x, want 0x05 (add eax)", buf[5])
	}
	base := binary.LittleEndian.Uint32(buf[1:5])
	offset := binary.LittleEndian.Uint32(buf[6:10])
	if base+offset != 59 {
		t.Errorf("base(%d) + offset(%d) = %d, want 59", base, offset, base+offset)
	}
}

func TestX86EmitSyscall(t *testing.T) {
	buf := x86EmitSyscall(nil, 11) // execve on i386
	if len(buf) != 10 {
		t.Fatalf("x86EmitSyscall produced %d bytes, want 10", len(buf))
	}
	base := binary.LittleEndian.Uint32(buf[1:5])
	offset := binary.LittleEndian.Uint32(buf[6:10])
	if base+offset != 11 {
		t.Errorf("base(%d) + offset(%d) = %d, want 11", base, offset, base+offset)
	}
}

func TestArm64EmitSyscall(t *testing.T) {
	buf := arm64EmitSyscall(nil, 221) // execve on aarch64
	// Two 4-byte ARM64 instructions
	if len(buf) != 8 {
		t.Fatalf("arm64EmitSyscall produced %d bytes, want 8", len(buf))
	}
	// Decode mov x8, #base
	insn1 := binary.LittleEndian.Uint32(buf[0:4])
	base := (insn1 >> 5) & 0xFFFF
	// Decode add x8, x8, #offset
	insn2 := binary.LittleEndian.Uint32(buf[4:8])
	offset := (insn2 >> 10) & 0xFFF
	if base+offset != 221 {
		t.Errorf("base(%d) + offset(%d) = %d, want 221", base, offset, base+offset)
	}
}
