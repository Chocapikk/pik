package stager

import "encoding/binary"

// .shstrtab content: "\0.text\0.shstrtab\0"
var shstrtab = []byte("\x00.text\x00.shstrtab\x00")

// makeELF64Wrapper returns an ELF64 wrapping function for the given e_machine.
// Includes fake .text and .shstrtab section headers to look like a normal stripped binary.
func makeELF64Wrapper(machine uint16) func([]byte) []byte {
	return func(shellcode []byte) []byte {
		const ehdrSize = 64
		const phdrSize = 56
		const shdrSize = 64 // Elf64_Shdr size
		const headerSize = ehdrSize + phdrSize
		const baseAddr = 0x400000
		const numSections = 3 // SHT_NULL + .text + .shstrtab

		entryPoint := baseAddr + headerSize

		elf := make([]byte, headerSize)

		copy(elf[0:], []byte{0x7f, 'E', 'L', 'F'})
		elf[4] = 2 // ELFCLASS64
		elf[5] = 1 // ELFDATA2LSB
		elf[6] = 1 // EV_CURRENT

		binary.LittleEndian.PutUint16(elf[0x10:], 2)       // e_type = ET_EXEC
		binary.LittleEndian.PutUint16(elf[0x12:], machine)  // e_machine
		binary.LittleEndian.PutUint32(elf[0x14:], 1)        // e_version
		binary.LittleEndian.PutUint64(elf[0x18:], uint64(entryPoint))
		binary.LittleEndian.PutUint64(elf[0x20:], ehdrSize) // e_phoff
		binary.LittleEndian.PutUint16(elf[0x34:], ehdrSize) // e_ehsize
		binary.LittleEndian.PutUint16(elf[0x36:], phdrSize) // e_phentsize
		binary.LittleEndian.PutUint16(elf[0x38:], 1)        // e_phnum
		binary.LittleEndian.PutUint16(elf[0x3A:], shdrSize) // e_shentsize
		binary.LittleEndian.PutUint16(elf[0x3C:], numSections) // e_shnum
		binary.LittleEndian.PutUint16(elf[0x3E:], 2)        // e_shstrndx = 2 (.shstrtab)

		// Program header at ehdrSize
		binary.LittleEndian.PutUint32(elf[ehdrSize:], 1)              // p_type = PT_LOAD
		binary.LittleEndian.PutUint32(elf[ehdrSize+4:], 7)            // p_flags = RWX
		binary.LittleEndian.PutUint64(elf[ehdrSize+8:], 0)            // p_offset
		binary.LittleEndian.PutUint64(elf[ehdrSize+16:], baseAddr)    // p_vaddr
		binary.LittleEndian.PutUint64(elf[ehdrSize+24:], baseAddr)    // p_paddr
		// p_filesz and p_memsz will be patched after we know total size
		// p_align
		binary.LittleEndian.PutUint64(elf[ehdrSize+48:], 0x1000)

		// Append shellcode
		elf = append(elf, shellcode...)

		// Append .shstrtab string table
		shstrtabOff := len(elf)
		elf = append(elf, shstrtab...)

		// Section header table offset
		shoff := len(elf)

		// Section 0: SHT_NULL (64 zero bytes)
		elf = append(elf, make([]byte, shdrSize)...)

		// Section 1: .text (SHT_PROGBITS)
		textShdr := make([]byte, shdrSize)
		binary.LittleEndian.PutUint32(textShdr[0:], 1)                  // sh_name = 1 (offset of ".text" in shstrtab)
		binary.LittleEndian.PutUint32(textShdr[4:], 1)                  // sh_type = SHT_PROGBITS
		binary.LittleEndian.PutUint64(textShdr[8:], 0x6)                // sh_flags = SHF_ALLOC|SHF_EXECINSTR
		binary.LittleEndian.PutUint64(textShdr[16:], uint64(entryPoint)) // sh_addr
		binary.LittleEndian.PutUint64(textShdr[24:], uint64(headerSize)) // sh_offset
		binary.LittleEndian.PutUint64(textShdr[32:], uint64(len(shellcode))) // sh_size
		binary.LittleEndian.PutUint64(textShdr[48:], 16)                // sh_addralign
		elf = append(elf, textShdr...)

		// Section 2: .shstrtab (SHT_STRTAB)
		strShdr := make([]byte, shdrSize)
		binary.LittleEndian.PutUint32(strShdr[0:], 7)                   // sh_name = 7 (offset of ".shstrtab")
		binary.LittleEndian.PutUint32(strShdr[4:], 3)                   // sh_type = SHT_STRTAB
		binary.LittleEndian.PutUint64(strShdr[24:], uint64(shstrtabOff)) // sh_offset
		binary.LittleEndian.PutUint64(strShdr[32:], uint64(len(shstrtab))) // sh_size
		binary.LittleEndian.PutUint64(strShdr[48:], 1)                  // sh_addralign
		elf = append(elf, strShdr...)

		// Patch e_shoff
		binary.LittleEndian.PutUint64(elf[0x28:], uint64(shoff))

		// Patch p_filesz and p_memsz to cover everything
		totalSize := len(elf)
		binary.LittleEndian.PutUint64(elf[ehdrSize+32:], uint64(totalSize))
		binary.LittleEndian.PutUint64(elf[ehdrSize+40:], uint64(totalSize+0x1000))

		return elf
	}
}

// makeELF32Wrapper returns an ELF32 wrapping function for the given e_machine.
// Includes fake .text and .shstrtab section headers.
func makeELF32Wrapper(machine uint16) func([]byte) []byte {
	return func(shellcode []byte) []byte {
		const ehdrSize = 52
		const phdrSize = 32
		const shdrSize = 40 // Elf32_Shdr size
		const headerSize = ehdrSize + phdrSize
		const baseAddr = 0x08048000
		const numSections = 3

		entryPoint := baseAddr + headerSize

		elf := make([]byte, headerSize)

		copy(elf[0:], []byte{0x7f, 'E', 'L', 'F'})
		elf[4] = 1 // ELFCLASS32
		elf[5] = 1 // ELFDATA2LSB
		elf[6] = 1 // EV_CURRENT

		binary.LittleEndian.PutUint16(elf[0x10:], 2)                   // e_type = ET_EXEC
		binary.LittleEndian.PutUint16(elf[0x12:], machine)             // e_machine
		binary.LittleEndian.PutUint32(elf[0x14:], 1)                   // e_version
		binary.LittleEndian.PutUint32(elf[0x18:], uint32(entryPoint))  // e_entry
		binary.LittleEndian.PutUint32(elf[0x1C:], ehdrSize)            // e_phoff
		binary.LittleEndian.PutUint16(elf[0x28:], ehdrSize)            // e_ehsize
		binary.LittleEndian.PutUint16(elf[0x2A:], phdrSize)            // e_phentsize
		binary.LittleEndian.PutUint16(elf[0x2C:], 1)                   // e_phnum
		binary.LittleEndian.PutUint16(elf[0x2E:], shdrSize)            // e_shentsize
		binary.LittleEndian.PutUint16(elf[0x30:], numSections)         // e_shnum
		binary.LittleEndian.PutUint16(elf[0x32:], 2)                   // e_shstrndx

		// Program header
		ph := ehdrSize
		binary.LittleEndian.PutUint32(elf[ph:], 1)          // p_type = PT_LOAD
		binary.LittleEndian.PutUint32(elf[ph+4:], 0)        // p_offset
		binary.LittleEndian.PutUint32(elf[ph+8:], baseAddr)  // p_vaddr
		binary.LittleEndian.PutUint32(elf[ph+12:], baseAddr) // p_paddr
		// p_filesz/p_memsz patched later
		binary.LittleEndian.PutUint32(elf[ph+24:], 7)        // p_flags = RWX
		binary.LittleEndian.PutUint32(elf[ph+28:], 0x1000)   // p_align

		// Shellcode
		elf = append(elf, shellcode...)

		// .shstrtab
		shstrtabOff := len(elf)
		elf = append(elf, shstrtab...)

		// Section header table
		shoff := len(elf)

		// Section 0: SHT_NULL
		elf = append(elf, make([]byte, shdrSize)...)

		// Section 1: .text
		textShdr := make([]byte, shdrSize)
		binary.LittleEndian.PutUint32(textShdr[0:], 1)                    // sh_name
		binary.LittleEndian.PutUint32(textShdr[4:], 1)                    // sh_type = SHT_PROGBITS
		binary.LittleEndian.PutUint32(textShdr[8:], 0x6)                  // sh_flags = ALLOC|EXECINSTR
		binary.LittleEndian.PutUint32(textShdr[12:], uint32(entryPoint))   // sh_addr
		binary.LittleEndian.PutUint32(textShdr[16:], uint32(headerSize))   // sh_offset
		binary.LittleEndian.PutUint32(textShdr[20:], uint32(len(shellcode))) // sh_size
		binary.LittleEndian.PutUint32(textShdr[32:], 16)                  // sh_addralign
		elf = append(elf, textShdr...)

		// Section 2: .shstrtab
		strShdr := make([]byte, shdrSize)
		binary.LittleEndian.PutUint32(strShdr[0:], 7)                     // sh_name
		binary.LittleEndian.PutUint32(strShdr[4:], 3)                     // sh_type = SHT_STRTAB
		binary.LittleEndian.PutUint32(strShdr[16:], uint32(shstrtabOff))   // sh_offset
		binary.LittleEndian.PutUint32(strShdr[20:], uint32(len(shstrtab))) // sh_size
		binary.LittleEndian.PutUint32(strShdr[32:], 1)                    // sh_addralign
		elf = append(elf, strShdr...)

		// Patch e_shoff
		binary.LittleEndian.PutUint32(elf[0x20:], uint32(shoff))

		// Patch p_filesz/p_memsz
		totalSize := len(elf)
		binary.LittleEndian.PutUint32(elf[ph+16:], uint32(totalSize))
		binary.LittleEndian.PutUint32(elf[ph+20:], uint32(totalSize+0x1000))

		return elf
	}
}
