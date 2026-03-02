package console

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"

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
		"check", "exploit", "lab", "sessions", "kill", "resource", "list", "search", "rank", "clear",
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

	termW := termWidth()

	// Group modules by directory prefix.
	type entry struct {
		shortName string
		mod       sdk.Exploit
	}
	groups := make(map[string][]entry)
	var groupOrder []string

	for _, mod := range modules {
		full := sdk.NameOf(mod)
		dir, short := splitModulePath(full)
		if _, seen := groups[dir]; !seen {
			groupOrder = append(groupOrder, dir)
		}
		groups[dir] = append(groups[dir], entry{short, mod})
	}

	// Compute short name column width.
	nameW := 4
	for _, entries := range groups {
		for _, e := range entries {
			if w := len(e.shortName); w > nameW {
				nameW = w
			}
		}
	}

	relW := 11
	maxDescW := termW - (4 + nameW + 2 + relW + 2 + 2 + 14)
	if maxDescW < 20 {
		maxDescW = 20
	}

	// Compute description column width from actual data.
	descW := 4
	for _, entries := range groups {
		for _, e := range entries {
			desc := e.mod.Info().Title()
			if len(desc) > maxDescW {
				desc = desc[:maxDescW-3] + "..."
			}
			if w := len(desc); w > descW {
				descW = w
			}
		}
	}

	output.Println()
	for _, dir := range groupOrder {
		output.Print("  %s\n", log.Muted(dir+"/"))
		for _, e := range groups[dir] {
			info := e.mod.Info()
			desc := info.Title()
			if len(desc) > maxDescW {
				desc = desc[:maxDescW-3] + "..."
			}
			cves := info.CVEs()
			cveStr := "-"
			if len(cves) == 1 {
				cveStr = cves[0]
			} else if len(cves) > 1 {
				cveStr = fmt.Sprintf("%d CVEs", len(cves))
			}
			output.Print("    %s  %s  %s  %s\n",
				log.Pad(log.Cyan(e.shortName), nameW),
				log.Pad(reliabilityStyle(info.Reliability), relW),
				log.Pad(desc, descW),
				log.Yellow(cveStr),
			)
		}
	}
	output.Println()
}

func splitModulePath(full string) (string, string) {
	idx := strings.LastIndex(full, "/")
	if idx < 0 {
		return "", full
	}
	return full[:idx], full[idx+1:]
}

func termWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w < 80 {
		return 120
	}
	return w
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
	output.Success("Using %s - %s", sdk.NameOf(mod), mod.Info().Title())
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
	output.Print("  %s  %s\n", log.Cyan("Description:"), info.Title())
	if info.Detail != "" {
		output.Print("\n%s\n", info.Detail)
	}
	output.Print("  %s  %s\n", log.Cyan("Authors:"), info.AuthorNames())
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

// --- Lab ---

func (c *Console) cmdLab(args []string) {
	if len(args) == 0 {
		output.Error("Usage: lab <start|stop|status|run>")
		return
	}
	switch strings.ToLower(args[0]) {
	case "start":
		c.labStart()
	case "stop":
		c.labStop()
	case "status":
		c.labStatus()
	case "run":
		c.labRun()
	default:
		output.Error("Unknown lab command: %s (try: start, stop, status, run)", args[0])
	}
}

func (c *Console) requireLabMgr() sdk.LabManager {
	mgr := sdk.GetLabManager()
	if mgr == nil {
		output.Error("Lab support not available")
	}
	return mgr
}

func (c *Console) labStart() {
	mgr := c.requireLabMgr()
	if mgr == nil || !c.requireMod() {
		return
	}
	info := c.mod.Info()
	if len(info.Lab.Services) == 0 {
		output.Warning("Module %s has no lab defined", sdk.NameOf(c.mod))
		return
	}
	labName := filepath.Base(sdk.NameOf(c.mod))
	ctx := context.Background()
	if err := mgr.Start(ctx, labName, info.Lab.Services); err != nil {
		output.Error("Lab start failed: %v", err)
		return
	}
	output.Success("Lab %s started", labName)
}

func (c *Console) labStop() {
	mgr := c.requireLabMgr()
	if mgr == nil || !c.requireMod() {
		return
	}
	labName := filepath.Base(sdk.NameOf(c.mod))
	ctx := context.Background()
	if err := mgr.Stop(ctx, labName); err != nil {
		output.Error("Lab stop failed: %v", err)
		return
	}
	output.Success("Lab %s stopped", labName)
}

func (c *Console) labStatus() {
	mgr := c.requireLabMgr()
	if mgr == nil {
		return
	}
	ctx := context.Background()
	labs, err := mgr.Status(ctx)
	if err != nil {
		output.Error("Lab status failed: %v", err)
		return
	}
	if len(labs) == 0 {
		output.Warning("No labs running")
		return
	}
	output.Println()
	for _, l := range labs {
		output.Print("  %s\n", log.Cyan(l.Name))
		for _, svc := range l.Services {
			state := log.Green(svc.State)
			if svc.State != "running" {
				state = log.Yellow(svc.State)
			}
			ports := ""
			if svc.Ports != "" {
				ports = " " + log.Gray(svc.Ports)
			}
			output.Print("    %s  %s  %s%s\n",
				log.Pad(svc.Name, 20),
				log.Pad(state, 12),
				svc.Image,
				ports,
			)
		}
	}
	output.Println()
}

func (c *Console) labRun() {
	mgr := c.requireLabMgr()
	if mgr == nil || !c.requireMod() {
		return
	}
	info := c.mod.Info()
	if len(info.Lab.Services) == 0 {
		output.Warning("Module %s has no lab defined", sdk.NameOf(c.mod))
		return
	}

	labName := filepath.Base(sdk.NameOf(c.mod))
	ctx := context.Background()

	// Start lab if not already running.
	if !mgr.IsRunning(ctx, labName) {
		if err := mgr.Start(ctx, labName, info.Lab.Services); err != nil {
			output.Error("Lab start failed: %v", err)
			return
		}
	} else {
		output.Success("Lab %s already running", labName)
	}

	// Derive target from port bindings.
	target := mgr.Target(info.Lab.Services)

	// Phase 1: wait for TCP port to open.
	output.Status("Waiting for %s", target)
	if err := mgr.WaitReady(ctx, target, 120*time.Second); err != nil {
		output.Error("Lab not ready: %v", err)
		return
	}

	// Set TARGET before probing (Check needs it in params).
	c.setOpt("TARGET", target)
	output.Success("TARGET => %s", target)

	// Phase 2: if module has Check(), retry until app is truly ready.
	if checker, ok := c.mod.(sdk.Checker); ok {
		output.Status("Probing application readiness")
		params := c.buildParams()
		if err := mgr.WaitProbe(ctx, 120*time.Second, func() error {
			_, err := checker.Check(c.buildModuleRun(params, ""))
			return err
		}); err != nil {
			output.Error("Application not ready: %v", err)
			return
		}
	}
	output.Success("Lab ready at %s", target)

	// Auto-detect LHOST from Docker gateway if not set.
	if c.getOpt("LHOST") == "" {
		gw := mgr.DockerGateway()
		if gw == "" {
			output.Error("Set LHOST before running (container needs to reach your host)")
			return
		}
		c.setOpt("LHOST", gw)
		output.Success("LHOST => %s (docker gateway)", gw)
	}

	c.cmdExploit()
}
