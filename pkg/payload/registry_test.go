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
		if pl.Platform == "" {
			t.Errorf("payload %q has empty platform", pl.Name)
		}
	}
}

func TestListForPlatform(t *testing.T) {
	linux := ListForPlatform("linux")
	if len(linux) == 0 {
		t.Error("no linux payloads")
	}
	for _, pl := range linux {
		if pl.Platform != "linux" {
			t.Errorf("ListForPlatform(linux) returned %q platform", pl.Platform)
		}
	}

	windows := ListForPlatform("windows")
	if len(windows) == 0 {
		t.Error("no windows payloads")
	}
	for _, pl := range windows {
		if pl.Platform != "windows" {
			t.Errorf("ListForPlatform(windows) returned %q platform", pl.Platform)
		}
	}

	all := ListForPlatform("")
	if len(all) != len(ListPayloads()) {
		t.Error("ListForPlatform('') should return all payloads")
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

func TestDefaultPayload(t *testing.T) {
	linux := DefaultPayload("linux")
	if linux == nil || linux.Name != "cmd/bash/reverse_tcp" {
		t.Errorf("DefaultPayload(linux) = %v", linux)
	}

	windows := DefaultPayload("windows")
	if windows == nil || windows.Name != "cmd/powershell/reverse_tcp" {
		t.Errorf("DefaultPayload(windows) = %v", windows)
	}

	fallback := DefaultPayload("unknown")
	if fallback == nil || fallback.Name != "cmd/bash/reverse_tcp" {
		t.Errorf("DefaultPayload(unknown) = %v", fallback)
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
