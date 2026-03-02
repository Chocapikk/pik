//go:build !windows

package sigutil

import (
	"os"
	"os/signal"
	"syscall"
)

// NotifySuspend registers SIGTSTP on the channel (Ctrl+Z to background).
func NotifySuspend(ch chan<- os.Signal) { signal.Notify(ch, syscall.SIGTSTP) }

// StopSuspend unregisters the channel from SIGTSTP.
func StopSuspend(ch chan<- os.Signal) { signal.Stop(ch) }
