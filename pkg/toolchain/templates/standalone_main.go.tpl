package main

import (
	"github.com/Chocapikk/pik/sdk"
	_ "github.com/Chocapikk/pik/pkg/cli"
	_ "github.com/Chocapikk/pik/pkg/lab"
	_ "github.com/Chocapikk/pik/pkg/protocol/{{.Proto}}"
{{- if .XMLUtil}}
	_ "github.com/Chocapikk/pik/pkg/xmlutil"
{{- end}}
{{- if .Faker}}
	_ "github.com/Chocapikk/pik/pkg/fake"
{{- end}}
{{- if .HTTPServer}}
	_ "github.com/Chocapikk/pik/pkg/httpsrv"
{{- end}}
	_ "{{.ImportPath}}"
)

func main() {
	mod := sdk.Get("{{.ModuleName}}")
	if mod == nil {
		panic("module {{.ModuleName}} not found")
	}
	sdk.Run(mod, sdk.WithConsole(), sdk.WithLab())
}
