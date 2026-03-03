package sdk

// RefType identifies the kind of reference.
type RefType string

const (
	RefCVE         RefType = "CVE"
	RefGHSA        RefType = "GHSA"
	RefEDB         RefType = "EDB"
	RefPacketstorm RefType = "PACKETSTORM"
	RefVulnCheck   RefType = "VULNCHECK"
	RefURL         RefType = "URL"
)

// Reference is a vulnerability reference.
type Reference struct {
	Type RefType
	ID   string
	Repo string // optional: "owner/repo" for repo-scoped advisories (GHSA)
}

// URL returns the full URL for this reference.
func (r Reference) URL() string {
	switch r.Type {
	case RefCVE:
		return "https://nvd.nist.gov/vuln/detail/" + r.ID
	case RefGHSA:
		if r.Repo != "" {
			return "https://github.com/" + r.Repo + "/security/advisories/" + r.ID
		}
		return "https://github.com/advisories/" + r.ID
	case RefEDB:
		return "https://www.exploit-db.com/exploits/" + r.ID
	case RefPacketstorm:
		return "https://packetstormsecurity.com/files/" + r.ID
	case RefVulnCheck:
		return "https://www.vulncheck.com/advisories/" + r.ID
	case RefURL:
		return r.ID
	default:
		return r.ID
	}
}

func (r Reference) String() string {
	if r.Type == RefURL {
		return r.ID
	}
	return string(r.Type) + "-" + r.ID
}

func CVE(id string) Reference        { return Reference{Type: RefCVE, ID: "CVE-" + id} }
// GHSA creates a GitHub Security Advisory reference.
// Use GHSA("xxxx-yyyy-zzzz") for global advisories,
// or GHSA("xxxx-yyyy-zzzz", "owner/repo") for repo-scoped ones.
func GHSA(id string, repo ...string) Reference {
	ref := Reference{Type: RefGHSA, ID: "GHSA-" + id}
	if len(repo) > 0 {
		ref.Repo = repo[0]
	}
	return ref
}
func EDB(id string) Reference        { return Reference{Type: RefEDB, ID: id} }
func Packetstorm(id string) Reference  { return Reference{Type: RefPacketstorm, ID: id} }
func VulnCheck(slug string) Reference  { return Reference{Type: RefVulnCheck, ID: slug} }
func URL(u string) Reference           { return Reference{Type: RefURL, ID: u} }
