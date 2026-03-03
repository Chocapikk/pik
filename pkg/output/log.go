package output

import (
	"fmt"
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
	fmt.Fprintf(log.Output(), format, args...)
}

func Println(args ...any) {
	fmt.Fprintln(log.Output(), args...)
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

// BannerModuleCount and BannerPayloadCount are set by the console before Banner().
var BannerModuleCount int
var BannerPayloadCount int

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
	// These lines use a small gap instead of right-aligning.
	closeGapLines := map[int]bool{
		len(bannerLines) - 1: true, // github link
	}

	fmt.Fprintln(log.Output())
	for i, line := range bannerLines {
		right := meta[i]
		if right != "" {
			var gap string
			if closeGapLines[i] {
				gap = "  "
			} else {
				gap = strings.Repeat(" ", maxWidth-len(line)+2)
			}
			fmt.Fprintf(log.Output(), "%s%s%s\n", log.Amber(line), gap, log.Muted(right))
		} else {
			fmt.Fprintln(log.Output(), log.Amber(line))
		}
	}
	fmt.Fprintln(log.Output())
	if BannerModuleCount > 0 {
		fmt.Fprintf(log.Output(), "  %s %s %s %s\n\n",
			log.Amber(fmt.Sprintf("%d", BannerModuleCount)), log.White("exploits,"),
			log.Amber(fmt.Sprintf("%d", BannerPayloadCount)), log.White("payloads"))
	}
}
