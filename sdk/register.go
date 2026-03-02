package sdk

import "github.com/Chocapikk/pik/pkg/core"

//go:generate go run gen.go

// Register adds an exploit to the global registry.
// Wraps core.Register with adjusted caller depth for the SDK indirection.
func Register(mod core.Exploit) { core.RegisterFrom(mod, 3) }
