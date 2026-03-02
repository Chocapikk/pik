package sdk

// runFn is set by pkg/cli via SetRunner during init.
var runFn func(Exploit, RunOptions)

// RunOptions configures standalone binary behavior.
type RunOptions struct {
	Console bool // Add interactive console subcommand.
}

// RunOption is a functional option for Run.
type RunOption func(*RunOptions)

// WithConsole enables the interactive console subcommand in standalone binaries.
func WithConsole() RunOption {
	return func(o *RunOptions) { o.Console = true }
}

// SetRunner registers the standalone runner function.
// Called from pkg/cli's init() to break the import cycle.
func SetRunner(fn func(Exploit, RunOptions)) { runFn = fn }

// Run starts a standalone single-module CLI.
// Requires importing _ "github.com/Chocapikk/pik/pkg/cli" to register the runner.
func Run(mod Exploit, opts ...RunOption) {
	if runFn == nil {
		panic("sdk.Run: import _ \"github.com/Chocapikk/pik/pkg/cli\" to register the runner")
	}
	var options RunOptions
	for _, opt := range opts {
		opt(&options)
	}
	runFn(mod, options)
}
