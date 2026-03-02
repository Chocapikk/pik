package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Chocapikk/pik/pkg/core"
	"github.com/Chocapikk/pik/pkg/output"
)

// resolveModule looks up a module by name and exits on failure.
func resolveModule(name string) core.Exploit {
	mod := core.Get(name)
	if mod == nil {
		output.Error("module %q not found", name)
		os.Exit(1)
	}
	return mod
}

// parseOpts parses -s KEY=VALUE flags into Params.
func parseOpts(sets []string, params core.Params) error {
	for _, pair := range sets {
		key, val, ok := strings.Cut(pair, "=")
		if !ok {
			return fmt.Errorf("invalid option %q (expected KEY=VALUE)", pair)
		}
		params.Set(key, val)
	}
	return nil
}

// newParams creates Params from a values map.
func newParams(values map[string]string) core.Params {
	return core.NewParams(context.Background(), values)
}

// defaultParams creates Params pre-filled with module option defaults.
func defaultParams(mod core.Exploit) core.Params {
	values := make(map[string]string)
	for _, opt := range mod.Options() {
		if opt.Default != "" {
			values[strings.ToUpper(opt.Name)] = opt.Default
		}
	}
	return newParams(values)
}

// flagParams creates Params from flag pointers and a target.
func flagParams(flagVals map[string]*string, target string) core.Params {
	values := make(map[string]string)
	values["TARGET"] = target
	for name, val := range flagVals {
		if *val != "" {
			values[strings.ToUpper(name)] = *val
		}
	}
	return newParams(values)
}

// readGoModModule reads the module path from a go.mod file.
func readGoModModule(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("read go.mod: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}
	return "", fmt.Errorf("no module directive in go.mod")
}

// readTargets reads a file of targets (one per line).
func readTargets(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		output.Error("failed to open %s: %v", path, err)
		return nil
	}
	defer f.Close()
	var targets []string
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			targets = append(targets, line)
		}
	}
	return targets
}
