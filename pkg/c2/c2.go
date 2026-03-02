package c2

import "time"

// Backend is the interface for C2 integrations (built-in shell, Sliver, etc.).
type Backend interface {
	Name() string
	Setup(lhost string, lport int) error
	GeneratePayload(targetOS, payloadType string) (string, error)
	WaitForSession(timeout time.Duration) error
	Shutdown() error
}

// ImplantGenerator is an optional interface for backends that can produce raw
// implant binaries. The runner uses this to feed CmdStager delivery instead of
// single-shot payload commands.
type ImplantGenerator interface {
	GenerateImplant(targetOS, arch string) ([]byte, error)
}

// Stager is an optional interface for backends that stage implants over HTTP.
// Returns the staging URL; the runner builds the fetch command via pkg/payload.
type Stager interface {
	StageImplant(targetOS, arch string) (url string, err error)
}

// TCPStager is an optional interface for backends that support TCP-based
// staging. Returns a small stager binary (patched with host:port) ready for
// CmdStager chunking. The backend manages the TCP listener internally.
type TCPStager interface {
	TCPStageImplant(targetOS, arch string) ([]byte, error)
}

// Factory creates a Backend from a config path.
type Factory func(configPath string) Backend

var factories = map[string]Factory{}

// RegisterFactory registers a named C2 backend factory.
// Called from init() in each backend package.
func RegisterFactory(name string, factory Factory) {
	factories[name] = factory
}

// Resolve returns a Backend for the given C2 type.
// Returns nil if the type is "shell" or unregistered (runner defaults to built-in shell).
func Resolve(c2Type, configPath string) Backend {
	if factory, ok := factories[c2Type]; ok {
		return factory(configPath)
	}
	return nil
}
