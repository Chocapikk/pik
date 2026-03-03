package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/Chocapikk/pik/pkg/log"
	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/pkg/runner"
	"github.com/Chocapikk/pik/sdk"
)

func requireLab() sdk.LabManager {
	mgr := sdk.GetLabManager()
	if mgr == nil {
		output.Error("lab support not available (import _ \"github.com/Chocapikk/pik/pkg/lab\")")
	}
	return mgr
}

func labCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lab",
		Short: "Manage lab environments",
	}
	cmd.AddCommand(labStartCmd(), labStopCmd(), labStatusCmd(), labRunCmd())
	return cmd
}

func labStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start [module]",
		Short: "Start a module's lab environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			mgr := requireLab()
			if mgr == nil {
				return fmt.Errorf("lab not available")
			}
			mod := resolveModule(args[0])
			info := mod.Info()
			if len(info.Lab.Services) == 0 {
				return fmt.Errorf("module %q has no lab defined", args[0])
			}
			labName := filepath.Base(sdk.NameOf(mod))
			ctx := context.Background()
			return mgr.Start(ctx, labName, info.Lab.Services)
		},
	}
}

func labStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop [module]",
		Short: "Stop a module's lab environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			mgr := requireLab()
			if mgr == nil {
				return fmt.Errorf("lab not available")
			}
			mod := resolveModule(args[0])
			labName := filepath.Base(sdk.NameOf(mod))
			ctx := context.Background()
			if err := mgr.Stop(ctx, labName); err != nil {
				return err
			}
			output.Success("Lab %s stopped", labName)
			return nil
		},
	}
}

func labRunCmd() *cobra.Command {
	var sets []string

	cmd := &cobra.Command{
		Use:   "run [module]",
		Short: "Start lab, wait for ready, and exploit",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			mgr := requireLab()
			if mgr == nil {
				return fmt.Errorf("lab not available")
			}
			mod := resolveModule(args[0])
			info := mod.Info()
			if len(info.Lab.Services) == 0 {
				return fmt.Errorf("module %q has no lab defined", args[0])
			}

			ctx := context.Background()
			labName := filepath.Base(sdk.NameOf(mod))

			// Start lab if not already running.
			if !mgr.IsRunning(ctx, labName) {
				if err := mgr.Start(ctx, labName, info.Lab.Services); err != nil {
					return err
				}
			} else {
				output.Success("Lab %s already running", labName)
			}

			// Derive target from port bindings.
			target := mgr.Target(ctx, labName)

			// Phase 1: wait for TCP port to open.
			output.Status("Waiting for %s", target)
			if err := mgr.WaitReady(ctx, target, 120*time.Second); err != nil {
				return fmt.Errorf("lab not ready: %w", err)
			}

			// Phase 2: if module has Check(), retry it until the app
			// layer is truly ready (not just TCP).
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

			// Build params and run exploit.
			params := defaultParams(mod)
			params.Set("TARGET", target)
			if err := parseOpts(sets, params); err != nil {
				return err
			}
			// Auto-detect LHOST from Docker gateway if not set.
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

func labStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "List running lab environments",
		RunE: func(*cobra.Command, []string) error {
			mgr := requireLab()
			if mgr == nil {
				return fmt.Errorf("lab not available")
			}
			ctx := context.Background()
			labs, err := mgr.Status(ctx)
			if err != nil {
				return err
			}
			if len(labs) == 0 {
				output.Warning("No labs running")
				return nil
			}
			output.Println()
			for _, l := range labs {
				output.Print("  %s\n", log.Cyan(l.Name))
				for _, svc := range l.Services {
					state := log.Green(svc.State)
					if svc.State != "running" {
						state = log.Yellow(svc.State)
					}
					ports := ""
					if svc.Ports != "" {
						ports = " " + log.Gray(svc.Ports)
					}
					output.Print("    %s  %s  %s%s\n",
						log.Pad(svc.Name, 20),
						log.Pad(state, 12),
						svc.Image,
						ports,
					)
				}
			}
			output.Println()
			return nil
		},
	}
}
