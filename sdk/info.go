package sdk

import (
	"strings"
)

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
	Name     string
	Platform string
	Type     string            // module-defined, e.g. "cmd", "dropper"
	Arches   []string
	Defaults map[string]string // per-target option overrides
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

const randPrefix = "{{rand:"

// Rand returns a placeholder that pkg/lab replaces with a random value.
// Same label across services = same generated value (shared credentials).
func Rand(label string) string { return randPrefix + label + "}}" }

// IsRand checks if a value is a Rand placeholder and returns the label.
func IsRand(v string) (string, bool) {
	if strings.HasPrefix(v, randPrefix) && strings.HasSuffix(v, "}}") {
		return v[len(randPrefix) : len(v)-2], true
	}
	return "", false
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

// --- Author ---

// Author describes a module contributor.
// Email is automatically formatted to <user[at]domain> for anti-scraping.
type Author struct {
	Name    string // real name or alias
	Handle  string // online handle (e.g. "Chocapikk")
	Email   string // contact email, must use <user[at]domain> format (Register panics on raw @)
	Company string // organization or team (e.g. "Horizon3 Attack Team")
}

// ObfuscateEmail formats a raw email to <user[at]domain>.
// Already obfuscated emails are returned as-is.
func ObfuscateEmail(email string) string {
	if email == "" || strings.Contains(email, "[at]") {
		return email
	}
	email = strings.TrimPrefix(strings.TrimSuffix(email, ">"), "<")
	return "<" + strings.Replace(email, "@", "[at]", 1) + ">"
}

func (a Author) String() string {
	parts := a.Name
	if a.Handle != "" && a.Handle != a.Name {
		parts += " (" + a.Handle + ")"
	}
	if a.Company != "" {
		parts += " @ " + a.Company
	}
	if a.Email != "" {
		parts += " " + ObfuscateEmail(a.Email)
	}
	return parts
}


// --- Info ---

type Info struct {
	Name        string // Software name (e.g. "OpenDCIM", "Langflow", "Next.js")
	Versions    string // Affected versions (e.g. "< 24.2", "1.0.0 - 1.2.9")
	Description string // Vulnerability title (e.g. "SQLi to RCE via Config Poisoning")
	Detail      string
	Authors     []Author
	Disclosure  string // "2026-01-15"
	Reliability Reliability
	Stance      Stance
	Privileged  bool // does exploitation yield privileged access?
	Notes       Notes
	Refs        []Reference
	Queries     []Query
	Targets     []Target
	Defaults    map[string]string
	Parsers     []Parser
	Lab         Lab
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

func (info Info) AuthorNames() string {
	names := make([]string, len(info.Authors))
	for i, a := range info.Authors {
		names[i] = a.String()
	}
	return strings.Join(names, ", ")
}

func (info Info) CVEs() []string {
	var cves []string
	for _, ref := range info.Refs {
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

// --- Helper constructors ---

// Authors returns its arguments as a slice, removing []Author{} noise from module code.
func Authors(authors ...Author) []Author { return authors }

// NewAuthor creates an Author with the given name. Chain .Handle(), .Email(), .Company() for more.
func NewAuthor(name string) Author { return Author{Name: name} }

func (a Author) WithHandle(h string) Author  { a.Handle = h; return a }
func (a Author) WithEmail(e string) Author   { a.Email = e; return a }
func (a Author) WithCompany(c string) Author { a.Company = c; return a }

// SafeNotes returns Notes with CrashSafe stability. Chain methods to add more.
func SafeNotes() Notes {
	return Notes{Stability: []string{CrashSafe}}
}

// Logs adds IOCInLogs to SideEffects.
func (n Notes) Logs() Notes {
	se := make([]string, len(n.SideEffects), len(n.SideEffects)+1)
	copy(se, n.SideEffects)
	n.SideEffects = append(se, IOCInLogs)
	return n
}

// ConfigChanges adds ConfigChanges to SideEffects.
func (n Notes) ConfigChanges() Notes {
	se := make([]string, len(n.SideEffects), len(n.SideEffects)+1)
	copy(se, n.SideEffects)
	n.SideEffects = append(se, ConfigChanges)
	return n
}

// Artifacts adds ArtifactsOnDisk to SideEffects.
func (n Notes) Artifacts() Notes {
	se := make([]string, len(n.SideEffects), len(n.SideEffects)+1)
	copy(se, n.SideEffects)
	n.SideEffects = append(se, ArtifactsOnDisk)
	return n
}

// Repeatable sets Reliability to RepeatableSession.
func (n Notes) Repeatable() Notes {
	r := make([]string, len(n.Reliability), len(n.Reliability)+1)
	copy(r, n.Reliability)
	n.Reliability = append(r, RepeatableSession)
	return n
}

// Opts builds a map from key-value pairs: Opts("RPORT", "7860").
func Opts(kv ...string) map[string]string {
	m := make(map[string]string, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return m
}

// --- Parsers ---

// Parser identifies an optional parser for standalone builds.
type Parser string

const XML Parser = "xml"

// LinuxCmd returns a single Linux command shell target.
func LinuxCmd() []Target {
	return []Target{{Name: "Unix/Linux Command Shell", Platform: "linux", Type: "cmd"}}
}

// SingleLab wraps a single service into a Lab.
func SingleLab(name, image string, ports ...string) Lab {
	return Lab{Services: []Service{NewLabService(name, image, ports...)}}
}
