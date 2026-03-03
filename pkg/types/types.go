// Package types defines shared types between console and tui packages.
package types

// Option represents a module option.
type Option struct {
	Name     string
	Value    string
	Required bool
	Desc     string
	Advanced bool
}

// FuzzyItem represents an item in the fuzzy picker.
// Implements list.Item interface (FilterValue).
type FuzzyItem struct {
	Name string
	Desc string
}

// FilterValue implements the list.Item interface for bubbletea lists.
func (i FuzzyItem) FilterValue() string { return i.Name + " " + i.Desc }

// FuzzySelectMsg requests the TUI to enter fuzzy selection mode.
type FuzzySelectMsg struct {
	Context string
	Items   []FuzzyItem
	Title   string
}

// SessionInteractMsg requests the TUI to interact with a session.
type SessionInteractMsg struct {
	ID int
}

// ClearOutputMsg requests the TUI to clear the output viewport.
type ClearOutputMsg struct{}
