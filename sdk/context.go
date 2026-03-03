package sdk

import (
	"context"
	"fmt"
	"time"
)

// Context is the execution context passed to exploits.
// Provides HTTP, logging, payload helpers, and timing.
type Context struct {
	values    map[string]string
	payload   string
	commands  []string
	target    Target
	startTime time.Time
	timing    bool

	// Function hooks injected by the runner.
	SendFn       func(HTTPRequest) (*HTTPResponse, error)
	DialFn       func() (Conn, error)
	StatusFn     func(string, ...any)
	SuccessFn    func(string, ...any)
	ErrorFn      func(string, ...any)
	WarningFn    func(string, ...any)
	Base64BashFn func(string) string
	CommentFn    func(string) string
	RandTextFn   func(int) string
}

// NewContext creates a Context with option values and payload command.
func NewContext(values map[string]string, payload string) *Context {
	return &Context{values: values, payload: payload}
}

// --- HTTP ---

// SendFactory creates a SendFn from module params.
// Registered by pkg/protocol/http via SetSendFactory.
type SendFactory func(Params) func(HTTPRequest) (*HTTPResponse, error)

var sendFactory SendFactory

// SetSendFactory registers the HTTP send implementation.
// Called by pkg/protocol/http's init().
func SetSendFactory(f SendFactory) { sendFactory = f }

// SendWith creates an HTTP send function using the registered factory.
func SendWith(params Params) func(HTTPRequest) (*HTTPResponse, error) {
	if sendFactory == nil {
		return nil
	}
	return sendFactory(params)
}

// PoolFactory configures connection pooling on a context for concurrent scanning.
// Registered by protocol packages that support pooling (e.g. HTTP).
type PoolFactory func(ctx context.Context, threads int, proxy string) context.Context

var poolFactory PoolFactory

// SetPoolFactory registers the connection pool implementation.
func SetPoolFactory(f PoolFactory) { poolFactory = f }

// WithPool applies connection pooling if a factory is registered.
// Returns ctx unchanged if no pool factory is available (e.g. TCP modules).
func WithPool(ctx context.Context, threads int, proxy string) context.Context {
	if poolFactory != nil {
		return poolFactory(ctx, threads, proxy)
	}
	return ctx
}

// Send dispatches an HTTP request through the runner's HTTP bridge.
func (c *Context) Send(req HTTPRequest) (*HTTPResponse, error) {
	if c.SendFn != nil {
		return c.SendFn(req)
	}
	return nil, fmt.Errorf("no HTTP client configured")
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
// Registered by pkg/protocol/tcp via SetDialFactory.
type DialFactory func(Params) (Conn, error)

var dialFactory DialFactory

// SetDialFactory registers the TCP dial implementation.
// Called by pkg/protocol/tcp's init().
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
// Used by TCP modules to pass to tcp.FromModule().
func (c *Context) Params() Params {
	cp := make(map[string]string, len(c.values))
	for k, v := range c.values {
		cp[k] = v
	}
	return NewParams(context.Background(), cp)
}

// Commands returns the CmdStager commands set by the runner.
// Empty when in single-shot mode.
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

func (c *Context) Base64Bash(cmd string) string {
	if c.Base64BashFn != nil {
		return c.Base64BashFn(cmd)
	}
	return cmd
}

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
