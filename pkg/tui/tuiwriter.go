package tui

import (
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

// outputMsg carries text that should be appended to the viewport.
type outputMsg string

// tuiWriter implements io.Writer and forwards output to the bubbletea program.
type tuiWriter struct {
	program *tea.Program
	mu      sync.Mutex
	pending string
}

func NewTUIWriter(p *tea.Program) *tuiWriter {
	return &tuiWriter{program: p}
}

func (w *tuiWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pending += string(b)
	for {
		idx := strings.IndexByte(w.pending, '\n')
		if idx < 0 {
			break
		}
		line := w.pending[:idx+1]
		w.pending = w.pending[idx+1:]
		msg := outputMsg(line)
		go w.program.Send(msg)
	}
	return len(b), nil
}
