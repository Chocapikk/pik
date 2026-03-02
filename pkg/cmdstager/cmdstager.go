package cmdstager

import "fmt"

// Flavor selects the encoding strategy for chunked delivery.
type Flavor string

const (
	FlavorPrintf  Flavor = "printf"
	FlavorBourne  Flavor = "bourne"
	DefaultFlavor        = FlavorPrintf
)

// Generate encodes a binary into shell commands using the specified flavor.
func Generate(binary []byte, flavor Flavor, opts Options) ([]string, error) {
	switch flavor {
	case FlavorPrintf, "":
		return Printf(binary, opts), nil
	case FlavorBourne:
		return Bourne(binary, opts), nil
	default:
		return nil, fmt.Errorf("unknown cmdstager flavor: %q", flavor)
	}
}
