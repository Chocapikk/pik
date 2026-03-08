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

// RandAlpha returns a random mixed-case alphabetic string of length n.
func RandAlpha(n int) string { return text.RandAlpha(n) }

// RandBool returns a random boolean.
func RandBool() bool { return text.RandBool() }

// Shuffle returns a shuffled copy of a string slice.
func Shuffle(items []string) []string { return text.Shuffle(items) }
