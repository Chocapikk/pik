package console

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Chocapikk/pik/pkg/log"
	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/pkg/payload"
	"github.com/Chocapikk/pik/sdk"
)

// --- Help ---

func (c *Console) cmdHelp(args []string) {
	if len(args) > 0 {
		name := strings.ToLower(args[0])
		if cmd, ok := c.commands[name]; ok && cmd.help != "" {
			output.Println()
			output.Print("  %s\n", cmd.help)
			output.Println()
			return
		}
		output.Error("No help available for %q", args[0])
		return
	}

	output.Println()
	for _, name := range []string{
		"use", "back", "previous", "show", "set", "setg", "unset", "unsetg", "target", "info",
		"check", "exploit", "sessions", "kill", "resource", "list", "search", "rank",
	} {
		cmd, ok := c.commands[name]
		if !ok || cmd.desc == "" {
			continue
		}
		output.Print("  %s %s\n", log.Pad(log.Cyan(name), 20), log.Gray(cmd.desc))
	}
	output.Print("  %s %s\n", log.Pad(log.Cyan("exit"), 20), log.Gray("Exit the console"))
	output.Println()
	output.Print("  %s %s\n", log.Gray("Aliases:"), log.Gray("run, rerun, rcheck, options, advanced, modules, ?"))
	output.Println()
}

// --- Module listing ---

func (c *Console) cmdList() {
	c.printModuleTable(sdk.List())
}

func (c *Console) cmdSearch(args []string) {
	if len(args) == 0 {
		output.Error("Usage: search <keyword>")
		return
	}
	query := strings.Join(args, " ")
	matches := sdk.Search(query)
	if len(matches) == 0 {
		output.Warning("No modules matching %q", query)
		return
	}
	c.printModuleTable(matches)
}

func (c *Console) printModuleTable(modules []sdk.Exploit) {
	if len(modules) == 0 {
		output.Warning("No modules loaded")
		return
	}
	output.Println()
	nameW, relW, descW := 4, 11, 11
	for _, mod := range modules {
		info := mod.Info()
		if w := len(sdk.NameOf(mod)); w > nameW {
			nameW = w
		}
		if w := len(info.Reliability.String()); w > relW {
			relW = w
		}
		if w := len(info.Description); w > descW {
			descW = w
		}
	}

	output.Print("  %s  %s  %s  %s\n",
		log.Pad(log.UnderlineText("Name"), nameW),
		log.Pad(log.UnderlineText("Reliability"), relW),
		log.Pad(log.UnderlineText("Description"), descW),
		log.UnderlineText("CVEs"),
	)
	for _, mod := range modules {
		info := mod.Info()
		cves := strings.Join(info.CVEs(), ", ")
		if cves == "" {
			cves = "-"
		}
		output.Print("  %s  %s  %s  %s\n",
			log.Pad(log.Cyan(sdk.NameOf(mod)), nameW),
			log.Pad(reliabilityStyle(info.Reliability), relW),
			log.Pad(info.Description, descW),
			log.Yellow(cves),
		)
	}
	output.Println()
}

func (c *Console) cmdRank() {
	rankings := sdk.Rankings()
	if len(rankings) == 0 {
		output.Warning("No modules loaded")
		return
	}
	output.Println()
	output.Print("  %s  %s  %s  %s\n",
		log.Pad(log.UnderlineText("#"), 5),
		log.Pad(log.UnderlineText("Author"), 20),
		log.Pad(log.UnderlineText("Modules"), 10),
		log.UnderlineText("CVEs"),
	)
	for i, rank := range rankings {
		output.Print("  %s  %s  %s  %s\n",
			log.Pad(log.Cyan(fmt.Sprintf("%d", i+1)), 5),
			log.Pad(log.White(rank.Name), 20),
			log.Pad(log.Cyan(fmt.Sprintf("%d", rank.Modules)), 10),
			log.Yellow(fmt.Sprintf("%d", rank.CVEs)),
		)
	}
	output.Println()
}

// --- Module selection ---

func (c *Console) cmdUse(args []string) {
	var name string

	if len(args) > 0 {
		name = args[0]
	}
	if name == "" {
		c.rl.Terminal.ExitRawMode()
		defer c.rl.Terminal.EnterRawMode()

		modules := sdk.List()
		items := make([]fuzzyItem, len(modules))
		for i, mod := range modules {
			info := mod.Info()
			cves := strings.Join(info.CVEs(), ", ")
			items[i] = fuzzyItem{name: sdk.NameOf(mod), desc: cves}
		}

		selected, ok := FuzzySelect("Select module", items)
		if !ok {
			return
		}
		name = selected
	}

	mod := sdk.Get(name)
	if mod == nil {
		output.Error("Module %q not found", name)
		return
	}

	// Save current module for `previous` command.
	if c.mod != nil {
		c.previousMod = c.mod
		c.previousIdx = c.targetIdx
	}

	c.mod = mod
	c.targetIdx = 0
	c.initOptions()
	c.updatePrompt()
	output.Success("Using %s - %s", sdk.NameOf(mod), mod.Info().Description)
}

func (c *Console) cmdBack() {
	if c.mod != nil {
		c.previousMod = c.mod
		c.previousIdx = c.targetIdx
	}
	c.mod = nil
	c.options = nil
	c.updatePrompt()
}

func (c *Console) cmdPrevious() {
	if c.previousMod == nil {
		output.Warning("No previous module")
		return
	}
	prev := c.previousMod
	prevIdx := c.previousIdx

	if c.mod != nil {
		c.previousMod = c.mod
		c.previousIdx = c.targetIdx
	}

	c.mod = prev
	c.targetIdx = prevIdx
	c.initOptions()
	c.updatePrompt()
	output.Success("Using %s - %s", sdk.NameOf(prev), prev.Info().Description)
}

// --- Info ---

func (c *Console) cmdInfo(args []string) {
	mod := c.mod
	if len(args) > 0 {
		mod = sdk.Get(args[0])
	}

	if mod == nil {
		output.Error("No module selected (use 'info <module>' or 'use <module>' first)")
		return
	}

	info := mod.Info()
	output.Println()
	output.Print("  %s  %s\n", log.Cyan("Name:"), sdk.NameOf(mod))
	output.Print("  %s  %s\n", log.Cyan("Description:"), info.Description)
	if info.Detail != "" {
		output.Print("\n%s\n", info.Detail)
	}
	output.Print("  %s  %s\n", log.Cyan("Authors:"), strings.Join(info.Authors, ", "))
	output.Print("  %s  %s\n", log.Cyan("Reliability:"), reliabilityStyle(info.Reliability))
	output.Print("  %s  %s\n", log.Cyan("Check:"), checkSupportStr(mod))
	output.Print("  %s  %s\n", log.Cyan("CVEs:"), strings.Join(info.CVEs(), ", "))
	if len(info.References) > 0 {
		output.Print("  %s\n", log.Cyan("References:"))
		for _, ref := range info.References {
			output.Print("    - %s\n", log.Blue(ref.URL()))
		}
	}
	output.Print("  %s  %s\n", log.Cyan("Targets:"), strings.Join(info.TargetStrings(), ", "))
	if len(info.Queries) > 0 {
		output.Println()
		output.Print("  %s\n", log.Cyan("Queries:"))
		for _, q := range info.Queries {
			output.Print("    %s %s\n", log.Pad(log.Gray(q.Engine+":"), 12), q.URL())
		}
	}
	output.Println()
}

// --- Show ---

func (c *Console) cmdShow(args []string) {
	if len(args) == 0 {
		output.Error("Usage: show <options|advanced|missing|payloads|targets|modules|sessions|info>")
		return
	}

	switch strings.ToLower(args[0]) {
	case "options":
		if c.mod == nil {
			c.showGlobals()
		} else {
			c.showOptions(false)
		}
	case "advanced":
		c.showOptions(true)
	case "missing":
		c.showMissing()
	case "payloads":
		c.showPayloads()
	case "targets":
		c.showTargets()
	case "modules":
		c.cmdList()
	case "sessions":
		c.cmdSessions(nil)
	case "info":
		c.cmdInfo(args[1:])
	default:
		output.Error("Unknown: show %s (try: options, advanced, missing, payloads, targets, modules, sessions)", args[0])
	}
}

// --- Set / Setg ---

func (c *Console) cmdSet(args []string) {
	if !c.requireMod() {
		return
	}

	// No args: dump all options.
	if len(args) == 0 {
		c.showOptions(false)
		return
	}

	name := strings.ToUpper(args[0])

	// One arg: print current value.
	if len(args) == 1 {
		if name == "PAYLOAD" {
			c.selectPayload()
			return
		}
		if c.hasOpt(name) {
			val := c.getOpt(name)
			if val == "" {
				val = log.Muted("(not set)")
			}
			output.Print("  %s => %s\n", name, val)
			return
		}
		c.suggestOption(name)
		return
	}

	value := strings.Join(args[1:], " ")

	if name == "PAYLOAD" && value == "?" {
		c.selectPayload()
		return
	}

	if !c.setOpt(name, value) {
		c.suggestOption(name)
		return
	}
	output.Success("%s => %s", name, value)
	c.warnSSLPort(name)
}

func (c *Console) cmdUnset(args []string) {
	if len(args) == 0 {
		output.Error("Usage: unset <option>")
		return
	}
	name := strings.ToUpper(args[0])
	if !c.setOpt(name, "") {
		output.Error("Unknown option: %s", name)
		return
	}
	output.Success("Unset %s", name)
}

func (c *Console) cmdSetg(args []string) {
	// No args: dump globals.
	if len(args) == 0 {
		c.showGlobals()
		return
	}

	if len(args) < 2 {
		name := strings.ToUpper(args[0])
		if val, ok := c.globals[name]; ok {
			output.Print("  %s => %s\n", name, val)
		} else {
			output.Print("  %s => %s\n", name, log.Muted("(not set)"))
		}
		return
	}

	name := strings.ToUpper(args[0])
	value := strings.Join(args[1:], " ")
	c.globals[name] = value

	// Also set on current module if loaded.
	if c.mod != nil {
		c.setOpt(name, value)
	}
	output.Success("%s => %s (global)", name, value)
}

func (c *Console) cmdUnsetg(args []string) {
	if len(args) == 0 {
		output.Error("Usage: unsetg <option>")
		return
	}
	name := strings.ToUpper(args[0])
	delete(c.globals, name)
	output.Success("Unset global %s", name)
}

// suggestOption prints an error with a "did you mean?" hint.
func (c *Console) suggestOption(name string) {
	lower := strings.ToLower(name)
	var closest string
	for _, opt := range c.options {
		if strings.Contains(strings.ToLower(opt.Name), lower) || strings.HasPrefix(strings.ToLower(opt.Name), lower[:min(3, len(lower))]) {
			closest = opt.Name
			break
		}
	}
	if closest != "" {
		output.Error("Unknown option: %s. Did you mean %s?", name, log.Cyan(closest))
	} else {
		output.Error("Unknown option: %s", name)
	}
}

// --- Target ---

func (c *Console) cmdTarget(args []string) {
	if !c.requireMod() {
		return
	}
	if len(args) == 0 {
		c.showTargets()
		return
	}
	targets := c.mod.Info().Targets
	if len(targets) == 0 {
		output.Warning("Module has no targets defined")
		return
	}
	idx, err := strconv.Atoi(args[0])
	if err != nil || idx < 0 || idx >= len(targets) {
		output.Error("Invalid target index (0-%d)", len(targets)-1)
		return
	}
	c.targetIdx = idx
	c.importTargetDefaults()
	output.Success("Target => %d - %s", idx, targets[idx].Name)
}

// --- Resource ---

func (c *Console) cmdResource(args []string) {
	if len(args) == 0 {
		output.Error("Usage: resource <file.rc>")
		return
	}
	f, err := os.Open(args[0])
	if err != nil {
		output.Error("Cannot open %s: %v", args[0], err)
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		output.Print("  %s\n", log.Gray(line))
		if c.exec(line) {
			return
		}
	}
}

// --- Payload selector ---

func (c *Console) selectPayload() {
	platform := ""
	if c.mod != nil {
		platform = c.mod.Info().Platform()
	}

	payloads := payload.ListForPlatform(platform)
	items := make([]fuzzyItem, len(payloads))
	for i, pl := range payloads {
		items[i] = fuzzyItem{name: pl.Name, desc: pl.Description}
	}

	c.rl.Terminal.ExitRawMode()
	defer c.rl.Terminal.EnterRawMode()

	selected, ok := FuzzySelect("Select payload", items)
	if !ok {
		return
	}
	c.setOpt("PAYLOAD", selected)
	output.Success("PAYLOAD => %s", selected)
}
