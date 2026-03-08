package sdk

import (
	"regexp"

	"github.com/Chocapikk/pik/pkg/encode"
)

// JSONBody serializes a value to a JSON string for use in Request.Body.
func JSONBody(v any) string {
	return encode.JSON(v)
}

// Base64Decode decodes a base64 string.
func Base64Decode(s string) (string, error) {
	data, err := encode.Base64Decode(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Base64Encode encodes a string to base64.
func Base64Encode(s string) string {
	return encode.Base64([]byte(s))
}

// UTF16LEBase64 encodes a string as UTF-16LE then base64.
// Used for PowerShell -EncodedCommand / -e payloads.
func UTF16LEBase64(s string) string {
	return encode.Base64(encode.UTF16LE(s))
}

// RegexFind returns the first capturing group match of pattern in s, or empty string.
func RegexFind(pattern, s string) string {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return ""
	}
	match := re.FindStringSubmatch(s)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}

// HexEncode encodes a string to hexadecimal.
func HexEncode(s string) string { return encode.Hex([]byte(s)) }

// ROT13 applies ROT13 substitution cipher.
func ROT13(s string) string { return encode.ROT13(s) }

// Reverse returns the string reversed byte-by-byte.
func Reverse(s string) string { return encode.Reverse(s) }

// --- Binary packing ---

// Buffer is a fluent binary packet builder for crafting protocol messages.
// Re-exported from pkg/encode.
type Buffer = encode.Buffer

// NewBuffer creates a new binary packet builder.
func NewBuffer() *Buffer {
	return encode.NewBuffer()
}
