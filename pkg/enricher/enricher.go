// Package enricher registers protocol option enrichers.
// Protocol client factories are registered separately by their own init().
package enricher

import "github.com/Chocapikk/pik/sdk"

func init() {
	sdk.RegisterEnricher(enrichHTTP)
	sdk.RegisterEnricher(enrichTCP)
}
