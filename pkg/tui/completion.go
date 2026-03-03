package tui

import (
	"sort"
	"strings"

	"github.com/Chocapikk/pik/pkg/console"
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

var defaultCommands = func() []string {
	cmds := []string{
		"help", "use", "back", "previous", "info",
		"show", "set", "unset", "setg", "unsetg",
		"check", "exploit", "run", "sessions", "kill",
		"target", "resource", "list", "search", "rank",
		"lab", "clear", "exit", "quit",
	}
	sort.Strings(cmds)
	return cmds
}()

func completeInput(c *console.Console, input string) []string {
	parts := strings.Fields(input)
	trailing := strings.HasSuffix(input, " ")

	if len(parts) == 0 || (len(parts) == 1 && !trailing) {
		prefix := ""
		if len(parts) == 1 {
			prefix = strings.ToLower(parts[0])
		}
		return filterPrefix(defaultCommands, prefix)
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
		return filterPrefix(c.OptionNames(), strings.ToUpper(argPrefix))
	case "show":
		return filterPrefix(showSubcommands, argPrefix)
	case "lab":
		return filterPrefix(labSubcommands, argPrefix)
	case "help":
		return filterPrefix(defaultCommands, argPrefix)
	}

	return nil
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
