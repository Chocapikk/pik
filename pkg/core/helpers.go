package core

import (
	"fmt"
	"math"
	"strings"
)

// Sprintf is fmt.Sprintf, re-exported so modules don't need to import fmt.
func Sprintf(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}

// Errorf is fmt.Errorf, re-exported so modules don't need to import fmt.
func Errorf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}

// Dedent strips the common leading whitespace from all non-empty lines.
func Dedent(s string) string {
	lines := strings.Split(s, "\n")

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

	for i, line := range lines {
		if len(line) >= minIndent {
			lines[i] = line[minIndent:]
		}
	}

	result := strings.Join(lines, "\n")
	return strings.TrimLeft(strings.TrimRight(result, " \t\n"), "\n")
}
