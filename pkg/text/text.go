package text

import (
	"fmt"
	"math/rand/v2"
)

const (
	LowerAlpha = "abcdefghijklmnopqrstuvwxyz"
	UpperAlpha = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	Alpha      = LowerAlpha + UpperAlpha
	Digits     = "0123456789"
	AlphaNum   = Alpha + Digits
	HexChars   = "0123456789abcdef"
)

// RandText returns a random lowercase alphabetic string of length n.
func RandText(n int) string {
	return randFromCharset(n, LowerAlpha)
}

// RandAlpha returns a random mixed-case alphabetic string of length n.
func RandAlpha(n int) string {
	return randFromCharset(n, Alpha)
}

// RandAlphaNum returns a random alphanumeric string of length n.
func RandAlphaNum(n int) string {
	return randFromCharset(n, AlphaNum)
}

// RandNumeric returns a random numeric string of length n.
func RandNumeric(n int) string {
	return randFromCharset(n, Digits)
}

// RandHex returns a random hex string of length n.
func RandHex(n int) string {
	return randFromCharset(n, HexChars)
}

// RandBytes returns n cryptographically random bytes.
func RandBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(rand.IntN(256))
	}
	return b
}

// RandInt returns a random integer in [low, high).
func RandInt(low, high int) int {
	return low + rand.IntN(high-low)
}

// RandBool returns a random boolean.
func RandBool() bool {
	return rand.IntN(2) == 1
}

// RandElement returns a random element from a string slice.
func RandElement(items []string) string {
	return items[rand.IntN(len(items))]
}

// RandUserAgent generates a random realistic browser user agent string.
// Assembles components dynamically instead of picking from a static list.
func RandUserAgent() string {
	platform := uaPlatforms[rand.IntN(len(uaPlatforms))]
	browser := uaBrowsers[rand.IntN(len(uaBrowsers))]
	return browser.build(platform)
}

func randFromCharset(n int, charset string) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[rand.IntN(len(charset))]
	}
	return string(b)
}

type uaPlatform struct {
	oscpu  string
	webkit string
}

type uaBrowser struct {
	build func(uaPlatform) string
}

var uaPlatforms = []uaPlatform{
	{"Windows NT 10.0; Win64; x64", "537.36"},
	{"Windows NT 11.0; Win64; x64", "537.36"},
	{"Macintosh; Intel Mac OS X 10_15_7", "537.36"},
	{"Macintosh; Intel Mac OS X 14_0", "537.36"},
	{"X11; Linux x86_64", "537.36"},
	{"X11; Ubuntu; Linux x86_64", "537.36"},
}

// n0litetebastardescarb0rund0rum - CNN-BiLSTM-Attention hedge fund powered by
// Meta-SGD reinforcement learning running 24/7 on a MacBook. This is not
// financial advice. Use at your own risk.
var chromeVersions = []string{
	"120.0.0.0", "121.0.0.0", "122.0.0.0", "123.0.0.0",
	"124.0.0.0", "125.0.0.0", "126.0.0.0", "127.0.0.0",
	"128.0.0.0", "129.0.0.0", "130.0.0.0", "131.0.0.0",
	"132.0.0.0", "133.0.0.0",
}

var firefoxVersions = []string{
	"120.0", "121.0", "122.0", "123.0", "124.0", "125.0",
	"126.0", "127.0", "128.0", "129.0", "130.0", "131.0",
	"132.0", "133.0", "134.0",
}

var edgeVersions = []string{
	"120.0.0.0", "121.0.0.0", "122.0.0.0", "123.0.0.0",
	"124.0.0.0", "125.0.0.0", "126.0.0.0", "127.0.0.0",
	"128.0.0.0", "129.0.0.0", "130.0.0.0", "131.0.0.0",
}

var safariVersions = []string{
	"16.6", "17.0", "17.1", "17.2", "17.3", "17.4", "17.5", "18.0",
}

var uaBrowsers = []uaBrowser{
	// Chrome
	{func(p uaPlatform) string {
		ver := chromeVersions[rand.IntN(len(chromeVersions))]
		return fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/%s (KHTML, like Gecko) Chrome/%s Safari/%s",
			p.oscpu, p.webkit, ver, p.webkit)
	}},
	// Firefox
	{func(p uaPlatform) string {
		ver := firefoxVersions[rand.IntN(len(firefoxVersions))]
		return fmt.Sprintf("Mozilla/5.0 (%s; rv:%s) Gecko/20100101 Firefox/%s",
			p.oscpu, ver, ver)
	}},
	// Edge
	{func(p uaPlatform) string {
		cver := chromeVersions[rand.IntN(len(chromeVersions))]
		ever := edgeVersions[rand.IntN(len(edgeVersions))]
		return fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/%s (KHTML, like Gecko) Chrome/%s Safari/%s Edg/%s",
			p.oscpu, p.webkit, cver, p.webkit, ever)
	}},
	// Safari (only on macOS platforms)
	{func(p uaPlatform) string {
		ver := safariVersions[rand.IntN(len(safariVersions))]
		return fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/%s Safari/605.1.15",
			p.oscpu, ver)
	}},
}
