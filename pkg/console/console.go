package console

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Chocapikk/pik/pkg/c2"
	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/sdk"
)

type command struct {
	fn   func([]string)
	desc string
	help string
}

// Console holds the state for the interactive REPL.
type Console struct {
	mod           sdk.Exploit
	options       []Option
	targetIdx     int
	activeBackend c2.Backend
	commands      map[string]command
	globals       map[string]string
	previousMod   sdk.Exploit
	previousIdx   int
	program       *tea.Program
}

// exec runs a single console line. Returns true if the console should exit.
func (c *Console) exec(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return false
	}

	parts := strings.Fields(line)
	name := strings.ToLower(parts[0])

	if name == "exit" || name == "quit" {
		return true
	}

	cmd, ok := c.commands[name]
	if !ok {
		output.Error("Unknown command: %s (type 'help' for commands)", name)
		return false
	}

	defer func() {
		if r := recover(); r != nil {
			output.Error("Command panicked: %v", r)
		}
	}()

	cmd.fn(parts[1:])
	return false
}

func (c *Console) registerCommands() {
	c.commands = map[string]command{
		"help": {func(a []string) { c.cmdHelp(a) }, "Show this help", "Usage: help [command]\n\nWithout arguments, shows all commands.\nWith a command name, shows detailed help for that command."},
		"?":    {func(a []string) { c.cmdHelp(a) }, "", ""},

		"use":      {func(a []string) { c.cmdUse(a) }, "Select a module", "Usage: use [module]\n\nWithout arguments, opens a fuzzy finder.\nWith a module name (or partial name), selects it directly."},
		"back":     {func(_ []string) { c.cmdBack() }, "Deselect current module", "Usage: back\n\nDeselects the current module and returns to the global context."},
		"previous": {func(_ []string) { c.cmdPrevious() }, "Switch to previous module", "Usage: previous\n\nSwitches back to the previously selected module."},
		"info":     {func(a []string) { c.cmdInfo(a) }, "Show module details", "Usage: info [module]\n\nWithout arguments, shows info for the current module.\nWith a module name, shows info for that module."},

		"show":   {func(a []string) { c.cmdShow(a) }, "Show options/payloads/modules/targets", "Usage: show <subcommand>\n\nSubcommands: options, advanced, missing, payloads, targets, modules, sessions, info"},
		"set":    {func(a []string) { c.cmdSet(a) }, "Set an option value", "Usage: set [option] [value]\n\nWithout arguments, dumps all current option values.\nWith one argument, prints the current value of that option.\nWith two arguments, sets the option to the value."},
		"unset":  {func(a []string) { c.cmdUnset(a) }, "Clear an option value", "Usage: unset <option>\n\nClears the value of a module option."},
		"setg":   {func(a []string) { c.cmdSetg(a) }, "Set a global option", "Usage: setg <option> <value>\n\nSets a global option that persists across module changes.\nWithout arguments, dumps all global options."},
		"unsetg": {func(a []string) { c.cmdUnsetg(a) }, "Clear a global option", "Usage: unsetg <option>\n\nClears a global option."},

		"check":   {func(_ []string) { c.cmdCheck() }, "Check if target is vulnerable", "Usage: check\n\nRuns the module's Check() method against the current TARGET."},
		"exploit": {func(_ []string) { c.cmdExploit() }, "Run the exploit", "Usage: exploit\n\nRuns the module's Exploit() method. Starts a C2 listener and delivers the payload."},
		"run":     {func(_ []string) { c.cmdExploit() }, "", ""},
		"rerun":   {func(_ []string) { c.cmdExploit() }, "", ""},
		"rcheck":  {func(_ []string) { c.cmdCheck() }, "", ""},

		"sessions": {func(a []string) { c.cmdSessions(a) }, "List or interact with sessions", "Usage: sessions [id]\n\nWithout arguments, lists all active sessions.\nWith an ID, interacts with that session."},
		"kill":     {func(a []string) { c.cmdKill(a) }, "Kill a session", "Usage: kill <id>\n\nTerminates the session with the given ID."},
		"target":   {func(a []string) { c.cmdTarget(a) }, "Set exploit target (show targets to list)", "Usage: target [id]\n\nWithout arguments, shows available targets.\nWith an ID, selects that target."},
		"resource": {func(a []string) { c.cmdResource(a) }, "Run commands from a .rc file", "Usage: resource <file>\n\nExecutes commands from the given file, one per line."},

		"list":    {func(_ []string) { c.cmdList() }, "List all modules", "Usage: list\n\nDisplays all registered modules with reliability and CVEs."},
		"modules": {func(_ []string) { c.cmdList() }, "", ""},
		"search":  {func(a []string) { c.cmdSearch(a) }, "Search modules by keyword", "Usage: search <keyword>\n\nSearches modules by name, description, or CVE."},
		"rank":    {func(_ []string) { c.cmdRank() }, "Contributor leaderboard", "Usage: rank\n\nDisplays authors ranked by module count and CVEs."},

		"lab": {func(a []string) { c.cmdLab(a) }, "Manage lab environments", "Usage: lab <start|stop|status|run>\n\nstart   Start the lab for the current module\nstop    Stop the lab for the current module\nstatus  List all running labs\nrun     Start lab, wait for ready, and exploit"},

		"clear": {func(_ []string) { c.SendClear() }, "Clear the screen", ""},
		"cls":   {func(_ []string) { c.SendClear() }, "", ""},

		"options":  {func(_ []string) { c.cmdShow([]string{"options"}) }, "", ""},
		"advanced": {func(_ []string) { c.cmdShow([]string{"advanced"}) }, "", ""},
	}
}

// applyFuzzyResult handles the result of a fuzzy selection.
func (c *Console) applyFuzzyResult(context, selected string) {
	switch context {
	case "module":
		c.cmdUseByName(selected)
	case "payload":
		c.setOpt("PAYLOAD", selected)
		output.Success("Payload set to %s", selected)
	}
}
