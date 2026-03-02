package sdk

import (
	"fmt"
	"strconv"
	"strings"
)

// --- Option types ---

type OptionType string

const (
	TypeString  OptionType = "string"
	TypeInt     OptionType = "int"
	TypeBool    OptionType = "bool"
	TypePort    OptionType = "port"
	TypePath    OptionType = "path"
	TypeAddress OptionType = "address"
	TypeEnum    OptionType = "enum"
)

// --- Option ---

type Option struct {
	Name     string
	Type     OptionType // defaults to TypeString if empty
	Default  string
	Desc     string
	Required bool
	Advanced bool
	Enums    []string // valid values for TypeEnum
}

// Validate checks if a value is valid for this option.
func (o Option) Validate(val string) error {
	if o.Required && val == "" {
		return fmt.Errorf("%s is required", o.Name)
	}
	if val == "" {
		return nil
	}
	switch o.Type {
	case TypeInt:
		if _, err := strconv.Atoi(val); err != nil {
			return fmt.Errorf("%s must be an integer", o.Name)
		}
	case TypePort:
		p, err := strconv.Atoi(val)
		if err != nil || p < 1 || p > 65535 {
			return fmt.Errorf("%s must be a port (1-65535)", o.Name)
		}
	case TypeBool:
		v := strings.ToLower(val)
		if v != "true" && v != "false" {
			return fmt.Errorf("%s must be true or false", o.Name)
		}
	case TypeEnum:
		if len(o.Enums) > 0 {
			for _, e := range o.Enums {
				if strings.EqualFold(val, e) {
					return nil
				}
			}
			return fmt.Errorf("%s must be one of: %s", o.Name, strings.Join(o.Enums, ", "))
		}
	}
	return nil
}

// --- Option constructors ---

func OptTargetURI(def string) Option {
	return Option{Name: "TARGETURI", Type: TypePath, Default: def, Desc: "Base path to the application"}
}

func OptString(name, def, desc string) Option {
	return Option{Name: name, Type: TypeString, Default: def, Desc: desc}
}

func OptRequired(name, def, desc string) Option {
	return Option{Name: name, Type: TypeString, Default: def, Desc: desc, Required: true}
}

func OptInt(name string, def int, desc string) Option {
	return Option{Name: name, Type: TypeInt, Default: fmt.Sprintf("%d", def), Desc: desc}
}

func OptPort(name string, def int, desc string) Option {
	return Option{Name: name, Type: TypePort, Default: fmt.Sprintf("%d", def), Desc: desc}
}

func OptBool(name string, def bool, desc string) Option {
	return Option{Name: name, Type: TypeBool, Default: fmt.Sprintf("%t", def), Desc: desc}
}

func OptEnum(name, def, desc string, values ...string) Option {
	return Option{Name: name, Type: TypeEnum, Default: def, Desc: desc, Enums: values}
}

// OptAdvanced marks any option as advanced.
func OptAdvanced(opt Option) Option {
	opt.Advanced = true
	return opt
}

func OptAddress(name, def, desc string) Option {
	return Option{Name: name, Type: TypeAddress, Default: def, Desc: desc}
}

// --- Enrichers ---

type OptionEnricher func(mod Exploit, opts []Option) []Option

var enrichers []OptionEnricher

func RegisterEnricher(fn OptionEnricher) {
	enrichers = append(enrichers, fn)
}

// ResolveOptions returns the full option set: module + enrichers + target defaults + module defaults.
func ResolveOptions(mod Exploit) []Option {
	opts := make([]Option, len(mod.Options()))
	copy(opts, mod.Options())

	for _, enrich := range enrichers {
		opts = enrich(mod, opts)
	}

	// Module default overrides
	for name, val := range mod.Info().DefaultOptions {
		for i := range opts {
			if opts[i].Name == name {
				opts[i].Default = val
				break
			}
		}
	}

	return opts
}

func HasOpt(opts []Option, name string) bool {
	for _, opt := range opts {
		if strings.EqualFold(opt.Name, name) {
			return true
		}
	}
	return false
}
