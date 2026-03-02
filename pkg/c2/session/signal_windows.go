//go:build windows

package session

import "os"

func notifySuspend(_ chan<- os.Signal) {}
func stopSuspend(_ chan<- os.Signal)   {}
