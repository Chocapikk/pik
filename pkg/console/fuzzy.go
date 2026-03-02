package console

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// fuzzyItem is an item in the fuzzy finder list.
type fuzzyItem struct {
	name string
	desc string
}

func (i fuzzyItem) FilterValue() string { return i.name + " " + i.desc }

// fuzzyDelegate renders list items with custom styling.
type fuzzyDelegate struct {
	nameStyle     lipgloss.Style
	descStyle     lipgloss.Style
	selectedStyle lipgloss.Style
}

func newFuzzyDelegate() fuzzyDelegate {
	return fuzzyDelegate{
		nameStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true),
		descStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
		selectedStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true),
	}
}

func (d fuzzyDelegate) Height() int                             { return 1 }
func (d fuzzyDelegate) Spacing() int                            { return 0 }
func (d fuzzyDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d fuzzyDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	fi, ok := item.(fuzzyItem)
	if !ok {
		return
	}

	var line string
	prefix, nameStyle := "  ", d.nameStyle
	if index == m.Index() {
		prefix, nameStyle = "> ", d.selectedStyle
	}
	rendered := nameStyle.Render(fi.name)
	pad := 35 - lipgloss.Width(rendered)
	if pad < 0 {
		pad = 0
	}
	line = fmt.Sprintf("%s%s%s %s",
		prefix,
		rendered,
		strings.Repeat(" ", pad),
		d.descStyle.Render(fi.desc),
	)

	fmt.Fprint(w, line)
}

// fuzzyModel is the bubbletea model for the fuzzy finder.
type fuzzyModel struct {
	list     list.Model
	selected string
	aborted  bool
}

func (m fuzzyModel) Init() tea.Cmd { return nil }

func (m fuzzyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		height := max(msg.Height-4, 5)
		m.list.SetHeight(height)
		return m, nil
	case tea.KeyMsg:
		// Don't intercept keys when filtering
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if item, ok := m.list.SelectedItem().(fuzzyItem); ok {
				m.selected = item.name
			}
			return m, tea.Quit
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "ctrl+c"))):
			m.aborted = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m fuzzyModel) View() string {
	return "\n" + m.list.View()
}

var titleStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("14")).
	Bold(true).
	Padding(0, 1)

// FuzzySelect opens an interactive fuzzy finder and returns the selected item name.
// Returns empty string and false if aborted.
func FuzzySelect(title string, items []fuzzyItem) (string, bool) {
	if len(items) == 0 {
		return "", false
	}

	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}

	delegate := newFuzzyDelegate()
	listModel := list.New(listItems, delegate, 70, 15)
	listModel.Title = title
	listModel.Styles.Title = titleStyle
	listModel.SetFilteringEnabled(true)
	listModel.SetShowStatusBar(true)
	listModel.SetShowHelp(true)
	listModel.FilterInput.Prompt = "  filter: "
	listModel.FilterInput.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	listModel.KeyMap.Quit = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back"))

	// Start in filter mode for immediate typing
	listModel.SetFilteringEnabled(true)

	model := fuzzyModel{list: listModel}
	program := tea.NewProgram(model, tea.WithOutput(os.Stderr))
	result, err := program.Run()
	if err != nil {
		return "", false
	}

	fm := result.(fuzzyModel)
	if fm.aborted || fm.selected == "" {
		return "", false
	}

	// Clear the line after selection
	fmt.Fprint(os.Stderr, strings.Repeat("\n", 1))
	return fm.selected, true
}
