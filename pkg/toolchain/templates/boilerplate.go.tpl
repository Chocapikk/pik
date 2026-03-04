package main

import (
	"github.com/Chocapikk/pik/sdk"
	_ "github.com/Chocapikk/pik/pkg/cli"
)

type {{.StructName}} struct{ sdk.Pik }

func (m *{{.StructName}}) Info() sdk.Info {
	return sdk.Info{
		Description: "{{.Name}} exploit",
		Detail: sdk.Dedent(` + "`" + `
			TODO: Describe the vulnerability and exploitation chain.
		` + "`" + `),
		Authors: sdk.Authors(sdk.NewAuthor("TODO").WithHandle("TODO").WithEmail("<TODO[at]TODO>")),
		Disclosure:  "TODO",
		Reliability: sdk.Typical,
		Notes:       sdk.SafeNotes().Logs(),

		Refs: sdk.Refs(
			// sdk.CVE("2026-XXXXX"),
			// sdk.VulnCheck("advisory-slug"),
		),
		Queries: sdk.Dorks(
			// sdk.Shodan(` + "`" + `http.title:"{{.Name}}"` + "`" + `),
			// sdk.FOFA(` + "`" + `title="{{.Name}}"` + "`" + `),
		),

		Targets: []sdk.Target{sdk.TargetLinux("amd64")},
	}
}

func (m *{{.StructName}}) Check(run *sdk.Context) (sdk.CheckResult, error) {
	// TODO: Implement vulnerability check
	return sdk.Unknown(nil)
}

func (m *{{.StructName}}) Exploit(run *sdk.Context) error {
	// TODO: Implement exploitation
	return sdk.Errorf("not implemented")
}

func main() {
	sdk.Run(&{{.StructName}}{})
}
