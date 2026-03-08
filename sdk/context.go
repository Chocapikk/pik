package sdk

import (
	"context"
	"fmt"
	"time"
)

// Context is the execution context passed to exploits.
// Provides protocol dispatch, logging, payload helpers, and timing.
type Context struct {
	values    map[string]string
	payload   string
	commands  []string
	target    Target
	startTime time.Time
	timing    bool
	senders   map[string]any // protocol -> send function

	// Function hooks injected by the runner.
	DialFn       func() (Conn, error)
	StatusFn     func(string, ...any)
	SuccessFn    func(string, ...any)
	ErrorFn      func(string, ...any)
	WarningFn    func(string, ...any)
	CommentFn  func(string) string
	RandTextFn func(int) string
	EncoderFn  func(string) string
}

// NewContext creates a Context with option values and payload command.
func NewContext(values map[string]string, payload string) *Context {
	return &Context{values: values, payload: payload, senders: make(map[string]any)}
}

// RegisterSender adds a protocol send function to the context.
// Called by the runner for each registered protocol factory.
func (c *Context) RegisterSender(proto string, fn any) {
	c.senders[proto] = fn
}

// --- Send (polymorphic dispatch) ---

// Send dispatches a request to the appropriate protocol handler.
// The request type determines which protocol is used.
func (c *Context) Send(req Sendable) (*HTTPResponse, error) {
	switch r := req.(type) {
	case HTTPRequest:
		fn, ok := c.senders["http"]
		if !ok {
			return nil, fmt.Errorf("no HTTP client configured")
		}
		return fn.(func(HTTPRequest) (*HTTPResponse, error))(r)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", req.protocol())
	}
}

// --- Protocol factories ---

// SenderFactory creates a protocol-specific send function from module params.
type SenderFactory struct {
	Proto   string
	Factory func(Params) any
}

var senderFactories []SenderFactory

// RegisterSenderFactory registers a protocol's send factory.
// Called by protocol packages via init().
func RegisterSenderFactory(proto string, factory func(Params) any) {
	senderFactories = append(senderFactories, SenderFactory{Proto: proto, Factory: factory})
}

// WireSenders creates and registers all protocol senders on a context.
// Called by the runner in BuildContext.
func WireSenders(ctx *Context, params Params) {
	for _, sf := range senderFactories {
		if fn := sf.Factory(params); fn != nil {
			ctx.RegisterSender(sf.Proto, fn)
		}
	}
}

// PoolFactory configures connection pooling on a context for concurrent scanning.
// Registered by protocol packages that support pooling (e.g. HTTP).
type PoolFactory func(ctx context.Context, threads int, proxy string) context.Context

var poolFactory PoolFactory

// SetPoolFactory registers the connection pool implementation.
func SetPoolFactory(f PoolFactory) { poolFactory = f }

// WithPool applies connection pooling if a factory is registered.
func WithPool(ctx context.Context, threads int, proxy string) context.Context {
	if poolFactory != nil {
		return poolFactory(ctx, threads, proxy)
	}
	return ctx
}

// --- TCP ---

// Conn is a raw TCP connection returned by Dial.
type Conn interface {
	Send([]byte) error
	Recv(int) ([]byte, error)
	SendRecv(data []byte, recvSize int) ([]byte, error)
	Close() error
}

// DialFactory creates a Conn from module params.
type DialFactory func(Params) (Conn, error)

var dialFactory DialFactory

// SetDialFactory registers the TCP dial implementation.
func SetDialFactory(f DialFactory) { dialFactory = f }

// DialWith creates a Conn using the registered factory.
func DialWith(params Params) (Conn, error) {
	if dialFactory == nil {
		return nil, fmt.Errorf("no TCP client registered (import pkg/protocol/tcp)")
	}
	return dialFactory(params)
}

// Dial opens a raw TCP connection to the target.
func (c *Context) Dial() (Conn, error) {
	if c.DialFn != nil {
		return c.DialFn()
	}
	return nil, fmt.Errorf("no TCP client configured")
}

// --- Params ---

func (c *Context) Get(key string) string { return c.values[key] }
func (c *Context) Payload() string       { return c.payload }

// Params returns an sdk.Params built from the context values.
func (c *Context) Params() Params {
	cp := make(map[string]string, len(c.values))
	for k, v := range c.values {
		cp[k] = v
	}
	return NewParams(context.Background(), cp)
}

// Commands returns the CmdStager commands set by the runner.
func (c *Context) Commands() []string { return c.commands }

// SetCommands is called by the runner to inject CmdStager commands.
func (c *Context) SetCommands(cmds []string) { c.commands = cmds }

// Target returns the selected target from module metadata.
func (c *Context) Target() Target { return c.target }

// SetTarget is called by the runner to set the active target.
func (c *Context) SetTarget(t Target) { c.target = t }

// --- Logging ---

func (c *Context) Status(format string, args ...any) {
	if c.StatusFn != nil {
		c.StatusFn(format, args...)
	}
}

func (c *Context) Success(format string, args ...any) {
	if c.SuccessFn != nil {
		c.SuccessFn(format, args...)
	}
}

func (c *Context) Error(format string, args ...any) {
	if c.ErrorFn != nil {
		c.ErrorFn(format, args...)
	}
}

func (c *Context) Warning(format string, args ...any) {
	if c.WarningFn != nil {
		c.WarningFn(format, args...)
	}
}

// --- Payload helpers ---

func (c *Context) CommentTrail(cmd string) string {
	if c.CommentFn != nil {
		return c.CommentFn(cmd)
	}
	return cmd + " #"
}

func (c *Context) RandText(n int) string {
	if c.RandTextFn != nil {
		return c.RandTextFn(n)
	}
	return "x"
}

// EncodedPayload returns the payload wrapped with the selected encoder.
// Modules should use this instead of manually calling Base64Bash(Payload()).
func (c *Context) EncodedPayload() string {
	if c.EncoderFn != nil {
		return c.EncoderFn(c.payload)
	}
	return c.payload
}

// --- Timing ---

func (c *Context) Elapsed(start bool) float64 {
	if start {
		c.startTime = time.Now()
		c.timing = true
		return 0
	}
	if !c.timing {
		return 0
	}
	return time.Since(c.startTime).Seconds()
}
