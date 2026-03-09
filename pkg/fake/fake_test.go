package fake

import (
	"testing"

	"github.com/Chocapikk/pik/sdk"
)

func TestFakerRegistered(t *testing.T) {
	// init() should have registered the faker
	f := sdk.Fake()
	if f == nil {
		t.Fatal("Fake() returned nil after init")
	}
}

func TestFakerGeneratesNonEmpty(t *testing.T) {
	f := sdk.Fake()
	checks := []struct {
		name string
		fn   func() string
	}{
		{"DomainName", f.DomainName},
		{"URL", f.URL},
		{"IPv4Address", f.IPv4Address},
		{"IPv6Address", f.IPv6Address},
		{"Email", f.Email},
		{"Username", f.Username},
		{"FirstName", f.FirstName},
		{"LastName", f.LastName},
		{"Name", f.Name},
	}
	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			got := c.fn()
			if got == "" {
				t.Errorf("%s returned empty", c.name)
			}
		})
	}
}
