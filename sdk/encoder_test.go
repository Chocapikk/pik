package sdk

import "testing"

func TestEncoderRegistry(t *testing.T) {
	// Save and restore state
	old := encoders
	encoders = nil
	defer func() { encoders = old }()

	RegisterEncoder(&Encoder{Name: "base64", Platform: "linux", Desc: "Base64 encoder", Fn: Base64Encode})
	RegisterEncoder(&Encoder{Name: "rot13", Platform: "", Desc: "ROT13", Fn: ROT13})
	RegisterEncoder(&Encoder{Name: "winb64", Platform: "windows", Desc: "Windows B64", Fn: Base64Encode})

	// ListEncoders for linux: base64 + rot13 (any platform)
	linux := ListEncoders("linux")
	if len(linux) != 2 {
		t.Errorf("linux encoders = %d, want 2", len(linux))
	}

	// ListEncoders for windows: winb64 + rot13
	win := ListEncoders("windows")
	if len(win) != 2 {
		t.Errorf("windows encoders = %d, want 2", len(win))
	}

	// ListEncoders for all
	all := ListEncoders("")
	if len(all) != 3 {
		t.Errorf("all encoders = %d, want 3", len(all))
	}

	// EncoderNames
	names := EncoderNames("linux")
	if len(names) != 2 || names[0] != "base64" {
		t.Errorf("names = %v", names)
	}

	// GetEncoder
	enc := GetEncoder("rot13")
	if enc == nil || enc.Name != "rot13" {
		t.Error("GetEncoder(rot13) failed")
	}
	if GetEncoder("missing") != nil {
		t.Error("should return nil for missing")
	}

	// Test encoder function
	if got := enc.Fn("Hello"); got != "Uryyb" {
		t.Errorf("rot13.Fn = %q", got)
	}
}
