package encode

import (
	"bytes"
	"encoding/binary"
	"strings"
)

// Buffer is a fluent binary packet builder with big-endian encoding.
// Designed for crafting protocol messages.
type Buffer struct {
	buf bytes.Buffer
}

// NewBuffer creates a new binary buffer.
func NewBuffer() *Buffer {
	return &Buffer{}
}

// Byte appends a single byte.
func (b *Buffer) Byte(v int) *Buffer {
	b.buf.WriteByte(byte(v))
	return b
}

// Bytes appends raw bytes.
func (b *Buffer) Bytes(v []byte) *Buffer {
	b.buf.Write(v)
	return b
}

// Uint16 appends a big-endian uint16.
func (b *Buffer) Uint16(v int) *Buffer {
	binary.Write(&b.buf, binary.BigEndian, uint16(v))
	return b
}

// Uint32 appends a big-endian uint32.
func (b *Buffer) Uint32(v int) *Buffer {
	binary.Write(&b.buf, binary.BigEndian, uint32(v))
	return b
}

// Uint64 appends a big-endian uint64.
func (b *Buffer) Uint64(v int) *Buffer {
	binary.Write(&b.buf, binary.BigEndian, uint64(v))
	return b
}

// Uint16LE appends a little-endian uint16.
func (b *Buffer) Uint16LE(v int) *Buffer {
	binary.Write(&b.buf, binary.LittleEndian, uint16(v))
	return b
}

// Uint32LE appends a little-endian uint32.
func (b *Buffer) Uint32LE(v int) *Buffer {
	binary.Write(&b.buf, binary.LittleEndian, uint32(v))
	return b
}

// String appends a length-prefixed string: 4-byte big-endian length + data.
func (b *Buffer) String(s string) *Buffer {
	b.Uint32(len(s))
	b.buf.WriteString(s)
	return b
}

// NameList appends a comma-joined name list with 4-byte length prefix.
func (b *Buffer) NameList(names ...string) *Buffer {
	return b.String(strings.Join(names, ","))
}

// Zeroes appends n zero bytes.
func (b *Buffer) Zeroes(n int) *Buffer {
	b.buf.Write(make([]byte, n))
	return b
}

// Len returns the current buffer length.
func (b *Buffer) Len() int {
	return b.buf.Len()
}

// Build returns the buffer contents as a byte slice.
func (b *Buffer) Build() []byte {
	return b.buf.Bytes()
}

