package encode

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf16"
)

// Base64 encodes data to standard base64.
func Base64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// Base64Raw encodes data to base64 without padding.
func Base64Raw(data []byte) string {
	return base64.RawStdEncoding.EncodeToString(data)
}

// Base64Decode decodes a standard base64 string.
func Base64Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// Base64URL encodes data to URL-safe base64.
func Base64URL(data []byte) string {
	return base64.URLEncoding.EncodeToString(data)
}

// Hex encodes data to a hex string.
func Hex(data []byte) string {
	return hex.EncodeToString(data)
}

// HexDecode decodes a hex string.
func HexDecode(s string) ([]byte, error) {
	return hex.DecodeString(s)
}

// URL percent-encodes a string.
func URL(s string) string {
	return url.QueryEscape(s)
}

// URLDecode decodes a percent-encoded string.
func URLDecode(s string) (string, error) {
	return url.QueryUnescape(s)
}

// URLPath encodes a string for use in a URL path segment.
func URLPath(s string) string {
	return url.PathEscape(s)
}

// UTF16LE encodes a string to UTF-16 Little Endian bytes.
// Used for PowerShell -EncodedCommand.
func UTF16LE(s string) []byte {
	encoded := utf16.Encode([]rune(s))
	b := make([]byte, len(encoded)*2)
	for i, v := range encoded {
		b[i*2] = byte(v)
		b[i*2+1] = byte(v >> 8)
	}
	return b
}

// XOR applies XOR encryption with a repeating key.
func XOR(data, key []byte) []byte {
	out := make([]byte, len(data))
	for i := range data {
		out[i] = data[i] ^ key[i%len(key)]
	}
	return out
}

// URLRaw percent-encodes a string preserving RFC 3986 unreserved characters.
// Unlike url.QueryEscape, it encodes spaces as %20 (not +) and only escapes
// characters that are not in: A-Z a-z 0-9 - _ . ~
func URLRaw(s string) string {
	var b strings.Builder
	b.Grow(len(s) * 3)
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~':
			b.WriteByte(c)
		default:
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

// ROT13 applies ROT13 substitution cipher.
func ROT13(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, c := range s {
		switch {
		case c >= 'a' && c <= 'z':
			b.WriteRune('a' + (c-'a'+13)%26)
		case c >= 'A' && c <= 'Z':
			b.WriteRune('A' + (c-'A'+13)%26)
		default:
			b.WriteRune(c)
		}
	}
	return b.String()
}
