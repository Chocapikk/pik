package console

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chzyer/readline"

	"github.com/Chocapikk/pik/pkg/c2"
	"github.com/Chocapikk/pik/pkg/c2/shell"
	"github.com/Chocapikk/pik/pkg/cmdstager"
	"github.com/Chocapikk/pik/sdk"
	pikhttp "github.com/Chocapikk/pik/pkg/http"
	"github.com/Chocapikk/pik/pkg/log"
	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/pkg/payload"
	"github.com/Chocapikk/pik/pkg/text"
)

var (
	promptBase  = log.Cyan("pik")
	promptArrow = log.Gray(" > ")
	optEmpty    = log.Gray("(not set)")
	optReq      = log.Red("yes")
	optNo       = log.Gray("no")
	divider     = log.Gray(strings.Repeat("\u2500", 70))
)

type option struct {
	Name     string
	Value    string
	Required bool
	Desc     string
	Advanced bool
}

// Console is the interactive REPL.
type Console struct {
	rl            *readline.Instance
	mod           sdk.Exploit
	options       []option
	activeBackend c2.Backend
}

// Run starts the interactive console.
func Run() error {
	output.Banner()

	cons := &Console{}
	if err := cons.initReadline(); err != nil {
		return err
	}
	defer cons.rl.Close()
	defer cons.shutdownBackend()

	for {
		line, err := cons.rl.Readline()
		if err != nil { // EOF or ctrl+D
			output.Println()
			return nil
		}
		if cons.exec(line) {
			return nil
		}
	}
}

// exec runs a single console line. Returns true if the console should exit.
func (c *Console) exec(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return false
	}

	parts := strings.Fields(line)
	command := strings.ToLower(parts[0])
	args := parts[1:]

	switch command {
	case "exit", "quit":
		return true
	case "help", "?":
		c.cmdHelp()
	case "list", "modules":
		c.cmdList()
	case "use":
		c.cmdUse(args)
	case "back":
		c.cmdBack()
	case "info":
		c.cmdInfo(args)
	case "show":
		c.cmdShow(args)
	case "set":
		c.cmdSet(args)
	case "unset":
		c.cmdUnset(args)
	case "check":
		c.cmdCheck()
	case "exploit", "run":
		c.cmdExploit()
	case "sessions":
		c.cmdSessions(args)
	case "kill":
		c.cmdKill(args)
	case "rank":
		c.cmdRank()
	case "resource":
		c.cmdResource(args)
	default:
		output.Error("Unknown command: %s (type 'help' for commands)", command)
	}
	return false
}

func (c *Console) initReadline() error {
	commands := []string{
		"use", "set", "unset", "show", "back", "info",
		"check", "exploit", "run", "list", "modules",
		"sessions", "kill", "resource",
		"help", "exit", "quit",
	}

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
			readline.PcItem("modules"),
		),
		readline.PcItem("info", readline.PcItemDynamic(func(line string) []string {
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

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          c.buildPrompt(),
		AutoComplete:    completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return err
	}
	c.rl = rl
	return nil
}

func (c *Console) buildPrompt() string {
	if c.mod != nil {
		return promptBase + " " + log.Red(sdk.NameOf(c.mod)) + promptArrow
	}
	return promptBase + promptArrow
}

func (c *Console) updatePrompt() {
	c.rl.SetPrompt(c.buildPrompt())
}

func (c *Console) optionNames() []string {
	names := make([]string, len(c.options))
	for i, opt := range c.options {
		names[i] = opt.Name
	}
	return names
}

func (c *Console) initOptions() {
	c.options = []option{
		{Name: "TARGET", Required: true, Desc: "Target URL/IP"},
	}

	for _, opt := range sdk.ResolveOptions(c.mod) {
		c.options = append(c.options, option{
			Name:     opt.Name,
			Value:    opt.Default,
			Required: opt.Required,
			Desc:     opt.Desc,
			Advanced: opt.Advanced,
		})
	}

	// Smart default for PAYLOAD based on platform
	if c.hasOpt("PAYLOAD") && c.getOpt("PAYLOAD") == "" {
		if defPayload := payload.DefaultPayload(c.mod.Info().Platform()); defPayload != nil {
			c.setOpt("PAYLOAD", defPayload.Name)
		}
	}
}

func (c *Console) hasOpt(name string) bool {
	for _, opt := range c.options {
		if strings.EqualFold(opt.Name, name) {
			return true
		}
	}
	return false
}

func (c *Console) getOpt(name string) string {
	for _, opt := range c.options {
		if strings.EqualFold(opt.Name, name) {
			return opt.Value
		}
	}
	return ""
}

func (c *Console) setOpt(name, value string) bool {
	for i := range c.options {
		if strings.EqualFold(c.options[i].Name, name) {
			c.options[i].Value = value
			return true
		}
	}
	return false
}

// buildParams creates a sdk.Params from all current option values.
func (c *Console) buildParams() sdk.Params {
	values := make(map[string]string, len(c.options))
	for _, opt := range c.options {
		if opt.Value != "" {
			values[strings.ToUpper(opt.Name)] = opt.Value
		}
	}
	return sdk.NewParams(context.Background(), values)
}

// --- Commands ---

func (c *Console) cmdHelp() {
	help := []struct{ cmd, desc string }{
		{"use <module>", "Select a module (fuzzy search if no args)"},
		{"back", "Deselect current module"},
		{"show options", "Show current module options"},
		{"show advanced", "Show advanced options"},
		{"show payloads", "List available payloads"},
		{"show modules", "List all modules"},
		{"set <OPT> <value>", "Set an option value"},
		{"unset <OPT>", "Clear an option value"},
		{"info [module]", "Show module details"},
		{"check", "Check if target is vulnerable"},
		{"exploit / run", "Run the exploit"},
		{"sessions", "List active sessions"},
		{"sessions <id>", "Interact with a session"},
		{"kill <id>", "Kill a session"},
		{"resource <file>", "Run commands from a .rc file"},
		{"list", "List all modules"},
		{"rank", "Contributor leaderboard"},
		{"exit / quit", "Exit the console"},
	}

	output.Println()
	for _, entry := range help {
		output.Print("  %-25s %s\n", log.Cyan(entry.cmd), log.Gray(entry.desc))
	}
	output.Println()
}

func (c *Console) cmdList() {
	modules := sdk.List()
	if len(modules) == 0 {
		output.Warning("No modules loaded")
		return
	}
	output.Println()
	output.Print("  %-20s  %-10s  %-35s  %s\n",
		log.UnderlineText("Name"),
		log.UnderlineText("Reliability"),
		log.UnderlineText("Description"),
		log.UnderlineText("CVEs"),
	)
	for _, mod := range modules {
		info := mod.Info()
		cves := strings.Join(info.CVEs(), ", ")
		if cves == "" {
			cves = "-"
		}
		output.Print("  %-20s  %-10s  %-35s  %s\n",
			log.Cyan(sdk.NameOf(mod)),
			reliabilityStyle(info.Reliability),
			info.Description,
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
	output.Print("  %-5s  %-20s  %-10s  %s\n",
		log.UnderlineText("#"),
		log.UnderlineText("Author"),
		log.UnderlineText("Modules"),
		log.UnderlineText("CVEs"),
	)
	for i, rank := range rankings {
		output.Print("  %-5s  %-20s  %-10s  %s\n",
			log.Cyan(fmt.Sprintf("%d", i+1)),
			log.White(rank.Name),
			log.Cyan(fmt.Sprintf("%d", rank.Modules)),
			log.Yellow(fmt.Sprintf("%d", rank.CVEs)),
		)
	}
	output.Println()
}

func (c *Console) cmdUse(args []string) {
	var name string

	if len(args) > 0 {
		name = args[0]
	}
	if name == "" {
		c.rl.Terminal.ExitRawMode()
		defer func() {
			c.rl.Terminal.EnterRawMode()
		}()

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

	c.mod = mod
	c.initOptions()
	c.updatePrompt()
	output.Success("Using %s - %s", sdk.NameOf(mod), mod.Info().Description)
}

func (c *Console) cmdBack() {
	c.mod = nil
	c.options = nil
	c.updatePrompt()
}

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
			output.Print("    %-12s %s\n", log.Gray(q.Engine+":"), q.URL())
		}
	}
	output.Println()
}

func (c *Console) cmdShow(args []string) {
	if len(args) == 0 {
		output.Error("Usage: show <options|payloads|modules>")
		return
	}

	switch strings.ToLower(args[0]) {
	case "options":
		c.showOptions(false)
	case "advanced":
		c.showOptions(true)
	case "payloads":
		c.showPayloads()
	case "modules":
		c.cmdList()
	default:
		output.Error("Unknown: show %s (try: options, advanced, payloads, modules)", args[0])
	}
}

func (c *Console) showOptions(advanced bool) {
	if c.mod == nil {
		output.Error("No module selected")
		return
	}

	label := "Options"
	if advanced {
		label = "Advanced options"
	}

	output.Println()
	output.Print("  %s: %s\n", label, log.Cyan(sdk.NameOf(c.mod)))
	output.Println(divider)
	output.Print("  %-20s  %-35s  %-8s  %s\n",
		log.UnderlineText("Option"),
		log.UnderlineText("Value"),
		log.UnderlineText("Required"),
		log.UnderlineText("Description"),
	)

	for _, opt := range c.options {
		if opt.Advanced != advanced {
			continue
		}
		displayVal := optEmpty
		if opt.Value != "" {
			displayVal = log.White(opt.Value)
		}
		required := optNo
		if opt.Required {
			required = optReq
		}
		output.Print("  %-20s  %-35s  %-8s  %s\n",
			log.Cyan(opt.Name),
			displayVal,
			required,
			log.Gray(opt.Desc),
		)
	}
	output.Println()
}

func (c *Console) showPayloads() {
	platform := ""
	if c.mod != nil {
		platform = c.mod.Info().Platform()
	}

	payloads := payload.ListForPlatform(platform)
	if len(payloads) == 0 {
		output.Warning("No payloads available")
		return
	}

	output.Println()
	output.Print("  %-35s  %s\n",
		log.UnderlineText("Payload"),
		log.UnderlineText("Description"),
	)
	current := c.getOpt("PAYLOAD")
	for _, pl := range payloads {
		marker := "  "
		displayName := log.Cyan(pl.Name)
		if pl.Name == current {
			marker = log.Green("* ")
			displayName = log.Green(pl.Name)
		}
		output.Print("%s%-35s  %s\n", marker, displayName, log.Gray(pl.Description))
	}
	output.Println()
}

func (c *Console) cmdSet(args []string) {
	if c.mod == nil {
		output.Error("No module selected")
		return
	}

	if len(args) < 2 {
		output.Error("Usage: set <option> <value>")
		return
	}

	name := strings.ToUpper(args[0])
	value := strings.Join(args[1:], " ")

	if name == "PAYLOAD" && (value == "" || value == "?") {
		c.selectPayload()
		return
	}

	if !c.setOpt(name, value) {
		output.Error("Unknown option: %s", name)
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

func (c *Console) cmdCheck() {
	if c.mod == nil {
		output.Error("No module selected")
		return
	}

	checker, ok := c.mod.(sdk.Checker)
	if !ok {
		output.Warning("Module %s does not support check", sdk.NameOf(c.mod))
		return
	}

	target := c.getOpt("TARGET")
	if target == "" {
		output.Error("TARGET not set")
		return
	}

	params := c.buildParams()
	run := c.buildModuleRun(params, "")
	output.Status("Checking %s", target)
	result, err := checker.Check(run)
	if err != nil {
		output.Error("Check failed: %v", err)
		return
	}
	if !result.Code.IsVulnerable() {
		output.Warning("%s - %s%s", target, result.Code, result.FormatReason())
		return
	}
	output.Success("%s - %s%s", target, result.Code, result.FormatReason())
}

func (c *Console) cmdExploit() {
	if c.mod == nil {
		output.Error("No module selected")
		return
	}

	target := c.getOpt("TARGET")
	if target == "" {
		output.Error("TARGET not set")
		return
	}
	lhost := c.getOpt("LHOST")
	if lhost == "" {
		output.Error("LHOST not set")
		return
	}

	params := c.buildParams()

	payloadName := c.getOpt("PAYLOAD")
	selectedPayload := payload.GetPayload(payloadName)
	if selectedPayload == nil {
		output.Error("Payload %q not found", payloadName)
		return
	}
	payloadCmd := selectedPayload.Generate(lhost, params.Lport())

	backend := c.ensureBackend(lhost, params.Lport())
	if backend == nil {
		return
	}

	if checker, ok := c.mod.(sdk.Checker); ok {
		output.Status("Checking %s", target)
		checkRun := c.buildModuleRun(params, "")
		result, err := checker.Check(checkRun)
		if err != nil {
			output.Error("Check failed: %v", err)
			return
		}
		if !result.Code.IsVulnerable() {
			output.Warning("%s - %s", target, result.Code)
			return
		}
		output.Success("%s - %s", target, result.Code)
	}

	// CmdStager path: module supports chunked delivery AND backend provides raw implant.
	if c.tryCmdStager(target, params, backend) {
		return
	}

	// Single-shot path.
	output.Status("Exploiting %s", target)
	exploitRun := c.buildModuleRun(params, payloadCmd)
	if err := c.mod.Exploit(exploitRun); err != nil {
		output.Error("Exploit failed: %v", err)
		return
	}

	sessionTimeout := time.Duration(params.IntOr("WAITSESSION", 60)) * time.Second
	output.Status("Waiting for session...")
	if err := backend.WaitForSession(sessionTimeout); err != nil {
		output.Error("No session received: %v", err)
	}
}

// tryCmdStager attempts CmdStager delivery if both module and backend support it.
// Returns true if it handled the exploit (success or failure), false to fall through.
func (c *Console) tryCmdStager(target string, params sdk.Params, backend c2.Backend) bool {
	if _, ok := c.mod.(sdk.CmdStager); !ok {
		return false
	}
	gen, ok := backend.(c2.ImplantGenerator)
	if !ok {
		return false
	}

	binary, err := gen.GenerateImplant(c.mod.Info().Platform(), params.Arch())
	if err != nil {
		output.Error("Implant generation failed: %v", err)
		return true
	}

	tempPath := params.Get("CMDSTAGER_TEMPPATH")
	if tempPath == "" {
		tempPath = fmt.Sprintf("/tmp/.%s", text.RandText(6))
	}

	flavor := cmdstager.Flavor(params.GetOr("CMDSTAGER", string(cmdstager.DefaultFlavor)))
	opts := cmdstager.Options{
		TempPath: tempPath,
		LineMax:  params.IntOr("CMDSTAGER_LINEMAX", 2047),
	}

	commands, err := cmdstager.Generate(binary, flavor, opts)
	if err != nil {
		output.Error("CmdStager failed: %v", err)
		return true
	}

	output.InfoBox("CmdStager",
		"Target", target,
		"Flavor", string(flavor),
		"Implant", output.HumanSize(len(binary)),
		"Chunks", fmt.Sprintf("%d commands", len(commands)),
		"Drop path", tempPath,
	)

	stagerRun := c.buildModuleRun(params, "")
	stagerRun.SetCommands(commands)

	output.Status("Exploiting %s", target)
	if err := c.mod.Exploit(stagerRun); err != nil {
		output.Error("Exploit failed: %v", err)
		return true
	}

	sessionTimeout := time.Duration(params.IntOr("WAITSESSION", 60)) * time.Second
	output.Status("Waiting for session...")
	if err := backend.WaitForSession(sessionTimeout); err != nil {
		output.Error("No session received: %v", err)
	}
	return true
}

// ensureBackend returns the active backend, setting up a new one if needed.
func (c *Console) ensureBackend(lhost string, lport int) c2.Backend {
	if c.activeBackend != nil {
		return c.activeBackend
	}
	backend := c.resolveC2()
	if backend == nil {
		return nil
	}
	if err := backend.Setup(lhost, lport); err != nil {
		output.Error("C2 setup failed: %v", err)
		return nil
	}
	c.activeBackend = backend
	return backend
}

func (c *Console) cmdSessions(args []string) {
	handler := c.sessionHandler()
	if handler == nil {
		output.Warning("No active listener with session support")
		return
	}

	if len(args) > 0 {
		id, ok := c.parseSessionID(args[0])
		if !ok {
			return
		}
		if err := handler.Interact(id); err != nil {
			output.Error("%v", err)
		}
		return
	}

	sessions := handler.Sessions()
	if len(sessions) == 0 {
		output.Warning("No active sessions")
		return
	}

	output.Println()
	output.Print("  %-6s  %-25s  %s\n",
		log.UnderlineText("ID"),
		log.UnderlineText("Remote Address"),
		log.UnderlineText("Opened"),
	)
	for _, sess := range sessions {
		output.Print("  %-6s  %-25s  %s\n",
			log.Cyan(strconv.Itoa(sess.ID)),
			log.White(sess.RemoteAddr),
			log.Gray(sess.CreatedAt.Format("15:04:05")),
		)
	}
	output.Println()
}

func (c *Console) cmdKill(args []string) {
	if len(args) == 0 {
		output.Error("Usage: kill <session_id>")
		return
	}

	handler := c.sessionHandler()
	if handler == nil {
		output.Warning("No active listener with session support")
		return
	}

	id, ok := c.parseSessionID(args[0])
	if !ok {
		return
	}
	if err := handler.Kill(id); err != nil {
		output.Error("%v", err)
	}
}

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

func (c *Console) parseSessionID(raw string) (int, bool) {
	id, err := strconv.Atoi(raw)
	if err != nil {
		output.Error("Invalid session ID: %s", raw)
		return 0, false
	}
	return id, true
}

func (c *Console) shutdownBackend() {
	if c.activeBackend != nil {
		_ = c.activeBackend.Shutdown()
		c.activeBackend = nil
	}
}

func (c *Console) sessionHandler() c2.SessionHandler {
	if c.activeBackend == nil {
		return nil
	}
	handler, ok := c.activeBackend.(c2.SessionHandler)
	if !ok {
		return nil
	}
	return handler
}

func reliabilityStyle(rel sdk.Reliability) string {
	switch {
	case rel >= sdk.Certain:
		return log.Green(rel.String())
	case rel >= sdk.VeryReliable:
		return log.Cyan(rel.String())
	case rel >= sdk.Reliable:
		return log.Blue(rel.String())
	case rel >= sdk.Typical:
		return log.White(rel.String())
	case rel >= sdk.Difficult:
		return log.Yellow(rel.String())
	default:
		return log.Red(rel.String())
	}
}

func checkSupportStr(m sdk.Exploit) string {
	if sdk.CanCheck(m) {
		return log.Green("yes")
	}
	return log.Gray("no")
}

func (c *Console) warnSSLPort(changed string) {
	if changed != "SSL" && changed != "RPORT" {
		return
	}
	if !c.hasOpt("SSL") || !c.hasOpt("RPORT") {
		return
	}
	ssl := strings.EqualFold(c.getOpt("SSL"), "true")
	rport := c.getOpt("RPORT")
	switch {
	case ssl && rport == "80":
		output.Warning("SSL is enabled but RPORT is 80 - did you mean 443?")
	case !ssl && rport == "443":
		output.Warning("RPORT is 443 but SSL is disabled - consider 'set SSL true'")
	}
}

// buildModuleRun creates a *sdk.Context from Params, wiring HTTP, logging, and payload helpers.
func (c *Console) buildModuleRun(params sdk.Params, payloadCmd string) *sdk.Context {
	run := sdk.NewContext(params.Map(), payloadCmd)
	run.StatusFn = output.Status
	run.SuccessFn = output.Success
	run.ErrorFn = output.Error
	run.WarningFn = output.Warning
	run.Base64BashFn = payload.Base64Bash
	run.CommentFn = payload.CommentTrail
	run.RandTextFn = text.RandText

	httpRun := pikhttp.FromModule(params)
	run.SendFn = func(req sdk.Request) (*sdk.Response, error) {
		httpReq := pikhttp.Request{
			Method:      req.Method,
			Path:        req.Path,
			Query:       url.Values(req.Query),
			Form:        url.Values(req.Form),
			ContentType: req.ContentType,
			Headers:     req.Headers,
			Timeout:     time.Duration(req.Timeout) * time.Second,
			NoRedirect:  req.NoRedirect,
		}
		resp, err := httpRun.Send(httpReq)
		if err != nil {
			return nil, err
		}
		modResp := &sdk.Response{
			StatusCode: resp.StatusCode,
			Body:       resp.Body,
		}
		modResp.SetContainsFn(resp.ContainsAny)
		return modResp, nil
	}

	return run
}

func (c *Console) resolveC2() c2.Backend {
	backend := c2.Resolve(c.getOpt("C2"), c.getOpt("C2CONFIG"))
	if backend == nil {
		return shell.New()
	}
	return backend
}
