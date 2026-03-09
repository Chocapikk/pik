package encode

import (
	"bytes"
	"testing"
)

func TestBufferByte(t *testing.T) {
	b := NewBuffer().Byte(0x14).Byte(0xFF).Build()
	if !bytes.Equal(b, []byte{0x14, 0xFF}) {
		t.Errorf("Byte = %v", b)
	}
}

func TestBufferBytes(t *testing.T) {
	b := NewBuffer().Bytes([]byte{1, 2, 3}).Build()
	if !bytes.Equal(b, []byte{1, 2, 3}) {
		t.Errorf("Bytes = %v", b)
	}
}

func TestBufferUint16(t *testing.T) {
	b := NewBuffer().Uint16(0x0102).Build()
	want := []byte{0x01, 0x02}
	if !bytes.Equal(b, want) {
		t.Errorf("Uint16 = %v, want %v", b, want)
	}
}

func TestBufferUint32(t *testing.T) {
	b := NewBuffer().Uint32(0x01020304).Build()
	want := []byte{0x01, 0x02, 0x03, 0x04}
	if !bytes.Equal(b, want) {
		t.Errorf("Uint32 = %v, want %v", b, want)
	}
}

func TestBufferUint64(t *testing.T) {
	b := NewBuffer().Uint64(1).Build()
	if len(b) != 8 {
		t.Fatalf("Uint64 len = %d", len(b))
	}
	if b[7] != 1 {
		t.Errorf("Uint64 = %v", b)
	}
}

func TestBufferUint16LE(t *testing.T) {
	b := NewBuffer().Uint16LE(0x0102).Build()
	want := []byte{0x02, 0x01}
	if !bytes.Equal(b, want) {
		t.Errorf("Uint16LE = %v, want %v", b, want)
	}
}

func TestBufferUint32LE(t *testing.T) {
	b := NewBuffer().Uint32LE(0x01020304).Build()
	want := []byte{0x04, 0x03, 0x02, 0x01}
	if !bytes.Equal(b, want) {
		t.Errorf("Uint32LE = %v, want %v", b, want)
	}
}

func TestBufferString(t *testing.T) {
	b := NewBuffer().String("hi").Build()
	// 4-byte length prefix (big-endian) + "hi"
	if len(b) != 6 {
		t.Fatalf("String len = %d, want 6", len(b))
	}
	if b[3] != 2 { // length = 2
		t.Errorf("length byte = %d", b[3])
	}
	if string(b[4:]) != "hi" {
		t.Errorf("string data = %q", b[4:])
	}
}

func TestBufferNameList(t *testing.T) {
	b := NewBuffer().NameList("a", "b", "c").Build()
	// "a,b,c" = 5 bytes, + 4 prefix = 9
	if len(b) != 9 {
		t.Fatalf("NameList len = %d, want 9", len(b))
	}
	if string(b[4:]) != "a,b,c" {
		t.Errorf("NameList data = %q", b[4:])
	}
}

func TestBufferZeroes(t *testing.T) {
	b := NewBuffer().Zeroes(4).Build()
	if !bytes.Equal(b, []byte{0, 0, 0, 0}) {
		t.Errorf("Zeroes = %v", b)
	}
}

func TestBufferLen(t *testing.T) {
	buf := NewBuffer().Byte(1).Uint16(2)
	if buf.Len() != 3 {
		t.Errorf("Len = %d, want 3", buf.Len())
	}
}

func TestBufferFluent(t *testing.T) {
	// Test full fluent chain like SSH packet building
	b := NewBuffer().
		Byte(0x14).
		Uint32(3).
		String("ssh-rsa").
		NameList("aes128-ctr", "aes256-ctr").
		Zeroes(2).
		Build()

	if len(b) == 0 {
		t.Error("fluent chain produced empty buffer")
	}
	if b[0] != 0x14 {
		t.Errorf("first byte = 0x%02x, want 0x14", b[0])
	}
}

func TestJSONEncode(t *testing.T) {
	got := JSON(map[string]int{"a": 1})
	if got != `{"a":1}` {
		t.Errorf("JSON = %q", got)
	}
}

func TestReverseEncode(t *testing.T) {
	if got := Reverse("abc"); got != "cba" {
		t.Errorf("Reverse = %q", got)
	}
	if got := Reverse(""); got != "" {
		t.Error("Reverse empty should be empty")
	}
}
