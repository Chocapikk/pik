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
