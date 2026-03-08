package c2

import (
	"fmt"
	"time"

	"github.com/Chocapikk/pik/pkg/c2/session"
)

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

// SessionHandler is an optional interface for backends that support
// multiple concurrent sessions.
type SessionHandler interface {
	Sessions() []*session.Session
	Interact(id int) error
	Kill(id int) error
}

// SessionBase provides default SessionHandler methods by wrapping a session.Manager.
// Embed in a listener to get Sessions, Interact, Kill, WaitForSession for free.
type SessionBase struct {
	Manager *session.Manager
}

func (b *SessionBase) WaitForSession(timeout time.Duration) error {
	_, err := b.Manager.Accept(timeout)
	if err != nil {
		return fmt.Errorf("no session received: %w", err)
	}
	return nil
}

func (b *SessionBase) Sessions() []*session.Session { return b.Manager.List() }
func (b *SessionBase) Interact(id int) error         { return b.Manager.Interact(id) }
func (b *SessionBase) Kill(id int) error              { return b.Manager.Kill(id) }

func (b *SessionBase) ShutdownManager() error {
	if b.Manager != nil {
		b.Manager.Close()
	}
	return nil
}

// PayloadGen is a function that generates a reverse shell command for a given host:port.
type PayloadGen func(lhost string, lport int) string

// PayloadMap is a map of payload names to generators.
type PayloadMap map[string]PayloadGen

// ResolvePayload looks up a payload by type in the map, falling back to the given default.
func ResolvePayload(payloads PayloadMap, lhost string, lport int, payloadType string, fallback PayloadGen) (string, error) {
	if gen, ok := payloads[payloadType]; ok {
		return gen(lhost, lport), nil
	}
	return fallback(lhost, lport), nil
}

// Configurable is an optional interface for backends that need
// external configuration (e.g. Sliver operator config file).
type Configurable interface {
	Configure(configPath string)
}

var registry = map[string]Backend{}

// Register registers a C2 backend by its Name().
// Called from init() in each backend package.
func Register(b Backend) {
	registry[b.Name()] = b
}

// Resolve returns a Backend for the given C2 type.
// Returns nil if the type is "shell" or unregistered (runner defaults to built-in shell).
func Resolve(c2Type, configPath string) Backend {
	b, ok := registry[c2Type]
	if !ok {
		return nil
	}
	if configPath != "" {
		if cb, ok := b.(Configurable); ok {
			cb.Configure(configPath)
		}
	}
	return b
}
