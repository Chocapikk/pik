package output

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Chocapikk/pik/pkg/log"
)

var spinFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner starts an animated spinner with the given label.
func Spinner(label string) func(string) {
	var mu sync.Mutex
	currentLabel := label
	done := make(chan struct{})
	frame := 0

	go func() {
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				mu.Lock()
				l := currentLabel
				mu.Unlock()
				fmt.Fprintf(log.Output(), "\r%s %s", log.Amber(spinFrames[frame%len(spinFrames)]), l)
				frame++
			}
		}
	}()

	return func(result string) {
		close(done)
		mu.Lock()
		defer mu.Unlock()
		fmt.Fprintf(log.Output(), "\r%s\r", strings.Repeat(" ", 80))
		if result != "" {
			Success("%s", result)
		}
	}
}

func Accent(s string) string { return log.Amber(s) }

func HumanSize(bytes int) string {
	switch {
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func InfoBox(title string, pairs ...string) {
	border := log.Muted(strings.Repeat("─", 45))
	fmt.Fprintf(log.Output(), "\n %s %s %s\n", log.Muted("┌"), log.Amber(title), log.Muted(strings.Repeat("─", max(0, 43-log.VisualLen(title)))))
	for i := 0; i+1 < len(pairs); i += 2 {
		fmt.Fprintf(log.Output(), " %s  %s %s\n", log.Muted("│"), log.Pad(log.Blue(pairs[i]), 14), log.Amber(pairs[i+1]))
	}
	fmt.Fprintf(log.Output(), " %s\n\n", border)
}
