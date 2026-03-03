package console

import (
	"sort"
	"strings"

	"github.com/Chocapikk/pik/sdk"
)

type completionState struct {
	prefix  string
	matches []string
	index   int
}

var showSubcommands = []string{
	"options", "advanced", "missing", "payloads",
	"targets", "modules", "sessions", "info",
}

var labSubcommands = []string{"start", "stop", "status", "run"}

// complete returns completion candidates for the given input line.
func (c *Console) complete(input string) []string {
	parts := strings.Fields(input)
	trailing := strings.HasSuffix(input, " ")

	// No input or completing the command name
	if len(parts) == 0 || (len(parts) == 1 && !trailing) {
		prefix := ""
		if len(parts) == 1 {
			prefix = strings.ToLower(parts[0])
		}
		return filterPrefix(c.commandNames(), prefix)
	}

	cmd := strings.ToLower(parts[0])
	argPrefix := ""
	if !trailing && len(parts) > 1 {
		argPrefix = parts[len(parts)-1]
	}

	switch cmd {
	case "use", "info", "search":
		return filterPrefix(sdk.Names(), argPrefix)
	case "set", "setg", "unset":
		return filterPrefix(c.optionNames(), strings.ToUpper(argPrefix))
	case "unsetg":
		keys := make([]string, 0, len(c.globals))
		for k := range c.globals {
			keys = append(keys, k)
		}
		return filterPrefix(keys, strings.ToUpper(argPrefix))
	case "show":
		return filterPrefix(showSubcommands, argPrefix)
	case "lab":
		return filterPrefix(labSubcommands, argPrefix)
	case "help":
		return filterPrefix(c.commandNames(), argPrefix)
	}

	return nil
}

func (c *Console) commandNames() []string {
	var names []string
	for name, cmd := range c.commands {
		if cmd.desc != "" {
			names = append(names, name)
		}
	}
	names = append(names, "exit", "quit")
	sort.Strings(names)
	return names
}

func filterPrefix(items []string, prefix string) []string {
	if prefix == "" {
		return items
	}
	lower := strings.ToLower(prefix)
	var matches []string
	for _, item := range items {
		if strings.HasPrefix(strings.ToLower(item), lower) {
			matches = append(matches, item)
		}
	}
	return matches
}
