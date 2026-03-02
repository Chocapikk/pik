package output

import (
	"fmt"
	"os"
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
				fmt.Fprintf(os.Stderr, "\r%s %s", log.Cyan(spinFrames[frame%len(spinFrames)]), l)
				frame++
			}
		}
	}()

	return func(result string) {
		close(done)
		mu.Lock()
		defer mu.Unlock()
		fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", 80))
		if result != "" {
			Success("%s", result)
		}
	}
}

func Accent(s string) string    { return log.Cyan(s) }
func Dim(s string) string       { return log.DimText(s) }

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
	border := log.Gray(strings.Repeat("─", 45))
	fmt.Fprintf(os.Stderr, "\n %s %s %s\n", log.Gray("┌"), log.Cyan(title), log.Gray(strings.Repeat("─", max(0, 43-len(title)))))
	for i := 0; i+1 < len(pairs); i += 2 {
		fmt.Fprintf(os.Stderr, " %s  %-14s %s\n", log.Gray("│"), log.Blue(pairs[i]), log.Cyan(pairs[i+1]))
	}
	fmt.Fprintf(os.Stderr, " %s\n\n", border)
}
