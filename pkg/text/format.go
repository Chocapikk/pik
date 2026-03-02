package text

import (
	"math"
	"strings"
)

// Dedent strips the common leading whitespace from all non-empty lines.
// Leading/trailing blank lines are trimmed.
func Dedent(s string) string {
	lines := strings.Split(s, "\n")

	// Find minimum indentation across non-empty lines
	minIndent := math.MaxInt
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" {
			continue
		}
		indent := len(line) - len(trimmed)
		if indent < minIndent {
			minIndent = indent
		}
	}

	if minIndent == math.MaxInt {
		return s
	}

	// Strip common indent
	for i, line := range lines {
		if len(line) >= minIndent {
			lines[i] = line[minIndent:]
		}
	}

	result := strings.Join(lines, "\n")
	return strings.TrimLeft(strings.TrimRight(result, " \t\n"), "\n")
}
