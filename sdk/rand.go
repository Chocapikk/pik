package sdk

import "github.com/Chocapikk/pik/pkg/text"

// RandTextDefault generates random alphanumeric text without needing a Context.
func RandTextDefault(n int) string {
	return text.RandAlphaNum(n)
}

// RandInt returns a random int between min and max (inclusive).
func RandInt(min, max int) int {
	return text.RandInt(min, max+1)
}
