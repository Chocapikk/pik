package payload

import (
	"strings"
	"testing"
)

func TestPHPReverseShellDrop(t *testing.T) {
	got := PHPReverseShellDrop("10.0.0.1", 4444)
	if got == "" {
		t.Fatal("PHPReverseShellDrop returned empty string")
	}
	if !strings.Contains(got, "<?php") {
		t.Errorf("PHPReverseShellDrop should contain '<?php', got %q", got)
	}
}

func TestPHPSystemDrop(t *testing.T) {
	got := PHPSystemDrop("id")
	if got == "" {
		t.Fatal("PHPSystemDrop returned empty string")
	}
	if !strings.Contains(got, "<?php") {
		t.Errorf("PHPSystemDrop should contain '<?php', got %q", got)
	}
}

func TestPHPEvalReverseShell(t *testing.T) {
	got := PHPEvalReverseShell("10.0.0.1", 4444)
	if got == "" {
		t.Fatal("PHPEvalReverseShell returned empty string")
	}
}

func TestPHPEvalSystemExec(t *testing.T) {
	got := PHPEvalSystemExec("id")
	if got == "" {
		t.Fatal("PHPEvalSystemExec returned empty string")
	}
}

func TestNames(t *testing.T) {
	names := Names()
	if len(names) == 0 {
		t.Fatal("Names() returned empty slice")
	}
}

func TestBase64Python(t *testing.T) {
	got := Base64Python("id")
	if !strings.Contains(got, "python3") {
		t.Errorf("Base64Python should contain 'python3', got %q", got)
	}
}

func TestBase64Perl(t *testing.T) {
	got := Base64Perl("id")
	if !strings.Contains(got, "perl") {
		t.Errorf("Base64Perl should contain 'perl', got %q", got)
	}
}

func TestBase64PowerShell(t *testing.T) {
	got := Base64PowerShell("whoami")
	if !strings.Contains(got, "powershell") {
		t.Errorf("Base64PowerShell should contain 'powershell', got %q", got)
	}
}

func TestDeliverGzipBashDec(t *testing.T) {
	got := NewCmd("id").Encode(GzipBase64Enc).Deliver(GzipBashDec).String()
	if !strings.Contains(got, "gunzip") {
		t.Errorf("GzipBashDec should contain 'gunzip', got %q", got)
	}
	if !strings.Contains(got, "base64 -d") {
		t.Errorf("GzipBashDec should contain 'base64 -d', got %q", got)
	}
}

func TestDeliverPerlDec(t *testing.T) {
	got := NewCmd("id").Encode(Base64Enc).Deliver(PerlDec).String()
	if !strings.Contains(got, "perl") {
		t.Errorf("PerlDec should contain 'perl', got %q", got)
	}
	if !strings.Contains(got, "MIME::Base64") {
		t.Errorf("PerlDec should contain 'MIME::Base64', got %q", got)
	}
}

func TestDeliverRubyDec(t *testing.T) {
	got := NewCmd("id").Encode(Base64Enc).Deliver(RubyDec).String()
	if !strings.Contains(got, "ruby") {
		t.Errorf("RubyDec should contain 'ruby', got %q", got)
	}
	if !strings.Contains(got, "Base64.decode64") {
		t.Errorf("RubyDec should contain 'Base64.decode64', got %q", got)
	}
}

func TestDeliverPHPDec(t *testing.T) {
	got := NewCmd("id").Encode(Base64Enc).Deliver(PHPDec).String()
	if !strings.Contains(got, "php") {
		t.Errorf("PHPDec should contain 'php', got %q", got)
	}
	if !strings.Contains(got, "base64_decode") {
		t.Errorf("PHPDec should contain 'base64_decode', got %q", got)
	}
}

func TestVarSplitShort(t *testing.T) {
	// Short string (under 50 chars) should use variable splitting
	got := NewCmd("id").VarSplit().String()
	if strings.Contains(got, "base64") {
		t.Errorf("VarSplit on short string should not use base64, got %q", got)
	}
	// Should contain variable assignments
	if !strings.Contains(got, "=") {
		t.Errorf("VarSplit should contain variable assignments, got %q", got)
	}
}

func TestVarSplitLong(t *testing.T) {
	// Long string (over 50 chars) should fall back to base64
	long := strings.Repeat("A", 51)
	got := NewCmd(long).VarSplit().String()
	if !strings.Contains(got, "base64") {
		t.Errorf("VarSplit on long string should fall back to base64, got %q", got)
	}
}

func TestEncodeBase64URL(t *testing.T) {
	got := NewCmd("hello world!").Encode(Base64URLEnc).String()
	if got == "" {
		t.Fatal("Base64URLEnc produced empty string")
	}
	// URL-safe base64 should not contain + or /
	if strings.ContainsAny(got, "+/") {
		t.Errorf("Base64URLEnc should not contain + or /, got %q", got)
	}
}
