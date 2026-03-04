package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/Chocapikk/pik/pkg/log"
	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/sdk"
)

func infoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info [module]",
		Short: "Show module details",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			mod := resolveModule(args[0])
			info := mod.Info()

			output.Print("Name:         %s\n", sdk.NameOf(mod))
			output.Print("Description:  %s\n", info.Title())
			if info.Detail != "" {
				output.Print("\n%s\n\n", info.Detail)
			}
			output.Print("Authors:      %s\n", info.AuthorNames())
			if info.Disclosure != "" {
				output.Print("Disclosed:    %s\n", info.Disclosure)
			}
			output.Print("Reliability:  %s\n", info.Reliability)
			stance := info.Stance
			if stance == "" {
				stance = "aggressive"
			}
			output.Print("Stance:       %s\n", stance)
			if info.Privileged {
				output.Print("Privileged:   yes\n")
			}
			if len(info.Notes.Stability) > 0 {
				output.Print("Stability:    %s\n", strings.Join(info.Notes.Stability, ", "))
			}
			if len(info.Notes.SideEffects) > 0 {
				output.Print("Side effects: %s\n", strings.Join(info.Notes.SideEffects, ", "))
			}
			output.Print("CVEs:         %s\n", strings.Join(info.CVEs(), ", "))
			if len(info.Refs) > 0 {
				urls := make([]string, len(info.Refs))
				for i, ref := range info.Refs {
					urls[i] = ref.URL()
				}
				output.Print("References:   %s\n", strings.Join(urls, "\n              "))
			}
			output.Print("Targets:      %s\n", strings.Join(info.TargetStrings(), ", "))
			if len(info.Queries) > 0 {
				output.Print("\nQueries:\n")
				for _, q := range info.Queries {
					output.Print("  %s %s\n", log.Pad(q.Engine+":", 12), q.URL())
				}
			}
		},
	}
}
