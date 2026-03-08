package sdk

// Late-binding hooks for PHP payload generation.
// Implementations live in pkg/payload, registered via init().
var (
	phpReverseShellFn func(string, int) string
	phpSystemFn       func(string) string
)

// SetPHPReverseShell registers the PHP reverse shell drop implementation.
func SetPHPReverseShell(fn func(string, int) string) { phpReverseShellFn = fn }

// SetPHPSystem registers the PHP system exec drop implementation.
func SetPHPSystem(fn func(string) string) { phpSystemFn = fn }

// PHPReverseShell returns a self-deleting PHP reverse shell for file drop.
// Reads LHOST/LPORT from the context automatically.
func PHPReverseShell(run *Context) string {
	p := run.Params()
	return phpReverseShellFn(p.Lhost(), p.Lport())
}

// PHPSystem returns a self-deleting PHP system exec for file drop.
func PHPSystem(cmd string) string {
	return phpSystemFn(cmd)
}
