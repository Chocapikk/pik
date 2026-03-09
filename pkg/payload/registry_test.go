package payload

import "testing"

func TestListPayloads(t *testing.T) {
	all := ListPayloads()
	if len(all) == 0 {
		t.Fatal("ListPayloads returned empty")
	}

	for _, pl := range all {
		if pl.Name == "" {
			t.Error("payload with empty name")
		}
		if pl.Generate == nil {
			t.Errorf("payload %q has nil Generate func", pl.Name)
		}
		if pl.Type == "" {
			t.Errorf("payload %q has empty type", pl.Name)
		}
	}
}

func TestListFor(t *testing.T) {
	// Filter by type only.
	cmd := ListFor("cmd", "")
	if len(cmd) == 0 {
		t.Error("no cmd payloads")
	}
	for _, pl := range cmd {
		if pl.Type != "cmd" {
			t.Errorf("ListFor(cmd, '') returned type %q", pl.Type)
		}
	}

	py := ListFor("py", "")
	if len(py) == 0 {
		t.Error("no py payloads")
	}
	for _, pl := range py {
		if pl.Type != "py" {
			t.Errorf("ListFor(py, '') returned type %q", pl.Type)
		}
	}

	// Filter by type + platform.
	linux := ListFor("cmd", "linux")
	if len(linux) == 0 {
		t.Error("no cmd/linux payloads")
	}
	for _, pl := range linux {
		if pl.Platform != "linux" {
			t.Errorf("ListFor(cmd, linux) returned platform %q", pl.Platform)
		}
	}

	windows := ListFor("cmd", "windows")
	if len(windows) == 0 {
		t.Error("no cmd/windows payloads")
	}
	for _, pl := range windows {
		if pl.Platform != "windows" {
			t.Errorf("ListFor(cmd, windows) returned platform %q", pl.Platform)
		}
	}

	// py/ payloads are cross-platform, should appear for any platform.
	pyLinux := ListFor("py", "linux")
	if len(pyLinux) != len(py) {
		t.Errorf("ListFor(py, linux) = %d, want %d (py payloads are cross-platform)", len(pyLinux), len(py))
	}

	// No filter = all.
	all := ListFor("", "")
	if len(all) != len(ListPayloads()) {
		t.Error("ListFor('', '') should return all payloads")
	}
}

func TestGetPayload(t *testing.T) {
	pl := GetPayload("cmd/bash/reverse_tcp")
	if pl == nil {
		t.Fatal("GetPayload(cmd/bash/reverse_tcp) = nil")
	}
	if pl.Name != "cmd/bash/reverse_tcp" {
		t.Errorf("name = %q", pl.Name)
	}

	result := pl.Generate("10.0.0.1", 4444)
	if result == "" {
		t.Error("Generate returned empty")
	}
}

func TestGetPayloadNotFound(t *testing.T) {
	if pl := GetPayload("nonexistent"); pl != nil {
		t.Errorf("GetPayload(nonexistent) = %+v, want nil", pl)
	}
}

func TestDefaultFor(t *testing.T) {
	linux := DefaultFor("cmd", "linux")
	if linux == nil || linux.Name != "cmd/bash/reverse_tcp" {
		t.Errorf("DefaultFor(cmd, linux) = %v", linux)
	}

	windows := DefaultFor("cmd", "windows")
	if windows == nil || windows.Name != "cmd/powershell/reverse_tcp" {
		t.Errorf("DefaultFor(cmd, windows) = %v", windows)
	}

	py := DefaultFor("py", "")
	if py == nil || py.Name != "py/reverse_tcp" {
		t.Errorf("DefaultFor(py, '') = %v", py)
	}

	fallback := DefaultFor("", "unknown")
	if fallback == nil || fallback.Name != "cmd/bash/reverse_tcp" {
		t.Errorf("DefaultFor('', unknown) = %v", fallback)
	}
}

func TestPayloadNamesUnique(t *testing.T) {
	seen := make(map[string]bool)
	for _, pl := range ListPayloads() {
		if seen[pl.Name] {
			t.Errorf("duplicate payload name: %q", pl.Name)
		}
		seen[pl.Name] = true
	}
}
