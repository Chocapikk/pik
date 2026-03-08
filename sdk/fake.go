package sdk

// Faker generates realistic fake data (domains, emails, IPs, etc.).
// Set by pkg/fake via init() - only loaded when imported.
type Faker interface {
	DomainName() string
	URL() string
	IPv4Address() string
	IPv6Address() string
	Email() string
	Username() string
	FirstName() string
	LastName() string
	Name() string
}

var fakeFn Faker

// SetFaker registers the faker implementation (called from pkg/fake init).
func SetFaker(f Faker) { fakeFn = f }

// Fake returns the registered faker. Panics if not imported.
func Fake() Faker {
	if fakeFn == nil {
		panic("no faker registered (import _ \"github.com/Chocapikk/pik/pkg/fake\")")
	}
	return fakeFn
}
