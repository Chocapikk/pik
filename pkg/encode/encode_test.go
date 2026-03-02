package encode

import (
	"bytes"
	"testing"
)

func TestBase64RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"hello", []byte("hello world")},
		{"binary", []byte{0x00, 0xff, 0x80, 0x7f}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := Base64(tt.data)
			decoded, err := Base64Decode(encoded)
			if err != nil {
				t.Fatalf("Base64Decode: %v", err)
			}
			if !bytes.Equal(decoded, tt.data) {
				t.Errorf("round trip failed: got %v, want %v", decoded, tt.data)
			}
		})
	}
}

func TestBase64Raw(t *testing.T) {
	encoded := Base64Raw([]byte("test"))
	if encoded[len(encoded)-1] == '=' {
		t.Error("Base64Raw should not have padding")
	}
}

func TestBase64URL(t *testing.T) {
	data := []byte{0xfb, 0xef, 0xbe}
	encoded := Base64URL(data)
	if encoded == Base64(data) {
		t.Error("Base64URL should differ from standard Base64 for this input")
	}
}

func TestHexRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"hello", []byte("hello")},
		{"binary", []byte{0xde, 0xad, 0xbe, 0xef}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := Hex(tt.data)
			decoded, err := HexDecode(encoded)
			if err != nil {
				t.Fatalf("HexDecode: %v", err)
			}
			if !bytes.Equal(decoded, tt.data) {
				t.Errorf("round trip failed")
			}
		})
	}
}

func TestHex(t *testing.T) {
	got := Hex([]byte{0xde, 0xad})
	if got != "dead" {
		t.Errorf("Hex = %q, want %q", got, "dead")
	}
}

func TestURLRoundTrip(t *testing.T) {
	tests := []string{
		"hello world",
		"a=b&c=d",
		"special: !@#$%^&*()",
		"",
	}
	for _, input := range tests {
		encoded := URL(input)
		decoded, err := URLDecode(encoded)
		if err != nil {
			t.Fatalf("URLDecode(%q): %v", encoded, err)
		}
		if decoded != input {
			t.Errorf("URL round trip: got %q, want %q", decoded, input)
		}
	}
}

func TestURLPath(t *testing.T) {
	got := URLPath("hello world/test")
	if got == "hello world/test" {
		t.Error("URLPath should encode spaces")
	}
}

func TestUTF16LE(t *testing.T) {
	got := UTF16LE("A")
	if len(got) != 2 {
		t.Fatalf("UTF16LE(A) length = %d, want 2", len(got))
	}
	if got[0] != 0x41 || got[1] != 0x00 {
		t.Errorf("UTF16LE(A) = %v, want [0x41 0x00]", got)
	}
}

func TestUTF16LEMultiChar(t *testing.T) {
	got := UTF16LE("AB")
	if len(got) != 4 {
		t.Fatalf("UTF16LE(AB) length = %d, want 4", len(got))
	}
	if got[0] != 0x41 || got[2] != 0x42 {
		t.Errorf("UTF16LE(AB) = %v", got)
	}
}

func TestXOR(t *testing.T) {
	data := []byte("hello")
	key := []byte{0x42}
	encrypted := XOR(data, key)
	decrypted := XOR(encrypted, key)
	if !bytes.Equal(decrypted, data) {
		t.Errorf("XOR round trip failed")
	}
}

func TestXORRepeatingKey(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04}
	key := []byte{0xff, 0x00}
	got := XOR(data, key)
	want := []byte{0xfe, 0x02, 0xfc, 0x04}
	if !bytes.Equal(got, want) {
		t.Errorf("XOR repeating key = %v, want %v", got, want)
	}
}

func TestURLRaw(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"hello world", "hello%20world"},
		{"a+b", "a%2Bb"},
		{"test~value", "test~value"},
		{"foo/bar", "foo%2Fbar"},
	}
	for _, tt := range tests {
		got := URLRaw(tt.input)
		if got != tt.want {
			t.Errorf("URLRaw(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestROT13(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "uryyb"},
		{"HELLO", "URYYB"},
		{"Hello World!", "Uryyb Jbeyq!"},
		{"", ""},
		{"123", "123"},
	}
	for _, tt := range tests {
		got := ROT13(tt.input)
		if got != tt.want {
			t.Errorf("ROT13(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestROT13Involution(t *testing.T) {
	input := "The Quick Brown Fox"
	if got := ROT13(ROT13(input)); got != input {
		t.Errorf("ROT13(ROT13(x)) = %q, want %q", got, input)
	}
}
