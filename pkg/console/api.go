package console

import (
	"github.com/Chocapikk/pik/pkg/c2"
	"github.com/Chocapikk/pik/pkg/log"
	"github.com/Chocapikk/pik/pkg/types"
	"github.com/Chocapikk/pik/sdk"
)

// Shared message type aliases (from pkg/types to break import cycles).
type ClearOutputMsg = types.ClearOutputMsg
type FuzzyItem = types.FuzzyItem
type FuzzySelectMsg = types.FuzzySelectMsg
type SessionInteractMsg = types.SessionInteractMsg

// New creates a Console with registered commands.
func New() *Console {
	c := &Console{globals: make(map[string]string)}
	c.registerCommands()
	return c
}

// --- TUI integration ---

// SetProgram sets the message sender (TUI program) for async communication.
func (c *Console) SetProgram(p MsgSender) { c.program = p }

// Program returns the message sender.
func (c *Console) Program() MsgSender { return c.program }

// SendClear sends a clear message to the TUI.
func (c *Console) SendClear() {
	if c.program != nil {
		c.program.Send(ClearOutputMsg{})
	}
}

// --- Module and options ---

// Mod returns the currently selected module.
func (c *Console) Mod() sdk.Exploit { return c.mod }

// SetMod sets the current module and initializes options.
func (c *Console) SetMod(mod sdk.Exploit) {
	c.mod = mod
	c.targetIdx = 0
	c.initOptions()
}

// Options returns the current module options.
func (c *Console) Options() []Option { return c.options }

// OptionNames returns option names for completion.
func (c *Console) OptionNames() []string { return c.optionNames() }

// SetOpt sets a module option value.
func (c *Console) SetOpt(name, value string) bool { return c.setOpt(name, value) }

// --- Commands ---

// Exec runs a console command line. Returns true if the console should exit.
func (c *Console) Exec(line string) bool { return c.exec(line) }

// UseByName selects a module by name or numeric index.
func (c *Console) UseByName(name string) { c.cmdUseByName(name) }

// ApplyFuzzyResult handles the result of a fuzzy selection.
func (c *Console) ApplyFuzzyResult(context, selected string) { c.applyFuzzyResult(context, selected) }

// SessionHandler returns the C2 session handler, or nil.
func (c *Console) SessionHandler() c2.SessionHandler { return c.sessionHandler() }

// ShutdownBackend shuts down the C2 backend.
func (c *Console) ShutdownBackend() { c.shutdownBackend() }

// BuildPrompt returns the styled prompt string.
func (c *Console) BuildPrompt() string {
	promptBase := log.Amber("pik")
	promptArrow := log.Muted(" > ")
	if c.mod != nil {
		return promptBase + " " + log.White(sdk.NameOf(c.mod)) + promptArrow
	}
	return promptBase + promptArrow
}
