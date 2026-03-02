package console

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/chzyer/readline"

	"github.com/Chocapikk/pik/pkg/c2"
	"github.com/Chocapikk/pik/pkg/log"
	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/sdk"
)

var (
	promptBase  = log.Amber("pik")
	promptArrow = log.Muted(" > ")
)

type command struct {
	fn   func([]string)
	desc string
}

// Console is the interactive REPL.
type Console struct {
	rl            *readline.Instance
	mod           sdk.Exploit
	options       []option
	targetIdx     int
	activeBackend c2.Backend
	commands      map[string]command
}

// Run starts the interactive console.
func Run() error {
	output.Banner()

	cons := &Console{}
	cons.registerCommands()
	if err := cons.initReadline(); err != nil {
		return err
	}
	defer cons.rl.Close()
	defer cons.shutdownBackend()

	for {
		line, err := cons.rl.Readline()
		if err != nil {
			output.Println()
			return nil
		}
		if cons.exec(line) {
			return nil
		}
	}
}

func (c *Console) registerCommands() {
	c.commands = map[string]command{
		"help":     {func(_ []string) { c.cmdHelp() }, "Show this help"},
		"?":       {func(_ []string) { c.cmdHelp() }, ""},
		"use":     {func(a []string) { c.cmdUse(a) }, "Select a module"},
		"back":    {func(_ []string) { c.cmdBack() }, "Deselect current module"},
		"info":    {func(a []string) { c.cmdInfo(a) }, "Show module details"},
		"show":    {func(a []string) { c.cmdShow(a) }, "Show options/payloads/modules"},
		"set":     {func(a []string) { c.cmdSet(a) }, "Set an option value"},
		"unset":   {func(a []string) { c.cmdUnset(a) }, "Clear an option value"},
		"check":   {func(_ []string) { c.cmdCheck() }, "Check if target is vulnerable"},
		"exploit": {func(_ []string) { c.cmdExploit() }, "Run the exploit"},
		"run":     {func(_ []string) { c.cmdExploit() }, ""},
		"sessions": {func(a []string) { c.cmdSessions(a) }, "List or interact with sessions"},
		"kill":     {func(a []string) { c.cmdKill(a) }, "Kill a session"},
		"target":   {func(a []string) { c.cmdTarget(a) }, "Set exploit target (show targets to list)"},
		"resource": {func(a []string) { c.cmdResource(a) }, "Run commands from a .rc file"},
		"list":     {func(_ []string) { c.cmdList() }, "List all modules"},
		"modules":  {func(_ []string) { c.cmdList() }, ""},
		"rank":     {func(_ []string) { c.cmdRank() }, "Contributor leaderboard"},
		"search":   {func(a []string) { c.cmdSearch(a) }, "Search modules by keyword"},
	}
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
	cmd.fn(parts[1:])
	return false
}

func (c *Console) initReadline() error {
	var commands []string
	for name := range c.commands {
		commands = append(commands, name)
	}
	commands = append(commands, "exit", "quit")

	completer := readline.NewPrefixCompleter(
		readline.PcItem("use", readline.PcItemDynamic(func(line string) []string {
			return sdk.Names()
		})),
		readline.PcItem("set", readline.PcItemDynamic(func(line string) []string {
			return c.optionNames()
		})),
		readline.PcItem("unset", readline.PcItemDynamic(func(line string) []string {
			return c.optionNames()
		})),
		readline.PcItem("show",
			readline.PcItem("options"),
			readline.PcItem("advanced"),
			readline.PcItem("payloads"),
			readline.PcItem("targets"),
			readline.PcItem("modules"),
		),
		readline.PcItem("info", readline.PcItemDynamic(func(line string) []string {
			return sdk.Names()
		})),
		readline.PcItem("search", readline.PcItemDynamic(func(line string) []string {
			return sdk.Names()
		})),
	)
	for _, cmd := range commands {
		switch cmd {
		case "use", "set", "unset", "show", "info":
			continue
		}
		completer.Children = append(completer.Children, readline.PcItem(cmd))
	}

	historyFile := ""
	if home, err := os.UserHomeDir(); err == nil {
		historyFile = filepath.Join(home, ".pik_history")
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:            c.buildPrompt(),
		AutoComplete:      completer,
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistoryFile:       historyFile,
		HistorySearchFold: true,
	})
	if err != nil {
		return err
	}
	c.rl = rl
	return nil
}

func (c *Console) buildPrompt() string {
	if c.mod != nil {
		return promptBase + " " + log.White(sdk.NameOf(c.mod)) + promptArrow
	}
	return promptBase + promptArrow
}

func (c *Console) updatePrompt() {
	c.rl.SetPrompt(c.buildPrompt())
}
