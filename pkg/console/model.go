package console

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/sdk"
)

type mode int

const (
	modeNormal mode = iota
	modeFuzzy
)

// --- Messages ---

type commandDoneMsg struct{}

type sessionInteractMsg struct {
	id int
}
type sessionDoneMsg struct{ err error }

// sessionCmd wraps session InteractTUI for tea.Exec.
type sessionCmd struct {
	session interface{ InteractTUI() }
}

func (s *sessionCmd) Run() error            { s.session.InteractTUI(); return nil }
func (s *sessionCmd) SetStdin(_ io.Reader)  {}
func (s *sessionCmd) SetStdout(_ io.Writer) {}
func (s *sessionCmd) SetStderr(_ io.Writer) {}

type fuzzySelectMsg struct {
	context string
	items   []fuzzyItem
	title   string
}


// --- Model ---

type consoleModel struct {
	console   *Console
	input     textinput.Model
	viewport  viewport.Model
	output    []string
	mode          mode
	activeTab     tab
	tabBarFocused bool
	tuiFocused    bool // true = keys go to tab content, false = keys go to console input
	splitRatio    int  // percentage of height for tab content (0 = auto)

	browser list.Model
	search  browseSearch
	opts    optionsPanel

	fuzzy fuzzyModel

	sessionCursor    int
	sessionBtnCursor int
	sessionBtnMode   bool // true when navigating buttons row

	history []string
	historyIdx int
	historyTmp string
	compState  *completionState

	width, height int
	ready         bool
	busy          bool
	quitting      bool
}

func newConsoleModel(c *Console) consoleModel {
	ti := textinput.New()
	ti.Focus()
	ti.Prompt = ""
	ti.CharLimit = 0

	return consoleModel{
		console:    c,
		input:      ti,
		activeTab:  tabBrowse,
		browser:    newBrowser(80, 20),
		search:     newBrowseSearch(),
		opts:       newOptionsPanel(),
		history:    loadHistory(),
		historyIdx: -1,
	}
}

func (m consoleModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.bannerCmd())
}

func (m consoleModel) bannerCmd() tea.Cmd {
	return func() tea.Msg {
		output.Status("pik - exploit framework")
		if m.console.mod != nil {
			output.Success("Using %s - %s", sdk.NameOf(m.console.mod), m.console.mod.Info().Title())
		}
		return nil
	}
}

// --- Update ---

func (m consoleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		th := m.tabHeight()
		oh := m.outputHeight()
		m.browser.SetSize(m.width, th)
		m.viewport.Width = m.width
		m.viewport.Height = oh
		if !m.ready {
			m.viewport = viewport.New(m.width, oh)
			m.viewport.MouseWheelEnabled = true
			m.ready = true
		}
		m.viewport.SetContent(strings.Join(m.output, ""))
		return m, nil

	case outputMsg:
		m.output = append(m.output, string(msg))
		m.viewport.SetContent(strings.Join(m.output, ""))
		m.viewport.GotoBottom()
		return m, nil

	case ClearOutputMsg:
		m.output = nil
		m.viewport.SetContent("")
		return m, nil

	case commandDoneMsg:
		m.busy = false
		m.refreshConfig()
		return m, nil

	case sessionInteractMsg:
		handler := m.console.sessionHandler()
		if handler == nil {
			return m, nil
		}
		sessions := handler.Sessions()
		for _, sess := range sessions {
			if sess.ID == msg.id {
				return m, tea.Exec(&sessionCmd{session: sess}, func(err error) tea.Msg {
					return sessionDoneMsg{err: err}
				})
			}
		}
		return m, nil

	case sessionDoneMsg:
		// Re-enable mouse after tea.Exec returns
		return m, func() tea.Msg { return tea.EnableMouseAllMotion() }

	case fuzzySelectMsg:
		m.mode = modeFuzzy
		m.fuzzy = newFuzzyModel(msg.title, msg.items)
		m.fuzzy.context = msg.context
		return m, nil

	case tea.KeyMsg:
		if m.mode == modeFuzzy {
			return m.updateFuzzy(msg)
		}
		// Tab switching: F1-F4
		if newTab, ok := m.tabFromKey(msg); ok {
			m.activeTab = newTab
			return m, nil
		}
		// Global keys
		switch msg.Type {
		case tea.KeyCtrlD:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyCtrlC:
			m.input.SetValue("")
			m.compState = nil
			return m, nil
		}
		// Route to active tab content or input
		return m.routeKey(msg)
	}

	// Handle mouse events
	if mouseMsg, ok := msg.(tea.MouseMsg); ok {
		return m.handleMouse(mouseMsg)
	}

	// Pass other messages to active tab components
	if m.activeTab == tabBrowse {
		var cmd tea.Cmd
		m.browser, cmd = m.browser.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m consoleModel) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	isClick := msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft

	tabH := m.tabHeight()
	dividerY := 1 + tabH
	outputY := dividerY + 1

	// Click on tab bar (row 0)
	if msg.Y == 0 && isClick {
		x := 0
		for i, label := range tabLabels {
			tabW := len(fmt.Sprintf("F%d:%s", i+1, label)) + 2
			if msg.X >= x && msg.X < x+tabW {
				m.activeTab = tab(i)
				m.tuiFocused = true
				m.tabBarFocused = false
				m.input.Blur()
				return m, nil
			}
			x += tabW + 1
		}
		return m, nil
	}

	// Click in TUI zone (tab content)
	if msg.Y >= 1 && msg.Y < dividerY && isClick {
		m.tuiFocused = true
		m.input.Blur()
		if m.activeTab == tabBrowse {
			m.search.input.Focus()
		}
	}

	// Click in console zone (output + input)
	if msg.Y >= outputY && isClick {
		m.tuiFocused = false
		m.search.input.Blur()
		m.input.Focus()
	}
	isScroll := msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown

	// Drag divider to resize: any click/motion on or near the divider
	if msg.Action == tea.MouseActionMotion && msg.Button == tea.MouseButtonLeft && msg.Y > 2 && msg.Y < m.height-3 {
		// Set split ratio based on mouse Y position
		contentArea := m.height - 3 // tabBar(1) + divider(1) + input(1)
		newTabH := msg.Y - 1        // subtract tab bar
		ratio := newTabH * 100 / contentArea
		if ratio < 20 {
			ratio = 20
		}
		if ratio > 80 {
			ratio = 80
		}
		m.splitRatio = ratio
		// Recalculate sizes
		th := m.tabHeight()
		oh := m.outputHeight()
		m.browser.SetSize(m.width, th)
		m.viewport.Width = m.width
		m.viewport.Height = oh
		m.viewport.SetContent(strings.Join(m.output, ""))
		return m, nil
	}

	// Output area: scroll viewport
	if msg.Y >= outputY {
		if isScroll {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	// Tab content area
	if msg.Y >= 1 && msg.Y < dividerY {
		contentY := msg.Y - 1

		switch m.activeTab {
		case tabBrowse:
			// Mouse scroll in browse = navigate list
			if isScroll {
				if msg.Button == tea.MouseButtonWheelUp {
					m.browser.CursorUp()
				} else {
					m.browser.CursorDown()
				}
				return m, nil
			}
			var cmd tea.Cmd
			m.browser, cmd = m.browser.Update(msg)
			return m, cmd

		case tabConfig:
			if isClick && contentY >= 3 {
				optIdx := contentY - 3
				maxIdx := m.visibleOptionCount() + len(m.actionButtons()) - 1
				if optIdx >= 0 && optIdx <= maxIdx {
					m.opts.cursor = optIdx
				}
			}
			return m, nil

		case tabSessions:
			if isClick && contentY >= 2 {
				handler := m.console.sessionHandler()
				if handler != nil {
					sessions := handler.Sessions()
					sessIdx := contentY - 2
					if sessIdx >= 0 && sessIdx < len(sessions) {
						m.sessionCursor = sessIdx
					}
				}
			}
			return m, nil
		}
	}

	return m, nil
}

func (m consoleModel) tabFromKey(msg tea.KeyMsg) (tab, bool) {
	switch msg.String() {
	case "f1":
		return tabBrowse, true
	case "f2":
		return tabConfig, true
	case "f3":
		return tabSessions, true
	}
	return 0, false
}

func (m consoleModel) routeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// When editing an option in Config tab, ALL keys go to the editor
	if m.activeTab == tabConfig && m.opts.editing {
		return m.updateConfigEditing(msg)
	}


	// Tab bar navigation
	if m.tabBarFocused {
		switch msg.Type {
		case tea.KeyLeft:
			m.activeTab = (m.activeTab - 1 + tabCount) % tabCount
			return m, nil
		case tea.KeyRight:
			m.activeTab = (m.activeTab + 1) % tabCount
			return m, nil
		case tea.KeyDown, tea.KeyEnter:
			m.tabBarFocused = false
			return m, nil
		case tea.KeyEscape:
			m.tabBarFocused = false
			return m, nil
		}
		return m, nil
	}

	// TUI focused: all keys go to tab content
	if m.tuiFocused {
		switch msg.Type {
		case tea.KeyEscape:
			// Esc in TUI: switch to console focus
			m.tuiFocused = false
			m.search.input.Blur()
			m.input.Focus()
			return m, nil
		case tea.KeyUp:
			if m.isAtTopOfTab() {
				m.tabBarFocused = true
				return m, nil
			}
			return m.routeToTab(msg)
		case tea.KeyTab:
			m.activeTab = (m.activeTab + 1) % tabCount
			return m, nil
		case tea.KeyPgUp, tea.KeyPgDown:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
		return m.routeToTab(msg)
	}

	// Console focused: keys go to input
	switch msg.Type {
	case tea.KeyPgUp, tea.KeyPgDown:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	return m.updateInput(msg)
}

func (m consoleModel) routeToTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.activeTab {
	case tabBrowse:
		return m.updateBrowseTab(msg)
	case tabConfig:
		return m.updateConfigTab(msg)
	case tabSessions:
		return m.updateSessionsTab(msg)
	}
	return m, nil
}

func (m consoleModel) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		if m.busy {
			return m, nil
		}
		line := strings.TrimSpace(m.input.Value())
		m.input.SetValue("")
		m.compState = nil
		m.historyIdx = -1
		if line == "" {
			return m, nil
		}
		m.history = append(m.history, line)
		appendHistory(line)
		prompt := m.console.BuildPrompt()
		m.output = append(m.output, prompt+line+"\n")
		m.viewport.SetContent(strings.Join(m.output, ""))
		m.viewport.GotoBottom()

		if line == "exit" || line == "quit" {
			m.quitting = true
			return m, tea.Quit
		}

		m.busy = true
		return m, m.execCmd(line)

	case tea.KeyUp:
		if len(m.history) == 0 {
			return m, nil
		}
		if m.historyIdx < 0 {
			m.historyTmp = m.input.Value()
			m.historyIdx = len(m.history) - 1
		} else if m.historyIdx > 0 {
			m.historyIdx--
		}
		m.input.SetValue(m.history[m.historyIdx])
		m.input.CursorEnd()
		m.compState = nil
		return m, nil

	case tea.KeyDown:
		if m.historyIdx < 0 {
			return m, nil
		}
		if m.historyIdx < len(m.history)-1 {
			m.historyIdx++
			m.input.SetValue(m.history[m.historyIdx])
		} else {
			m.historyIdx = -1
			m.input.SetValue(m.historyTmp)
		}
		m.input.CursorEnd()
		m.compState = nil
		return m, nil

	case tea.KeyTab:
		return m.handleTab()
	}

	m.compState = nil
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m consoleModel) handleTab() (tea.Model, tea.Cmd) {
	input := m.input.Value()
	if m.compState != nil && m.compState.prefix == input {
		m.compState.index = (m.compState.index + 1) % len(m.compState.matches)
	} else {
		matches := m.console.complete(input)
		if len(matches) == 0 {
			return m, nil
		}
		m.compState = &completionState{
			prefix:  input,
			matches: matches,
			index:   0,
		}
	}

	match := m.compState.matches[m.compState.index]
	parts := strings.Fields(m.compState.prefix)
	if len(parts) <= 1 && !strings.HasSuffix(m.compState.prefix, " ") {
		m.input.SetValue(match + " ")
	} else {
		parts[len(parts)-1] = match
		m.input.SetValue(strings.Join(parts, " ") + " ")
	}
	m.input.CursorEnd()
	return m, nil
}

func (m consoleModel) updateFuzzy(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.fuzzy.list.FilterState() != list.Filtering {
		switch msg.String() {
		case "enter":
			if item, ok := m.fuzzy.list.SelectedItem().(fuzzyItem); ok {
				ctx := m.fuzzy.context
				m.mode = modeNormal
				m.console.applyFuzzyResult(ctx, item.name)
				m.refreshConfig()
				if ctx == "module" {
					m.activeTab = tabConfig
				}
			}
			return m, nil
		case "esc", "ctrl+c":
			m.mode = modeNormal
			return m, nil
		}
	}

	var cmd tea.Cmd
	var model tea.Model
	model, cmd = m.fuzzy.Update(msg)
	m.fuzzy = model.(fuzzyModel)
	return m, cmd
}

func (m consoleModel) isAtTopOfTab() bool {
	switch m.activeTab {
	case tabBrowse:
		return m.browser.Index() == 0
	case tabConfig:
		return m.opts.cursor == 0
	case tabSessions:
		return m.sessionCursor == 0
	}
	return true
}

func (m consoleModel) execCmd(line string) tea.Cmd {
	c := m.console
	return func() tea.Msg {
		c.exec(line)
		return commandDoneMsg{}
	}
}

// --- View ---

func (m consoleModel) View() string {
	if !m.ready {
		return "Initializing..."
	}
	if m.mode == modeFuzzy {
		return m.fuzzy.View()
	}

	tabBar := m.renderTabBar()
	tabContent := m.renderActiveTab()
	divider := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(strings.Repeat("─", m.width))
	outputView := m.viewport.View()
	prompt := m.console.BuildPrompt()
	inputLine := prompt + m.input.View()

	return lipgloss.JoinVertical(lipgloss.Left, tabBar, tabContent, divider, outputView, inputLine)
}

// tabHeight returns the height for the tab content area.
func (m consoleModel) tabHeight() int {
	contentArea := m.height - 3 // tabBar(1) + divider(1) + input(1)
	if contentArea < 10 {
		contentArea = 10
	}
	ratio := m.splitRatio
	if ratio == 0 {
		ratio = 50 // default 50/50
	}
	h := contentArea * ratio / 100
	if h < 4 {
		h = 4
	}
	return h
}

// outputHeight returns the height for the output viewport.
func (m consoleModel) outputHeight() int {
	contentArea := m.height - 3
	if contentArea < 10 {
		contentArea = 10
	}
	h := contentArea - m.tabHeight()
	if h < 3 {
		h = 3
	}
	return h
}

// contentHeight kept for compatibility.
func (m consoleModel) contentHeight() int {
	return m.tabHeight()
}

