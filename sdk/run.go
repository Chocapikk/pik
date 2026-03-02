package sdk

// runFn is set by pkg/cli via SetRunner during init.
var runFn func(Exploit)

// SetRunner registers the standalone runner function.
// Called from pkg/cli's init() to break the import cycle.
func SetRunner(fn func(Exploit)) { runFn = fn }

// Run starts a standalone single-module CLI.
// Requires importing _ "github.com/Chocapikk/pik/pkg/cli" to register the runner.
func Run(mod Exploit) {
	if runFn == nil {
		panic("sdk.Run: import _ \"github.com/Chocapikk/pik/pkg/cli\" to register the runner")
	}
	runFn(mod)
}
