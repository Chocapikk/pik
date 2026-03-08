package fake

import (
	"github.com/Chocapikk/pik/sdk"
	"github.com/brianvoe/gofakeit/v7"
)

func init() {
	sdk.SetFaker(&faker{gofakeit.New(0)})
}

type faker struct{ f *gofakeit.Faker }

func (f *faker) DomainName() string   { return f.f.DomainName() }
func (f *faker) URL() string          { return f.f.URL() }
func (f *faker) IPv4Address() string  { return f.f.IPv4Address() }
func (f *faker) IPv6Address() string  { return f.f.IPv6Address() }
func (f *faker) Email() string        { return f.f.Email() }
func (f *faker) Username() string     { return f.f.Username() }
func (f *faker) FirstName() string    { return f.f.FirstName() }
func (f *faker) LastName() string     { return f.f.LastName() }
func (f *faker) Name() string         { return f.f.Name() }
