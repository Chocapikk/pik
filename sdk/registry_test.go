package sdk

import "testing"

// testMod is a minimal exploit for registry tests.
type testMod struct {
	Pik
	info Info
}

func (m *testMod) Info() Info             { return m.info }
func (m *testMod) Exploit(*Context) error { return nil }

func withCleanRegistry(t *testing.T, fn func()) {
	t.Helper()
	old := entries
	entries = nil
	defer func() { entries = old }()
	fn()
}

func seedRegistry() (*testMod, *testMod) {
	a := &testMod{info: Info{
		Name:        "AppA",
		Description: "SQL Injection in AppA",
		Refs:        Refs(CVE("2026-1111")),
	}}
	b := &testMod{info: Info{
		Name:        "AppB",
		Description: "RCE via Command Injection",
		Refs:        Refs(CVE("2026-2222")),
	}}
	entries = []entry{
		{name: "exploit/linux/http/app_a_sqli", mod: a},
		{name: "exploit/linux/http/app_b_rce", mod: b},
	}
	return a, b
}

func TestRegisterPublic(t *testing.T) {
	withCleanRegistry(t, func() {
		mod := &testMod{info: Info{Name: "TestModule"}}
		// Register() calls register(mod, 2) -> callerModuleName(2)
		// -> runtime.Caller(3) which is the test runner.
		// This exercises the public Register wrapper.
		Register(mod)

		if len(entries) != 1 {
			t.Fatalf("entries = %d", len(entries))
		}
		if entries[0].name == "" {
			t.Error("name should not be empty")
		}
	})
}

func TestRegisterInternal(t *testing.T) {
	withCleanRegistry(t, func() {
		mod := &testMod{info: Info{Name: "TestModule"}}
		register(mod, 0)

		if len(entries) != 1 {
			t.Fatalf("entries = %d", len(entries))
		}
		if entries[0].name == "" {
			t.Error("name should not be empty")
		}
	})
}

func TestRegisterDuplicatePanic(t *testing.T) {
	withCleanRegistry(t, func() {
		mod := &testMod{info: Info{Name: "Dup"}}
		register(mod, 0)

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic on duplicate registration")
			}
		}()
		// Same caller location = same derived name -> panic
		register(mod, 0)
	})
}

func TestRegisterRawEmailPanic(t *testing.T) {
	withCleanRegistry(t, func() {
		mod := &testMod{info: Info{
			Name:    "BadEmail",
			Authors: Authors(NewAuthor("Test").WithEmail("raw@email.com")),
		}}

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic on raw email")
			}
		}()
		register(mod, 0)
	})
}

func TestRegisterValidEmail(t *testing.T) {
	withCleanRegistry(t, func() {
		mod := &testMod{info: Info{
			Name:    "GoodEmail",
			Authors: Authors(NewAuthor("Test").WithEmail("<test[at]example.com>")),
		}}
		register(mod, 0)
		if len(entries) != 1 {
			t.Error("should register with valid email")
		}
	})
}

func TestRegisterNoEmail(t *testing.T) {
	withCleanRegistry(t, func() {
		mod := &testMod{info: Info{
			Name:    "NoEmail",
			Authors: Authors(NewAuthor("Test")),
		}}
		register(mod, 0)
		if len(entries) != 1 {
			t.Error("should register with no email")
		}
	})
}

func TestModuleNameFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		// Path with modules/ marker
		{"/home/user/pik/modules/exploit/linux/http/app_sqli.go", "exploit/linux/http/app_sqli"},
		{"/go/src/modules/exploit/test_rce.go", "exploit/test_rce"},
		// Fallback: no modules/ marker
		{"/some/random/path/myfile.go", "myfile"},
		{"/usr/local/standalone.go", "standalone"},
		// Edge: nested modules/ uses last occurrence
		{"/modules/old/modules/exploit/new.go", "exploit/new"},
	}
	for _, tt := range tests {
		if got := moduleNameFromPath(tt.path); got != tt.want {
			t.Errorf("moduleNameFromPath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestCallerModuleName(t *testing.T) {
	// Called from test file (not under modules/) -> fallback to base filename
	name := callerModuleName(0)
	if name == "" {
		t.Error("callerModuleName should return non-empty")
	}
	if Contains(name, ".go") {
		t.Errorf("name should not contain .go: %q", name)
	}
}

func TestCallerModuleNamePanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for bad skip")
		}
	}()
	callerModuleName(999)
}

func TestGet(t *testing.T) {
	withCleanRegistry(t, func() {
		a, _ := seedRegistry()

		// Exact match
		if got := Get("exploit/linux/http/app_a_sqli"); got != a {
			t.Error("exact match failed")
		}

		// Short name (base name, case-insensitive)
		if got := Get("app_b_rce"); got == nil {
			t.Error("short name match failed")
		}

		// Not found
		if got := Get("nonexistent"); got != nil {
			t.Error("should return nil for missing")
		}
	})
}

func TestNameOf(t *testing.T) {
	withCleanRegistry(t, func() {
		a, b := seedRegistry()
		if got := NameOf(a); got != "exploit/linux/http/app_a_sqli" {
			t.Errorf("NameOf(a) = %q", got)
		}
		if got := NameOf(b); got != "exploit/linux/http/app_b_rce" {
			t.Errorf("NameOf(b) = %q", got)
		}
		unknown := &testMod{}
		if got := NameOf(unknown); got != "unknown" {
			t.Errorf("NameOf(unknown) = %q", got)
		}
	})
}

func TestList(t *testing.T) {
	withCleanRegistry(t, func() {
		seedRegistry()
		list := List()
		if len(list) != 2 {
			t.Errorf("List len = %d", len(list))
		}
	})
}

func TestNames(t *testing.T) {
	withCleanRegistry(t, func() {
		seedRegistry()
		names := Names()
		if len(names) != 2 {
			t.Fatalf("Names len = %d", len(names))
		}
		if names[0] != "exploit/linux/http/app_a_sqli" {
			t.Errorf("Names[0] = %q", names[0])
		}
	})
}

func TestSearch(t *testing.T) {
	withCleanRegistry(t, func() {
		seedRegistry()

		// Search by name
		if got := Search("AppA"); len(got) != 1 {
			t.Errorf("Search(AppA) = %d", len(got))
		}

		// Search by description
		if got := Search("command injection"); len(got) != 1 {
			t.Errorf("Search(command injection) = %d", len(got))
		}

		// Search by CVE
		if got := Search("2026-1111"); len(got) != 1 {
			t.Errorf("Search(CVE) = %d", len(got))
		}

		// Search by path
		if got := Search("app_b"); len(got) != 1 {
			t.Errorf("Search(path) = %d", len(got))
		}

		// No match
		if got := Search("zzzzz"); len(got) != 0 {
			t.Errorf("Search(no match) = %d", len(got))
		}
	})
}

func TestRankings(t *testing.T) {
	withCleanRegistry(t, func() {
		entries = []entry{
			{name: "a", mod: &testMod{info: Info{
				Authors: Authors(NewAuthor("Alice"), NewAuthor("Bob")),
				Refs:    Refs(CVE("2026-1")),
			}}},
			{name: "b", mod: &testMod{info: Info{
				Authors: Authors(NewAuthor("Alice")),
				Refs:    Refs(CVE("2026-2"), CVE("2026-3")),
			}}},
		}

		ranks := Rankings()
		if len(ranks) != 2 {
			t.Fatalf("Rankings len = %d", len(ranks))
		}
		// Alice should be first (2 modules)
		if ranks[0].Name != "Alice" || ranks[0].Modules != 2 || ranks[0].CVEs != 3 {
			t.Errorf("ranks[0] = %+v", ranks[0])
		}
		if ranks[1].Name != "Bob" || ranks[1].Modules != 1 || ranks[1].CVEs != 1 {
			t.Errorf("ranks[1] = %+v", ranks[1])
		}
	})
}

func TestRankingsTieBreaker(t *testing.T) {
	withCleanRegistry(t, func() {
		// Both have 1 module, but Charlie has more CVEs
		entries = []entry{
			{name: "a", mod: &testMod{info: Info{
				Authors: Authors(NewAuthor("Charlie")),
				Refs:    Refs(CVE("2026-1"), CVE("2026-2")),
			}}},
			{name: "b", mod: &testMod{info: Info{
				Authors: Authors(NewAuthor("Dave")),
				Refs:    Refs(CVE("2026-3")),
			}}},
		}

		ranks := Rankings()
		if len(ranks) != 2 {
			t.Fatalf("len = %d", len(ranks))
		}
		// Charlie should be first (same modules=1, but more CVEs)
		if ranks[0].Name != "Charlie" {
			t.Errorf("tie-breaker: ranks[0] = %+v, ranks[1] = %+v", ranks[0], ranks[1])
		}
	})
}
