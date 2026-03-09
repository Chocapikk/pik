package sdk

import "testing"

func TestOptionValidateRequired(t *testing.T) {
	opt := Option{Name: "TEST", Required: true}
	if err := opt.Validate(""); err == nil {
		t.Error("required empty should fail")
	}
	if err := opt.Validate("value"); err != nil {
		t.Errorf("required with value: %v", err)
	}
}

func TestOptionValidateInt(t *testing.T) {
	opt := Option{Name: "NUM", Type: TypeInt}
	if err := opt.Validate("42"); err != nil {
		t.Errorf("valid int: %v", err)
	}
	if err := opt.Validate("abc"); err == nil {
		t.Error("invalid int should fail")
	}
	if err := opt.Validate(""); err != nil {
		t.Errorf("empty int should pass: %v", err)
	}
}

func TestOptionValidatePort(t *testing.T) {
	opt := Option{Name: "PORT", Type: TypePort}
	if err := opt.Validate("80"); err != nil {
		t.Errorf("valid port: %v", err)
	}
	if err := opt.Validate("0"); err == nil {
		t.Error("port 0 should fail")
	}
	if err := opt.Validate("65536"); err == nil {
		t.Error("port 65536 should fail")
	}
	if err := opt.Validate("abc"); err == nil {
		t.Error("non-numeric port should fail")
	}
}

func TestOptionValidateBool(t *testing.T) {
	opt := Option{Name: "FLAG", Type: TypeBool}
	if err := opt.Validate("true"); err != nil {
		t.Errorf("true: %v", err)
	}
	if err := opt.Validate("false"); err != nil {
		t.Errorf("false: %v", err)
	}
	if err := opt.Validate("TRUE"); err != nil {
		t.Errorf("TRUE: %v", err)
	}
	if err := opt.Validate("yes"); err == nil {
		t.Error("yes should fail")
	}
}

func TestOptionValidateEnum(t *testing.T) {
	opt := Option{Name: "MODE", Type: TypeEnum, Enums: []string{"fast", "slow"}}
	if err := opt.Validate("fast"); err != nil {
		t.Errorf("valid enum: %v", err)
	}
	if err := opt.Validate("FAST"); err != nil {
		t.Errorf("case-insensitive enum: %v", err)
	}
	if err := opt.Validate("medium"); err == nil {
		t.Error("invalid enum should fail")
	}
}

func TestOptionValidateString(t *testing.T) {
	opt := Option{Name: "STR", Type: TypeString}
	if err := opt.Validate("anything"); err != nil {
		t.Errorf("string validation: %v", err)
	}
}

func TestOptConstructors(t *testing.T) {
	tests := []struct {
		name string
		opt  Option
		typ  OptionType
	}{
		{"TargetURI", OptTargetURI("/app"), TypePath},
		{"String", OptString("A", "b", "desc"), TypeString},
		{"Required", OptRequired("A", "b", "desc"), TypeString},
		{"Int", OptInt("A", 10, "desc"), TypeInt},
		{"Port", OptPort("A", 80, "desc"), TypePort},
		{"Bool", OptBool("A", true, "desc"), TypeBool},
		{"Enum", OptEnum("A", "x", "desc", "x", "y"), TypeEnum},
		{"Address", OptAddress("A", "0.0.0.0", "desc"), TypeAddress},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.opt.Type != tt.typ {
				t.Errorf("Type = %q, want %q", tt.opt.Type, tt.typ)
			}
		})
	}

	req := OptRequired("X", "", "desc")
	if !req.Required {
		t.Error("OptRequired should set Required")
	}
}

func TestOptAdvanced(t *testing.T) {
	opt := OptAdvanced(OptString("A", "", "desc"))
	if !opt.Advanced {
		t.Error("OptAdvanced should set Advanced")
	}
}

func TestResolveOptions(t *testing.T) {
	old := enrichers
	enrichers = nil
	defer func() { enrichers = old }()

	RegisterEnricher(func(mod Exploit, opts []Option) []Option {
		return append(opts, OptString("INJECTED", "val", "from enricher"))
	})

	mod := &mockExploit{
		info: Info{Name: "Test", Defaults: map[string]string{"CUSTOM": "override"}},
		opts: []Option{OptString("CUSTOM", "", "test")},
	}

	resolved := ResolveOptions(mod)

	// Should have CUSTOM + INJECTED
	if len(resolved) != 2 {
		t.Fatalf("ResolveOptions len = %d", len(resolved))
	}
	// CUSTOM should have default overridden
	if resolved[0].Default != "override" {
		t.Errorf("CUSTOM default = %q", resolved[0].Default)
	}
	if resolved[1].Name != "INJECTED" {
		t.Errorf("enricher option = %q", resolved[1].Name)
	}
}

func TestHasOpt(t *testing.T) {
	opts := []Option{
		{Name: "TARGET"},
		{Name: "LPORT"},
	}
	if !HasOpt(opts, "TARGET") {
		t.Error("should find TARGET")
	}
	if !HasOpt(opts, "lport") {
		t.Error("should find LPORT case-insensitive")
	}
	if HasOpt(opts, "MISSING") {
		t.Error("should not find MISSING")
	}
}
