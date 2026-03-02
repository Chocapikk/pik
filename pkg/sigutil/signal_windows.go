//go:build windows

package sigutil

import "os"

// NotifySuspend is a no-op on Windows (no SIGTSTP).
func NotifySuspend(_ chan<- os.Signal) {}

// StopSuspend is a no-op on Windows.
func StopSuspend(_ chan<- os.Signal) {}
