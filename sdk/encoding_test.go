package sdk

import "testing"

func TestBase64EncodeDecode(t *testing.T) {
	encoded := Base64Encode("hello world")
	decoded, err := Base64Decode(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded != "hello world" {
		t.Errorf("round trip = %q", decoded)
	}
}

func TestBase64DecodeInvalid(t *testing.T) {
	_, err := Base64Decode("not-valid-base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestUTF16LEBase64(t *testing.T) {
	got := UTF16LEBase64("A")
	if got == "" {
		t.Error("UTF16LEBase64 should not be empty")
	}
	// "A" -> UTF16LE [0x41, 0x00] -> base64 "QQA="
	if got != "QQA=" {
		t.Errorf("UTF16LEBase64(A) = %q, want QQA=", got)
	}
}

func TestRegexFind(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		want    string
	}{
		{`version[:\s]+(\d+\.\d+)`, "version: 1.5", "1.5"},
		{`id=(\d+)`, "item?id=42&name=test", "42"},
		{`no-match`, "hello", ""},
		{`(bad`, "test", ""},  // invalid regex
		{`(\w+)`, "hello", "hello"},
	}
	for _, tt := range tests {
		got := RegexFind(tt.pattern, tt.input)
		if got != tt.want {
			t.Errorf("RegexFind(%q, %q) = %q, want %q", tt.pattern, tt.input, got, tt.want)
		}
	}
}

func TestHexEncode(t *testing.T) {
	if got := HexEncode("AB"); got != "4142" {
		t.Errorf("HexEncode = %q", got)
	}
}

func TestROT13SDK(t *testing.T) {
	if got := ROT13("Hello"); got != "Uryyb" {
		t.Errorf("ROT13 = %q", got)
	}
	if got := ROT13(ROT13("test")); got != "test" {
		t.Error("ROT13 should be involution")
	}
}

func TestReverse(t *testing.T) {
	if got := Reverse("abcd"); got != "dcba" {
		t.Errorf("Reverse = %q", got)
	}
	if got := Reverse(""); got != "" {
		t.Errorf("Reverse empty = %q", got)
	}
}

func TestJSONBody(t *testing.T) {
	got := JSONBody(map[string]string{"key": "val"})
	if got != `{"key":"val"}` {
		t.Errorf("JSONBody = %q", got)
	}
}

func TestNewBuffer(t *testing.T) {
	b := NewBuffer().Byte(0x01).Uint16(0x0203).Build()
	if len(b) != 3 || b[0] != 0x01 {
		t.Errorf("NewBuffer = %v", b)
	}
}

func TestPHPEvalWrap(t *testing.T) {
	got := PHPEvalWrap("echo 1;")
	if !Contains(got, "eval(base64_decode('") {
		t.Errorf("PHPEvalWrap = %q", got)
	}
}
