package core

import (
	"encoding/base64"
	"net/url"
)

// Query is a search engine dork for finding targets.
type Query struct {
	Engine string
	Dork   string
	Scope  string // optional, e.g. "service" or "leak" for LeakIX
}

// URL returns the direct search URL for this query.
func (q Query) URL() string {
	d := url.QueryEscape(q.Dork)
	switch q.Engine {
	case "Shodan":
		return "https://www.shodan.io/search?query=" + d
	case "ZoomEye":
		return "https://www.zoomeye.ai/searchResult?q=" + d
	case "FOFA":
		return "https://en.fofa.info/result?qbase64=" + base64.StdEncoding.EncodeToString([]byte(q.Dork))
	case "Censys":
		return "https://search.censys.io/search?resource=hosts&q=" + d
	case "LeakIX":
		scope := q.Scope
		if scope == "" {
			scope = "service"
		}
		return "https://leakix.net/search?scope=" + scope + "&q=" + d
	case "Google":
		return "https://www.google.com/search?q=" + d
	case "Hunter":
		return "https://hunter.io/search?query=" + d
	default:
		return ""
	}
}

func Shodan(dork string) Query        { return Query{Engine: "Shodan", Dork: dork} }
func ZoomEye(dork string) Query       { return Query{Engine: "ZoomEye", Dork: dork} }
func FOFA(dork string) Query          { return Query{Engine: "FOFA", Dork: dork} }
func LeakIX(dork, scope string) Query { return Query{Engine: "LeakIX", Dork: dork, Scope: scope} }
func Google(dork string) Query        { return Query{Engine: "Google", Dork: dork} }
func Censys(dork string) Query        { return Query{Engine: "Censys", Dork: dork} }
func Hunter(dork string) Query        { return Query{Engine: "Hunter", Dork: dork} }
