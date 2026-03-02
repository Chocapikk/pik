package stager

import (
	"crypto/rand"
	"encoding/binary"
	"math/big"
)

// patchRel32 patches a 4-byte relative offset at position off in buf,
// pointing to target. The offset is relative to off+4 (next instruction).
func patchRel32(buf []byte, off, target int) {
	rel := int32(target - (off + 4))
	binary.LittleEndian.PutUint32(buf[off:], uint32(rel))
}

func appendU64(buf []byte, val uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, val)
	return append(buf, b...)
}

func appendU32(buf []byte, val uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, val)
	return append(buf, b...)
}

func uint16BE(v uint16) uint16 {
	return (v >> 8) | (v << 8)
}

// randSplit splits n into two random addends (a, b) where a + b = n and a > 0.
func randSplit(n uint32) (uint32, uint32) {
	if n <= 1 {
		return n, 0
	}
	a, _ := rand.Int(rand.Reader, big.NewInt(int64(n-1)))
	base := uint32(a.Int64()) + 1
	return base, n - base
}

// x64EmitSyscall emits an obfuscated syscall number load into eax for x86_64.
// Instead of "mov eax, N", emits "mov eax, A; add eax, B" where A+B=N.
func x64EmitSyscall(buf []byte, nr uint32) []byte {
	base, offset := randSplit(nr)
	buf = append(buf, 0xb8) // mov eax, imm32
	buf = appendU32(buf, base)
	buf = append(buf, 0x05) // add eax, imm32
	buf = appendU32(buf, offset)
	return buf
}

// x86EmitSyscall emits an obfuscated syscall number load into eax for i386.
func x86EmitSyscall(buf []byte, nr uint32) []byte {
	return x64EmitSyscall(buf, nr) // same encoding
}

// arm64EmitSyscall emits an obfuscated syscall number load into x8 for aarch64.
// Instead of "mov x8, #N", emits "mov x8, #A; add x8, x8, #B".
func arm64EmitSyscall(buf []byte, nr uint32) []byte {
	base, offset := randSplit(nr)
	// mov x8, #base -> 0xD2800008 | (base << 5)
	insn1 := uint32(0xD2800008) | (base&0xFFFF)<<5
	buf = arm64Emit(buf, insn1)
	// add x8, x8, #offset -> 0x91000108 | (offset << 10)
	insn2 := uint32(0x91000108) | (offset&0xFFF)<<10
	buf = arm64Emit(buf, insn2)
	return buf
}
