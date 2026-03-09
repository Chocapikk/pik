package text

import (
	"strings"
	"testing"
)

func TestRandTextLength(t *testing.T) {
	for _, n := range []int{0, 1, 8, 32, 100} {
		got := RandText(n)
		if len(got) != n {
			t.Errorf("RandText(%d) length = %d", n, len(got))
		}
	}
}

func TestRandTextCharset(t *testing.T) {
	got := RandText(1000)
	for _, c := range got {
		if !strings.ContainsRune(LowerAlpha, c) {
			t.Errorf("RandText contains invalid char %q", c)
			break
		}
	}
}

func TestRandAlphaCharset(t *testing.T) {
	got := RandAlpha(1000)
	for _, c := range got {
		if !strings.ContainsRune(Alpha, c) {
			t.Errorf("RandAlpha contains invalid char %q", c)
			break
		}
	}
}

func TestRandAlphaNumCharset(t *testing.T) {
	got := RandAlphaNum(1000)
	for _, c := range got {
		if !strings.ContainsRune(AlphaNum, c) {
			t.Errorf("RandAlphaNum contains invalid char %q", c)
			break
		}
	}
}

func TestRandNumericCharset(t *testing.T) {
	got := RandNumeric(1000)
	for _, c := range got {
		if !strings.ContainsRune(Digits, c) {
			t.Errorf("RandNumeric contains invalid char %q", c)
			break
		}
	}
}

func TestRandHexCharset(t *testing.T) {
	got := RandHex(1000)
	for _, c := range got {
		if !strings.ContainsRune(HexChars, c) {
			t.Errorf("RandHex contains invalid char %q", c)
			break
		}
	}
}

func TestRandBytesLength(t *testing.T) {
	for _, n := range []int{0, 1, 16, 256} {
		got := RandBytes(n)
		if len(got) != n {
			t.Errorf("RandBytes(%d) length = %d", n, len(got))
		}
	}
}

func TestRandIntRange(t *testing.T) {
	for i := 0; i < 1000; i++ {
		got := RandInt(5, 10)
		if got < 5 || got >= 10 {
			t.Errorf("RandInt(5, 10) = %d, out of range", got)
		}
	}
}

func TestRandBoolDistribution(t *testing.T) {
	trues := 0
	n := 10000
	for i := 0; i < n; i++ {
		if RandBool() {
			trues++
		}
	}
	ratio := float64(trues) / float64(n)
	if ratio < 0.4 || ratio > 0.6 {
		t.Errorf("RandBool distribution skewed: %.2f true ratio", ratio)
	}
}

func TestRandElement(t *testing.T) {
	items := []string{"a", "b", "c"}
	for i := 0; i < 100; i++ {
		got := RandElement(items)
		found := false
		for _, item := range items {
			if got == item {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("RandElement returned %q, not in items", got)
		}
	}
}

func TestRandUserAgent(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 50; i++ {
		got := RandUserAgent()
		if !strings.HasPrefix(got, "Mozilla/5.0") {
			t.Fatalf("RandUserAgent() = %q, doesn't look like a user agent", got)
		}
		seen[got] = true
	}
	if len(seen) < 3 {
		t.Errorf("RandUserAgent() produced only %d unique UAs in 50 calls, expected more variety", len(seen))
	}
}

func TestRandTextUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		got := RandText(16)
		if seen[got] {
			t.Errorf("RandText(16) produced duplicate: %q", got)
		}
		seen[got] = true
	}
}

func TestShuffle(t *testing.T) {
	original := []string{"alpha", "bravo", "charlie", "delta", "echo"}
	origCopy := make([]string, len(original))
	copy(origCopy, original)

	result := Shuffle(original)

	// Length must match
	if len(result) != len(original) {
		t.Errorf("Shuffle returned %d items, want %d", len(result), len(original))
	}

	// Original slice must be unchanged
	for i, v := range original {
		if v != origCopy[i] {
			t.Errorf("original[%d] changed from %q to %q", i, origCopy[i], v)
		}
	}

	// All elements must be present in the result
	counts := make(map[string]int)
	for _, v := range original {
		counts[v]++
	}
	for _, v := range result {
		counts[v]--
	}
	for k, v := range counts {
		if v != 0 {
			t.Errorf("element %q count mismatch: %d", k, v)
		}
	}
}

func TestShuffleEmpty(t *testing.T) {
	result := Shuffle(nil)
	if len(result) != 0 {
		t.Errorf("Shuffle(nil) returned %d items, want 0", len(result))
	}

	result = Shuffle([]string{})
	if len(result) != 0 {
		t.Errorf("Shuffle([]) returned %d items, want 0", len(result))
	}
}

func TestShuffleSingleElement(t *testing.T) {
	result := Shuffle([]string{"only"})
	if len(result) != 1 || result[0] != "only" {
		t.Errorf("Shuffle single element = %v, want [only]", result)
	}
}
