package sdk

import "testing"

func TestReferenceURL(t *testing.T) {
	tests := []struct {
		ref  Reference
		want string
	}{
		{CVE("2026-1234"), "https://nvd.nist.gov/vuln/detail/CVE-2026-1234"},
		{GHSA("xxxx-yyyy-zzzz"), "https://github.com/advisories/GHSA-xxxx-yyyy-zzzz"},
		{GHSA("xxxx-yyyy-zzzz", "owner/repo"), "https://github.com/owner/repo/security/advisories/GHSA-xxxx-yyyy-zzzz"},
		{EDB("12345"), "https://www.exploit-db.com/exploits/12345"},
		{Packetstorm("54321"), "https://packetstormsecurity.com/files/54321"},
		{VulnCheck("vc-slug"), "https://www.vulncheck.com/advisories/vc-slug"},
		{URL("https://example.com"), "https://example.com"},
	}
	for _, tt := range tests {
		if got := tt.ref.URL(); got != tt.want {
			t.Errorf("%s.URL() = %q, want %q", tt.ref, got, tt.want)
		}
	}
}

func TestReferenceString(t *testing.T) {
	tests := []struct {
		ref  Reference
		want string
	}{
		{CVE("2026-1234"), "CVE-CVE-2026-1234"},
		{EDB("99"), "EDB-99"},
		{URL("https://example.com"), "https://example.com"},
	}
	for _, tt := range tests {
		if got := tt.ref.String(); got != tt.want {
			t.Errorf("Reference.String() = %q, want %q", got, tt.want)
		}
	}
}

func TestReferenceURLDefaultType(t *testing.T) {
	ref := Reference{Type: "UNKNOWN", ID: "test-id"}
	if got := ref.URL(); got != "test-id" {
		t.Errorf("default URL = %q", got)
	}
}

func TestRefs(t *testing.T) {
	refs := Refs(CVE("2026-1"), EDB("2"))
	if len(refs) != 2 {
		t.Errorf("Refs() len = %d", len(refs))
	}
}
