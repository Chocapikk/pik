package sdk

import "testing"

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		path    string
		pattern string
		want    bool
	}{
		// Exact match
		{"/foo", "/foo", true},
		{"/foo", "/bar", false},
		// Suffix (starts with *)
		{"*.js", "*.js", true},
		{"/app/main.js", "*.js", true},
		{"/app/main.css", "*.js", false},
		// Prefix (ends with *)
		{"/api/v1/users", "/api/*", true},
		{"/web/index", "/api/*", false},
		// Contains (both *)
		{"/foo/bar/baz", "*bar*", true},
		{"/foo/qux/baz", "*bar*", false},
	}
	for _, tt := range tests {
		if got := matchPattern(tt.path, tt.pattern); got != tt.want {
			t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.path, tt.pattern, got, tt.want)
		}
	}
}

func TestServerMuxServeAndMatch(t *testing.T) {
	mux := &ServerMux{}
	mux.ServeRoute("/payload.js", "text/javascript", []byte("alert(1)"))
	mux.ServeRoute("*.css", "text/css", []byte("body{}"))

	ct, body, ok := mux.Match("/payload.js")
	if !ok || ct != "text/javascript" || string(body) != "alert(1)" {
		t.Errorf("Match exact: ok=%v ct=%q body=%q", ok, ct, body)
	}

	ct, body, ok = mux.Match("/styles/main.css")
	if !ok || ct != "text/css" {
		t.Errorf("Match suffix: ok=%v ct=%q", ok, ct)
	}

	_, _, ok = mux.Match("/missing")
	if ok {
		t.Error("should not match /missing")
	}
}

func TestServerMuxHitTracking(t *testing.T) {
	mux := &ServerMux{}
	mux.ServeRoute("/a", "text/plain", []byte("a"))
	mux.ServeRoute("/b", "text/plain", []byte("b"))

	if mux.allHit([]string{"/a"}) {
		t.Error("should not be hit yet")
	}

	mux.Match("/a")
	if !mux.allHit([]string{"/a"}) {
		t.Error("should be hit after Match")
	}
	if mux.allHit([]string{"/a", "/b"}) {
		t.Error("/b not hit yet")
	}

	mux.Match("/b")
	if !mux.allHit([]string{"/a", "/b"}) {
		t.Error("both should be hit")
	}
}

func TestServerMuxWaitRoutesSuccess(t *testing.T) {
	mux := &ServerMux{}
	mux.ServeRoute("/a", "text/plain", []byte("a"))

	// Hit it immediately in a goroutine
	go func() {
		mux.Match("/a")
	}()

	err := mux.WaitRoutes(2, "/a")
	if err != nil {
		t.Errorf("WaitRoutes = %v", err)
	}
}

func TestServerMuxWaitRoutesTimeout(t *testing.T) {
	mux := &ServerMux{}
	mux.ServeRoute("/never", "text/plain", []byte("x"))

	err := mux.WaitRoutes(1, "/never")
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestWithHTTPServer(t *testing.T) {
	type testMod struct {
		Pik
		WithHTTPServer
	}
	mod := &testMod{}
	if _, ok := any(mod).(HTTPServerModule); !ok {
		t.Error("should implement HTTPServerModule")
	}
	// Call the marker method for coverage
	mod.HTTPServer()
}
