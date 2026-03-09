package sdk

import (
	"testing"
)

// --- Reliability ---

func TestReliabilityString(t *testing.T) {
	tests := []struct {
		r    Reliability
		want string
	}{
		{Unstable, "unstable"},
		{Unlikely, "unlikely"},
		{Difficult, "difficult"},
		{Typical, "typical"},
		{Reliable, "reliable"},
		{VeryReliable, "very reliable"},
		{Certain, "certain"},
		{Reliability(999), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.r.String(); got != tt.want {
			t.Errorf("Reliability(%d).String() = %q, want %q", tt.r, got, tt.want)
		}
	}
}

// --- CheckCode ---

func TestCheckCodeString(t *testing.T) {
	tests := []struct {
		c    CheckCode
		want string
	}{
		{CheckUnknown, "unknown"},
		{CheckSafe, "safe"},
		{CheckDetected, "detected"},
		{CheckAppears, "appears"},
		{CheckVulnerable, "vulnerable"},
		{CheckCode(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.c.String(); got != tt.want {
			t.Errorf("CheckCode(%d).String() = %q, want %q", tt.c, got, tt.want)
		}
	}
}

func TestCheckCodeIsVulnerable(t *testing.T) {
	tests := []struct {
		c    CheckCode
		want bool
	}{
		{CheckUnknown, false},
		{CheckSafe, false},
		{CheckDetected, false},
		{CheckAppears, true},
		{CheckVulnerable, true},
	}
	for _, tt := range tests {
		if got := tt.c.IsVulnerable(); got != tt.want {
			t.Errorf("CheckCode(%d).IsVulnerable() = %v, want %v", tt.c, got, tt.want)
		}
	}
}

func TestCheckResultFormatReason(t *testing.T) {
	r1 := CheckResult{Reason: "found it"}
	if got := r1.FormatReason(); got != " - found it" {
		t.Errorf("FormatReason() = %q, want %q", got, " - found it")
	}

	r2 := CheckResult{}
	if got := r2.FormatReason(); got != "" {
		t.Errorf("FormatReason() empty = %q, want empty", got)
	}
}

// --- Check result constructors ---

func TestVulnerable(t *testing.T) {
	r, err := Vulnerable("sqli works", "version", "1.0")
	if err != nil {
		t.Fatal(err)
	}
	if r.Code != CheckVulnerable {
		t.Errorf("Code = %v, want Vulnerable", r.Code)
	}
	if r.Reason != "sqli works" {
		t.Errorf("Reason = %q", r.Reason)
	}
	if r.Details["version"] != "1.0" {
		t.Errorf("Details = %v", r.Details)
	}
}

func TestVulnerableNoDetails(t *testing.T) {
	r, _ := Vulnerable("test")
	if r.Details != nil {
		t.Errorf("Details should be nil, got %v", r.Details)
	}
}

func TestSafe(t *testing.T) {
	r, err := Safe("patched")
	if err != nil {
		t.Fatal(err)
	}
	if r.Code != CheckSafe || r.Reason != "patched" {
		t.Errorf("Safe() = %+v", r)
	}
}

func TestDetected(t *testing.T) {
	r, err := Detected("banner found")
	if err != nil {
		t.Fatal(err)
	}
	if r.Code != CheckDetected || r.Reason != "banner found" {
		t.Errorf("Detected() = %+v", r)
	}
}

func TestUnknown(t *testing.T) {
	r, err := Unknown(Errorf("timeout"))
	if err == nil {
		t.Fatal("expected error")
	}
	if r.Code != CheckUnknown {
		t.Errorf("Code = %v, want Unknown", r.Code)
	}
}

// --- Target ---

func TestTargetString(t *testing.T) {
	tests := []struct {
		target Target
		want   string
	}{
		{Target{Platform: "linux"}, "linux"},
		{Target{Name: "Custom", Platform: "linux"}, "Custom"},
		{Target{Platform: "linux", Arches: []string{"amd64", "arm64"}}, "linux (amd64, arm64)"},
		{Target{Name: "Named", Arches: []string{"x86"}}, "Named (x86)"},
	}
	for _, tt := range tests {
		if got := tt.target.String(); got != tt.want {
			t.Errorf("Target.String() = %q, want %q", got, tt.want)
		}
	}
}

func TestTargetSupportsArch(t *testing.T) {
	noArch := Target{Platform: "linux"}
	if !noArch.SupportsArch("anything") {
		t.Error("empty arches should support any arch")
	}

	withArch := Target{Platform: "linux", Arches: []string{"amd64", "arm64"}}
	if !withArch.SupportsArch("amd64") {
		t.Error("should support amd64")
	}
	if withArch.SupportsArch("x86") {
		t.Error("should not support x86")
	}
}

func TestTargetLinuxWindows(t *testing.T) {
	lt := TargetLinux("amd64")
	if lt.Platform != "linux" || len(lt.Arches) != 1 || lt.Arches[0] != "amd64" {
		t.Errorf("TargetLinux = %+v", lt)
	}

	wt := TargetWindows()
	if wt.Platform != "windows" || len(wt.Arches) != 0 {
		t.Errorf("TargetWindows = %+v", wt)
	}
}

func TestLinuxCmd(t *testing.T) {
	targets := LinuxCmd()
	if len(targets) != 1 {
		t.Fatalf("LinuxCmd() len = %d", len(targets))
	}
	if targets[0].Platform != "linux" || targets[0].Type != "cmd" {
		t.Errorf("LinuxCmd()[0] = %+v", targets[0])
	}
}

// --- Author ---

func TestObfuscateEmail(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"<user[at]domain>", "<user[at]domain>"},
		{"user@domain.com", "<user[at]domain.com>"},
		{"<user@domain.com>", "<user[at]domain.com>"},
	}
	for _, tt := range tests {
		if got := ObfuscateEmail(tt.input); got != tt.want {
			t.Errorf("ObfuscateEmail(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestAuthorString(t *testing.T) {
	tests := []struct {
		author Author
		want   string
	}{
		{Author{Name: "Alice"}, "Alice"},
		{Author{Name: "Alice", Handle: "alice99"}, "Alice (alice99)"},
		{Author{Name: "Alice", Handle: "Alice"}, "Alice"},
		{Author{Name: "Alice", Company: "ACME"}, "Alice @ ACME"},
		{Author{Name: "Alice", Email: "<a[at]b.com>"}, "Alice <a[at]b.com>"},
		{
			Author{Name: "Alice", Handle: "a99", Company: "ACME", Email: "<a[at]b.com>"},
			"Alice (a99) @ ACME <a[at]b.com>",
		},
	}
	for _, tt := range tests {
		if got := tt.author.String(); got != tt.want {
			t.Errorf("Author.String() = %q, want %q", got, tt.want)
		}
	}
}

func TestAuthorBuilderChain(t *testing.T) {
	a := NewAuthor("Bob").WithHandle("b0b").WithEmail("<b[at]x.com>").WithCompany("Corp")
	if a.Name != "Bob" || a.Handle != "b0b" || a.Email != "<b[at]x.com>" || a.Company != "Corp" {
		t.Errorf("Author chain = %+v", a)
	}
}

// --- Info ---

func TestInfoTitle(t *testing.T) {
	tests := []struct {
		info Info
		want string
	}{
		{Info{Name: "App", Versions: "< 1.0", Description: "RCE"}, "App < 1.0 - RCE"},
		{Info{Name: "App", Versions: "< 1.0"}, "App < 1.0"},
		{Info{Description: "RCE"}, "RCE"},
		{Info{Name: "App"}, "App"},
		{Info{}, ""},
	}
	for _, tt := range tests {
		if got := tt.info.Title(); got != tt.want {
			t.Errorf("Info.Title() = %q, want %q", got, tt.want)
		}
	}
}

func TestAuthorsHelper(t *testing.T) {
	a := Authors(NewAuthor("A"), NewAuthor("B"))
	if len(a) != 2 {
		t.Errorf("Authors len = %d", len(a))
	}
}

func TestInfoAuthorNames(t *testing.T) {
	info := Info{Authors: []Author{{Name: "A"}, {Name: "B"}}}
	got := info.AuthorNames()
	if got != "A, B" {
		t.Errorf("AuthorNames() = %q", got)
	}
}

func TestInfoCVEs(t *testing.T) {
	info := Info{Refs: []Reference{
		CVE("2026-1234"),
		URL("https://example.com"),
		CVE("2026-5678"),
	}}
	cves := info.CVEs()
	if len(cves) != 2 || cves[0] != "CVE-2026-1234" || cves[1] != "CVE-2026-5678" {
		t.Errorf("CVEs() = %v", cves)
	}
}

func TestInfoPlatform(t *testing.T) {
	info := Info{Targets: []Target{{Platform: "linux"}}}
	if got := info.Platform(); got != "linux" {
		t.Errorf("Platform() = %q", got)
	}

	info2 := Info{Targets: []Target{{Platform: "windows"}}}
	if got := info2.Platform(); got != "windows" {
		t.Errorf("Platform() = %q", got)
	}

	info3 := Info{}
	if got := info3.Platform(); got != "linux" {
		t.Errorf("Platform() default = %q", got)
	}
}

func TestInfoTargetStrings(t *testing.T) {
	info := Info{Targets: []Target{
		{Platform: "linux"},
		{Name: "Custom", Arches: []string{"amd64"}},
	}}
	got := info.TargetStrings()
	if len(got) != 2 || got[0] != "linux" || got[1] != "Custom (amd64)" {
		t.Errorf("TargetStrings() = %v", got)
	}
}

// --- Notes ---

func TestNotesChain(t *testing.T) {
	n := SafeNotes().Logs().ConfigChanges().Artifacts().Repeatable()
	if len(n.Stability) != 1 || n.Stability[0] != CrashSafe {
		t.Errorf("Stability = %v", n.Stability)
	}
	if len(n.SideEffects) != 3 {
		t.Errorf("SideEffects = %v", n.SideEffects)
	}
	if n.SideEffects[0] != IOCInLogs || n.SideEffects[1] != ConfigChanges || n.SideEffects[2] != ArtifactsOnDisk {
		t.Errorf("SideEffects order = %v", n.SideEffects)
	}
	if len(n.Reliability) != 1 || n.Reliability[0] != RepeatableSession {
		t.Errorf("Reliability = %v", n.Reliability)
	}
}

// --- Opts ---

func TestOpts(t *testing.T) {
	m := Opts("A", "1", "B", "2")
	if m["A"] != "1" || m["B"] != "2" || len(m) != 2 {
		t.Errorf("Opts = %v", m)
	}
}

func TestOptsOdd(t *testing.T) {
	m := Opts("A", "1", "B")
	if m["A"] != "1" || len(m) != 1 {
		t.Errorf("Opts odd = %v", m)
	}
}

// --- Lab ---

func TestNewLabServiceChain(t *testing.T) {
	s := NewLabService("web", "nginx:latest", "80", "443").
		WithEnv("DEBUG", "1").
		WithEnv("MODE", "dev").
		WithCmd("nginx", "-g", "daemon off;").
		WithVolume("/data:/var/data").
		WithHealthcheck("curl -f http://localhost").
		WithPostStart("echo hello")

	if s.Name != "web" || s.Image != "nginx:latest" {
		t.Errorf("Name/Image = %q/%q", s.Name, s.Image)
	}
	if len(s.Ports) != 2 {
		t.Errorf("Ports = %v", s.Ports)
	}
	if s.Env["DEBUG"] != "1" || s.Env["MODE"] != "dev" {
		t.Errorf("Env = %v", s.Env)
	}
	if len(s.Cmd) != 3 {
		t.Errorf("Cmd = %v", s.Cmd)
	}
	if len(s.Volumes) != 1 || s.Volumes[0] != "/data:/var/data" {
		t.Errorf("Volumes = %v", s.Volumes)
	}
	if len(s.Healthcheck) != 1 {
		t.Errorf("Healthcheck = %v", s.Healthcheck)
	}
	if len(s.PostStart) != 1 {
		t.Errorf("PostStart = %v", s.PostStart)
	}
}

func TestSingleLab(t *testing.T) {
	lab := SingleLab("web", "img:1.0", "8080")
	if len(lab.Services) != 1 {
		t.Fatalf("Services = %d", len(lab.Services))
	}
	if lab.Services[0].Name != "web" || lab.Services[0].Image != "img:1.0" {
		t.Errorf("Service = %+v", lab.Services[0])
	}
}

func TestRandIsRand(t *testing.T) {
	placeholder := Rand("password")
	label, ok := IsRand(placeholder)
	if !ok || label != "password" {
		t.Errorf("IsRand(%q) = %q, %v", placeholder, label, ok)
	}

	_, ok = IsRand("not-a-rand")
	if ok {
		t.Error("should not be rand")
	}
}

// --- Feature ---

func TestFeatureConstants(t *testing.T) {
	if XML != "xml" {
		t.Errorf("XML = %q", XML)
	}
	if FakeData != "fake" {
		t.Errorf("FakeData = %q", FakeData)
	}
}
