package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Chocapikk/pik/pkg/runner"
)

func runCmd() *cobra.Command {
	var target string
	var sets []string

	cmd := &cobra.Command{
		Use:   "run [module]",
		Short: "Exploit a target",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			mod := resolveModule(args[0])
			if target == "" {
				return fmt.Errorf("specify -t <target>")
			}
			ctx := context.Background()
			params := defaultParams(mod)
			params.Set("TARGET", target)
			if err := parseOpts(sets, params); err != nil {
				return err
			}
			if params.Lhost() == "" {
				return fmt.Errorf("specify -s LHOST=<ip>")
			}
			return runner.RunSingle(ctx, mod, params, runner.RunOpts{})
		},
	}

	cmd.Flags().StringVarP(&target, "target", "t", "", "Target URL/IP")
	cmd.Flags().StringArrayVarP(&sets, "set", "s", nil, "Set option (KEY=VALUE)")
	return cmd
}
