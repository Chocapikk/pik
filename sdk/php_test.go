package sdk

import "testing"

func TestPHPLateBinding(t *testing.T) {
	// Save and restore
	oldRS := phpReverseShellFn
	oldSys := phpSystemFn
	oldES := phpEvalShellFn
	oldESys := phpEvalSystemFn
	defer func() {
		phpReverseShellFn = oldRS
		phpSystemFn = oldSys
		phpEvalShellFn = oldES
		phpEvalSystemFn = oldESys
	}()

	SetPHPReverseShell(func(host string, port int) string {
		return Sprintf("reverse:%s:%d", host, port)
	})
	SetPHPSystem(func(cmd string) string {
		return "system:" + cmd
	})
	SetPHPEvalShell(func(host string, port int) string {
		return Sprintf("evalshell:%s:%d", host, port)
	})
	SetPHPEvalSystem(func(cmd string) string {
		return "evalsys:" + cmd
	})

	ctx := NewContext(map[string]string{"LHOST": "10.0.0.1", "LPORT": "4444"}, "")

	if got := PHPReverseShell(ctx); got != "reverse:10.0.0.1:4444" {
		t.Errorf("PHPReverseShell = %q", got)
	}
	if got := PHPSystem("id"); got != "system:id" {
		t.Errorf("PHPSystem = %q", got)
	}
	if got := PHPEvalShell(ctx); got != "evalshell:10.0.0.1:4444" {
		t.Errorf("PHPEvalShell = %q", got)
	}
	if got := PHPEvalSystem("whoami"); got != "evalsys:whoami" {
		t.Errorf("PHPEvalSystem = %q", got)
	}
}
