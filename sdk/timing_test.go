package sdk

import (
	"testing"
	"time"
)

func TestSleepCheckVulnerable(t *testing.T) {
	ctx := NewContext(nil, "")
	result, err := SleepCheck(ctx, func(delay int) error {
		// Simulate a real delay
		time.Sleep(time.Duration(delay) * time.Second)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Code != CheckVulnerable {
		t.Errorf("expected Vulnerable, got %s: %s", result.Code, result.Reason)
	}
}

func TestSleepCheckSafe(t *testing.T) {
	ctx := NewContext(nil, "")
	result, err := SleepCheck(ctx, func(delay int) error {
		// No delay - returns immediately
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Code != CheckSafe {
		t.Errorf("expected Safe, got %s: %s", result.Code, result.Reason)
	}
}

func TestSleepCheckWithErrors(t *testing.T) {
	ctx := NewContext(nil, "")
	result, err := SleepCheck(ctx, func(delay int) error {
		return Errorf("connection refused")
	})
	if err != nil {
		t.Fatal(err)
	}
	// All rounds error out -> hits=0 -> Safe
	if result.Code != CheckSafe {
		t.Errorf("expected Safe on errors, got %s", result.Code)
	}
}
