package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Chocapikk/pik/pkg/runner"
)

func checkCmd() *cobra.Command {
	var target, file, outputFile string
	var threads int
	var sets []string

	cmd := &cobra.Command{
		Use:   "check [module]",
		Short: "Check targets for vulnerability",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			mod := resolveModule(args[0])
			ctx := context.Background()
			params := defaultParams(mod)
			params.Set("TARGET", target)
			if err := parseOpts(sets, params); err != nil {
				return err
			}

			if file != "" {
				scan := &runner.Scanner{Module: mod, Targets: readTargets(file), Threads: threads, BaseParams: params, OutputFile: outputFile}
				scan.Run(ctx)
				return nil
			}
			if target == "" {
				return fmt.Errorf("specify -t <target> or -f <file>")
			}
			return runner.RunSingle(ctx, mod, params, runner.RunOpts{CheckOnly: true})
		},
	}

	cmd.Flags().StringVarP(&target, "target", "t", "", "Target URL/IP")
	cmd.Flags().StringVarP(&file, "file", "f", "", "File with targets")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file")
	cmd.Flags().IntVar(&threads, "threads", 10, "Threads")
	cmd.Flags().StringArrayVarP(&sets, "set", "s", nil, "Set option (KEY=VALUE)")
	return cmd
}
