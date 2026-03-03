package console

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Chocapikk/pik/pkg/log"
	"github.com/Chocapikk/pik/pkg/payload"
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

// --- Browse search bar ---

type browseSearch struct {
	input textinput.Model
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
	table         table.Model
	editing       bool
	editor        textinput.Model
	showAdvanced  bool
	labMenuOpen   bool
	labMenuCursor int
	btnMode       bool
	btnCursor     int
}

func newOptionsPanel() optionsPanel {
	editor := textinput.New()
	editor.Prompt = ""
	editor.CharLimit = 0

	cols := []table.Column{
		{Title: "Option", Width: 16},
		{Title: "Value", Width: 30},
		{Title: "Req", Width: 3},
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithRows([]table.Row{}),
		table.WithHeight(10),
		table.WithFocused(false),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.Foreground(lipgloss.Color("214")).Bold(true)
	s.Selected = s.Selected.Foreground(lipgloss.Color("214")).Bold(true)
	t.SetStyles(s)

	return optionsPanel{table: t, editor: editor}
}

// --- Tab bar rendering ---

func (m consoleModel) renderTabBar() string {
	var parts []string
	for i, label := range tabLabels {
		shortcut := fmt.Sprintf("F%d:", i+1)
		if tab(i) == m.activeTab && m.tabBarFocused {
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
	m.browser.SetSize(m.width, h-1)
	return padToHeight(searchLine+"\n"+m.browser.View(), h)
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
	l.DisableQuitKeybindings()
	return l
}

// --- Config tab ---

func (m consoleModel) renderConfigTab(h int) string {
	if m.console.mod == nil {
		return padToHeight("\n"+log.White("  No module selected")+"\n\n"+log.White("  Select a module in Browse (F1) or type 'use <module>'"), h)
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, "  "+log.Amber(sdk.NameOf(m.console.mod))+"  "+log.White(m.console.mod.Info().Title()))
	lines = append(lines, "")

	// Options table
	m.opts.table.SetHeight(h - 6) // header(3) + buttons(~3)
	lines = append(lines, m.opts.table.View())

	// Action buttons
	btns := m.actionButtons()
	if len(btns) > 0 {
		lines = append(lines, "")
		var rendered []string
		for i, btn := range btns {
			style := btnStyle
			if m.opts.btnMode && i == m.opts.btnCursor {
				style = btnFocusedStyle
			}
			rendered = append(rendered, style.Render(btn))
		}
		lines = append(lines, "  "+strings.Join(rendered, "  "))

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

	return padToHeight(strings.Join(lines, "\n"), h)
}

func (m *consoleModel) refreshOptionsTable() {
	var rows []table.Row
	for _, opt := range m.console.options {
		if opt.Advanced && !m.opts.showAdvanced {
			continue
		}
		val := opt.Value
		if val == "" {
			val = "(not set)"
		}
		req := ""
		if opt.Required {
			req = "*"
		}
		rows = append(rows, table.Row{opt.Name, val, req})
	}
	m.opts.table.SetRows(rows)

	cols := []table.Column{
		{Title: "Option", Width: 16},
		{Title: "Value", Width: max(m.width-24, 12)},
		{Title: "", Width: 3},
	}
	m.opts.table.SetColumns(cols)
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

var (
	btnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("214")).
			Padding(0, 1).Bold(true)
	btnFocusedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("14")).
			Padding(0, 1).Bold(true)
	labSubMenu = []string{"Start", "Stop", "Status", "Run"}
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
		if i == m.sessionCursor && !m.sessionBtnMode {
			prefix = log.Amber("> ")
		}
		lines = append(lines, fmt.Sprintf("%s%s  %s  %s",
			prefix,
			log.Pad(log.Cyan(strconv.Itoa(sess.ID)), 6),
			log.Pad(log.White(sess.RemoteAddr), 25),
			log.White(sess.CreatedAt.Format("15:04:05")),
		))
	}

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
	switch msg.Type {
	case tea.KeyEnter:
		if item, ok := m.browser.SelectedItem().(fuzzyItem); ok {
			m.console.cmdUseByName(item.name)
			m.refreshConfig()
			m.activeTab = tabConfig
			m.search.input.SetValue("")
			m.resetBrowserFilter()
		}
		return m, nil
	case tea.KeyEscape:
		m.search.input.SetValue("")
		m.resetBrowserFilter()
		return m, nil
	case tea.KeyRunes, tea.KeyBackspace:
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
	m.opts.table.Focus()
	if m.opts.editing {
		return m.updateConfigEditing(msg)
	}

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
	btns := m.actionButtons()
	inButtons := m.opts.table.Cursor() >= optCount-1 && m.opts.btnMode

	switch msg.Type {
	case tea.KeyDown:
		if !m.opts.btnMode && m.opts.table.Cursor() >= optCount-1 && len(btns) > 0 {
			m.opts.btnMode = true
			m.opts.btnCursor = 0
			return m, nil
		}
		if m.opts.btnMode {
			return m, nil
		}
		var cmd tea.Cmd
		m.opts.table, cmd = m.opts.table.Update(msg)
		return m, cmd
	case tea.KeyUp:
		if m.opts.btnMode {
			m.opts.btnMode = false
			return m, nil
		}
		var cmd tea.Cmd
		m.opts.table, cmd = m.opts.table.Update(msg)
		return m, cmd
	case tea.KeyLeft:
		if m.opts.btnMode && m.opts.btnCursor > 0 {
			m.opts.btnCursor--
		}
		return m, nil
	case tea.KeyRight:
		if m.opts.btnMode && m.opts.btnCursor < len(btns)-1 {
			m.opts.btnCursor++
		}
		return m, nil
	case tea.KeyEnter:
		if m.opts.btnMode {
			_ = inButtons
			return m, m.handleActionButton(m.opts.btnCursor)
		}
		cursor := m.opts.table.Cursor()
		if cursor < m.visibleOptionCount() {
			opt := m.visibleOptionAt(cursor)
			if opt == nil {
				return m, nil
			}
			if strings.EqualFold(opt.Name, "PAYLOAD") {
				// Build payload fuzzy select directly (no Send, avoids deadlock)
				platform := ""
				if m.console.mod != nil {
					platform = m.console.mod.Info().Platform()
				}
				payloads := payload.ListForPlatform(platform)
				items := make([]fuzzyItem, len(payloads))
				for i, pl := range payloads {
					items[i] = fuzzyItem{name: pl.Name, desc: pl.Description}
				}
				m.mode = modeFuzzy
				m.fuzzy = newFuzzyModel("Select payload", items)
				m.fuzzy.context = "payload"
				return m, nil
			}
			m.opts.editing = true
			m.opts.editor.SetValue(opt.Value)
			m.opts.editor.Focus()
			m.opts.editor.CursorEnd()
		}
		return m, nil
	}
	return m, nil
}

func (m consoleModel) updateConfigEditing(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		cursor := m.opts.table.Cursor()
		opt := m.visibleOptionAt(cursor)
		if opt != nil {
			m.console.setOpt(opt.Name, m.opts.editor.Value())
			m.refreshOptionsTable()
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
			case 0:
				if m.console.program != nil {
					go m.console.program.Send(sessionInteractMsg{id: sess.ID})
				}
				m.sessionBtnMode = false
			case 1:
				handler.Kill(sess.ID)
				if m.sessionCursor > 0 && m.sessionCursor >= len(sessions)-1 {
					m.sessionCursor--
				}
				m.sessionBtnMode = false
			}
		} else {
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
		m.refreshOptionsTable()
		return nil
	}
	if cmd == "" {
		return nil
	}
	c := m.console
	m.busy = true
	return func() tea.Msg {
		c.exec(cmd)
		return commandDoneMsg{}
	}
}

func (m *consoleModel) refreshConfig() {
	m.opts.table.SetCursor(0)
	m.opts.editing = false
	m.opts.editor.Blur()
	m.opts.labMenuOpen = false
	m.opts.btnMode = false
	m.opts.btnCursor = 0
	m.opts.table.Focus()
	m.refreshOptionsTable()
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

