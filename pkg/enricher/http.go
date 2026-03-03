package enricher

import (
	"strings"

	"github.com/Chocapikk/pik/sdk"
)

func enrichHTTP(mod sdk.Exploit, opts []sdk.Option) []sdk.Option {
	if !strings.Contains(sdk.NameOf(mod), "/http/") {
		return opts
	}

	if !sdk.HasOption(mod, "TARGETURI") {
		opts = append(opts, sdk.OptTargetURI("/"))
	}

	return append(opts,
		sdk.OptAdvanced(sdk.OptBool("SSL", false, "Use SSL/TLS")),
		sdk.OptAdvanced(sdk.OptString("USER_AGENT", "random", "HTTP User-Agent")),
		sdk.OptAdvanced(sdk.OptInt("HTTP_TIMEOUT", 10, "HTTP request timeout in seconds")),
		sdk.OptAdvanced(sdk.OptBool("FOLLOW_REDIRECTS", true, "Follow HTTP redirects")),
		sdk.OptAdvanced(sdk.OptBool("KEEP_COOKIES", true, "Persist cookies across requests")),
		sdk.OptAdvanced(sdk.OptBool("HTTP_TRACE", false, "Show HTTP request/response traces")),
	)
}
