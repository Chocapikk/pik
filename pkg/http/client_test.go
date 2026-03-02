package http

import (
	"testing"
)

func TestNormalizeURI(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{"single root", []string{"/"}, "/"},
		{"single path", []string{"/api"}, "/api"},
		{"two segments", []string{"/app", "/api"}, "/app/api"},
		{"trailing slash preserved", []string{"/app", "/api/"}, "/app/api/"},
		{"no leading slash added", []string{"app", "api"}, "/app/api"},
		{"double slash removed", []string{"/app/", "/api"}, "/app/api"},
		{"empty parts ignored", []string{"", "/api"}, "/api"},
		{"all empty", []string{"", ""}, "/"},
		{"root plus path", []string{"/", "/test"}, "/test"},
		{"three segments", []string{"/a", "/b", "/c"}, "/a/b/c"},
		{"trailing slash only on last", []string{"/a/", "/b/"}, "/a/b/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeURI(tt.parts...)
			if got != tt.want {
				t.Errorf("NormalizeURI(%v) = %q, want %q", tt.parts, got, tt.want)
			}
		})
	}
}
