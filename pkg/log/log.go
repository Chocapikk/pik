// Package log provides zero-dependency colored terminal output.
package log

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// ANSI escape codes - LeakIX-inspired amber palette.
const (
	Reset     = "\x1b[0m"
	Bold      = "\x1b[1m"
	Dim       = "\x1b[2m"
	Underline = "\x1b[4m"

	// 256-color: amber accent (#fbbf24)
	FgAmber   = "\x1b[38;5;214m"
	BoldAmber = "\x1b[1;38;5;214m"

	// Standard colors
	BoldGreen   = "\x1b[1;32m"
	BoldRed     = "\x1b[1;31m"
	BoldYellow  = "\x1b[1;38;5;220m"
	BoldWhite   = "\x1b[1;37m"
	BoldBlue    = "\x1b[1;34m"
	BoldMagenta = "\x1b[1;35m"
	FgWhite     = "\x1b[37m"
	FgDim       = "\x1b[38;5;245m"
)

// Styled text helpers.

func Style(code, text string) string { return code + text + Reset }
func Amber(s string) string          { return Style(BoldAmber, s) }
func Green(s string) string          { return Style(BoldGreen, s) }
func Red(s string) string            { return Style(BoldRed, s) }
func Yellow(s string) string         { return Style(BoldYellow, s) }
func Blue(s string) string           { return Style(BoldBlue, s) }
func White(s string) string          { return Style(BoldWhite, s) }
func Muted(s string) string          { return Style(FgDim, s) }
func DimText(s string) string        { return Style(Dim, s) }
func BoldText(s string) string       { return Style(Bold, s) }
func UnderlineText(s string) string  { return Style(Underline, s) }

// Backward compat aliases.
func Cyan(s string) string   { return Amber(s) }
func Gray(s string) string   { return Muted(s) }

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// VisualLen returns the display width of a string, ignoring ANSI codes.
func VisualLen(s string) int { return len(ansiRe.ReplaceAllString(s, "")) }

// Pad right-pads a (possibly ANSI-colored) string to width based on visual length.
func Pad(s string, width int) string {
	gap := width - VisualLen(s)
	if gap <= 0 {
		return s
	}
	return s + strings.Repeat(" ", gap)
}

// Log functions.

func Status(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "%s %s\n", Amber(">>"), fmt.Sprintf(format, args...))
}

func Success(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "%s %s\n", Green("++"), fmt.Sprintf(format, args...))
}

func Error(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "%s %s\n", Red("!!"), fmt.Sprintf(format, args...))
}

func Warning(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "%s %s\n", Yellow("**"), fmt.Sprintf(format, args...))
}

func Verbose(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "%s %s\n", DimText(".."), fmt.Sprintf(format, args...))
}

func Debug(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "%s %s\n", Style(BoldMagenta, "##"), fmt.Sprintf(format, args...))
}
