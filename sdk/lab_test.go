package sdk

import "testing"

func TestLabManagerLateBinding(t *testing.T) {
	old := labMgr
	defer func() { labMgr = old }()

	labMgr = nil
	if GetLabManager() != nil {
		t.Error("should be nil without registration")
	}

	// Can't easily mock the full interface, just test Set/Get
	SetLabManager(nil)
	if GetLabManager() != nil {
		t.Error("should still be nil")
	}
}
