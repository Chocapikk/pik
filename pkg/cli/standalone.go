package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Chocapikk/pik/sdk"
	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/pkg/runner"
)

func init() {
	sdk.SetRunner(RunStandaloneWith)
}

// RunStandaloneWith starts a single-module CLI for a directly-provided exploit.
func RunStandaloneWith(mod sdk.Exploit) {
	name := sdk.NameOf(mod)
	if name == "unknown" {
		name = mod.Info().Description
	}

	var target, file, outputFile string
	var threads int
	var checkOnly bool

	opts := sdk.ResolveOptions(mod)
	defaults := make(map[string]string, len(opts))
	flagVals := make(map[string]*string, len(opts))
	for _, opt := range opts {
		val := new(string)
		*val = opt.Default
		flagVals[opt.Name] = val
		defaults[opt.Name] = opt.Default
	}

	var targetIdx int

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
			params := flagParams(flagVals, defaults, target)
			params.Set("TARGET_INDEX", fmt.Sprintf("%d", targetIdx))

			if file != "" {
				scan := &runner.Scanner{Module: mod, Targets: readTargets(file), Threads: threads, BaseParams: params, OutputFile: outputFile}
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
	cmd.Flags().IntVar(&threads, "threads", 10, "Threads")
	cmd.Flags().BoolVar(&checkOnly, "check", false, "Check only")
	cmd.Flags().BoolP("verbose", "v", false, "Verbose")
	cmd.Flags().Bool("debug", false, "Debug")
	if len(mod.Info().Targets) > 1 {
		cmd.Flags().IntVar(&targetIdx, "exploit-target", 0, buildTargetFlag(mod))
	}

	for _, opt := range sdk.ResolveOptions(mod) {
		flagName := strings.ToLower(strings.ReplaceAll(opt.Name, "_", "-"))
		if cmd.Flags().Lookup(flagName) != nil {
			continue
		}
		cmd.Flags().StringVar(flagVals[opt.Name], flagName, opt.Default, opt.Desc)
	}

	if err := cmd.Execute(); err != nil {
		output.Error("%v", err)
		os.Exit(1)
	}
}
