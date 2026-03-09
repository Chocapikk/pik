package runner

import (
	"context"
	"strings"
	"testing"

	"github.com/Chocapikk/pik/sdk"
)

// --- Fake module for testing ---

type fakeModule struct {
	sdk.Pik
	info sdk.Info
}

func (m *fakeModule) Info() sdk.Info                    { return m.info }
func (m *fakeModule) Exploit(run *sdk.Context) error    { return nil }

// --- resolveTarget ---

func TestResolveTarget_NoTargets(t *testing.T) {
	mod := &fakeModule{info: sdk.Info{
		Targets: nil,
	}}
	params := sdk.NewParams(context.Background(), nil)

	got := resolveTarget(mod, params)

	// With no targets, should return a Target with Platform from Info().Platform().
	// Info().Platform() defaults to "linux" when no targets are defined.
	if got.Platform != "linux" {
		t.Errorf("resolveTarget with no targets: Platform = %q, want %q", got.Platform, "linux")
	}
	if got.Name != "" {
		t.Errorf("resolveTarget with no targets: Name = %q, want empty", got.Name)
	}
	if got.Type != "" {
		t.Errorf("resolveTarget with no targets: Type = %q, want empty", got.Type)
	}
}

func TestResolveTarget_SingleTarget(t *testing.T) {
	target := sdk.Target{Name: "Linux Command Shell", Platform: "linux", Type: "cmd"}
	mod := &fakeModule{info: sdk.Info{
		Targets: []sdk.Target{target},
	}}
	params := sdk.NewParams(context.Background(), nil)

	got := resolveTarget(mod, params)

	if got.Name != target.Name {
		t.Errorf("Name = %q, want %q", got.Name, target.Name)
	}
	if got.Platform != target.Platform {
		t.Errorf("Platform = %q, want %q", got.Platform, target.Platform)
	}
	if got.Type != target.Type {
		t.Errorf("Type = %q, want %q", got.Type, target.Type)
	}
}

func TestResolveTarget_MultipleTargets(t *testing.T) {
	targets := []sdk.Target{
		{Name: "Linux", Platform: "linux", Type: "cmd"},
		{Name: "Windows", Platform: "windows", Type: "cmd"},
		{Name: "Python", Platform: "", Type: "py"},
	}
	mod := &fakeModule{info: sdk.Info{Targets: targets}}

	tests := []struct {
		name         string
		targetIndex  string
		wantName     string
	}{
		{"default (no index)", "", "Linux"},
		{"index 0", "0", "Linux"},
		{"index 1", "1", "Windows"},
		{"index 2", "2", "Python"},
		{"negative index clamps to 0", "-1", "Linux"},
		{"out of bounds clamps to 0", "5", "Linux"},
		{"non-numeric defaults to 0", "abc", "Linux"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := map[string]string{}
			if tt.targetIndex != "" {
				values["TARGET_INDEX"] = tt.targetIndex
			}
			params := sdk.NewParams(context.Background(), values)

			got := resolveTarget(mod, params)
			if got.Name != tt.wantName {
				t.Errorf("resolveTarget with TARGET_INDEX=%q: Name = %q, want %q",
					tt.targetIndex, got.Name, tt.wantName)
			}
		})
	}
}

func TestResolveTarget_PlatformFallback(t *testing.T) {
	// When no targets defined but module has windows targets in a different field,
	// Platform() on an empty Targets slice returns "linux" (default).
	mod := &fakeModule{info: sdk.Info{}}
	params := sdk.NewParams(context.Background(), nil)

	got := resolveTarget(mod, params)
	if got.Platform != "linux" {
		t.Errorf("Platform = %q, want %q", got.Platform, "linux")
	}
}

// --- remotePath ---

func TestRemotePath_Default(t *testing.T) {
	params := sdk.NewParams(context.Background(), nil)

	got := remotePath(params)

	if !strings.HasPrefix(got, "/tmp/.") {
		t.Errorf("remotePath() = %q, want prefix /tmp/.", got)
	}
	// /tmp/. + 8 random chars = 14 chars total
	if len(got) != 14 {
		t.Errorf("remotePath() length = %d, want 14", len(got))
	}
}

func TestRemotePath_Uniqueness(t *testing.T) {
	params := sdk.NewParams(context.Background(), nil)

	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		got := remotePath(params)
		if seen[got] {
			t.Errorf("remotePath() produced duplicate: %q", got)
		}
		seen[got] = true
	}
}

func TestRemotePath_CustomPath(t *testing.T) {
	params := sdk.NewParams(context.Background(), map[string]string{
		"REMOTE_PATH": "/var/tmp/exploit",
	})

	got := remotePath(params)
	if got != "/var/tmp/exploit" {
		t.Errorf("remotePath() = %q, want %q", got, "/var/tmp/exploit")
	}
}

func TestRemotePath_RandomCharset(t *testing.T) {
	params := sdk.NewParams(context.Background(), nil)

	// The random suffix should be lowercase alpha (from text.RandText).
	for i := 0; i < 50; i++ {
		got := remotePath(params)
		suffix := got[len("/tmp/."):]
		for _, c := range suffix {
			if c < 'a' || c > 'z' {
				t.Errorf("remotePath() suffix contains non-lowercase char %q in %q", c, got)
				return
			}
		}
	}
}

// --- resolveEncoder ---

func TestResolveEncoder_DefaultLinux(t *testing.T) {
	// With no explicit ENCODER and linux platform, resolveEncoder should
	// return a working function (either cmd/base64 if registered, or identity).
	params := sdk.NewParams(context.Background(), nil)

	enc := resolveEncoder(params, "linux")

	input := "echo hello"
	got := enc(input)
	if got == "" {
		t.Error("resolveEncoder returned function that produces empty string")
	}
	// The result should be different from input if an encoder is active,
	// or equal if identity fallback is used. Either way, it must be non-empty.
}

func TestResolveEncoder_ExplicitEncoder(t *testing.T) {
	// When ENCODER is set but doesn't match any registered encoder,
	// it falls through to the platform default or identity.
	params := sdk.NewParams(context.Background(), map[string]string{
		"ENCODER": "nonexistent_encoder",
	})

	enc := resolveEncoder(params, "linux")

	// Should still return something callable (identity fallback).
	input := "test payload"
	got := enc(input)
	if got == "" {
		t.Error("resolveEncoder returned function that produces empty string")
	}
}

// --- resolveTarget with sdk helpers ---

func TestResolveTarget_LinuxCmd(t *testing.T) {
	mod := &fakeModule{info: sdk.Info{Targets: sdk.LinuxCmd()}}
	params := sdk.NewParams(context.Background(), nil)

	got := resolveTarget(mod, params)
	if got.Platform != "linux" {
		t.Errorf("Platform = %q, want linux", got.Platform)
	}
	if got.Type != "cmd" {
		t.Errorf("Type = %q, want cmd", got.Type)
	}
}

func TestResolveTarget_MultiCmd(t *testing.T) {
	mod := &fakeModule{info: sdk.Info{Targets: sdk.MultiCmd()}}

	// Index 0 should be Linux.
	params := sdk.NewParams(context.Background(), map[string]string{"TARGET_INDEX": "0"})
	got := resolveTarget(mod, params)
	if got.Platform != "linux" {
		t.Errorf("MultiCmd index 0: Platform = %q, want linux", got.Platform)
	}

	// Index 1 should be Windows.
	params = sdk.NewParams(context.Background(), map[string]string{"TARGET_INDEX": "1"})
	got = resolveTarget(mod, params)
	if got.Platform != "windows" {
		t.Errorf("MultiCmd index 1: Platform = %q, want windows", got.Platform)
	}
}

func TestResolveTarget_PythonCmd(t *testing.T) {
	mod := &fakeModule{info: sdk.Info{Targets: sdk.PythonCmd()}}
	params := sdk.NewParams(context.Background(), nil)

	got := resolveTarget(mod, params)
	if got.Type != "py" {
		t.Errorf("Type = %q, want py", got.Type)
	}
}
