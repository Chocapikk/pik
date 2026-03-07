package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Chocapikk/pik/sdk"
	"github.com/Chocapikk/pik/pkg/output"
)

// resolveModule looks up a module by name and exits on failure.
func resolveModule(name string) sdk.Exploit {
	mod := sdk.Get(name)
	if mod == nil {
		output.Error("module %q not found", name)
		os.Exit(1)
	}
	return mod
}

// parseOpts parses -s KEY=VALUE flags into Params.
func parseOpts(sets []string, params sdk.Params) error {
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
func newParams(values map[string]string) sdk.Params {
	return sdk.NewParams(context.Background(), values)
}

// defaultParams creates Params pre-filled with module option defaults.
func defaultParams(mod sdk.Exploit) sdk.Params {
	values := make(map[string]string)
	for _, opt := range mod.Options() {
		if opt.Default != "" {
			values[strings.ToUpper(opt.Name)] = opt.Default
		}
	}
	return newParams(values)
}


// protoFromPath extracts the protocol name from a module path.
// e.g. "exploit/tcp/multi/erlang_ssh_rce" -> "tcp"
//      "exploit/http/linux/opendcim"      -> "http"
func protoFromPath(modulePath string) string {
	for _, seg := range strings.Split(modulePath, "/") {
		switch seg {
		case "http", "tcp":
			return seg
		}
	}
	return "http"
}

// hasParser checks if a module declares a parser dependency in Info().Parsers.
func hasParser(mod sdk.Exploit, name sdk.Parser) bool {
	for _, p := range mod.Info().Parsers {
		if p == name {
			return true
		}
	}
	return false
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

// buildTargetHelp returns a long description listing available targets and options.
func buildTargetHelp(mod sdk.Exploit) string {
	lines := []string{mod.Info().Title()}

	targets := mod.Info().Targets
	if len(targets) > 0 {
		lines = append(lines, "", "Targets:")
		for i, t := range targets {
			name := t.Name
			if name == "" {
				name = t.Platform
			}
			arches := strings.Join(t.Arches, ", ")
			if arches == "" {
				arches = "cmd"
			}
			lines = append(lines, fmt.Sprintf("  %d  %s (%s) [%s]", i, name, t.Type, arches))
		}
	}

	opts := sdk.ResolveOptions(mod)
	if len(opts) > 0 {
		lines = append(lines, "", "Options (-s KEY=VALUE):")
		for _, opt := range opts {
			req := ""
			if opt.Required {
				req = " (required)"
			}
			def := ""
			if opt.Default != "" {
				def = fmt.Sprintf(" [%s]", opt.Default)
			}
			lines = append(lines, fmt.Sprintf("  %-16s %s%s%s", opt.Name, opt.Desc, def, req))
		}
	}

	return strings.Join(lines, "\n")
}

// buildTargetFlag returns a description for the --exploit-target flag.
func buildTargetFlag(mod sdk.Exploit) string {
	targets := mod.Info().Targets
	names := make([]string, len(targets))
	for i, t := range targets {
		name := t.Name
		if name == "" {
			name = t.Type
		}
		names[i] = fmt.Sprintf("%d=%s", i, name)
	}
	return "Exploit target [" + strings.Join(names, ", ") + "]"
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
