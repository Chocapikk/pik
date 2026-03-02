package http

import (
	"strings"

	"github.com/Chocapikk/pik/pkg/core"
)

func init() {
	core.RegisterEnricher(enrichHTTP)
}

func enrichHTTP(mod core.Exploit, opts []core.Option) []core.Option {
	if !strings.Contains(core.NameOf(mod), "/http/") {
		return opts
	}

	if !core.HasOption(mod, "TARGETURI") {
		opts = append(opts, core.OptTargetURI("/"))
	}

	return append(opts,
		core.OptBool("SSL", false, "Use SSL/TLS"),
		core.OptString("USER_AGENT", "random", "HTTP User-Agent"),
		core.OptInt("HTTP_TIMEOUT", 10, "HTTP request timeout in seconds"),
		core.OptBool("FOLLOW_REDIRECTS", true, "Follow HTTP redirects"),
		core.OptBool("KEEP_COOKIES", true, "Persist cookies across requests"),
	)
}
