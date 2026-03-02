package console

import (
	"context"
	"strings"

	"github.com/Chocapikk/pik/sdk"
	"github.com/Chocapikk/pik/pkg/payload"
)

type option struct {
	Name     string
	Value    string
	Required bool
	Desc     string
	Advanced bool
}

func (c *Console) initOptions() {
	c.options = []option{
		{Name: "TARGET", Required: true, Desc: "Target URL/IP"},
	}

	for _, opt := range sdk.ResolveOptions(c.mod) {
		c.options = append(c.options, option{
			Name:     opt.Name,
			Value:    opt.Default,
			Required: opt.Required,
			Desc:     opt.Desc,
			Advanced: opt.Advanced,
		})
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
	for name, val := range targets[c.targetIdx].DefaultOptions {
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
			return true
		}
	}
	return false
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
