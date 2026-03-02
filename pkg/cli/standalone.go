package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Chocapikk/pik/pkg/console"
	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/pkg/runner"
	"github.com/Chocapikk/pik/sdk"
)

func init() {
	sdk.SetRunner(RunStandaloneWith)
}

// RunStandaloneWith starts a single-module CLI for a directly-provided exploit.
func RunStandaloneWith(mod sdk.Exploit, runOpts sdk.RunOptions) {
	name := sdk.NameOf(mod)
	if name == "unknown" {
		name = mod.Info().Description
	}

	var target, file, outputFile string
	var sets []string
	var threads, targetIdx int
	var checkOnly, jsonOutput bool

	cmd := &cobra.Command{
		Use:           name,
		Short:         mod.Info().Description,
		Long:          buildTargetHelp(mod),
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			v, _ := cmd.Flags().GetBool("verbose")
			d, _ := cmd.Flags().GetBool("debug")
			if d {
				output.EnableDebug()
			} else if v {
				output.SetVerbose(true)
			}
			output.Banner()
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx := context.Background()
			params := defaultParams(mod)
			params.Set("TARGET", target)
			params.Set("TARGET_INDEX", fmt.Sprintf("%d", targetIdx))
			if err := parseOpts(sets, params); err != nil {
				return err
			}

			if file != "" {
				scan := &runner.Scanner{Module: mod, Targets: readTargets(file), Threads: threads, BaseParams: params, OutputFile: outputFile, JSONOutput: jsonOutput}
				scan.Run(ctx)
				return nil
			}
			if target == "" {
				return fmt.Errorf("specify -t <target> or -f <file>")
			}
			checkOnlyMode := checkOnly || params.Lhost() == ""
			return runner.RunSingle(ctx, mod, params, runner.RunOpts{CheckOnly: checkOnlyMode})
		},
	}

	cmd.Flags().StringVarP(&target, "target", "t", "", "Target URL/IP")
	cmd.Flags().StringVarP(&file, "file", "f", "", "File with targets")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file")
	cmd.Flags().StringArrayVarP(&sets, "set", "s", nil, "Set option (KEY=VALUE)")
	cmd.Flags().IntVar(&threads, "threads", 10, "Threads")
	cmd.Flags().BoolVar(&checkOnly, "check", false, "Check only")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output (with -o)")
	cmd.Flags().BoolP("verbose", "v", false, "Verbose")
	cmd.Flags().Bool("debug", false, "Debug")
	if len(mod.Info().Targets) > 1 {
		cmd.Flags().IntVar(&targetIdx, "exploit-target", 0, buildTargetFlag(mod))
	}

	if runOpts.Console {
		cmd.AddCommand(&cobra.Command{
			Use:              "console",
			Short:            "Interactive console with module pre-loaded",
			PersistentPreRun: func(_ *cobra.Command, _ []string) {},
			RunE: func(_ *cobra.Command, _ []string) error {
				return console.RunWith(mod)
			},
		})
	}

	if err := cmd.Execute(); err != nil {
		output.Error("%v", err)
		os.Exit(1)
	}
}
