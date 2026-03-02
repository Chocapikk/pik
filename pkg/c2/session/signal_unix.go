//go:build !windows

package session

import (
	"os"
	"os/signal"
	"syscall"
)

func notifySuspend(ch chan<- os.Signal)  { signal.Notify(ch, syscall.SIGTSTP) }
func stopSuspend(ch chan<- os.Signal)    { signal.Stop(ch) }
