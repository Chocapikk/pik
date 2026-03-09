package sdk

import (
	"context"
	"testing"
)

func TestNewParams(t *testing.T) {
	p := NewParams(context.Background(), nil)
	if p.Get("X") != "" {
		t.Error("nil map should return empty")
	}
}

func TestParamsGetSet(t *testing.T) {
	p := NewParams(context.Background(), map[string]string{"TARGET": "10.0.0.1"})
	if got := p.Get("TARGET"); got != "10.0.0.1" {
		t.Errorf("Get = %q", got)
	}

	// Get uppercases the key
	p.Set("lhost", "10.0.0.2")
	if got := p.Get("LHOST"); got != "10.0.0.2" {
		t.Errorf("Get after Set = %q", got)
	}
}

func TestParamsGetOr(t *testing.T) {
	p := NewParams(context.Background(), map[string]string{"A": "val"})
	if got := p.GetOr("A", "def"); got != "val" {
		t.Errorf("GetOr existing = %q", got)
	}
	if got := p.GetOr("B", "def"); got != "def" {
		t.Errorf("GetOr missing = %q", got)
	}
}

func TestParamsInt(t *testing.T) {
	p := NewParams(context.Background(), map[string]string{"PORT": "8080", "BAD": "abc"})
	if got := p.Int("PORT"); got != 8080 {
		t.Errorf("Int = %d", got)
	}
	if got := p.Int("BAD"); got != 0 {
		t.Errorf("Int bad = %d", got)
	}
	if got := p.Int("MISSING"); got != 0 {
		t.Errorf("Int missing = %d", got)
	}
}

func TestParamsIntOr(t *testing.T) {
	p := NewParams(context.Background(), map[string]string{"PORT": "9090"})
	if got := p.IntOr("PORT", 80); got != 9090 {
		t.Errorf("IntOr = %d", got)
	}
	if got := p.IntOr("MISSING", 80); got != 80 {
		t.Errorf("IntOr missing = %d", got)
	}
}

func TestParamsShortcuts(t *testing.T) {
	p := NewParams(context.Background(), map[string]string{
		"TARGET": "10.0.0.1",
		"LHOST":  "10.0.0.2",
		"LPORT":  "5555",
		"ARCH":   "arm64",
		"TUNNEL": "https://tun.nel",
	})
	if p.Target() != "10.0.0.1" {
		t.Errorf("Target = %q", p.Target())
	}
	if p.Lhost() != "10.0.0.2" {
		t.Errorf("Lhost = %q", p.Lhost())
	}
	if p.Lport() != 5555 {
		t.Errorf("Lport = %d", p.Lport())
	}
	if p.Arch() != "arm64" {
		t.Errorf("Arch = %q", p.Arch())
	}
	if p.Tunnel() != "https://tun.nel" {
		t.Errorf("Tunnel = %q", p.Tunnel())
	}
}

func TestParamsLportDefault(t *testing.T) {
	p := NewParams(context.Background(), nil)
	if got := p.Lport(); got != 4444 {
		t.Errorf("Lport default = %d", got)
	}
}

func TestParamsArchDefault(t *testing.T) {
	p := NewParams(context.Background(), nil)
	if got := p.Arch(); got != "amd64" {
		t.Errorf("Arch default = %q", got)
	}
}

func TestParamsMap(t *testing.T) {
	orig := map[string]string{"A": "1", "B": "2"}
	p := NewParams(context.Background(), orig)
	m := p.Map()
	m["A"] = "changed"
	if p.Get("A") != "1" {
		t.Error("Map should return a copy")
	}
}

func TestParamsClone(t *testing.T) {
	p := NewParams(context.Background(), map[string]string{"X": "1"})
	c := p.Clone()
	c.Set("X", "2")
	if p.Get("X") != "1" {
		t.Error("Clone should be independent")
	}
}
