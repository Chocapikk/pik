package payload

import (
	"strings"
	"testing"
)

func TestReverseShellsContainHostPort(t *testing.T) {
	lhost := "10.0.0.1"
	lport := 4444

	generators := []struct {
		name string
		fn   func(string, int) string
	}{
		{"Bash", Bash},
		{"BashMin", BashMin},
		{"BashFD", BashFD},
		{"BashReadLine", BashReadLine},
		{"Python", Python},
		{"PythonMin", PythonMin},
		{"PythonPTY", PythonPTY},
		{"Perl", Perl},
		{"Ruby", Ruby},
		{"PHP", PHP},
		{"PHPMin", PHPMin},
		{"PHPExec", PHPExec},
		{"Netcat", Netcat},
		{"NetcatMkfifo", NetcatMkfifo},
		{"NetcatOpenbsd", NetcatOpenbsd},
		{"PowerShell", PowerShell},
		{"PowerShellConPTY", PowerShellConPTY},
		{"Java", Java},
		{"Socat", Socat},
		{"Lua", Lua},
		{"NodeJS", NodeJS},
		{"Awk", Awk},
		// TLS reverse shells
		{"BashTLS", BashTLS},
		{"PythonTLS", PythonTLS},
		{"NcatTLS", NcatTLS},
		{"SocatTLS", SocatTLS},
		// HTTP reverse shells
		{"CurlHTTP", CurlHTTP},
		{"WgetHTTP", WgetHTTP},
		{"PHPHTTP", PHPHTTP},
		{"PythonHTTP", PythonHTTP},
	}

	for _, gen := range generators {
		t.Run(gen.name, func(t *testing.T) {
			result := gen.fn(lhost, lport)
			if !strings.Contains(result, lhost) {
				t.Errorf("%s does not contain lhost %q", gen.name, lhost)
			}
			if !strings.Contains(result, "4444") {
				t.Errorf("%s does not contain lport %d", gen.name, lport)
			}
			if result == "" {
				t.Errorf("%s returned empty string", gen.name)
			}
		})
	}
}

func TestReverseShellsNonEmpty(t *testing.T) {
	if got := Bash("127.0.0.1", 1234); got == "" {
		t.Error("Bash returned empty")
	}
	if got := Python("127.0.0.1", 1234); got == "" {
		t.Error("Python returned empty")
	}
}
