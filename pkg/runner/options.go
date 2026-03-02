package runner

import "github.com/Chocapikk/pik/sdk"

func init() {
	sdk.RegisterEnricher(enrichBase)
	sdk.RegisterEnricher(enrichC2)
	sdk.RegisterEnricher(enrichCmdStager)
	sdk.RegisterEnricher(enrichScan)
}

func enrichBase(mod sdk.Exploit, opts []sdk.Option) []sdk.Option {
	if !sdk.HasOpt(opts, "RPORT") {
		opts = append(opts, sdk.OptAdvanced(sdk.OptPort("RPORT", 80, "Target port")))
	}
	return append(opts,
		sdk.Option{Name: "LHOST", Type: sdk.TypeAddress, Required: true, Desc: "Callback host for payload"},
		sdk.OptPort("LPORT", 4444, "Callback port for payload"),
		sdk.OptEnum("PAYLOAD", "reverse_bash", "Payload type", "reverse_bash", "reverse_python", "reverse_perl", "reverse_powershell"),
		sdk.OptAdvanced(sdk.OptString("PROXIES", "", "Proxy URL (http://host:port or socks5://host:port)")),
	)
}

func enrichC2(_ sdk.Exploit, opts []sdk.Option) []sdk.Option {
	if !sdk.HasOpt(opts, "LHOST") {
		return opts
	}
	return append(opts,
		sdk.OptAdvanced(sdk.OptEnum("C2", "shell", "C2 backend", "shell", "sslshell", "httpshell", "sliver")),
		sdk.OptAdvanced(sdk.OptString("C2CONFIG", "", "C2 config file (sliver)")),
		sdk.OptAdvanced(sdk.OptAddress("SRVHOST", "", "Local bind address (default: LHOST)")),
		sdk.OptAdvanced(sdk.OptPort("SRVPORT", 0, "Local bind port (default: LPORT)")),
		sdk.OptAdvanced(sdk.OptString("TUNNEL", "", "Tunnel URL for staging (ngrok, bore, cloudflared)")),
		sdk.OptAdvanced(sdk.OptString("REMOTE_PATH", "", "Remote drop path (default: random /tmp)")),
		sdk.OptAdvanced(sdk.OptInt("WAITSESSION", 30, "Session wait timeout in seconds")),
		sdk.OptAdvanced(sdk.OptString("ARCH", "amd64", "Target architecture")),
		sdk.OptAdvanced(sdk.OptEnum("FETCH_COMMAND", "curl", "Staging download method", "curl", "wget", "python", "perl", "php", "certutil", "powershell", "tcp")),
	)
}

func enrichCmdStager(mod sdk.Exploit, opts []sdk.Option) []sdk.Option {
	if _, ok := mod.(sdk.CmdStager); !ok {
		return opts
	}
	return append(opts,
		sdk.OptAdvanced(sdk.OptEnum("CMDSTAGER", "printf", "CmdStager flavor", "printf", "bourne")),
		sdk.OptAdvanced(sdk.OptInt("CMDSTAGER_LINEMAX", 2047, "Max command line length")),
	)
}

func enrichScan(_ sdk.Exploit, opts []sdk.Option) []sdk.Option {
	return append(opts, sdk.OptAdvanced(sdk.OptInt("THREADS", 10, "Scan threads")))
}

