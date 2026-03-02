package http

import (
	"strings"

	"github.com/Chocapikk/pik/sdk"
)

func init() {
	sdk.RegisterEnricher(enrichHTTP)
}

func enrichHTTP(mod sdk.Exploit, opts []sdk.Option) []sdk.Option {
	if !strings.Contains(sdk.NameOf(mod), "/http/") {
		return opts
	}

	if !sdk.HasOption(mod, "TARGETURI") {
		opts = append(opts, sdk.OptTargetURI("/"))
	}

	return append(opts,
		sdk.OptBool("SSL", false, "Use SSL/TLS"),
		sdk.OptString("USER_AGENT", "random", "HTTP User-Agent"),
		sdk.OptInt("HTTP_TIMEOUT", 10, "HTTP request timeout in seconds"),
		sdk.OptBool("FOLLOW_REDIRECTS", true, "Follow HTTP redirects"),
		sdk.OptBool("KEEP_COOKIES", true, "Persist cookies across requests"),
	)
}
