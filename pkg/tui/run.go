package tui

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Chocapikk/pik/pkg/console"
	"github.com/Chocapikk/pik/pkg/log"
	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/pkg/payload"
	"github.com/Chocapikk/pik/sdk"
)

// programSender wraps *tea.Program to satisfy console.MsgSender.
type programSender struct{ p *tea.Program }

func (s programSender) Send(msg any) { s.p.Send(msg) }

// Run starts the TUI with no pre-selected module.
func Run() error {
	return RunWith(nil)
}

// RunWith starts the TUI with an optional pre-selected module.
func RunWith(mod sdk.Exploit) error {
	cons := console.New()
	if mod != nil {
		cons.SetMod(mod)
	}

	model := NewModel(cons)
	p := tea.NewProgram(model,
		tea.WithAltScreen(),
		tea.WithOutput(os.Stderr),
		tea.WithMouseAllMotion(),
	)
	cons.SetProgram(programSender{p})

	writer := NewTUIWriter(p)
	log.SetOutput(writer)
	defer log.SetOutput(os.Stderr)

	output.BannerModuleCount = len(sdk.List())
	output.BannerPayloadCount = len(payload.ListPayloads())

	_, err := p.Run()
	cons.ShutdownBackend()
	return err
}
