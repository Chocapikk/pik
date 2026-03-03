package enricher

import (
	"strings"

	"github.com/Chocapikk/pik/sdk"
)

func enrichTCP(mod sdk.Exploit, opts []sdk.Option) []sdk.Option {
	if !strings.Contains(sdk.NameOf(mod), "/tcp/") {
		return opts
	}

	return append(opts,
		sdk.OptAdvanced(sdk.OptInt("TCP_TIMEOUT", 10, "TCP connection timeout in seconds")),
		sdk.OptAdvanced(sdk.OptBool("TCP_TRACE", false, "Show TCP send/recv hex traces")),
	)
}
