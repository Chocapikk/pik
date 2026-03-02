package sdk

import (
	"github.com/Chocapikk/pik/pkg/cli"
	"github.com/Chocapikk/pik/pkg/core"
)

// Run starts a standalone single-module CLI.
func Run(mod core.Exploit) {
	cli.RunStandaloneWith(mod)
}
