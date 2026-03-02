package core

import (
	"context"
	"strconv"
	"strings"
)

// Params holds the resolved option values for a module run.
type Params struct {
	Ctx    context.Context
	values map[string]string
}

// NewParams creates a Params with initial values.
func NewParams(ctx context.Context, values map[string]string) Params {
	if values == nil {
		values = make(map[string]string)
	}
	return Params{Ctx: ctx, values: values}
}

// Get returns the value of a parameter.
func (p Params) Get(key string) string {
	return p.values[strings.ToUpper(key)]
}

// GetOr returns the value of a parameter, or a default if not set.
func (p Params) GetOr(key, def string) string {
	val := p.values[strings.ToUpper(key)]
	if val == "" {
		return def
	}
	return val
}

// Int returns the value as an integer, or 0 if not parseable.
func (p Params) Int(key string) int {
	val, _ := strconv.Atoi(p.Get(key))
	return val
}

// IntOr returns the value as an integer, or a default if not parseable.
func (p Params) IntOr(key string, def int) int {
	val, err := strconv.Atoi(p.Get(key))
	if err != nil {
		return def
	}
	return val
}

// Set sets a parameter value.
func (p Params) Set(key, value string) {
	p.values[strings.ToUpper(key)] = value
}

// Target returns the TARGET parameter.
func (p Params) Target() string { return p.Get("TARGET") }

// Lhost returns the LHOST parameter.
func (p Params) Lhost() string { return p.Get("LHOST") }

// Lport returns the LPORT parameter as an integer.
func (p Params) Lport() int { return p.IntOr("LPORT", 4444) }

// Arch returns the ARCH parameter, defaulting to "amd64".
func (p Params) Arch() string { return p.GetOr("ARCH", "amd64") }

// Srvhost returns the local bind address. Falls back to LHOST.
func (p Params) Srvhost() string { return p.GetOr("SRVHOST", p.Lhost()) }

// Srvport returns the local bind port. Falls back to LPORT.
func (p Params) Srvport() int { return p.IntOr("SRVPORT", p.Lport()) }

// Tunnel returns the tunnel URL if set.
func (p Params) Tunnel() string { return p.Get("TUNNEL") }

// Map returns a copy of all parameter values.
func (p Params) Map() map[string]string {
	result := make(map[string]string, len(p.values))
	for k, v := range p.values {
		result[k] = v
	}
	return result
}

// Clone returns a copy of Params with an independent values map.
func (p Params) Clone() Params {
	return NewParams(p.Ctx, p.Map())
}
