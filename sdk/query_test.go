package sdk

import (
	"strings"
	"testing"
)

func TestQueryURL(t *testing.T) {
	tests := []struct {
		query Query
		check func(string) bool
	}{
		{Shodan(`http.title:"test"`), func(u string) bool { return strings.Contains(u, "shodan.io/search") }},
		{ZoomEye("app:nginx"), func(u string) bool { return strings.Contains(u, "zoomeye.ai") }},
		{FOFA(`title="test"`), func(u string) bool { return strings.Contains(u, "fofa.info/result?qbase64=") }},
		{Censys("services.http"), func(u string) bool { return strings.Contains(u, "censys.io") }},
		{Google("inurl:test"), func(u string) bool { return strings.Contains(u, "google.com/search") }},
		{Hunter("query"), func(u string) bool { return strings.Contains(u, "hunter.io") }},
		{LeakIX("plugin:test", "leak"), func(u string) bool {
			return strings.Contains(u, "leakix.net") && strings.Contains(u, "scope=leak")
		}},
		{LeakIXPlugin("TestPlugin"), func(u string) bool {
			return strings.Contains(u, "scope=leak") && strings.Contains(u, "plugin%3ATestPlugin")
		}},
	}
	for _, tt := range tests {
		url := tt.query.URL()
		if !tt.check(url) {
			t.Errorf("%s query URL = %q", tt.query.Engine, url)
		}
	}
}

func TestQueryURLDefault(t *testing.T) {
	q := Query{Engine: "Unknown", Dork: "test"}
	if got := q.URL(); got != "" {
		t.Errorf("Unknown engine URL = %q, want empty", got)
	}
}

func TestLeakIXDefaultScope(t *testing.T) {
	q := Query{Engine: "LeakIX", Dork: "test"}
	url := q.URL()
	if !strings.Contains(url, "scope=service") {
		t.Errorf("LeakIX default scope should be service, got %q", url)
	}
}

func TestDorks(t *testing.T) {
	d := Dorks(Shodan("a"), FOFA("b"))
	if len(d) != 2 {
		t.Errorf("Dorks() len = %d", len(d))
	}
}
