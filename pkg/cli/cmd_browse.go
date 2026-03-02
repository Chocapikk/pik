package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Chocapikk/pik/pkg/log"
	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/sdk"
)

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available modules",
		Run: func(*cobra.Command, []string) {
			modules := sdk.List()
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
				output.Print("  %s %s %s [%s]\n",
					log.Pad(sdk.NameOf(mod), 20),
					log.Pad(info.Reliability.String(), 12),
					log.Pad(info.Title(), 40),
					cves,
				)
			}
		},
	}
}

func rankCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rank",
		Short: "Show contributor leaderboard",
		Run: func(*cobra.Command, []string) {
			rankings := sdk.Rankings()
			if len(rankings) == 0 {
				output.Warning("No modules loaded")
				return
			}
			for i, rank := range rankings {
				output.Print("  #%s %s %d modules, %d CVEs\n",
					log.Pad(fmt.Sprintf("%d", i+1), 3),
					log.Pad(rank.Name, 20),
					rank.Modules, rank.CVEs,
				)
			}
		},
	}
}
