//go:build windows

package httpshell

import "os"

func notifySuspend(_ chan<- os.Signal) {}
func stopSuspend(_ chan<- os.Signal)   {}
