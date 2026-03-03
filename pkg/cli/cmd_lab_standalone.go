package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/pkg/runner"
	"github.com/Chocapikk/pik/sdk"
)

// standaloneLabCmd builds lab subcommands for a standalone binary.
// No module arg needed - the module is already known.
func standaloneLabCmd(mod sdk.Exploit) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lab",
		Short: "Manage lab environment",
	}
	cmd.AddCommand(
		standaloneLabStart(mod),
		standaloneLabStop(mod),
		standaloneLabRun(mod),
		labStatusCmd(), // status is shared, no module arg
	)
	return cmd
}

func standaloneLabStart(mod sdk.Exploit) *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start lab environment",
		RunE: func(*cobra.Command, []string) error {
			mgr := requireLab()
			if mgr == nil {
				return fmt.Errorf("lab not available")
			}
			info := mod.Info()
			if len(info.Lab.Services) == 0 {
				return fmt.Errorf("module has no lab defined")
			}
			return mgr.Start(context.Background(), labName(mod), info.Lab.Services)
		},
	}
}

func standaloneLabStop(mod sdk.Exploit) *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop lab environment",
		RunE: func(*cobra.Command, []string) error {
			mgr := requireLab()
			if mgr == nil {
				return fmt.Errorf("lab not available")
			}
			ctx := context.Background()
			if err := mgr.Stop(ctx, labName(mod)); err != nil {
				return err
			}
			output.Success("Lab %s stopped", labName(mod))
			return nil
		},
	}
}

func standaloneLabRun(mod sdk.Exploit) *cobra.Command {
	var sets []string

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Start lab, wait for ready, and exploit",
		RunE: func(*cobra.Command, []string) error {
			mgr := requireLab()
			if mgr == nil {
				return fmt.Errorf("lab not available")
			}
			info := mod.Info()
			if len(info.Lab.Services) == 0 {
				return fmt.Errorf("module has no lab defined")
			}

			ctx := context.Background()
			name := labName(mod)

			if !mgr.IsRunning(ctx, name) {
				if err := mgr.Start(ctx, name, info.Lab.Services); err != nil {
					return err
				}
			} else {
				output.Success("Lab %s already running", name)
			}

			target := mgr.Target(ctx, name)
			output.Status("Waiting for %s", target)
			if err := mgr.WaitReady(ctx, target, 120*time.Second); err != nil {
				return fmt.Errorf("lab not ready: %w", err)
			}

			if checker, ok := mod.(sdk.Checker); ok {
				params := defaultParams(mod)
				params.Set("TARGET", target)
				if err := parseOpts(sets, params); err != nil {
					return err
				}
				output.Status("Probing application readiness")
				if err := mgr.WaitProbe(ctx, 120*time.Second, func() error {
					_, err := checker.Check(runner.BuildContext(params, ""))
					return err
				}); err != nil {
					return fmt.Errorf("application not ready: %w", err)
				}
			}
			output.Success("Lab ready at %s", target)

			params := defaultParams(mod)
			params.Set("TARGET", target)
			if err := parseOpts(sets, params); err != nil {
				return err
			}
			if params.Lhost() == "" {
				gw := mgr.DockerGateway()
				if gw == "" {
					return fmt.Errorf("specify -s LHOST=<ip>")
				}
				params.Set("LHOST", gw)
				output.Success("LHOST => %s (docker gateway)", gw)
			}
			return runner.RunSingle(ctx, mod, params, runner.RunOpts{})
		},
	}

	cmd.Flags().StringArrayVarP(&sets, "set", "s", nil, "Set option (KEY=VALUE)")
	return cmd
}

func labName(mod sdk.Exploit) string {
	return filepath.Base(sdk.NameOf(mod))
}
