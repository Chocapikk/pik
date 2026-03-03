package console

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Chocapikk/pik/pkg/c2"
	"github.com/Chocapikk/pik/pkg/log"
	"github.com/Chocapikk/pik/sdk"
)

// Option represents a module option visible to the TUI.
type Option struct {
	Name     string
	Value    string
	Required bool
	Desc     string
	Advanced bool
}

// New creates a Console with registered commands. Used by the TUI package.
func New() *Console {
	c := &Console{globals: make(map[string]string)}
	c.registerCommands()
	return c
}

// SetProgram sets the bubbletea program reference for message sending.
func (c *Console) SetProgram(p *tea.Program) { c.program = p }

// Exec runs a console command line. Returns true if the console should exit.
func (c *Console) Exec(line string) bool { return c.exec(line) }

// UseByName selects a module by name or numeric index.
func (c *Console) UseByName(name string) { c.cmdUseByName(name) }

// SetOpt sets a module option value.
func (c *Console) SetOpt(name, value string) bool { return c.setOpt(name, value) }

// GetOpt returns a module option value.
func (c *Console) GetOpt(name string) string { return c.getOpt(name) }

// Complete returns tab completion candidates for the given input.
func (c *Console) Complete(input string) []string { return c.complete(input) }

// ApplyFuzzyResult handles the result of a fuzzy selection.
func (c *Console) ApplyFuzzyResult(context, selected string) { c.applyFuzzyResult(context, selected) }

// SessionHandler returns the C2 session handler, or nil.
func (c *Console) SessionHandler() c2.SessionHandler { return c.sessionHandler() }

// ShutdownBackend shuts down the C2 backend.
func (c *Console) ShutdownBackend() { c.shutdownBackend() }

// Mod returns the currently selected module.
func (c *Console) Mod() sdk.Exploit { return c.mod }

// SetMod sets the current module and initializes options.
func (c *Console) SetMod(mod sdk.Exploit) {
	c.mod = mod
	c.targetIdx = 0
	c.initOptions()
}

// Options returns the current module options as exported types.
func (c *Console) Options() []Option {
	result := make([]Option, len(c.options))
	for i, o := range c.options {
		result[i] = Option{
			Name:     o.Name,
			Value:    o.Value,
			Required: o.Required,
			Desc:     o.Desc,
			Advanced: o.Advanced,
		}
	}
	return result
}

// OptionNames returns option names for completion.
func (c *Console) OptionNames() []string { return c.optionNames() }

// BuildPrompt returns the styled prompt string.
func (c *Console) BuildPrompt() string {
	promptBase := log.Amber("pik")
	promptArrow := log.Muted(" > ")
	if c.mod != nil {
		return promptBase + " " + log.White(sdk.NameOf(c.mod)) + promptArrow
	}
	return promptBase + promptArrow
}

// RunCheck runs the check command.
func (c *Console) RunCheck() { c.cmdCheck() }

// RunExploit runs the exploit command.
func (c *Console) RunExploit() { c.cmdExploit() }

// RunLab runs a lab subcommand.
func (c *Console) RunLab(args []string) { c.cmdLab(args) }

// RunSet runs the set command.
func (c *Console) RunSet(args []string) { c.cmdSet(args) }

// SendClear sends a clear message to the TUI.
func (c *Console) SendClear() {
	if c.program != nil {
		c.program.Send(ClearOutputMsg{})
	}
}

// ClearOutputMsg requests the TUI to clear the output viewport.
type ClearOutputMsg struct{}
