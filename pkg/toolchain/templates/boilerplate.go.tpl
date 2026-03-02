package main

import "github.com/Chocapikk/pik/sdk"

type {{.StructName}} struct{ sdk.Pik }

func (m *{{.StructName}}) Info() sdk.Info {
	return sdk.Info{
		Description: "{{.Name}} exploit",
		Detail:      sdk.Dedent(` + "`" + `
			TODO: Describe the vulnerability and exploitation chain.
		` + "`" + `),
		Authors:     []string{"TODO"},
		Reliability: sdk.Typical,
		References: []sdk.Reference{
			// sdk.CVE("2026-XXXXX"),
		},
		Queries: []sdk.Query{
			// sdk.Shodan(` + "`" + `http.title:"{{.Name}}"` + "`" + `),
			// sdk.FOFA(` + "`" + `title="{{.Name}}"` + "`" + `),
		},
		Targets: []sdk.Target{sdk.TargetLinux("amd64")},
	}
}

func (m *{{.StructName}}) Check(run *sdk.Context) (sdk.CheckResult, error) {
	// TODO: Implement vulnerability check
	return sdk.CheckResult{Code: sdk.CheckUnknown, Reason: "not implemented"}, nil
}

func (m *{{.StructName}}) Exploit(run *sdk.Context) error {
	// TODO: Implement exploitation
	return sdk.Errorf("not implemented")
}

func main() {
	sdk.Run(&{{.StructName}}{})
}
