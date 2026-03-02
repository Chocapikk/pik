//go:build windows

package httpshell

import (
	"os"

	"github.com/Chocapikk/pik/pkg/sigutil"
)

func notifySuspend(ch chan<- os.Signal) { sigutil.NotifySuspend(ch) }
func stopSuspend(ch chan<- os.Signal)   { sigutil.StopSuspend(ch) }
