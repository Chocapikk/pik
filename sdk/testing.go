package sdk

import "testing"

// TestRegister registers a module under a specific name for testing purposes.
// It cleans up the registration when the test completes.
func TestRegister(t *testing.T, name string, mod Exploit) {
	t.Helper()
	mu.Lock()
	defer mu.Unlock()

	for _, e := range entries {
		if e.name == name {
			t.Fatalf("test module %q already registered", name)
		}
	}

	entries = append(entries, entry{name, mod})

	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		for i, e := range entries {
			if e.name == name {
				entries = append(entries[:i], entries[i+1:]...)
				return
			}
		}
	})
}
