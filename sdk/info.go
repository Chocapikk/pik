package sdk

import "strings"

// --- Reliability ---

type Reliability int

const (
	Unstable     Reliability = 0
	Unlikely     Reliability = 100
	Difficult    Reliability = 200
	Typical      Reliability = 300
	Reliable     Reliability = 400
	VeryReliable Reliability = 500
	Certain      Reliability = 600
)

func (r Reliability) String() string {
	switch r {
	case Unstable:
		return "unstable"
	case Unlikely:
		return "unlikely"
	case Difficult:
		return "difficult"
	case Typical:
		return "typical"
	case Reliable:
		return "reliable"
	case VeryReliable:
		return "very reliable"
	case Certain:
		return "certain"
	default:
		return "unknown"
	}
}

// --- Check ---

type CheckCode int

const (
	CheckUnknown    CheckCode = iota
	CheckSafe
	CheckDetected
	CheckAppears
	CheckVulnerable
)

func (c CheckCode) String() string {
	switch c {
	case CheckUnknown:
		return "unknown"
	case CheckSafe:
		return "safe"
	case CheckDetected:
		return "detected"
	case CheckAppears:
		return "appears"
	case CheckVulnerable:
		return "vulnerable"
	default:
		return "unknown"
	}
}

func (c CheckCode) IsVulnerable() bool {
	return c == CheckAppears || c == CheckVulnerable
}

type CheckResult struct {
	Code    CheckCode
	Reason  string
	Details map[string]string // version detected, banner, etc.
}

func (r CheckResult) FormatReason() string {
	if r.Reason != "" {
		return " - " + r.Reason
	}
	return ""
}

// Check result constructors - match MSF's CheckCode::Vulnerable() style.
func Vulnerable(reason string, details ...string) (CheckResult, error) {
	return CheckResult{Code: CheckVulnerable, Reason: reason, Details: pairs(details)}, nil
}

func Safe(reason string) (CheckResult, error) {
	return CheckResult{Code: CheckSafe, Reason: reason}, nil
}

func Detected(reason string) (CheckResult, error) {
	return CheckResult{Code: CheckDetected, Reason: reason}, nil
}

func Unknown(err error) (CheckResult, error) {
	return CheckResult{Code: CheckUnknown}, err
}


func pairs(kv []string) map[string]string {
	if len(kv) == 0 {
		return nil
	}
	m := make(map[string]string, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return m
}

// --- Stance ---

type Stance string

const (
	Aggressive Stance = "aggressive" // may crash target or leave artifacts
	Passive    Stance = "passive"    // read-only, no side effects
)

// --- Notes (MSF-style stability/side-effects metadata) ---

type Notes struct {
	Stability   []string // CRASH_SAFE, CRASH_UNSAFE, SERVICE_RESTART
	SideEffects []string // ARTIFACTS_ON_DISK, IOC_IN_LOGS, CONFIG_CHANGES
	Reliability []string // REPEATABLE_SESSION, UNRELIABLE_SESSION
}

// Standard stability tags.
const (
	CrashSafe       = "CRASH_SAFE"
	CrashUnsafe     = "CRASH_UNSAFE"
	ServiceRestart  = "SERVICE_RESTART"
	ArtifactsOnDisk = "ARTIFACTS_ON_DISK"
	IOCInLogs       = "IOC_IN_LOGS"
	ConfigChanges   = "CONFIG_CHANGES"
	RepeatableSession = "REPEATABLE_SESSION"
	AccountLockout    = "ACCOUNT_LOCKOUT_POSSIBLE"
)

// --- Target ---

type Target struct {
	Name           string
	Platform       string
	Type           string            // module-defined, e.g. "cmd", "dropper"
	Arches         []string
	DefaultOptions map[string]string // per-target option overrides
}

func (t Target) String() string {
	name := t.Platform
	if t.Name != "" {
		name = t.Name
	}
	if len(t.Arches) == 0 {
		return name
	}
	return name + " (" + strings.Join(t.Arches, ", ") + ")"
}

func (t Target) SupportsArch(arch string) bool {
	if len(t.Arches) == 0 {
		return true
	}
	for _, a := range t.Arches {
		if a == arch {
			return true
		}
	}
	return false
}

func TargetLinux(arches ...string) Target   { return Target{Platform: "linux", Arches: arches} }
func TargetWindows(arches ...string) Target { return Target{Platform: "windows", Arches: arches} }

// --- Lab ---

// Lab declares an optional Docker lab environment for testing a module.
type Lab struct {
	Services []Service
}

// Service describes a container in a lab environment.
// pkg/lab converts these to Docker SDK types at runtime.
type Service struct {
	Name        string            // container name suffix (e.g. "web", "db")
	Image       string            // Docker image (e.g. "vulhub/langflow:1.2.0")
	Ports       []string          // port bindings (e.g. "7860:7860")
	Env         map[string]string // environment variables
	Cmd         []string          // override entrypoint command
	Volumes     []string          // bind mounts (host:container)
	Healthcheck []string          // CMD-SHELL health check command
}

// NewLabService builds a Service for the common case: image + port bindings.
// Chain WithEnv(), WithCmd(), WithVolume(), and WithHealthcheck() for more.
func NewLabService(name, image string, ports ...string) Service {
	return Service{Name: name, Image: image, Ports: ports}
}

// WithEnv adds an environment variable.
func (s Service) WithEnv(key, value string) Service {
	if s.Env == nil {
		s.Env = make(map[string]string)
	}
	s.Env[key] = value
	return s
}

// WithCmd overrides the container command.
func (s Service) WithCmd(args ...string) Service {
	s.Cmd = args
	return s
}

// WithVolume adds a bind mount (host:container).
func (s Service) WithVolume(bind string) Service {
	s.Volumes = append(s.Volumes, bind)
	return s
}

// WithHealthcheck sets a CMD-SHELL health check.
func (s Service) WithHealthcheck(cmd string) Service {
	s.Healthcheck = []string{cmd}
	return s
}

// --- Info ---

type Info struct {
	Name           string // Software name (e.g. "OpenDCIM", "Langflow", "Next.js")
	Versions       string // Affected versions (e.g. "< 24.2", "1.0.0 - 1.2.9")
	Description    string // Vulnerability title (e.g. "SQLi to RCE via Config Poisoning")
	Detail         string
	Authors        []string
	DisclosureDate string // "2026-01-15"
	Reliability    Reliability
	Stance         Stance
	Privileged     bool // does exploitation yield privileged access?
	Notes          Notes
	References     []Reference
	Queries        []Query
	Targets        []Target
	DefaultOptions map[string]string
	Lab            Lab
}

// Title returns the formatted module title: "Name Versions - Description".
func (info Info) Title() string {
	parts := []string{}
	if info.Name != "" {
		parts = append(parts, info.Name)
	}
	if info.Versions != "" {
		parts = append(parts, info.Versions)
	}
	prefix := strings.Join(parts, " ")
	if prefix != "" && info.Description != "" {
		return prefix + " - " + info.Description
	}
	if prefix != "" {
		return prefix
	}
	return info.Description
}

func (info Info) CVEs() []string {
	var cves []string
	for _, ref := range info.References {
		if ref.Type == RefCVE {
			cves = append(cves, ref.ID)
		}
	}
	return cves
}

func (info Info) Platform() string {
	for _, t := range info.Targets {
		if t.Platform == "linux" || t.Platform == "windows" {
			return t.Platform
		}
	}
	return "linux"
}

func (info Info) TargetStrings() []string {
	result := make([]string, len(info.Targets))
	for i, t := range info.Targets {
		result[i] = t.String()
	}
	return result
}
