package sdk

import "testing"

func TestFakePanicWhenNotRegistered(t *testing.T) {
	old := fakeFn
	fakeFn = nil
	defer func() { fakeFn = old }()

	defer func() {
		if r := recover(); r == nil {
			t.Error("Fake() should panic when no faker registered")
		}
	}()
	Fake()
}

type stubFaker struct{}

func (stubFaker) DomainName() string   { return "example.com" }
func (stubFaker) URL() string          { return "https://example.com" }
func (stubFaker) IPv4Address() string  { return "10.0.0.1" }
func (stubFaker) IPv6Address() string  { return "::1" }
func (stubFaker) Email() string        { return "test@example.com" }
func (stubFaker) Username() string     { return "testuser" }
func (stubFaker) FirstName() string    { return "John" }
func (stubFaker) LastName() string     { return "Doe" }
func (stubFaker) Name() string         { return "John Doe" }

func TestSetFaker(t *testing.T) {
	old := fakeFn
	defer func() { fakeFn = old }()

	SetFaker(stubFaker{})
	f := Fake()
	if f.DomainName() != "example.com" {
		t.Errorf("DomainName = %q", f.DomainName())
	}
	if f.IPv4Address() != "10.0.0.1" {
		t.Errorf("IPv4Address = %q", f.IPv4Address())
	}
}
