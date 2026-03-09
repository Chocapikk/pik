package sdk

import (
	"strings"
	"testing"
)

func TestMultipart(t *testing.T) {
	body, ct := Multipart(map[string]string{"key": "value"})
	if !strings.HasPrefix(ct, "multipart/form-data; boundary=") {
		t.Errorf("content-type = %q", ct)
	}
	if !strings.Contains(body, "key") || !strings.Contains(body, "value") {
		t.Errorf("body missing parts: %q", body)
	}
	if !strings.HasSuffix(body, "--") {
		t.Errorf("body should end with boundary--: %q", body)
	}
}

func TestMultipartOrdered(t *testing.T) {
	body, ct := MultipartOrdered("BOUNDARY", "a", "1", "b", "2")
	if !strings.Contains(ct, "BOUNDARY") {
		t.Errorf("content-type = %q", ct)
	}
	// Check order: a should appear before b
	aIdx := strings.Index(body, `name="a"`)
	bIdx := strings.Index(body, `name="b"`)
	if aIdx < 0 || bIdx < 0 {
		t.Fatalf("missing parts in body: %q", body)
	}
	if aIdx >= bIdx {
		t.Error("parts should be ordered: a before b")
	}
}

func TestMultipartOrderedPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for odd args")
		}
	}()
	MultipartOrdered("B", "a", "1", "b")
}
