package main

import (
	"os"
	"path/filepath"

	_ "github.com/Chocapikk/pik/modules"
	"github.com/Chocapikk/pik/pkg/cli"
	"github.com/Chocapikk/pik/pkg/console"
	_ "github.com/Chocapikk/pik/pkg/lab" // register lab manager
	"github.com/Chocapikk/pik/sdk"
)

func init() {
	cli.ConsoleFunc = console.Run
}

func main() {
	name := filepath.Base(os.Args[0])
	if name != "pik" {
		if sdk.Get(name) != nil {
			cli.RunStandalone(name)
			return
		}
	}
	cli.Run()
}
