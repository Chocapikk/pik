package payload

import (
	"strings"
	"testing"
)

func TestPHPWebShellDefault(t *testing.T) {
	got := PHPWebShell("")
	if !strings.Contains(got, `"cmd"`) {
		t.Errorf("default param should be cmd: %q", got)
	}
	if !strings.Contains(got, "<?php") {
		t.Error("missing php tag")
	}
}

func TestPHPWebShellCustomParam(t *testing.T) {
	got := PHPWebShell("x")
	if !strings.Contains(got, `"x"`) {
		t.Errorf("custom param not found: %q", got)
	}
}

func TestPHPWebShellPassthru(t *testing.T) {
	got := PHPWebShellPassthru("")
	if !strings.Contains(got, "passthru") {
		t.Errorf("missing passthru: %q", got)
	}
}

func TestPHPWebShellPost(t *testing.T) {
	got := PHPWebShellPost("")
	if !strings.Contains(got, "$_POST") {
		t.Errorf("missing POST: %q", got)
	}
}

func TestPHPWebShellStealth(t *testing.T) {
	got := PHPWebShellStealth("")
	if !strings.Contains(got, "X_CMD") {
		t.Errorf("missing header key: %q", got)
	}
}

func TestPHPWebShellStealthCustomHeader(t *testing.T) {
	got := PHPWebShellStealth("X-Custom-Header")
	if !strings.Contains(got, "X_CUSTOM_HEADER") {
		t.Errorf("header not converted: %q", got)
	}
}

func TestPHPEval(t *testing.T) {
	got := PHPEval("")
	if !strings.Contains(got, "eval") || !strings.Contains(got, `"code"`) {
		t.Errorf("PHPEval = %q", got)
	}
}

func TestJSPWebShell(t *testing.T) {
	got := JSPWebShell("")
	if !strings.Contains(got, "Runtime") || !strings.Contains(got, `"cmd"`) {
		t.Errorf("JSPWebShell = %q", got)
	}
}

func TestASPWebShell(t *testing.T) {
	got := ASPWebShell("")
	if !strings.Contains(got, "WSCRIPT") {
		t.Errorf("ASPWebShell = %q", got)
	}
}

func TestASPXWebShell(t *testing.T) {
	got := ASPXWebShell("")
	if !strings.Contains(got, "cmd.exe") || !strings.Contains(got, `"cmd"`) {
		t.Errorf("ASPXWebShell = %q", got)
	}
}
