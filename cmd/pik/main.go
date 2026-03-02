package main

import (
	"os"
	"path/filepath"

	_ "github.com/Chocapikk/pik/modules"
	"github.com/Chocapikk/pik/pkg/cli"
	"github.com/Chocapikk/pik/pkg/console"
	"github.com/Chocapikk/pik/pkg/core"
)

func init() {
	cli.ConsoleFunc = console.Run
}

func main() {
	name := filepath.Base(os.Args[0])
	if name != "pik" {
		if core.Get(name) != nil {
			cli.RunStandalone(name)
			return
		}
	}
	cli.Run()
}
