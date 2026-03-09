package sdk

import (
	"strings"
	"testing"
	"time"
)

func TestSprintf(t *testing.T) {
	got := Sprintf("hello %s %d", "world", 42)
	if got != "hello world 42" {
		t.Errorf("Sprintf = %q", got)
	}
}

func TestErrorf(t *testing.T) {
	err := Errorf("fail: %d", 42)
	if err.Error() != "fail: 42" {
		t.Errorf("Errorf = %q", err)
	}
}

func TestReplace(t *testing.T) {
	got := Replace("hello world foo", "hello", "hi", "foo", "bar")
	if got != "hi world bar" {
		t.Errorf("Replace = %q", got)
	}
}

func TestContains(t *testing.T) {
	if !Contains("hello world", "world") {
		t.Error("Contains should find world")
	}
	if Contains("hello", "World") {
		t.Error("Contains should be case-sensitive")
	}
}

func TestContainsI(t *testing.T) {
	if !ContainsI("Hello World", "hello") {
		t.Error("ContainsI should be case-insensitive")
	}
	if ContainsI("hello", "xyz") {
		t.Error("ContainsI should not match non-substring")
	}
}

func TestDedent(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			"basic",
			"\n\t\tline1\n\t\tline2\n",
			"line1\nline2",
		},
		{
			"mixed indent",
			"\n    a\n      b\n    c\n",
			"a\n  b\nc",
		},
		{
			"no indent",
			"abc",
			"abc",
		},
		{
			"all empty",
			"\n\n\n",
			"\n\n\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Dedent(tt.input)
			if got != tt.want {
				t.Errorf("Dedent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSleep(t *testing.T) {
	start := time.Now()
	Sleep(0)
	if time.Since(start) > time.Second {
		t.Error("Sleep(0) took too long")
	}
}

func TestPoll(t *testing.T) {
	count := 0
	err := Poll(5, func() bool {
		count++
		return count >= 2
	})
	if err != nil {
		t.Errorf("Poll = %v", err)
	}
	if count < 2 {
		t.Errorf("Poll called %d times", count)
	}
}

func TestPollTimeout(t *testing.T) {
	err := Poll(1, func() bool { return false })
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestUTCDateOffset(t *testing.T) {
	got := UTCDateOffset(0)
	today := time.Now().UTC().Format("2006-01-02")
	if !strings.HasPrefix(got, today) {
		t.Errorf("UTCDateOffset(0) = %q, want prefix %q", got, today)
	}

	yesterday := UTCDateOffset(-1)
	y := time.Now().UTC().Add(-24 * time.Hour).Format("2006-01-02")
	if !strings.HasPrefix(yesterday, y) {
		t.Errorf("UTCDateOffset(-1) = %q, want prefix %q", yesterday, y)
	}
}
