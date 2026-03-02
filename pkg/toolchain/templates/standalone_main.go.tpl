package main

import (
	"github.com/Chocapikk/pik/sdk"
	_ "{{.ImportPath}}"
)

func main() {
	mods := sdk.List()
	if len(mods) == 0 {
		panic("no module registered")
	}
	sdk.Run(mods[0])
}
