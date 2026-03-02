package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/pkg/toolchain"
	"github.com/Chocapikk/pik/sdk"
)

const pikModule = "github.com/Chocapikk/pik"

func buildCmd() *cobra.Command {
	var outputPath, targetOS, targetArch string

	cmd := &cobra.Command{
		Use:   "build [module]",
		Short: "Build a standalone exploit binary",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return buildExploit(args[0], outputPath, targetOS, targetArch)
		},
	}

	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output path")
	cmd.Flags().StringVar(&targetOS, "os", "", "Target OS (linux, windows, darwin)")
	cmd.Flags().StringVar(&targetArch, "arch", "", "Target arch (amd64, arm64, 386)")
	return cmd
}

func buildExploit(name, outputPath, targetOS, targetArch string) error {
	mod := resolveModule(name)
	fullName := sdk.NameOf(mod)

	if outputPath == "" {
		outputPath = filepath.Base(fullName)
	}
	absOutput, _ := filepath.Abs(outputPath)

	importPath := pikModule + "/modules/" + filepath.Dir(fullName)

	// Use local sources if inside the pik repo, otherwise fetch from proxy.
	scaffoldOpts := toolchain.ScaffoldOpts{
		ImportPath: importPath,
		Version:    Version,
	}
	if modRoot, err := findModRoot(); err == nil {
		if goMod, err := readGoModModule(modRoot); err == nil && goMod == pikModule {
			scaffoldOpts.ModRoot = modRoot
			scaffoldOpts.Version = "v0.0.0" // replace overrides this
		}
	}

	output.Status("Building standalone binary for %s", fullName)

	srcDir, cleanup, err := toolchain.Scaffold(scaffoldOpts)
	if err != nil {
		return err
	}
	defer cleanup()

	if err := toolchain.Build(toolchain.BuildOpts{
		Dir: srcDir, Output: absOutput, OS: targetOS, Arch: targetArch,
	}); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	output.Success("Built %s (%s)", absOutput, humanSize(absOutput))
	return nil
}

func humanSize(path string) string {
	stat, err := os.Stat(path)
	if err != nil {
		return "unknown"
	}
	return fmt.Sprintf("%.1f MB", float64(stat.Size())/(1024*1024))
}

func findModRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("go.mod not found")
}
