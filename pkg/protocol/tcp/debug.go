package tcp

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	targetStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	sendArrow   = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("▶")
	recvArrow   = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Render("◀")
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	divider     = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(strings.Repeat("─", 60))
	lenStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
)

func debugSend(target string, data []byte) {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, divider)
	fmt.Fprintf(os.Stderr, "%s %s %s\n",
		sendArrow,
		targetStyle.Render(target),
		lenStyle.Render(fmt.Sprintf("(%d bytes)", len(data))),
	)
	fmt.Fprintln(os.Stderr, divider)
	printHexDump(data)
}

func debugRecv(target string, data []byte) {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, divider)
	fmt.Fprintf(os.Stderr, "%s %s %s\n",
		recvArrow,
		targetStyle.Render(target),
		lenStyle.Render(fmt.Sprintf("(%d bytes)", len(data))),
	)
	fmt.Fprintln(os.Stderr, divider)
	printHexDump(data)
	fmt.Fprintln(os.Stderr)
}

const maxDumpBytes = 512

func printHexDump(data []byte) {
	show := data
	truncated := false
	if len(show) > maxDumpBytes {
		show = show[:maxDumpBytes]
		truncated = true
	}

	dump := hex.Dump(show)
	// Style: offset in dim, hex in normal, ASCII in dim
	for _, line := range strings.Split(strings.TrimRight(dump, "\n"), "\n") {
		if len(line) < 10 {
			fmt.Fprintln(os.Stderr, dimStyle.Render(line))
			continue
		}
		// hex.Dump format: "00000000  xx xx xx ... |ascii...|"
		offset := line[:10]
		rest := line[10:]
		if pipeIdx := strings.LastIndex(rest, "|"); pipeIdx > 0 {
			hexPart := rest[:pipeIdx]
			asciiPart := rest[pipeIdx:]
			fmt.Fprintf(os.Stderr, "%s%s%s\n",
				dimStyle.Render(offset),
				hexPart,
				dimStyle.Render(asciiPart),
			)
		} else {
			fmt.Fprintf(os.Stderr, "%s%s\n", dimStyle.Render(offset), rest)
		}
	}

	if truncated {
		fmt.Fprintf(os.Stderr, "%s\n",
			dimStyle.Render(fmt.Sprintf("  ... truncated (%d bytes total)", len(data))),
		)
	}
}
