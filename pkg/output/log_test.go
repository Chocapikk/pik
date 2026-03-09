package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/Chocapikk/pik/pkg/log"
)

// helper to capture output and restore state
func withCapture(t *testing.T, fn func(buf *bytes.Buffer)) {
	t.Helper()
	var buf bytes.Buffer
	origWriter := log.Output()
	origVerbose := IsVerbose()
	origDebug := IsDebug()
	defer func() {
		log.SetOutput(origWriter)
		SetVerbose(origVerbose)
		SetDebug(origDebug)
	}()
	log.SetOutput(&buf)
	fn(&buf)
}

func TestSetVerboseAndIsVerbose(t *testing.T) {
	origV := IsVerbose()
	origD := IsDebug()
	defer func() {
		SetVerbose(origV)
		SetDebug(origD)
	}()

	SetVerbose(false)
	if IsVerbose() {
		t.Error("expected verbose to be false")
	}
	SetVerbose(true)
	if !IsVerbose() {
		t.Error("expected verbose to be true")
	}
}

func TestSetDebugAndIsDebug(t *testing.T) {
	origV := IsVerbose()
	origD := IsDebug()
	defer func() {
		SetVerbose(origV)
		SetDebug(origD)
	}()

	SetDebug(false)
	if IsDebug() {
		t.Error("expected debug to be false")
	}
	if IsVerbose() {
		t.Error("SetDebug(false) should also set verbose to false")
	}

	SetDebug(true)
	if !IsDebug() {
		t.Error("expected debug to be true")
	}
	if !IsVerbose() {
		t.Error("SetDebug(true) should also set verbose to true")
	}
}

func TestEnableDebug(t *testing.T) {
	origV := IsVerbose()
	origD := IsDebug()
	defer func() {
		SetVerbose(origV)
		SetDebug(origD)
	}()

	SetDebug(false)
	SetVerbose(false)
	EnableDebug()
	if !IsDebug() {
		t.Error("EnableDebug should set debug to true")
	}
	if !IsVerbose() {
		t.Error("EnableDebug should set verbose to true")
	}
}

func TestVerboseOnlyWritesWhenEnabled(t *testing.T) {
	withCapture(t, func(buf *bytes.Buffer) {
		SetVerbose(false)
		SetDebug(false)
		Verbose("should not appear")
		if buf.Len() != 0 {
			t.Errorf("Verbose should not write when verbose mode is off, got %q", buf.String())
		}

		SetVerbose(true)
		Verbose("visible %s", "msg")
		if !strings.Contains(buf.String(), "visible msg") {
			t.Errorf("Verbose should write when verbose mode is on, got %q", buf.String())
		}
	})
}

func TestDebugOnlyWritesWhenEnabled(t *testing.T) {
	withCapture(t, func(buf *bytes.Buffer) {
		SetDebug(false)
		Debug("should not appear")
		if buf.Len() != 0 {
			t.Errorf("Debug should not write when debug mode is off, got %q", buf.String())
		}

		SetDebug(true)
		Debug("visible %d", 42)
		if !strings.Contains(buf.String(), "visible 42") {
			t.Errorf("Debug should write when debug mode is on, got %q", buf.String())
		}
	})
}

func TestPrint(t *testing.T) {
	withCapture(t, func(buf *bytes.Buffer) {
		Print("hello %s", "world")
		got := buf.String()
		if got != "hello world" {
			t.Errorf("Print() = %q, want %q", got, "hello world")
		}
	})
}

func TestPrintln(t *testing.T) {
	withCapture(t, func(buf *bytes.Buffer) {
		Println("hello", "world")
		got := buf.String()
		if got != "hello world\n" {
			t.Errorf("Println() = %q, want %q", got, "hello world\n")
		}
	})
}

func TestHumanSize(t *testing.T) {
	tests := []struct {
		bytes int
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{2621440, "2.5 MB"},
	}
	for _, tt := range tests {
		got := HumanSize(tt.bytes)
		if got != tt.want {
			t.Errorf("HumanSize(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}

func TestBanner(t *testing.T) {
	withCapture(t, func(buf *bytes.Buffer) {
		BannerVersion = "1.0.0"
		BannerModuleCount = 5
		BannerPayloadCount = 3
		defer func() {
			BannerVersion = ""
			BannerModuleCount = 0
			BannerPayloadCount = 0
		}()

		Banner()
		got := buf.String()
		if len(got) == 0 {
			t.Error("Banner should produce output")
		}
		if !strings.Contains(got, "1.0.0") {
			t.Error("Banner should contain the version string")
		}
		if !strings.Contains(got, "5") {
			t.Error("Banner should contain module count")
		}
	})
}

func TestBannerDevVersion(t *testing.T) {
	withCapture(t, func(buf *bytes.Buffer) {
		BannerVersion = ""
		BannerModuleCount = 0
		BannerPayloadCount = 0

		Banner()
		got := buf.String()
		if !strings.Contains(got, "dev") {
			t.Error("Banner with empty version should show 'dev'")
		}
	})
}

func TestInfoBox(t *testing.T) {
	withCapture(t, func(buf *bytes.Buffer) {
		InfoBox("Test Title", "Key1", "Value1", "Key2", "Value2")
		got := buf.String()
		if len(got) == 0 {
			t.Error("InfoBox should produce output")
		}
		if !strings.Contains(got, "Test Title") {
			t.Error("InfoBox should contain the title")
		}
		if !strings.Contains(got, "Key1") {
			t.Error("InfoBox should contain key")
		}
		if !strings.Contains(got, "Value1") {
			t.Error("InfoBox should contain value")
		}
	})
}

func TestAccent(t *testing.T) {
	got := Accent("test")
	want := log.Amber("test")
	if got != want {
		t.Errorf("Accent(\"test\") = %q, want %q", got, want)
	}
}

func TestSpinner(t *testing.T) {
	withCapture(t, func(buf *bytes.Buffer) {
		stop := Spinner("loading")
		// Let the goroutine tick at least once (ticker is 80ms)
		time.Sleep(150 * time.Millisecond)
		stop("done")
		got := buf.String()
		if !strings.Contains(got, "done") {
			t.Errorf("Spinner stop with result should print success, got %q", got)
		}
	})
}

func TestSpinnerEmptyResult(t *testing.T) {
	withCapture(t, func(buf *bytes.Buffer) {
		stop := Spinner("working")
		stop("")
		// Should not print a success line when result is empty
	})
}
