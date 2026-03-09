package sdk

import "testing"

func TestRandTextDefault(t *testing.T) {
	got := RandTextDefault(10)
	if len(got) != 10 {
		t.Errorf("RandTextDefault(10) len = %d", len(got))
	}
}

func TestRandInt(t *testing.T) {
	for range 20 {
		v := RandInt(5, 10)
		if v < 5 || v > 10 {
			t.Errorf("RandInt(5, 10) = %d, out of range", v)
		}
	}
}

func TestRandAlpha(t *testing.T) {
	got := RandAlpha(8)
	if len(got) != 8 {
		t.Errorf("RandAlpha(8) len = %d", len(got))
	}
	for _, c := range got {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			t.Errorf("RandAlpha contains non-alpha: %c", c)
		}
	}
}

func TestRandBool(t *testing.T) {
	// Just verify it doesn't panic and returns both values over many calls
	trues, falses := 0, 0
	for range 100 {
		if RandBool() {
			trues++
		} else {
			falses++
		}
	}
	if trues == 0 || falses == 0 {
		t.Error("RandBool always returned same value")
	}
}

func TestShuffle(t *testing.T) {
	items := []string{"a", "b", "c", "d", "e"}
	shuffled := Shuffle(items)

	if len(shuffled) != len(items) {
		t.Errorf("Shuffle len = %d", len(shuffled))
	}

	// Original should be unchanged
	if items[0] != "a" {
		t.Error("Shuffle mutated original")
	}

	// All elements should be present
	seen := make(map[string]bool)
	for _, s := range shuffled {
		seen[s] = true
	}
	for _, s := range items {
		if !seen[s] {
			t.Errorf("Shuffle missing %q", s)
		}
	}
}
