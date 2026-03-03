package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Chocapikk/pik/pkg/log"
	"github.com/Chocapikk/pik/pkg/types"
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

	t := table.New(
		table.WithColumns(optionColumns(50)),
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

func optionColumns(w int) []table.Column {
	return []table.Column{
		{Title: "Option", Width: 16},
		{Title: "Value", Width: max(w-24, 12)},
		{Title: "", Width: 3},
	}
}

func browserColumns(w int) []table.Column {
	fixed := 3 + 11 + 5 + 4 // #, Reliability, Check, padding
	remaining := w - fixed
	nameW := remaining * 35 / 100
	descW := remaining * 35 / 100
	cveW := remaining - nameW - descW
	return []table.Column{
		{Title: "#", Width: 3},
		{Title: "Name", Width: nameW},
		{Title: "Reliability", Width: 11},
		{Title: "Check", Width: 5},
		{Title: "Description", Width: descW},
		{Title: "CVEs", Width: cveW},
	}
}

func (m consoleModel) renderBrowseTab(h int) string {
	searchLine := m.search.input.View()
	m.browser.SetHeight(h - 1)
	m.browser.SetWidth(m.width)
	m.browser.SetColumns(browserColumns(m.width))
	return padToHeight(searchLine+"\n"+m.browser.View(), h)
}

func newBrowserTable(w, h int) table.Model {
	rows := buildBrowserRows()

	t := table.New(
		table.WithColumns(browserColumns(w)),
		table.WithRows(rows),
		table.WithHeight(h),
		table.WithFocused(true),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.Foreground(lipgloss.Color("214")).Bold(true)
	s.Selected = s.Selected.Foreground(lipgloss.Color("214")).Bold(true)
	t.SetStyles(s)
	return t
}

// --- Config tab ---

func (m consoleModel) renderConfigTab(h int) string {
	if m.console.Mod() == nil {
		return padToHeight("\n"+log.White("  No module selected")+"\n\n"+log.White("  Select a module in Browse (F1) or type 'use <module>'"), h)
	}

	// Bottom section: buttons (fixed at bottom)
	var bottom []string
	btns := m.actionButtons()
	if len(btns) > 0 {
		var rendered []string
		for i, btn := range btns {
			style := btnStyle
			if m.opts.btnMode && i == m.opts.btnCursor {
				style = btnFocusedStyle
			}
			rendered = append(rendered, style.Render(btn))
		}
		bottom = append(bottom, "  "+strings.Join(rendered, "  "))

		if m.opts.labMenuOpen {
			for i, item := range labSubMenu {
				prefix := "     "
				if i == m.opts.labMenuCursor {
					prefix = "   " + log.Amber("> ")
				}
				bottom = append(bottom, prefix+log.White(item))
			}
		}
	}

	bottomH := len(bottom) + 1 // +1 for spacing
	tableH := h - 3 - bottomH  // 3 = header lines
	if tableH < 3 {
		tableH = 3
	}

	// Top section: header + options table
	var top []string
	top = append(top, "")
	top = append(top, "  "+log.Amber(sdk.NameOf(m.console.Mod()))+"  "+log.White(m.console.Mod().Info().Title()))
	top = append(top, "")

	m.opts.table.SetHeight(tableH)
	top = append(top, m.opts.table.View())

	// Pad between top and bottom to push buttons to bottom
	topStr := strings.Join(top, "\n")
	topLines := strings.Count(topStr, "\n") + 1
	padLines := h - topLines - len(bottom)
	if padLines < 1 {
		padLines = 1
	}

	var result []string
	result = append(result, topStr)
	for i := 0; i < padLines; i++ {
		result = append(result, "")
	}
	result = append(result, bottom...)

	return padToHeight(strings.Join(result, "\n"), h)
}

func (m *consoleModel) refreshOptionsTable() {
	var rows []table.Row
	for _, opt := range m.console.Options() {
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

	m.opts.table.SetColumns(optionColumns(m.width))
}

func (m consoleModel) visibleOptionCount() int {
	count := 0
	for _, opt := range m.console.Options() {
		if !opt.Advanced || m.opts.showAdvanced {
			count++
		}
	}
	return count
}

func (m consoleModel) visibleOptionAt(idx int) *types.Option {
	cur := 0
	for i := range m.console.Options() {
		if m.console.Options()[i].Advanced && !m.opts.showAdvanced {
			continue
		}
		if cur == idx {
			return &m.console.Options()[i]
		}
		cur++
	}
	return nil
}

func (m consoleModel) actionButtons() []string {
	if m.console.Mod() == nil {
		return nil
	}
	var btns []string
	if _, ok := m.console.Mod().(sdk.Checker); ok {
		btns = append(btns, "Check")
	}
	btns = append(btns, "Exploit")
	if len(m.console.Mod().Info().Lab.Services) > 0 {
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
	handler := m.console.SessionHandler()
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
		row := m.browser.SelectedRow()
		if len(row) >= 2 {
			m.console.UseByName(row[1]) // column 1 = Name
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

func buildBrowserRows() []table.Row {
	modules := sdk.List()
	rows := make([]table.Row, len(modules))
	for i, mod := range modules {
		info := mod.Info()
		cveList := info.CVEs()
		cves := "-"
		if len(cveList) == 1 {
			cves = cveList[0]
		} else if len(cveList) > 1 {
			cves = fmt.Sprintf("%d CVEs", len(cveList))
		}
		rel := info.Reliability.String()
		check := "no"
		if _, ok := mod.(sdk.Checker); ok {
			check = "yes"
		}
		rows[i] = table.Row{
			fmt.Sprintf("%d", i),
			sdk.NameOf(mod),
			rel,
			check,
			info.Title(),
			cves,
		}
	}
	return rows
}

func (m *consoleModel) filterBrowser(query string) {
	query = strings.ToLower(query)
	var filtered []table.Row
	for _, row := range m.allRows {
		match := false
		for _, cell := range row {
			if strings.Contains(strings.ToLower(cell), query) {
				match = true
				break
			}
		}
		if match {
			filtered = append(filtered, row)
		}
	}
	m.browser.SetRows(filtered)
}

func (m *consoleModel) resetBrowserFilter() {
	m.browser.SetRows(m.allRows)
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
				c.Exec(cmd)
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
				if m.console.Mod() != nil {
					platform = m.console.Mod().Info().Platform()
				}
				payloads := payload.ListForPlatform(platform)
				items := make([]fuzzyItem, len(payloads))
				for i, pl := range payloads {
					items[i] = fuzzyItem{Name: pl.Name, Desc: pl.Description}
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
			m.console.SetOpt(opt.Name, m.opts.editor.Value())
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
	handler := m.console.SessionHandler()
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
				if m.console.Program() != nil {
					go m.console.Program().Send(types.SessionInteractMsg{ID: sess.ID})
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
		c.Exec(cmd)
		return commandDoneMsg{}
	}
}

// refreshConfig resets the config tab fully (new module selected).
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

// refreshOptionsOnly updates option values without resetting focus state.
func (m *consoleModel) refreshOptionsOnly() {
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

