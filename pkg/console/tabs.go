package console

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Chocapikk/pik/pkg/log"
	"github.com/Chocapikk/pik/sdk"
)

// --- Tab types ---

type tab int

const (
	tabBrowse tab = iota
	tabConfig
	tabSessions
	tabCount
)

var tabLabels = []string{"Browse", "Config", "Sessions"}

// --- Options panel (Config tab) ---

// --- Browse search bar ---

type browseSearch struct {
	input   textinput.Model
	active  bool
}

func newBrowseSearch() browseSearch {
	ti := textinput.New()
	ti.Prompt = log.Amber("/ ")
	ti.Placeholder = "Search modules..."
	ti.CharLimit = 100
	return browseSearch{input: ti}
}

// --- Options panel (Config tab) ---

type optionsPanel struct {
	cursor        int
	editing       bool
	editor        textinput.Model
	labMenuOpen   bool
	labMenuCursor int
	showAdvanced  bool
}

func newOptionsPanel() optionsPanel {
	editor := textinput.New()
	editor.Prompt = ""
	editor.CharLimit = 0
	return optionsPanel{editor: editor}
}

// --- Tab bar rendering ---

func (m consoleModel) renderTabBar() string {
	var parts []string
	for i, label := range tabLabels {
		shortcut := fmt.Sprintf("F%d:", i+1)
		if tab(i) == m.activeTab && m.tabBarFocused {
			// Focused tab bar + active tab: inverse style
			parts = append(parts, log.Style(log.BoldAmber+"\x1b[7m", " "+shortcut+label+" "))
		} else if tab(i) == m.activeTab {
			parts = append(parts, log.Amber(" "+shortcut+label+" "))
		} else {
			parts = append(parts, log.White(" "+shortcut+label+" "))
		}
	}
	bar := strings.Join(parts, log.Amber("│"))
	pad := m.width - log.VisualLen(bar)
	if pad > 0 {
		bar += strings.Repeat(" ", pad)
	}
	return bar
}

// --- Tab content rendering ---

func (m consoleModel) renderActiveTab() string {
	h := m.tabHeight()
	switch m.activeTab {
	case tabBrowse:
		return m.renderBrowseTab(h)
	case tabConfig:
		return m.renderConfigTab(h)
	case tabSessions:
		return m.renderSessionsTab(h)
	}
	return ""
}

// --- Browse tab ---

func (m consoleModel) renderBrowseTab(h int) string {
	searchLine := m.search.input.View()
	// Resize list to fit below search bar
	m.browser.SetSize(m.width, h-1)
	listView := m.browser.View()
	return padToHeight(searchLine+"\n"+listView, h)
}

func newBrowser(w, h int) list.Model {
	modules := sdk.List()
	items := make([]list.Item, len(modules))
	for i, mod := range modules {
		info := mod.Info()
		cves := strings.Join(info.CVEs(), ", ")
		desc := info.Title()
		if cves != "" {
			desc += " - " + cves
		}
		items[i] = fuzzyItem{name: sdk.NameOf(mod), desc: desc}
	}

	delegate := newFuzzyDelegate()
	l := list.New(items, delegate, w, h)
	l.SetShowTitle(false)
	l.SetFilteringEnabled(false)
	l.SetShowStatusBar(true)
	l.SetShowHelp(false)
	l.FilterInput.Prompt = "/ "
	l.FilterInput.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	l.DisableQuitKeybindings()
	return l
}

// --- Config tab ---

func (m consoleModel) renderConfigTab(h int) string {
	var lines []string

	if m.console.mod == nil {
		lines = append(lines, "")
		lines = append(lines, log.White("  No module selected"))
		lines = append(lines, "")
		lines = append(lines, log.White("  Select a module in Browse (F1) or type 'use <module>'"))
		return padToHeight(strings.Join(lines, "\n"), h)
	}

	lines = append(lines, "")
	lines = append(lines, "  "+log.Amber(sdk.NameOf(m.console.mod))+"  "+log.White(m.console.mod.Info().Title()))
	lines = append(lines, "")

	// Option rows
	nameW := 16
	valW := max(m.width-nameW-8, 12)

	visibleIdx := 0
	for _, opt := range m.console.options {
		if opt.Advanced && !m.opts.showAdvanced {
			continue
		}
		prefix := "   "
		if m.activeTab == tabConfig && visibleIdx == m.opts.cursor {
			prefix = log.Amber(" > ")
		}

		val := opt.Value
		if m.opts.editing && visibleIdx == m.opts.cursor {
			val = m.opts.editor.View()
		} else if val == "" {
			val = log.White("(not set)")
		} else {
			val = log.White(val)
		}

		req := " "
		if opt.Required && opt.Value == "" {
			req = log.Red("*")
		} else if opt.Required {
			req = log.Green("*")
		}

		line := fmt.Sprintf("%s%s%s %s",
			prefix,
			log.Pad(log.Cyan(opt.Name), nameW),
			log.Pad(val, valW),
			req,
		)
		lines = append(lines, line)
		visibleIdx++
	}

	// Action buttons
	btns := m.actionButtons()
	if len(btns) > 0 {
		lines = append(lines, "")
		var rendered []string
		for i, btn := range btns {
			btnIdx := visibleIdx + i
			style := btnStyle
			if m.activeTab == tabConfig && m.opts.cursor == btnIdx {
				style = btnFocusedStyle
			}
			rendered = append(rendered, style.Render(btn))
		}
		lines = append(lines, "  "+strings.Join(rendered, "  "))

		// Lab submenu
		if m.opts.labMenuOpen {
			for i, item := range labSubMenu {
				prefix := "     "
				if i == m.opts.labMenuCursor {
					prefix = "   " + log.Amber("> ")
				}
				lines = append(lines, prefix+log.White(item))
			}
		}
	}

	// Scroll to keep cursor visible
	// Header takes 3 lines, buttons take ~2 lines
	content := strings.Join(lines, "\n")
	contentLines := strings.Split(content, "\n")

	// Find which line the cursor is on (3 header lines + cursor position)
	cursorLine := 3 + m.opts.cursor
	if m.opts.labMenuOpen {
		cursorLine = len(contentLines) - 1
	}

	// If content fits, no scroll needed
	if len(contentLines) <= h {
		return padToHeight(content, h)
	}

	// Scroll window so cursor is visible
	start := 0
	if cursorLine >= h-1 {
		start = cursorLine - h + 2
	}
	end := start + h
	if end > len(contentLines) {
		end = len(contentLines)
		start = end - h
		if start < 0 {
			start = 0
		}
	}

	return strings.Join(contentLines[start:end], "\n")
}

func (m consoleModel) visibleOptionCount() int {
	count := 0
	for _, opt := range m.console.options {
		if !opt.Advanced || m.opts.showAdvanced {
			count++
		}
	}
	return count
}

func (m consoleModel) visibleOptionAt(idx int) *option {
	cur := 0
	for i := range m.console.options {
		if m.console.options[i].Advanced && !m.opts.showAdvanced {
			continue
		}
		if cur == idx {
			return &m.console.options[i]
		}
		cur++
	}
	return nil
}

func (m consoleModel) actionButtons() []string {
	if m.console.mod == nil {
		return nil
	}
	var btns []string
	if _, ok := m.console.mod.(sdk.Checker); ok {
		btns = append(btns, "Check")
	}
	btns = append(btns, "Exploit")
	if len(m.console.mod.Info().Lab.Services) > 0 {
		btns = append(btns, "Lab")
	}
	if m.opts.showAdvanced {
		btns = append(btns, "Hide Advanced")
	} else {
		btns = append(btns, "Show Advanced")
	}
	return btns
}

var labSubMenu = []string{"Start", "Stop", "Status", "Run"}

func (m consoleModel) sessionInButtons() bool {
	return m.sessionBtnMode
}

var (
	btnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("214")).
			Padding(0, 1).Bold(true)
	btnFocusedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("14")).
			Padding(0, 1).Bold(true)
)

// --- Sessions tab ---

func (m consoleModel) renderSessionsTab(h int) string {
	handler := m.console.sessionHandler()
	if handler == nil {
		return padToHeight(log.White("  No active listener"), h)
	}

	sessions := handler.Sessions()
	if len(sessions) == 0 {
		return padToHeight(log.White("  No active sessions"), h)
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s  %s",
		log.Pad(log.UnderlineText("ID"), 6),
		log.Pad(log.UnderlineText("Remote Address"), 25),
		log.UnderlineText("Opened"),
	))

	for i, sess := range sessions {
		prefix := "  "
		if i == m.sessionCursor && !m.sessionInButtons() {
			prefix = log.Amber("> ")
		}
		lines = append(lines, fmt.Sprintf("%s%s  %s  %s",
			prefix,
			log.Pad(log.Cyan(strconv.Itoa(sess.ID)), 6),
			log.Pad(log.White(sess.RemoteAddr), 25),
			log.White(sess.CreatedAt.Format("15:04:05")),
		))
	}

	// Action buttons (only visible when a session is focused)
	if m.sessionBtnMode {
		lines = append(lines, "")
		sessBtns := []string{"Interact", "Kill"}
		var rendered []string
		for i, btn := range sessBtns {
			style := btnStyle
			if m.sessionBtnCursor == i {
				style = btnFocusedStyle
			}
			rendered = append(rendered, style.Render(btn))
		}
		lines = append(lines, "  "+strings.Join(rendered, "  "))
	}

	return padToHeight(strings.Join(lines, "\n"), h)
}

// --- Tab key handling ---

func (m consoleModel) updateBrowseTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Search bar is always active: runes go to search, navigation goes to list
	switch msg.Type {
	case tea.KeyEnter:
		if item, ok := m.browser.SelectedItem().(fuzzyItem); ok {
			m.console.cmdUseByName(item.name)
			m.refreshConfig()
			m.activeTab = tabConfig
			m.search.input.SetValue("")
			m.browser.ResetFilter()
		}
		return m, nil
	case tea.KeyEscape:
		m.search.input.SetValue("")
		m.browser.ResetFilter()
		return m, nil
	case tea.KeyRunes:
		// Type in search bar
		var cmd tea.Cmd
		m.search.input, cmd = m.search.input.Update(msg)
		// Apply filter to list
		query := m.search.input.Value()
		if query != "" {
			m.browser.SetFilteringEnabled(true)
			// Use the list's built-in filter by simulating key presses
			// Actually, filter the items manually
			m.filterBrowser(query)
		} else {
			m.resetBrowserFilter()
		}
		return m, cmd
	case tea.KeyBackspace:
		var cmd tea.Cmd
		m.search.input, cmd = m.search.input.Update(msg)
		query := m.search.input.Value()
		if query != "" {
			m.filterBrowser(query)
		} else {
			m.resetBrowserFilter()
		}
		return m, cmd
	}

	// Up/Down/PgUp/PgDown go to list navigation
	var cmd tea.Cmd
	m.browser, cmd = m.browser.Update(msg)
	return m, cmd
}

func (m *consoleModel) filterBrowser(query string) {
	query = strings.ToLower(query)
	modules := sdk.List()
	var items []list.Item
	for _, mod := range modules {
		info := mod.Info()
		name := strings.ToLower(sdk.NameOf(mod))
		cves := strings.ToLower(strings.Join(info.CVEs(), " "))
		desc := strings.ToLower(info.Title())
		if strings.Contains(name, query) || strings.Contains(cves, query) || strings.Contains(desc, query) {
			cvesStr := strings.Join(info.CVEs(), ", ")
			d := info.Title()
			if cvesStr != "" {
				d += " - " + cvesStr
			}
			items = append(items, fuzzyItem{name: sdk.NameOf(mod), desc: d})
		}
	}
	m.browser.SetItems(items)
}

func (m *consoleModel) resetBrowserFilter() {
	modules := sdk.List()
	items := make([]list.Item, len(modules))
	for i, mod := range modules {
		info := mod.Info()
		cves := strings.Join(info.CVEs(), ", ")
		desc := info.Title()
		if cves != "" {
			desc += " - " + cves
		}
		items[i] = fuzzyItem{name: sdk.NameOf(mod), desc: desc}
	}
	m.browser.SetItems(items)
}

func (m consoleModel) updateConfigTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.opts.editing {
		return m.updateConfigEditing(msg)
	}

	// Lab submenu open: handle its navigation
	if m.opts.labMenuOpen {
		switch msg.Type {
		case tea.KeyUp:
			if m.opts.labMenuCursor > 0 {
				m.opts.labMenuCursor--
			}
		case tea.KeyDown:
			if m.opts.labMenuCursor < len(labSubMenu)-1 {
				m.opts.labMenuCursor++
			}
		case tea.KeyEnter:
			cmd := "lab " + strings.ToLower(labSubMenu[m.opts.labMenuCursor])
			m.opts.labMenuOpen = false
			c := m.console
			m.busy = true
			return m, func() tea.Msg {
				c.exec(cmd)
				return commandDoneMsg{}
			}
		case tea.KeyEscape:
			m.opts.labMenuOpen = false
		}
		return m, nil
	}

	optCount := m.visibleOptionCount()
	btnCount := len(m.actionButtons())
	maxIdx := optCount + btnCount - 1
	if maxIdx < 0 {
		return m, nil
	}

	inButtons := m.opts.cursor >= optCount

	switch msg.Type {
	case tea.KeyUp:
		if inButtons {
			m.opts.cursor = optCount - 1
		} else if m.opts.cursor > 0 {
			m.opts.cursor--
		}
	case tea.KeyDown:
		if !inButtons && m.opts.cursor < optCount-1 {
			m.opts.cursor++
		} else if !inButtons && btnCount > 0 {
			m.opts.cursor = optCount
		}
	case tea.KeyLeft:
		if inButtons && m.opts.cursor > optCount {
			m.opts.cursor--
		}
	case tea.KeyRight:
		if inButtons && m.opts.cursor < maxIdx {
			m.opts.cursor++
		}
	case tea.KeyEnter:
		if !inButtons {
			opt := m.visibleOptionAt(m.opts.cursor)
			if opt == nil {
				return m, nil
			}
			if strings.EqualFold(opt.Name, "PAYLOAD") {
				m.console.cmdSet([]string{"PAYLOAD"})
				return m, nil
			}
			m.opts.editing = true
			m.opts.editor.SetValue(opt.Value)
			m.opts.editor.Focus()
			m.opts.editor.CursorEnd()
		} else {
			return m, m.handleActionButton(m.opts.cursor - optCount)
		}
	}
	return m, nil
}

func (m consoleModel) updateConfigEditing(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		opt := m.visibleOptionAt(m.opts.cursor)
		if opt != nil {
			m.console.setOpt(opt.Name, m.opts.editor.Value())
		}
		m.opts.editing = false
		m.opts.editor.Blur()
		return m, nil
	case tea.KeyEscape:
		m.opts.editing = false
		m.opts.editor.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.opts.editor, cmd = m.opts.editor.Update(msg)
	return m, cmd
}

func (m consoleModel) updateSessionsTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	handler := m.console.sessionHandler()
	if handler == nil {
		return m, nil
	}
	sessions := handler.Sessions()
	if len(sessions) == 0 {
		return m, nil
	}

	switch msg.Type {
	case tea.KeyUp:
		if m.sessionBtnMode {
			m.sessionBtnMode = false
		} else if m.sessionCursor > 0 {
			m.sessionCursor--
		}
	case tea.KeyDown:
		if !m.sessionBtnMode && m.sessionCursor < len(sessions)-1 {
			m.sessionCursor++
		} else if !m.sessionBtnMode {
			m.sessionBtnMode = true
			m.sessionBtnCursor = 0
		}
	case tea.KeyLeft:
		if m.sessionBtnMode && m.sessionBtnCursor > 0 {
			m.sessionBtnCursor--
		}
	case tea.KeyRight:
		if m.sessionBtnMode && m.sessionBtnCursor < 1 {
			m.sessionBtnCursor++
		}
	case tea.KeyEnter:
		if m.sessionCursor >= len(sessions) {
			return m, nil
		}
		sess := sessions[m.sessionCursor]
		if m.sessionBtnMode {
			switch m.sessionBtnCursor {
			case 0: // Interact
				if m.console.program != nil {
					go m.console.program.Send(sessionInteractMsg{id: sess.ID})
				}
				m.sessionBtnMode = false
			case 1: // Kill
				handler.Kill(sess.ID)
				if m.sessionCursor > 0 && m.sessionCursor >= len(sessions)-1 {
					m.sessionCursor--
				}
				m.sessionBtnMode = false
			}
		} else {
			// Enter on session row -> focus it, show buttons
			m.sessionBtnMode = true
			m.sessionBtnCursor = 0
		}
	case tea.KeyEscape:
		if m.sessionBtnMode {
			m.sessionBtnMode = false
		}
	}
	return m, nil
}

func (m *consoleModel) handleActionButton(idx int) tea.Cmd {
	btns := m.actionButtons()
	if idx < 0 || idx >= len(btns) {
		return nil
	}
	// Inject the command as text, same as if user typed it
	var cmd string
	switch btns[idx] {
	case "Check":
		cmd = "check"
	case "Exploit":
		cmd = "exploit"
	case "Lab":
		m.opts.labMenuOpen = true
		m.opts.labMenuCursor = 0
		return nil
	case "Show Advanced", "Hide Advanced":
		m.opts.showAdvanced = !m.opts.showAdvanced
		m.opts.cursor = 0
		return nil
	}
	if cmd == "" {
		return nil
	}
	// Simulate typing the command
	c := m.console
	m.busy = true
	return func() tea.Msg {
		c.exec(cmd)
		return commandDoneMsg{}
	}
}

func (m *consoleModel) refreshConfig() {
	m.opts.cursor = 0
	m.opts.editing = false
	m.opts.editor.Blur()
}

// --- Helpers ---

func padToHeight(content string, h int) string {
	lines := strings.Split(content, "\n")
	for len(lines) < h {
		lines = append(lines, "")
	}
	if len(lines) > h {
		lines = lines[:h]
	}
	return strings.Join(lines, "\n")
}
