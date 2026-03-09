//go:build !windows

package sigutil

import (
	"os"
	"testing"
)

func TestNotifySuspendDoesNotPanic(t *testing.T) {
	ch := make(chan os.Signal, 1)
	// Should not panic
	NotifySuspend(ch)
	// Clean up
	StopSuspend(ch)
}

func TestStopSuspendDoesNotPanic(t *testing.T) {
	ch := make(chan os.Signal, 1)
	NotifySuspend(ch)
	// Should not panic
	StopSuspend(ch)
}

func TestStopSuspendWithoutNotify(t *testing.T) {
	ch := make(chan os.Signal, 1)
	// Calling StopSuspend without prior NotifySuspend should not panic
	StopSuspend(ch)
}
