package output

import (
	"fmt"
	"os"

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

func Banner() {
	ver := BannerVersion
	if ver == "" {
		ver = "dev"
	}
	fmt.Fprintln(os.Stderr, log.Cyan(`
    ____  _ __
   / __ \(_) /__
  / /_/ / / //_/
 / ____/ / ,<
/_/   /_/_/|_|
`)+log.Gray("  v"+ver))
}
