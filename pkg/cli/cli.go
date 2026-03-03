package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/Chocapikk/pik/sdk"
	"github.com/Chocapikk/pik/pkg/output"
)

// Version is set at build time via ldflags.
var Version = "dev"

// ConsoleFunc is set by cmd/pik to provide the readline console.
var ConsoleFunc func() error

// TUIFunc is set by cmd/pik to provide the TUI dashboard.
var TUIFunc func() error

// Run starts the full framework CLI.
func Run() {
	output.BannerVersion = Version
	root := &cobra.Command{
		Use:     "pik",
		Short:   "Pik exploitation framework",
		Version: Version,
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			v, _ := cmd.Flags().GetBool("verbose")
			d, _ := cmd.Flags().GetBool("debug")
			if d {
				output.EnableDebug()
			} else if v {
				output.SetVerbose(true)
			}
			if cmd.Name() != "pik" && cmd.Name() != "console" {
				output.Banner()
			}
		},
		Run: func(*cobra.Command, []string) { runConsole() },
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().BoolP("verbose", "v", false, "Verbose output")
	root.PersistentFlags().Bool("debug", false, "Debug output (HTTP traces)")

	root.AddCommand(
		checkCmd(),
		runCmd(),
		infoCmd(),
		listCmd(),
		rankCmd(),
		buildCmd(),
		newCmd(),
		consoleCmd(),
		tuiCmd(),
		updateCmd(),
		labCmd(),
	)

	if err := root.Execute(); err != nil {
		output.Error("%v", err)
		os.Exit(1)
	}
}

// RunStandalone starts a single-module CLI by name.
func RunStandalone(name string) {
	mod := sdk.Get(name)
	if mod == nil {
		output.Error("module %q not found", name)
		os.Exit(1)
	}
	RunStandaloneWith(mod, sdk.RunOptions{})
}

func runConsole() {
	if ConsoleFunc != nil {
		_ = ConsoleFunc()
		return
	}
	output.Error("console not available in this build")
	os.Exit(1)
}

func runTUI() {
	if TUIFunc != nil {
		_ = TUIFunc()
		return
	}
	output.Error("TUI not available in this build")
	os.Exit(1)
}

func consoleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "console",
		Short: "Start the interactive readline console",
		Run:   func(*cobra.Command, []string) { runConsole() },
	}
}

func tuiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Start the TUI dashboard",
		Run:   func(*cobra.Command, []string) { runTUI() },
	}
}
