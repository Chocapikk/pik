package sdk

import (
	"fmt"
	"time"
)

// Context is the execution context passed to exploits.
// Provides HTTP, logging, payload helpers, and timing.
type Context struct {
	values    map[string]string
	payload   string
	commands  []string
	startTime time.Time
	timing    bool

	// Function hooks injected by the runner.
	SendFn       func(Request) (*Response, error)
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

func (c *Context) Send(req Request) (*Response, error) {
	if c.SendFn != nil {
		return c.SendFn(req)
	}
	return nil, fmt.Errorf("no HTTP client configured")
}

// --- Params ---

func (c *Context) Get(key string) string { return c.values[key] }
func (c *Context) Payload() string       { return c.payload }

// Commands returns the CmdStager commands set by the runner.
// Empty when in single-shot mode.
func (c *Context) Commands() []string { return c.commands }

// SetCommands is called by the runner to inject CmdStager commands.
func (c *Context) SetCommands(cmds []string) { c.commands = cmds }

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
