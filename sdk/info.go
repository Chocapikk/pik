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

// Check result constructors for clean module code.
func CheckOK(reason string, details ...string) (CheckResult, error) {
	return CheckResult{Code: CheckVulnerable, Reason: reason, Details: pairs(details)}, nil
}

func CheckFail(reason string) (CheckResult, error) {
	return CheckResult{Code: CheckSafe, Reason: reason}, nil
}

func CheckErr(err error) (CheckResult, error) {
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

// --- Info ---

type Info struct {
	Description    string
	Detail         string
	Authors        []string
	DisclosureDate string // "2026-01-15"
	Reliability    Reliability
	Stance         Stance
	Privileged     bool   // does exploitation yield privileged access?
	Notes          Notes
	References     []Reference
	Queries        []Query
	Targets        []Target
	DefaultOptions map[string]string
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

func (info Info) DefaultArch() string {
	for _, t := range info.Targets {
		if t.Platform == info.Platform() && len(t.Arches) > 0 {
			return t.Arches[0]
		}
	}
	return "amd64"
}

func (info Info) SupportsArch(arch string) bool {
	for _, t := range info.Targets {
		if t.Platform == info.Platform() {
			return t.SupportsArch(arch)
		}
	}
	return true
}

func (info Info) TargetStrings() []string {
	result := make([]string, len(info.Targets))
	for i, t := range info.Targets {
		result[i] = t.String()
	}
	return result
}
