package output

import (
	"fmt"
	"os"
	"strings"

	"github.com/Chocapikk/pik/pkg/log"
)

var (
	verboseMode bool
	debugMode   bool
)

func SetVerbose(v bool) { verboseMode = v }
func SetDebug(d bool)   { debugMode = d; verboseMode = d }
func EnableDebug()      { debugMode = true; verboseMode = true }
func IsDebug() bool     { return debugMode }
func IsVerbose() bool   { return verboseMode }

// Core log functions - delegate to pkg/log.
var (
	Status  = log.Status
	Success = log.Success
	Error   = log.Error
	Warning = log.Warning
)

func Verbose(format string, args ...any) {
	if verboseMode {
		log.Verbose(format, args...)
	}
}

func Debug(format string, args ...any) {
	if debugMode {
		log.Debug(format, args...)
	}
}

func Print(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
}

func Println(args ...any) {
	fmt.Fprintln(os.Stderr, args...)
}

// BannerVersion is set by the CLI package at init time.
var BannerVersion string

var bannerLines = []string{
	`           ███  █████`,
	`           ▒▒▒  ▒▒███`,
	` ████████  ████  ▒███ █████`,
	`▒▒███▒▒███▒▒███  ▒███▒▒███`,
	` ▒███ ▒███ ▒███  ▒██████▒`,
	` ▒███ ▒███ ▒███  ▒███▒▒███`,
	` ▒███████  █████ ████ █████`,
	` ▒███▒▒▒   ▒▒▒▒▒ ▒▒▒▒ ▒▒▒▒▒`,
	` ▒███`,
	` █████`,
}

func Banner() {
	ver := BannerVersion
	if ver == "" {
		ver = "dev"
	}

	maxWidth := 0
	for _, line := range bannerLines {
		if len(line) > maxWidth {
			maxWidth = len(line)
		}
	}

	meta := map[int]string{
		7:                          ver,
		len(bannerLines) - 1: "github.com/Chocapikk/pik",
	}

	// Link stays close to line, not right-aligned.
	linkIdx := len(bannerLines) - 1

	fmt.Fprintln(os.Stderr)
	for i, line := range bannerLines {
		right := meta[i]
		if right != "" {
			var gap string
			if i == linkIdx {
				gap = "  "
			} else {
				gap = strings.Repeat(" ", maxWidth-len(line)+2)
			}
			fmt.Fprintf(os.Stderr, "%s%s%s\n", log.Amber(line), gap, log.Muted(right))
		} else {
			fmt.Fprintln(os.Stderr, log.Amber(line))
		}
	}
	fmt.Fprintln(os.Stderr)
}
