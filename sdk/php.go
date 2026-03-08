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
func PHPReverseShell(lhost string, lport int) string {
	return phpReverseShellFn(lhost, lport)
}

// PHPSystem returns a self-deleting PHP system exec for file drop.
func PHPSystem(cmd string) string {
	return phpSystemFn(cmd)
}
