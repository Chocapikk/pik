package sdk

import (
	"fmt"
	"math"
	"strings"
	"time"
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

// --- time helpers ---

// Sleep pauses execution for the given number of seconds.
func Sleep(seconds int) {
	time.Sleep(time.Duration(seconds) * time.Second)
}

// Poll calls fn repeatedly until it returns true or timeout expires.
func Poll(timeoutSec int, fn func() bool) error {
	deadline := time.After(time.Duration(timeoutSec) * time.Second)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			return Errorf("poll timed out after %ds", timeoutSec)
		case <-ticker.C:
			if fn() {
				return nil
			}
		}
	}
}

// UTCDateOffset returns an ISO 8601 datetime string offset by days from now.
// Negative values produce past dates, positive values produce future dates.
func UTCDateOffset(days int) string {
	return time.Now().UTC().Add(time.Duration(days) * 24 * time.Hour).Format("2006-01-02T15:04:05Z")
}
