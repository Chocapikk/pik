package sdk

import (
	"fmt"
	"math"
	"strings"
)

// --- fmt re-exports ---

// Sprintf is fmt.Sprintf, re-exported so modules don't need to import fmt.
func Sprintf(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}

// Errorf is fmt.Errorf, re-exported so modules don't need to import fmt.
func Errorf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}

// --- string helpers ---

// Replace creates a string replacer and applies it.
func Replace(s string, oldNew ...string) string {
	return strings.NewReplacer(oldNew...).Replace(s)
}

// Contains checks if s contains substr.
func Contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// ContainsI checks if s contains substr (case-insensitive).
func ContainsI(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
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
