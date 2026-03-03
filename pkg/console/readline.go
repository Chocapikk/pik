package console

import (
	"os"
	"path/filepath"

	"github.com/chzyer/readline"

	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/pkg/payload"
	"github.com/Chocapikk/pik/sdk"
)

// Run starts the readline-based interactive console.
func Run() error {
	return RunWith(nil)
}

// RunWith starts the readline-based console with an optional pre-selected module.
func RunWith(mod sdk.Exploit) error {
	output.BannerModuleCount = len(sdk.List())
	output.BannerPayloadCount = len(payload.ListPayloads())
	output.Banner()

	cons := New()

	// Override clear for console mode (ANSI escape instead of TUI message).
	cons.commands["clear"] = command{func(_ []string) { output.Print("\033[2J\033[H") }, "Clear the screen", ""}
	cons.commands["cls"] = command{func(_ []string) { output.Print("\033[2J\033[H") }, "", ""}

	rl, err := cons.initReadline()
	if err != nil {
		return err
	}
	defer rl.Close()
	defer cons.shutdownBackend()

	if mod != nil {
		cons.SetMod(mod)
		rl.SetPrompt(cons.BuildPrompt())
		output.Success("Using %s - %s", sdk.NameOf(mod), mod.Info().Title())
	}

	for {
		line, err := rl.Readline()
		if err != nil {
			output.Println()
			return nil
		}
		if cons.exec(line) {
			return nil
		}
		rl.SetPrompt(cons.BuildPrompt())
	}
}

func (c *Console) initReadline() (*readline.Instance, error) {
	completer := readline.NewPrefixCompleter(
		readline.PcItem("use", readline.PcItemDynamic(func(line string) []string {
			return sdk.Names()
		})),
		readline.PcItem("set", readline.PcItemDynamic(func(line string) []string {
			return c.optionNames()
		})),
		readline.PcItem("setg", readline.PcItemDynamic(func(line string) []string {
			return c.optionNames()
		})),
		readline.PcItem("unset", readline.PcItemDynamic(func(line string) []string {
			return c.optionNames()
		})),
		readline.PcItem("show",
			readline.PcItem("options"),
			readline.PcItem("advanced"),
			readline.PcItem("missing"),
			readline.PcItem("payloads"),
			readline.PcItem("targets"),
			readline.PcItem("modules"),
			readline.PcItem("sessions"),
			readline.PcItem("info"),
		),
		readline.PcItem("info", readline.PcItemDynamic(func(line string) []string {
			return sdk.Names()
		})),
		readline.PcItem("search", readline.PcItemDynamic(func(line string) []string {
			return sdk.Names()
		})),
		readline.PcItem("lab",
			readline.PcItem("start"),
			readline.PcItem("stop"),
			readline.PcItem("status"),
			readline.PcItem("run"),
		),
		readline.PcItem("help", readline.PcItemDynamic(func(line string) []string {
			var names []string
			for name, cmd := range c.commands {
				if cmd.desc != "" {
					names = append(names, name)
				}
			}
			return names
		})),
	)

	for name := range c.commands {
		switch name {
		case "use", "set", "setg", "unset", "show", "info", "search", "help", "lab":
			continue
		}
		completer.Children = append(completer.Children, readline.PcItem(name))
	}
	for _, extra := range []string{"exit", "quit"} {
		completer.Children = append(completer.Children, readline.PcItem(extra))
	}

	historyFile := ""
	if home, err := os.UserHomeDir(); err == nil {
		historyFile = filepath.Join(home, ".pik_history")
	}

	return readline.NewEx(&readline.Config{
		Prompt:            c.BuildPrompt(),
		AutoComplete:      completer,
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistoryFile:       historyFile,
		HistorySearchFold: true,
	})
}
