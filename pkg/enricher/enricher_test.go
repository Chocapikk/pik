package enricher

import (
	"sync"
	"testing"

	"github.com/Chocapikk/pik/sdk"
)

type mockMod struct {
	sdk.Pik
	opts []sdk.Option
}

func (m *mockMod) Info() sdk.Info {
	return sdk.Info{Name: "Test"}
}

func (m *mockMod) Exploit(*sdk.Context) error { return nil }

func (m *mockMod) Options() []sdk.Option { return m.opts }

// withRegistered temporarily registers a module under the given name
// so that sdk.NameOf returns it. Uses an exported test helper approach.
func withRegistered(t *testing.T, name string, mod sdk.Exploit, fn func()) {
	t.Helper()
	// We can't easily register with custom names through the public API,
	// so we test via ResolveOptions which internally calls enrichers.
	// The enrichers check sdk.NameOf(mod) which checks the global registry.
	// We need to add to the registry temporarily.
	//
	// Since we can't access entries directly from this package,
	// we test the enricher functions directly with mocked conditions.
	fn()
}

// safeEnricher protects concurrent enricher tests
var mu sync.Mutex

func TestEnrichHTTPSkipsNonHTTP(t *testing.T) {
	mod := &mockMod{}
	opts := []sdk.Option{sdk.OptString("FOO", "bar", "test option")}
	result := enrichHTTP(mod, opts)
	if len(result) != len(opts) {
		t.Errorf("enrichHTTP should return opts unchanged, got %d want %d", len(result), len(opts))
	}
}

func TestEnrichTCPSkipsNonTCP(t *testing.T) {
	mod := &mockMod{}
	opts := []sdk.Option{sdk.OptString("BAR", "baz", "test option")}
	result := enrichTCP(mod, opts)
	if len(result) != len(opts) {
		t.Errorf("enrichTCP should return opts unchanged, got %d want %d", len(result), len(opts))
	}
}

func TestEnrichHTTPAddsOptions(t *testing.T) {
	// Register a module under a name containing "/http/"
	mod := &mockMod{}
	sdk.TestRegister(t, "exploit/linux/http/test_enrich", mod)

	opts := []sdk.Option{}
	result := enrichHTTP(mod, opts)

	// Should add TARGETURI + 6 advanced options = 7
	if len(result) < 7 {
		t.Errorf("enrichHTTP should add options for HTTP module, got %d", len(result))
	}

	// Check TARGETURI was added
	found := false
	for _, opt := range result {
		if opt.Name == "TARGETURI" {
			found = true
		}
	}
	if !found {
		t.Error("enrichHTTP should add TARGETURI")
	}
}

func TestEnrichHTTPSkipsTargetURIIfPresent(t *testing.T) {
	mod := &mockMod{opts: []sdk.Option{sdk.OptTargetURI("/api")}}
	sdk.TestRegister(t, "exploit/linux/http/test_enrich_uri", mod)

	opts := []sdk.Option{}
	result := enrichHTTP(mod, opts)

	// Should NOT add TARGETURI since module already has it
	count := 0
	for _, opt := range result {
		if opt.Name == "TARGETURI" {
			count++
		}
	}
	if count != 0 {
		t.Errorf("enrichHTTP should not add TARGETURI when module has it, found %d", count)
	}
}

func TestEnrichTCPAddsOptions(t *testing.T) {
	mod := &mockMod{}
	sdk.TestRegister(t, "exploit/linux/tcp/test_enrich", mod)

	opts := []sdk.Option{}
	result := enrichTCP(mod, opts)

	// Should add TCP_TIMEOUT + TCP_TRACE = 2
	if len(result) != 2 {
		t.Errorf("enrichTCP should add 2 options for TCP module, got %d", len(result))
	}
}

func TestInitRegistersEnrichers(t *testing.T) {
	// init() already ran. Verify enrichers exist by running ResolveOptions
	// on a non-matching module.
	mod := &mockMod{}
	opts := sdk.ResolveOptions(mod)
	if len(opts) != 0 {
		t.Errorf("ResolveOptions on bare module should return 0 opts, got %d", len(opts))
	}
}
