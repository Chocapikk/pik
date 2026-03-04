package console

import (
	"context"
	"strings"

	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/pkg/payload"
	"github.com/Chocapikk/pik/pkg/types"
	"github.com/Chocapikk/pik/sdk"
)

// Option is the shared option type.
type Option = types.Option

func (c *Console) initOptions() {
	c.options = []Option{
		{Name: "TARGET", Required: true, Desc: "Target URL/IP"},
	}

	for _, opt := range sdk.ResolveOptions(c.mod) {
		c.options = append(c.options, Option{
			Name:     opt.Name,
			Value:    opt.Default,
			Required: opt.Required,
			Desc:     opt.Desc,
			Advanced: opt.Advanced,
		})
	}

	// Overlay global options.
	for name, val := range c.globals {
		c.setOpt(name, val)
	}

	if c.hasOpt("PAYLOAD") && c.getOpt("PAYLOAD") == "" {
		if defPayload := payload.DefaultPayload(c.mod.Info().Platform()); defPayload != nil {
			c.setOpt("PAYLOAD", defPayload.Name)
		}
	}

	c.importTargetDefaults()
}

func (c *Console) importTargetDefaults() {
	targets := c.mod.Info().Targets
	if c.targetIdx < 0 || c.targetIdx >= len(targets) {
		return
	}
	for name, val := range targets[c.targetIdx].Defaults {
		c.setOpt(name, val)
	}
}

func (c *Console) optionNames() []string {
	names := make([]string, len(c.options))
	for i, opt := range c.options {
		names[i] = opt.Name
	}
	return names
}

func (c *Console) hasOpt(name string) bool {
	for _, opt := range c.options {
		if strings.EqualFold(opt.Name, name) {
			return true
		}
	}
	return false
}

func (c *Console) getOpt(name string) string {
	for _, opt := range c.options {
		if strings.EqualFold(opt.Name, name) {
			return opt.Value
		}
	}
	return ""
}

func (c *Console) setOpt(name, value string) bool {
	for i := range c.options {
		if strings.EqualFold(c.options[i].Name, name) {
			c.options[i].Value = value
			c.syncTargetPort(name, value)
			return true
		}
	}
	return false
}

// syncTargetPort keeps TARGET and RPORT in sync.
// Setting TARGET with a port updates RPORT.
// Setting RPORT updates the port in TARGET.
func (c *Console) syncTargetPort(name, value string) {
	upper := strings.ToUpper(name)
	if upper == "TARGET" && strings.Contains(value, ":") {
		parts := strings.SplitN(value, ":", 2)
		if len(parts) == 2 && c.hasOpt("RPORT") {
			for i := range c.options {
				if strings.EqualFold(c.options[i].Name, "RPORT") {
					c.options[i].Value = parts[1]
					break
				}
			}
		}
	}
	if upper == "RPORT" && c.hasOpt("TARGET") {
		target := c.getOpt("TARGET")
		if target != "" {
			host := target
			if idx := strings.LastIndex(target, ":"); idx >= 0 {
				host = target[:idx]
			}
			for i := range c.options {
				if strings.EqualFold(c.options[i].Name, "TARGET") {
					c.options[i].Value = host + ":" + value
					break
				}
			}
		}
	}
}

// requireMod checks that a module is selected. Returns false and prints an error if not.
func (c *Console) requireMod() bool {
	if c.mod == nil {
		output.Error("No module selected")
		return false
	}
	return true
}

// requireOpt checks that a required option is set. Returns the value and true, or prints an error and returns false.
func (c *Console) requireOpt(name string) (string, bool) {
	val := c.getOpt(name)
	if val == "" {
		output.Error("%s not set", name)
		return "", false
	}
	return val, true
}

func (c *Console) buildParams() sdk.Params {
	values := make(map[string]string, len(c.options))
	for _, opt := range c.options {
		if opt.Value != "" {
			values[strings.ToUpper(opt.Name)] = opt.Value
		}
	}
	return sdk.NewParams(context.Background(), values)
}
