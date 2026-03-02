package runner

import "github.com/Chocapikk/pik/pkg/core"

func init() {
	core.RegisterEnricher(enrichBase)
	core.RegisterEnricher(enrichC2)
	core.RegisterEnricher(enrichCmdStager)
	core.RegisterEnricher(enrichScan)
}

func enrichBase(mod core.Exploit, opts []core.Option) []core.Option {
	if !core.HasOpt(opts, "RPORT") {
		opts = append(opts, core.OptPort("RPORT", 80, "Target port"))
	}
	return append(opts,
		core.Option{Name: "LHOST", Type: core.TypeAddress, Required: true, Desc: "Callback host for payload"},
		core.OptPort("LPORT", 4444, "Callback port for payload"),
		core.OptEnum("PAYLOAD", "reverse_bash", "Payload type", "reverse_bash", "reverse_python", "reverse_perl", "reverse_powershell"),
	)
}

func enrichC2(_ core.Exploit, opts []core.Option) []core.Option {
	if !core.HasOpt(opts, "LHOST") {
		return opts
	}
	return append(opts,
		core.OptEnum("C2", "shell", "C2 backend", "shell", "sliver"),
		core.OptString("C2CONFIG", "", "C2 config file (sliver)"),
		core.OptAddress("SRVHOST", "", "Local bind address (default: LHOST)"),
		core.OptPort("SRVPORT", 0, "Local bind port (default: LPORT)"),
		core.OptString("TUNNEL", "", "Tunnel URL for staging (ngrok, bore, cloudflared)"),
		core.OptString("REMOTE_PATH", "", "Remote drop path (default: random /tmp)"),
		core.OptInt("WAITSESSION", 30, "Session wait timeout in seconds"),
		core.OptString("ARCH", "amd64", "Target architecture"),
		core.OptEnum("FETCH_COMMAND", "curl", "Staging download method", "curl", "wget", "python", "perl", "php", "certutil", "powershell", "tcp"),
	)
}

func enrichCmdStager(mod core.Exploit, opts []core.Option) []core.Option {
	if _, ok := mod.(core.CmdStager); !ok {
		return opts
	}
	return append(opts,
		core.OptEnum("DELIVERY", "staging", "Delivery method", "staging", "cmdstager"),
		core.OptEnum("CMDSTAGER", "printf", "CmdStager flavor", "printf", "bourne"),
		core.OptInt("CMDSTAGER_LINEMAX", 2047, "Max command line length"),
	)
}

func enrichScan(_ core.Exploit, opts []core.Option) []core.Option {
	return append(opts, core.OptInt("THREADS", 10, "Scan threads"))
}
