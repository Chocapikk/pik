package types

import (
	"testing"
)

func TestFuzzyItemFilterValue(t *testing.T) {
	item := FuzzyItem{Name: "exploit/http/rce", Desc: "Remote Code Execution"}
	got := item.FilterValue()
	want := "exploit/http/rce Remote Code Execution"
	if got != want {
		t.Errorf("FilterValue() = %q, want %q", got, want)
	}
}

func TestFuzzyItemFilterValueEmpty(t *testing.T) {
	item := FuzzyItem{}
	got := item.FilterValue()
	want := " "
	if got != want {
		t.Errorf("FilterValue() for empty item = %q, want %q", got, want)
	}
}

func TestOptionFields(t *testing.T) {
	opt := Option{
		Name:     "TARGET",
		Value:    "192.168.1.1",
		Required: true,
		Desc:     "Target address",
		Advanced: false,
	}
	if opt.Name != "TARGET" {
		t.Errorf("Option.Name = %q, want %q", opt.Name, "TARGET")
	}
	if opt.Value != "192.168.1.1" {
		t.Errorf("Option.Value = %q, want %q", opt.Value, "192.168.1.1")
	}
	if !opt.Required {
		t.Error("Option.Required should be true")
	}
	if opt.Desc != "Target address" {
		t.Errorf("Option.Desc = %q, want %q", opt.Desc, "Target address")
	}
	if opt.Advanced {
		t.Error("Option.Advanced should be false")
	}
}

func TestOptionAdvanced(t *testing.T) {
	opt := Option{
		Name:     "HTTP_TRACE",
		Value:    "false",
		Required: false,
		Desc:     "Enable HTTP tracing",
		Advanced: true,
	}
	if !opt.Advanced {
		t.Error("Option.Advanced should be true")
	}
}

func TestClearOutputMsg(t *testing.T) {
	// ClearOutputMsg is a zero-value struct, just verify it can be created
	msg := ClearOutputMsg{}
	_ = msg
}

func TestSessionInteractMsg(t *testing.T) {
	msg := SessionInteractMsg{ID: 7}
	if msg.ID != 7 {
		t.Errorf("SessionInteractMsg.ID = %d, want 7", msg.ID)
	}
}

func TestFuzzySelectMsg(t *testing.T) {
	items := []FuzzyItem{
		{Name: "mod1", Desc: "Module 1"},
		{Name: "mod2", Desc: "Module 2"},
	}
	msg := FuzzySelectMsg{
		Context: "use",
		Items:   items,
		Title:   "Select Module",
	}
	if msg.Context != "use" {
		t.Errorf("FuzzySelectMsg.Context = %q, want %q", msg.Context, "use")
	}
	if len(msg.Items) != 2 {
		t.Errorf("FuzzySelectMsg.Items length = %d, want 2", len(msg.Items))
	}
	if msg.Title != "Select Module" {
		t.Errorf("FuzzySelectMsg.Title = %q, want %q", msg.Title, "Select Module")
	}
}
