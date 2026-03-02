package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/Chocapikk/pik/pkg/core"
	"github.com/Chocapikk/pik/pkg/output"
)

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available modules",
		Run: func(*cobra.Command, []string) {
			modules := core.List()
			if len(modules) == 0 {
				output.Warning("No modules loaded")
				return
			}
			for _, mod := range modules {
				info := mod.Info()
				cves := strings.Join(info.CVEs(), ", ")
				if cves == "" {
					cves = "-"
				}
				output.Print("  %-20s %-12s %-40s [%s]\n", core.NameOf(mod), info.Reliability, info.Description, cves)
			}
		},
	}
}

func rankCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rank",
		Short: "Show contributor leaderboard",
		Run: func(*cobra.Command, []string) {
			rankings := core.Rankings()
			if len(rankings) == 0 {
				output.Warning("No modules loaded")
				return
			}
			for i, rank := range rankings {
				output.Print("  #%-3d %-20s %d modules, %d CVEs\n", i+1, rank.Name, rank.Modules, rank.CVEs)
			}
		},
	}
}
