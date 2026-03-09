package log

import (
	"bytes"
	"strings"
	"testing"
)

func TestStyle(t *testing.T) {
	got := Style("\x1b[1m", "hello")
	want := "\x1b[1mhello\x1b[0m"
	if got != want {
		t.Errorf("Style() = %q, want %q", got, want)
	}
}

func TestColorHelpers(t *testing.T) {
	tests := []struct {
		name string
		fn   func(string) string
		code string
	}{
		{"Amber", Amber, BoldAmber},
		{"Green", Green, BoldGreen},
		{"Red", Red, BoldRed},
		{"Yellow", Yellow, BoldYellow},
		{"Blue", Blue, BoldBlue},
		{"White", White, BoldWhite},
		{"Muted", Muted, FgDim},
		{"DimText", DimText, Dim},
		{"BoldText", BoldText, Bold},
		{"UnderlineText", UnderlineText, Underline},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn("test")
			want := tt.code + "test" + Reset
			if got != want {
				t.Errorf("%s(\"test\") = %q, want %q", tt.name, got, want)
			}
		})
	}
}

func TestVisualLen(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"hello", 5},
		{"\x1b[1mhello\x1b[0m", 5},
		{"\x1b[38;5;214mfoo\x1b[0m bar", 7},
		{"", 0},
		{"\x1b[1m\x1b[0m", 0},
	}
	for _, tt := range tests {
		got := VisualLen(tt.input)
		if got != tt.want {
			t.Errorf("VisualLen(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestPad(t *testing.T) {
	// Plain text padding
	got := Pad("hi", 10)
	if len(got) != 10 {
		t.Errorf("Pad(\"hi\", 10) length = %d, want 10", len(got))
	}
	if !strings.HasPrefix(got, "hi") {
		t.Errorf("Pad should preserve original text")
	}

	// ANSI text padding - visual length should be padded correctly
	colored := Amber("hi")
	padded := Pad(colored, 10)
	if VisualLen(padded) != 10 {
		t.Errorf("Pad(colored, 10) visual len = %d, want 10", VisualLen(padded))
	}

	// No padding needed
	got = Pad("hello world", 5)
	if got != "hello world" {
		t.Errorf("Pad with gap<=0 should return original string")
	}

	// Exact width
	got = Pad("abc", 3)
	if got != "abc" {
		t.Errorf("Pad with exact width should return original string")
	}
}

func TestOutputAndSetOutput(t *testing.T) {
	original := Output()
	defer SetOutput(original)

	var buf bytes.Buffer
	SetOutput(&buf)
	if Output() != &buf {
		t.Error("SetOutput/Output round-trip failed")
	}
}

func TestStatus(t *testing.T) {
	var buf bytes.Buffer
	original := Output()
	defer SetOutput(original)
	SetOutput(&buf)

	Status("hello %s", "world")
	got := buf.String()
	if !strings.Contains(got, "hello world") {
		t.Errorf("Status output should contain message, got %q", got)
	}
	if !strings.Contains(got, ">>") {
		t.Errorf("Status output should contain '>>' prefix, got %q", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Error("Status output should end with newline")
	}
}

func TestSuccess(t *testing.T) {
	var buf bytes.Buffer
	original := Output()
	defer SetOutput(original)
	SetOutput(&buf)

	Success("done %d", 42)
	got := buf.String()
	if !strings.Contains(got, "done 42") {
		t.Errorf("Success output should contain message, got %q", got)
	}
	if !strings.Contains(got, "++") {
		t.Errorf("Success output should contain '++' prefix, got %q", got)
	}
}

func TestError(t *testing.T) {
	var buf bytes.Buffer
	original := Output()
	defer SetOutput(original)
	SetOutput(&buf)

	Error("fail: %s", "oops")
	got := buf.String()
	if !strings.Contains(got, "fail: oops") {
		t.Errorf("Error output should contain message, got %q", got)
	}
	if !strings.Contains(got, "!!") {
		t.Errorf("Error output should contain '!!' prefix, got %q", got)
	}
}

func TestWarning(t *testing.T) {
	var buf bytes.Buffer
	original := Output()
	defer SetOutput(original)
	SetOutput(&buf)

	Warning("warn: %v", true)
	got := buf.String()
	if !strings.Contains(got, "warn: true") {
		t.Errorf("Warning output should contain message, got %q", got)
	}
	if !strings.Contains(got, "**") {
		t.Errorf("Warning output should contain '**' prefix, got %q", got)
	}
}

func TestVerbose(t *testing.T) {
	var buf bytes.Buffer
	original := Output()
	defer SetOutput(original)
	SetOutput(&buf)

	Verbose("verbose %s", "msg")
	got := buf.String()
	if !strings.Contains(got, "verbose msg") {
		t.Errorf("Verbose output should contain message, got %q", got)
	}
	if !strings.Contains(got, "..") {
		t.Errorf("Verbose output should contain '..' prefix, got %q", got)
	}
}

func TestDebug(t *testing.T) {
	var buf bytes.Buffer
	original := Output()
	defer SetOutput(original)
	SetOutput(&buf)

	Debug("debug %d", 99)
	got := buf.String()
	if !strings.Contains(got, "debug 99") {
		t.Errorf("Debug output should contain message, got %q", got)
	}
	if !strings.Contains(got, "##") {
		t.Errorf("Debug output should contain '##' prefix, got %q", got)
	}
}
