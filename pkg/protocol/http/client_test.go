package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	nethttp "net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Chocapikk/pik/sdk"
)

// ---------------------------------------------------------------------------
// NormalizeURI (preserved from original)
// ---------------------------------------------------------------------------

func TestNormalizeURI(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{"single root", []string{"/"}, "/"},
		{"single path", []string{"/api"}, "/api"},
		{"two segments", []string{"/app", "/api"}, "/app/api"},
		{"trailing slash preserved", []string{"/app", "/api/"}, "/app/api/"},
		{"no leading slash added", []string{"app", "api"}, "/app/api"},
		{"double slash removed", []string{"/app/", "/api"}, "/app/api"},
		{"empty parts ignored", []string{"", "/api"}, "/api"},
		{"all empty", []string{"", ""}, "/"},
		{"root plus path", []string{"/", "/test"}, "/test"},
		{"three segments", []string{"/a", "/b", "/c"}, "/a/b/c"},
		{"trailing slash only on last", []string{"/a/", "/b/"}, "/a/b/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeURI(tt.parts...)
			if got != tt.want {
				t.Errorf("NormalizeURI(%v) = %q, want %q", tt.parts, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// echoServer returns a test server that echoes request details back as JSON.
func echoServer() *httptest.Server {
	return httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		body, _ := io.ReadAll(r.Body)
		resp := map[string]any{
			"method":  r.Method,
			"path":    r.URL.Path,
			"query":   r.URL.RawQuery,
			"body":    string(body),
			"headers": flattenHeaders(r.Header),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

func flattenHeaders(h nethttp.Header) map[string]string {
	out := make(map[string]string)
	for k, v := range h {
		out[k] = strings.Join(v, ", ")
	}
	return out
}

// ---------------------------------------------------------------------------
// NewSession
// ---------------------------------------------------------------------------

func TestNewSession(t *testing.T) {
	s := NewSession()
	if s == nil {
		t.Fatal("NewSession returned nil")
	}
	if s.client == nil {
		t.Fatal("Session.client is nil")
	}
	if s.maxBody != defaultMaxBody {
		t.Errorf("maxBody = %d, want %d", s.maxBody, defaultMaxBody)
	}
}

func TestNewSessionWithOptions(t *testing.T) {
	transport := &nethttp.Transport{}
	s := NewSession(WithTransport(transport))
	if s.client.Transport != transport {
		t.Error("WithTransport did not set the transport")
	}
}

// ---------------------------------------------------------------------------
// Session.Get
// ---------------------------------------------------------------------------

func TestSessionGet(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	s := NewSession()
	resp, err := s.Get(srv.URL)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var result map[string]any
	if err := resp.JSON(&result); err != nil {
		t.Fatalf("JSON decode error: %v", err)
	}
	if result["method"] != "GET" {
		t.Errorf("method = %v, want GET", result["method"])
	}
}

// ---------------------------------------------------------------------------
// Session.PostForm
// ---------------------------------------------------------------------------

func TestSessionPostForm(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	s := NewSession()
	data := url.Values{"key": {"value"}, "foo": {"bar"}}
	resp, err := s.PostForm(srv.URL, data)
	if err != nil {
		t.Fatalf("PostForm error: %v", err)
	}

	var result map[string]any
	if err := resp.JSON(&result); err != nil {
		t.Fatalf("JSON decode error: %v", err)
	}
	if result["method"] != "POST" {
		t.Errorf("method = %v, want POST", result["method"])
	}
	body := result["body"].(string)
	parsed, _ := url.ParseQuery(body)
	if parsed.Get("key") != "value" {
		t.Errorf("form key = %q, want %q", parsed.Get("key"), "value")
	}
	if parsed.Get("foo") != "bar" {
		t.Errorf("form foo = %q, want %q", parsed.Get("foo"), "bar")
	}
}

// ---------------------------------------------------------------------------
// Session.Run
// ---------------------------------------------------------------------------

func TestSessionRun(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	s := NewSession()
	r := s.Run(context.Background(), srv.URL)
	if r.Target != srv.URL {
		t.Errorf("Target = %q, want %q", r.Target, srv.URL)
	}
	if r.TargetURI != "/" {
		t.Errorf("TargetURI = %q, want /", r.TargetURI)
	}
	if r.Session != s {
		t.Error("Run.Session does not match parent session")
	}
}

// ---------------------------------------------------------------------------
// NewRun
// ---------------------------------------------------------------------------

func TestNewRun(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	r := NewRun(context.Background(), srv.URL)
	if r.Session == nil {
		t.Fatal("NewRun Session is nil")
	}
	if r.TargetURI != "/" {
		t.Errorf("TargetURI = %q, want /", r.TargetURI)
	}
	// Target should already have a scheme
	if !strings.HasPrefix(r.Target, "http://") && !strings.HasPrefix(r.Target, "https://") {
		t.Errorf("Target %q missing scheme", r.Target)
	}
}

// ---------------------------------------------------------------------------
// Run.Send - various request configurations
// ---------------------------------------------------------------------------

func TestRunSendGET(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	r := NewRun(context.Background(), srv.URL)
	resp, err := r.Send(Request{})
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	var result map[string]any
	resp.JSON(&result)
	if result["method"] != "GET" {
		t.Errorf("method = %v, want GET", result["method"])
	}
}

func TestRunSendPOSTWithBody(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	r := NewRun(context.Background(), srv.URL)
	payload := "custom-body-content"
	resp, err := r.Send(Request{
		Method:      "POST",
		Body:        strings.NewReader(payload),
		ContentType: "text/plain",
	})
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	var result map[string]any
	resp.JSON(&result)
	if result["method"] != "POST" {
		t.Errorf("method = %v, want POST", result["method"])
	}
	if result["body"] != payload {
		t.Errorf("body = %v, want %v", result["body"], payload)
	}
}

func TestRunSendWithPath(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	r := NewRun(context.Background(), srv.URL)
	resp, err := r.Send(Request{Path: "api/test"})
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	var result map[string]any
	resp.JSON(&result)
	if result["path"] != "/api/test" {
		t.Errorf("path = %v, want /api/test", result["path"])
	}
}

func TestRunSendWithTargetURI(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	r := NewRun(context.Background(), srv.URL)
	r.TargetURI = "/app"
	resp, err := r.Send(Request{Path: "endpoint"})
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	var result map[string]any
	resp.JSON(&result)
	if result["path"] != "/app/endpoint" {
		t.Errorf("path = %v, want /app/endpoint", result["path"])
	}
}

func TestRunSendWithQuery(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	r := NewRun(context.Background(), srv.URL)
	resp, err := r.Send(Request{
		Query: url.Values{"a": {"1"}, "b": {"2"}},
	})
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	var result map[string]any
	resp.JSON(&result)
	query := result["query"].(string)
	parsed, _ := url.ParseQuery(query)
	if parsed.Get("a") != "1" {
		t.Errorf("query a = %q, want 1", parsed.Get("a"))
	}
	if parsed.Get("b") != "2" {
		t.Errorf("query b = %q, want 2", parsed.Get("b"))
	}
}

func TestRunSendWithHeaders(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	r := NewRun(context.Background(), srv.URL)
	resp, err := r.Send(Request{
		Headers: map[string]string{
			"X-Custom-Header": "custom-value",
		},
	})
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	var result map[string]any
	resp.JSON(&result)
	headers := result["headers"].(map[string]any)
	if headers["X-Custom-Header"] != "custom-value" {
		t.Errorf("X-Custom-Header = %v, want custom-value", headers["X-Custom-Header"])
	}
}

func TestRunSendWithForm(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	r := NewRun(context.Background(), srv.URL)
	resp, err := r.Send(Request{
		Method: "POST",
		Form:   url.Values{"username": {"admin"}, "password": {"secret"}},
	})
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	var result map[string]any
	resp.JSON(&result)
	body := result["body"].(string)
	parsed, _ := url.ParseQuery(body)
	if parsed.Get("username") != "admin" {
		t.Errorf("username = %q, want admin", parsed.Get("username"))
	}
	if parsed.Get("password") != "secret" {
		t.Errorf("password = %q, want secret", parsed.Get("password"))
	}
	headers := result["headers"].(map[string]any)
	ct, _ := headers["Content-Type"].(string)
	if !strings.Contains(ct, "application/x-www-form-urlencoded") {
		t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", ct)
	}
}

func TestRunSendWithBasicAuth(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	r := NewRun(context.Background(), srv.URL)
	resp, err := r.Send(Request{
		BasicAuth: [2]string{"user", "pass"},
	})
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	var result map[string]any
	resp.JSON(&result)
	headers := result["headers"].(map[string]any)
	auth, ok := headers["Authorization"].(string)
	if !ok || auth == "" {
		t.Error("Authorization header not set")
	}
	if !strings.HasPrefix(auth, "Basic ") {
		t.Errorf("Authorization = %q, expected Basic prefix", auth)
	}
}

func TestRunSendPUT(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	r := NewRun(context.Background(), srv.URL)
	resp, err := r.Send(Request{
		Method:      "PUT",
		Body:        strings.NewReader(`{"key":"value"}`),
		ContentType: "application/json",
	})
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	var result map[string]any
	resp.JSON(&result)
	if result["method"] != "PUT" {
		t.Errorf("method = %v, want PUT", result["method"])
	}
}

func TestRunSendDELETE(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	r := NewRun(context.Background(), srv.URL)
	resp, err := r.Send(Request{Method: "DELETE"})
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	var result map[string]any
	resp.JSON(&result)
	if result["method"] != "DELETE" {
		t.Errorf("method = %v, want DELETE", result["method"])
	}
}

// ---------------------------------------------------------------------------
// Response methods
// ---------------------------------------------------------------------------

func TestResponseBodyBytes(t *testing.T) {
	body := "hello world"
	resp := WrapResponse(&nethttp.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
	})

	data, err := resp.BodyBytes()
	if err != nil {
		t.Fatalf("BodyBytes error: %v", err)
	}
	if string(data) != body {
		t.Errorf("BodyBytes = %q, want %q", string(data), body)
	}

	// Second call should return cached result
	data2, err := resp.BodyBytes()
	if err != nil {
		t.Fatalf("BodyBytes (cached) error: %v", err)
	}
	if !bytes.Equal(data, data2) {
		t.Error("BodyBytes did not return cached data on second call")
	}
}

func TestResponseBodyString(t *testing.T) {
	body := "test string body"
	resp := WrapResponse(&nethttp.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
	})

	s, err := resp.BodyString()
	if err != nil {
		t.Fatalf("BodyString error: %v", err)
	}
	if s != body {
		t.Errorf("BodyString = %q, want %q", s, body)
	}
}

func TestResponseContains(t *testing.T) {
	resp := WrapResponse(&nethttp.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("the quick brown fox")),
	})

	if !resp.Contains("quick") {
		t.Error("Contains(quick) = false, want true")
	}
	if resp.Contains("lazy") {
		t.Error("Contains(lazy) = true, want false")
	}
}

func TestResponseContainsAll(t *testing.T) {
	resp := WrapResponse(&nethttp.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("alpha beta gamma")),
	})

	if !resp.ContainsAll("alpha", "beta", "gamma") {
		t.Error("ContainsAll(alpha,beta,gamma) = false, want true")
	}
	if resp.ContainsAll("alpha", "delta") {
		t.Error("ContainsAll(alpha,delta) = true, want false")
	}
}

func TestResponseContainsAny(t *testing.T) {
	resp := WrapResponse(&nethttp.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("alpha beta gamma")),
	})

	if !resp.ContainsAny("delta", "beta") {
		t.Error("ContainsAny(delta,beta) = false, want true")
	}
	if resp.ContainsAny("delta", "epsilon") {
		t.Error("ContainsAny(delta,epsilon) = true, want false")
	}
}

func TestResponseContainsAnyNone(t *testing.T) {
	resp := WrapResponse(&nethttp.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("nothing here")),
	})

	if resp.ContainsAny("x", "y", "z") {
		t.Error("ContainsAny should return false when no substring matches")
	}
}

func TestResponseJSON(t *testing.T) {
	payload := `{"name":"test","count":42}`
	resp := WrapResponse(&nethttp.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(payload)),
	})

	var result struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	if err := resp.JSON(&result); err != nil {
		t.Fatalf("JSON error: %v", err)
	}
	if result.Name != "test" {
		t.Errorf("Name = %q, want test", result.Name)
	}
	if result.Count != 42 {
		t.Errorf("Count = %d, want 42", result.Count)
	}
}

func TestResponseJSONInvalid(t *testing.T) {
	resp := WrapResponse(&nethttp.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("not json")),
	})
	var result map[string]any
	if err := resp.JSON(&result); err == nil {
		t.Error("JSON should fail on invalid JSON")
	}
}

func TestResponseHTML(t *testing.T) {
	html := `<html><body><h1>Title</h1><p class="test">content</p></body></html>`
	resp := WrapResponse(&nethttp.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(html)),
	})

	doc, err := resp.HTML()
	if err != nil {
		t.Fatalf("HTML error: %v", err)
	}
	title := doc.Find("h1").Text()
	if title != "Title" {
		t.Errorf("h1 text = %q, want Title", title)
	}
	content := doc.Find("p.test").Text()
	if content != "content" {
		t.Errorf("p.test text = %q, want content", content)
	}
}

func TestWrapResponse(t *testing.T) {
	raw := &nethttp.Response{
		StatusCode: 404,
		Body:       io.NopCloser(strings.NewReader("not found")),
	}
	resp := WrapResponse(raw)
	if resp.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", resp.StatusCode)
	}
	s, _ := resp.BodyString()
	if s != "not found" {
		t.Errorf("body = %q, want %q", s, "not found")
	}
}

// ---------------------------------------------------------------------------
// WithProxy
// ---------------------------------------------------------------------------

func TestWithProxyInvalidURL(t *testing.T) {
	// Should not panic with an invalid proxy URL
	s := NewSession(WithProxy("://invalid"))
	if s == nil {
		t.Fatal("NewSession with invalid proxy returned nil")
	}
}

func TestWithProxyValidURL(t *testing.T) {
	s := NewSession(WithProxy("http://127.0.0.1:8080"))
	if s == nil {
		t.Fatal("NewSession with proxy returned nil")
	}
	transport, ok := s.client.Transport.(*nethttp.Transport)
	if !ok {
		t.Fatal("Transport is not *http.Transport")
	}
	if transport.Proxy == nil {
		t.Error("Proxy function not set on transport")
	}
}

// ---------------------------------------------------------------------------
// WithTransport
// ---------------------------------------------------------------------------

func TestWithTransport(t *testing.T) {
	transport := &nethttp.Transport{MaxIdleConns: 100}
	s := NewSession(WithTransport(transport))
	if s.client.Transport != transport {
		t.Error("WithTransport did not set the transport")
	}
}

// ---------------------------------------------------------------------------
// WithPool / poolTransport
// ---------------------------------------------------------------------------

func TestWithPool(t *testing.T) {
	ctx := WithPool(context.Background(), 10, "")
	transport := poolTransport(ctx)
	if transport == nil {
		t.Fatal("poolTransport returned nil")
	}
	if transport.MaxIdleConnsPerHost != 10 {
		t.Errorf("MaxIdleConnsPerHost = %d, want 10", transport.MaxIdleConnsPerHost)
	}
	if transport.MaxConnsPerHost != 10 {
		t.Errorf("MaxConnsPerHost = %d, want 10", transport.MaxConnsPerHost)
	}
	if transport.MaxIdleConns != 20 {
		t.Errorf("MaxIdleConns = %d, want 20", transport.MaxIdleConns)
	}
}

func TestWithPoolWithProxy(t *testing.T) {
	ctx := WithPool(context.Background(), 5, "http://proxy.example.com:3128")
	transport := poolTransport(ctx)
	if transport == nil {
		t.Fatal("poolTransport returned nil")
	}
	if transport.Proxy == nil {
		t.Error("Proxy not set on pool transport")
	}
}

func TestPoolTransportMissing(t *testing.T) {
	ctx := context.Background()
	transport := poolTransport(ctx)
	if transport != nil {
		t.Error("poolTransport should return nil for context without pool")
	}
}

// ---------------------------------------------------------------------------
// applyRPORT
// ---------------------------------------------------------------------------

func TestApplyRPORT(t *testing.T) {
	tests := []struct {
		name   string
		target string
		rport  string
		want   string
	}{
		{"empty rport", "example.com", "", "example.com"},
		{"rport 80 ignored", "example.com", "80", "example.com"},
		{"rport 443 ignored", "example.com", "443", "example.com"},
		{"bare host with rport", "example.com", "8080", "example.com:8080"},
		{"bare host already has port", "example.com:9090", "8080", "example.com:9090"},
		{"scheme no port", "http://example.com", "8080", "http://example.com:8080"},
		{"scheme with existing port", "http://example.com:9090", "8080", "http://example.com:9090"},
		{"https no port", "https://example.com", "8443", "https://example.com:8443"},
		{"https with existing port", "https://example.com:9443", "8443", "https://example.com:9443"},
		{"scheme with path", "http://example.com/path", "8080", "http://example.com:8080/path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyRPORT(tt.target, tt.rport)
			if got != tt.want {
				t.Errorf("applyRPORT(%q, %q) = %q, want %q", tt.target, tt.rport, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// doRequest variations: NoRedirect, NoCookies, custom Timeout
// ---------------------------------------------------------------------------

func TestNoRedirect(t *testing.T) {
	redirected := false
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.URL.Path == "/redirect" {
			nethttp.Redirect(w, r, "/final", nethttp.StatusFound)
			return
		}
		redirected = true
		w.WriteHeader(200)
		w.Write([]byte("final"))
	}))
	defer srv.Close()

	run := NewRun(context.Background(), srv.URL)
	resp, err := run.Send(Request{
		Path:       "redirect",
		NoRedirect: true,
	})
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	if resp.StatusCode != 302 {
		t.Errorf("status = %d, want 302", resp.StatusCode)
	}
	if redirected {
		t.Error("server received the redirected request, but NoRedirect should have stopped it")
	}
}

func TestFollowRedirectByDefault(t *testing.T) {
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.URL.Path == "/redirect" {
			nethttp.Redirect(w, r, "/final", nethttp.StatusFound)
			return
		}
		w.WriteHeader(200)
		w.Write([]byte("arrived"))
	}))
	defer srv.Close()

	run := NewRun(context.Background(), srv.URL)
	resp, err := run.Send(Request{Path: "redirect"})
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200 (should follow redirect)", resp.StatusCode)
	}
	body, _ := resp.BodyString()
	if body != "arrived" {
		t.Errorf("body = %q, want %q", body, "arrived")
	}
}

func TestNoCookies(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		callCount++
		if callCount == 1 {
			nethttp.SetCookie(w, &nethttp.Cookie{Name: "sid", Value: "abc123"})
			w.WriteHeader(200)
			return
		}
		cookie := r.Header.Get("Cookie")
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(cookie))
	}))
	defer srv.Close()

	s := NewSession()

	// First request sets the cookie
	s.Send(srv.URL, Request{})

	// Second request with NoCookies should not send the cookie
	resp, err := s.Send(srv.URL, Request{NoCookies: true})
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	body, _ := resp.BodyString()
	if strings.Contains(body, "sid=abc123") {
		t.Error("NoCookies should prevent sending stored cookies")
	}
}

func TestCustomTimeout(t *testing.T) {
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	run := NewRun(context.Background(), srv.URL)
	_, err := run.Send(Request{
		Timeout: 50 * time.Millisecond,
	})
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}

// ---------------------------------------------------------------------------
// detectLexer
// ---------------------------------------------------------------------------

func TestDetectLexer(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        []byte
		want        string
	}{
		{"json content-type", "application/json", nil, "json"},
		{"json charset", "application/json; charset=utf-8", nil, "json"},
		{"xml content-type", "application/xml", nil, "xml"},
		{"text/xml", "text/xml", nil, "xml"},
		{"html content-type", "text/html", nil, "html"},
		{"javascript content-type", "application/javascript", nil, "javascript"},
		{"text/javascript", "text/javascript", nil, "javascript"},
		{"css content-type", "text/css", nil, "css"},

		// Body detection
		{"json object body", "", []byte(`{"key":"value"}`), "json"},
		{"json array body", "", []byte(`[1,2,3]`), "json"},
		{"xml declaration body", "", []byte(`<?xml version="1.0"?><root/>`), "xml"},
		{"soap body", "", []byte(`<soap:Envelope/>`), "xml"},
		{"doctype body", "", []byte(`<!DOCTYPE html><html></html>`), "html"},
		{"html tag body", "", []byte(`<html><body></body></html>`), "html"},
		{"empty body", "", []byte{}, ""},
		{"plain text", "", []byte("just plain text"), ""},
		{"whitespace before json", "", []byte("  {\"a\":1}"), "json"},

		// No content-type, no matching body
		{"unknown", "", []byte("random bytes"), ""},
		{"nil body", "", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectLexer(tt.contentType, tt.body)
			if got != tt.want {
				t.Errorf("detectLexer(%q, %q) = %q, want %q", tt.contentType, string(tt.body), got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// styledStatus
// ---------------------------------------------------------------------------

func TestStyledStatus(t *testing.T) {
	tests := []struct {
		code   int
		status string
	}{
		{200, "200 OK"},
		{201, "201 Created"},
		{301, "301 Moved Permanently"},
		{302, "302 Found"},
		{400, "400 Bad Request"},
		{404, "404 Not Found"},
		{500, "500 Internal Server Error"},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := styledStatus(tt.code, tt.status)
			if result == "" {
				t.Error("styledStatus returned empty string")
			}
			// The result should contain the status text (possibly wrapped in ANSI)
			if !strings.Contains(result, tt.status) {
				t.Errorf("styledStatus(%d, %q) = %q, doesn't contain status text", tt.code, tt.status, result)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// resolveBody
// ---------------------------------------------------------------------------

func TestResolveBodyNil(t *testing.T) {
	s := NewSession()
	body, bodyBytes, ct := s.resolveBody(Request{})
	if body != nil {
		t.Error("body should be nil for empty request")
	}
	if bodyBytes != nil {
		t.Error("bodyBytes should be nil for empty request")
	}
	if ct != "" {
		t.Errorf("content-type = %q, want empty", ct)
	}
}

func TestResolveBodyForm(t *testing.T) {
	s := NewSession()
	body, bodyBytes, ct := s.resolveBody(Request{
		Form: url.Values{"key": {"val"}},
	})
	if body == nil {
		t.Fatal("body should not be nil for form data")
	}
	if !strings.Contains(string(bodyBytes), "key=val") {
		t.Errorf("bodyBytes = %q, want key=val", string(bodyBytes))
	}
	if ct != "application/x-www-form-urlencoded" {
		t.Errorf("content-type = %q, want application/x-www-form-urlencoded", ct)
	}
}

func TestResolveBodyFormCustomContentType(t *testing.T) {
	s := NewSession()
	_, _, ct := s.resolveBody(Request{
		Form:        url.Values{"key": {"val"}},
		ContentType: "multipart/form-data",
	})
	if ct != "multipart/form-data" {
		t.Errorf("content-type = %q, want multipart/form-data", ct)
	}
}

func TestResolveBodyCustomReader(t *testing.T) {
	s := NewSession()
	s.trace = false
	body, bodyBytes, ct := s.resolveBody(Request{
		Body:        strings.NewReader("raw content"),
		ContentType: "text/plain",
	})
	if body == nil {
		t.Fatal("body should not be nil")
	}
	// When trace is false, bodyBytes is nil
	if bodyBytes != nil {
		t.Error("bodyBytes should be nil when trace is false")
	}
	if ct != "text/plain" {
		t.Errorf("content-type = %q, want text/plain", ct)
	}
}

func TestResolveBodyWithTrace(t *testing.T) {
	s := NewSession()
	s.trace = true
	body, bodyBytes, ct := s.resolveBody(Request{
		Body:        strings.NewReader("traced content"),
		ContentType: "application/octet-stream",
	})
	if body == nil {
		t.Fatal("body should not be nil")
	}
	if string(bodyBytes) != "traced content" {
		t.Errorf("bodyBytes = %q, want %q", string(bodyBytes), "traced content")
	}
	if ct != "application/octet-stream" {
		t.Errorf("content-type = %q, want application/octet-stream", ct)
	}
}

// ---------------------------------------------------------------------------
// FromModule
// ---------------------------------------------------------------------------

func TestFromModule(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	// Extract host:port from the test server URL
	u, _ := url.Parse(srv.URL)

	params := sdk.NewParams(context.Background(), map[string]string{
		"TARGET":    u.Host,
		"TARGETURI": "/custom/",
	})

	run := FromModule(params)
	if run.TargetURI != "/custom/" {
		t.Errorf("TargetURI = %q, want /custom/", run.TargetURI)
	}

	resp, err := run.Send(Request{Path: "endpoint"})
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	var result map[string]any
	resp.JSON(&result)
	if result["path"] != "/custom/endpoint" {
		t.Errorf("path = %v, want /custom/endpoint", result["path"])
	}
}

func TestFromModuleDefaultTargetURI(t *testing.T) {
	params := sdk.NewParams(context.Background(), map[string]string{
		"TARGET": "example.com",
	})
	run := FromModule(params)
	if run.TargetURI != "/" {
		t.Errorf("TargetURI = %q, want /", run.TargetURI)
	}
}

func TestFromModuleWithRPORT(t *testing.T) {
	params := sdk.NewParams(context.Background(), map[string]string{
		"TARGET": "example.com",
		"RPORT":  "8080",
	})
	run := FromModule(params)
	if !strings.Contains(run.Target, "8080") {
		t.Errorf("Target = %q, expected to contain port 8080", run.Target)
	}
}

func TestFromModuleWithHTTPTrace(t *testing.T) {
	params := sdk.NewParams(context.Background(), map[string]string{
		"TARGET":     "example.com",
		"HTTP_TRACE": "true",
	})
	run := FromModule(params)
	if !run.Session.trace {
		t.Error("trace should be true when HTTP_TRACE=true")
	}
}

func TestFromModuleHTTPTraceInsensitive(t *testing.T) {
	params := sdk.NewParams(context.Background(), map[string]string{
		"TARGET":     "example.com",
		"HTTP_TRACE": "TRUE",
	})
	run := FromModule(params)
	if !run.Session.trace {
		t.Error("trace should be true when HTTP_TRACE=TRUE (case insensitive)")
	}
}

func TestFromModuleHTTPTraceOff(t *testing.T) {
	params := sdk.NewParams(context.Background(), map[string]string{
		"TARGET":     "example.com",
		"HTTP_TRACE": "false",
	})
	run := FromModule(params)
	if run.Session.trace {
		t.Error("trace should be false when HTTP_TRACE=false")
	}
}

func TestFromModuleWithProxy(t *testing.T) {
	params := sdk.NewParams(context.Background(), map[string]string{
		"TARGET":  "example.com",
		"PROXIES": "http://127.0.0.1:8080",
	})
	run := FromModule(params)
	transport, ok := run.Session.client.Transport.(*nethttp.Transport)
	if !ok {
		t.Fatal("Transport is not *http.Transport")
	}
	if transport.Proxy == nil {
		t.Error("Proxy should be set from PROXIES param")
	}
}

func TestFromModuleWithPool(t *testing.T) {
	ctx := WithPool(context.Background(), 5, "")
	params := sdk.NewParams(ctx, map[string]string{
		"TARGET": "example.com",
	})
	run := FromModule(params)
	transport, ok := run.Session.client.Transport.(*nethttp.Transport)
	if !ok {
		t.Fatal("Transport is not *http.Transport")
	}
	// Should use the pool transport with its settings
	if transport.MaxConnsPerHost != 5 {
		t.Errorf("MaxConnsPerHost = %d, want 5", transport.MaxConnsPerHost)
	}
}

// ---------------------------------------------------------------------------
// MaxBody limit
// ---------------------------------------------------------------------------

func TestMaxBodyLimit(t *testing.T) {
	bigBody := strings.Repeat("x", 1000)
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Write([]byte(bigBody))
	}))
	defer srv.Close()

	run := NewRun(context.Background(), srv.URL)
	resp, err := run.Send(Request{MaxBody: 100})
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	data, _ := resp.BodyBytes()
	if len(data) > 100 {
		t.Errorf("body length = %d, want <= 100", len(data))
	}
}

// ---------------------------------------------------------------------------
// Multiple methods on same response (caching)
// ---------------------------------------------------------------------------

func TestResponseMethodChaining(t *testing.T) {
	body := `{"items":["a","b"],"count":2}`
	resp := WrapResponse(&nethttp.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
	})

	// BodyString first
	s, err := resp.BodyString()
	if err != nil {
		t.Fatalf("BodyString error: %v", err)
	}
	if s != body {
		t.Errorf("BodyString = %q, want %q", s, body)
	}

	// Contains should work on cached body
	if !resp.Contains("items") {
		t.Error("Contains(items) should be true")
	}

	// JSON should work on cached body
	var result map[string]any
	if err := resp.JSON(&result); err != nil {
		t.Fatalf("JSON error: %v", err)
	}
	if result["count"].(float64) != 2 {
		t.Errorf("count = %v, want 2", result["count"])
	}
}

// ---------------------------------------------------------------------------
// Integration: full round-trip with httptest
// ---------------------------------------------------------------------------

func TestFullRoundTrip(t *testing.T) {
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		switch r.URL.Path {
		case "/api/status":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		case "/api/data":
			if r.Method != "POST" {
				w.WriteHeader(405)
				return
			}
			body, _ := io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"received": string(body)})
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	run := NewRun(context.Background(), srv.URL)
	run.TargetURI = "/api"

	// GET /api/status
	resp, err := run.Send(Request{Path: "status"})
	if err != nil {
		t.Fatalf("GET /api/status error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if !resp.Contains("ok") {
		t.Error("response should contain 'ok'")
	}

	// POST /api/data
	resp, err = run.Send(Request{
		Method:      "POST",
		Path:        "data",
		Body:        strings.NewReader("payload"),
		ContentType: "text/plain",
	})
	if err != nil {
		t.Fatalf("POST /api/data error: %v", err)
	}
	if !resp.Contains("payload") {
		t.Error("response should contain 'payload'")
	}

	// 404
	resp, err = run.Send(Request{Path: "unknown"})
	if err != nil {
		t.Fatalf("GET /api/unknown error: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Session headers are applied
// ---------------------------------------------------------------------------

func TestSessionHeaders(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	s := NewSession()
	s.headers["X-Session-Header"] = "session-val"

	resp, err := s.Send(srv.URL, Request{})
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	var result map[string]any
	resp.JSON(&result)
	headers := result["headers"].(map[string]any)
	if headers["X-Session-Header"] != "session-val" {
		t.Errorf("X-Session-Header = %v, want session-val", headers["X-Session-Header"])
	}
}

func TestRequestHeadersOverrideSession(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	s := NewSession()
	s.headers["X-Test"] = "session"

	resp, err := s.Send(srv.URL, Request{
		Headers: map[string]string{"X-Test": "request"},
	})
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	var result map[string]any
	resp.JSON(&result)
	headers := result["headers"].(map[string]any)
	if headers["X-Test"] != "request" {
		t.Errorf("X-Test = %v, want request (request-level should override session-level)", headers["X-Test"])
	}
}

// ---------------------------------------------------------------------------
// AutoScheme with httptest
// ---------------------------------------------------------------------------

func TestAutoSchemePreservesExisting(t *testing.T) {
	result := AutoScheme("http://example.com")
	if result != "http://example.com" {
		t.Errorf("AutoScheme should preserve existing scheme, got %q", result)
	}

	result = AutoScheme("https://example.com")
	if result != "https://example.com" {
		t.Errorf("AutoScheme should preserve existing scheme, got %q", result)
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestSendToInvalidURL(t *testing.T) {
	s := NewSession()
	_, err := s.Send("http://[::1]:invalid", Request{})
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestRunSendCtxPassthrough(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	run := NewRun(ctx, srv.URL)
	_, err := run.Send(Request{})
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestRunSendRequestCtxOverride(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	// Run has a cancelled context, but request provides its own
	runCtx, runCancel := context.WithCancel(context.Background())
	runCancel()

	run := NewRun(runCtx, srv.URL)

	reqCtx := context.Background()
	resp, err := run.Send(Request{Ctx: reqCtx})
	if err != nil {
		t.Fatalf("Send error: %v (request context should override)", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

// --- Trace mode triggers debug functions ---

func TestSendWithTrace(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	s := NewSession()
	s.trace = true
	resp, err := s.Send(srv.URL, Request{
		Method:      "POST",
		Body:        strings.NewReader("trace body"),
		ContentType: "application/json",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d", resp.StatusCode)
	}
}

// --- Response error paths ---

type errorReadCloser struct{}

func (e *errorReadCloser) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (e *errorReadCloser) Close() error              { return nil }

func TestResponseBodyBytesError(t *testing.T) {
	resp := WrapResponse(&nethttp.Response{
		StatusCode: 200,
		Body:       &errorReadCloser{},
	})
	_, err := resp.BodyBytes()
	if err == nil {
		t.Error("expected error")
	}
}

func TestResponseBodyStringError(t *testing.T) {
	resp := WrapResponse(&nethttp.Response{
		StatusCode: 200,
		Body:       &errorReadCloser{},
	})
	_, err := resp.BodyString()
	if err == nil {
		t.Error("expected error")
	}
}

func TestResponseContainsError(t *testing.T) {
	resp := WrapResponse(&nethttp.Response{
		StatusCode: 200,
		Body:       &errorReadCloser{},
	})
	if resp.Contains("anything") {
		t.Error("should return false on error")
	}
}

func TestResponseContainsAllError(t *testing.T) {
	resp := WrapResponse(&nethttp.Response{
		StatusCode: 200,
		Body:       &errorReadCloser{},
	})
	if resp.ContainsAll("a") {
		t.Error("should return false on error")
	}
}

func TestResponseContainsAnyError(t *testing.T) {
	resp := WrapResponse(&nethttp.Response{
		StatusCode: 200,
		Body:       &errorReadCloser{},
	})
	if resp.ContainsAny("a") {
		t.Error("should return false on error")
	}
}

func TestResponseJSONError(t *testing.T) {
	resp := WrapResponse(&nethttp.Response{
		StatusCode: 200,
		Body:       &errorReadCloser{},
	})
	var m map[string]string
	if err := resp.JSON(&m); err == nil {
		t.Error("expected error")
	}
}

func TestResponseHTMLError(t *testing.T) {
	resp := WrapResponse(&nethttp.Response{
		StatusCode: 200,
		Body:       &errorReadCloser{},
	})
	_, err := resp.HTML()
	if err == nil {
		t.Error("expected error")
	}
}

// --- AutoScheme with test server ---

func TestAutoSchemeHTTPFallback(t *testing.T) {
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	// Extract host:port, AutoScheme will try HTTPS first (fail), then HTTP
	host := strings.TrimPrefix(srv.URL, "http://")
	got := AutoScheme(host)
	if !strings.HasPrefix(got, "http://") {
		t.Errorf("expected http:// prefix, got %q", got)
	}
}

// --- printHeaders/printBody/debugRequest/debugResponse via trace ---

func TestDebugFunctions(t *testing.T) {
	// Just test they don't panic
	h := nethttp.Header{}
	h.Set("Content-Type", "application/json")
	printHeaders(h, ">")

	printBody([]byte(`{"a":1}`), "application/json")
	printBody([]byte(`<html>`), "text/html")
	printBody([]byte(`plain text`), "text/plain")
	printBody([]byte(`body`), "")
}

// --- option.go init: test the registered sender factory ---

func TestSenderFactoryViaSend(t *testing.T) {
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Header().Set("X-Test", "passed")
		w.Write([]byte("factory ok"))
	}))
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	params := sdk.NewParams(context.Background(), map[string]string{
		"TARGET": u.Host,
	})

	ctx := sdk.NewContext(map[string]string{"TARGET": u.Host}, "")
	sdk.WireSenders(ctx, params)

	// Normal request
	resp, err := ctx.Send(sdk.HTTPRequest{Method: "GET"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d", resp.StatusCode)
	}
	if resp.Headers["X-Test"] != "passed" {
		t.Errorf("header = %q", resp.Headers["X-Test"])
	}
}

func TestSenderFactoryFireAndForget(t *testing.T) {
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	params := sdk.NewParams(context.Background(), map[string]string{
		"TARGET": u.Host,
	})

	ctx := sdk.NewContext(map[string]string{"TARGET": u.Host}, "")
	sdk.WireSenders(ctx, params)

	// Fire and forget returns status 0
	resp, err := ctx.Send(sdk.HTTPRequest{Method: "GET", FireAndForget: true})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 0 {
		t.Errorf("fire-and-forget status = %d, want 0", resp.StatusCode)
	}
}

func TestSenderFactoryWithTimeout(t *testing.T) {
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	params := sdk.NewParams(context.Background(), map[string]string{
		"TARGET": u.Host,
	})

	ctx := sdk.NewContext(map[string]string{"TARGET": u.Host}, "")
	sdk.WireSenders(ctx, params)

	// Request with explicit timeout
	resp, err := ctx.Send(sdk.HTTPRequest{Method: "GET", Timeout: 5})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d", resp.StatusCode)
	}
}

func TestEmptyBasicAuthNotSent(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	run := NewRun(context.Background(), srv.URL)
	resp, err := run.Send(Request{
		BasicAuth: [2]string{"", ""},
	})
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	var result map[string]any
	resp.JSON(&result)
	headers := result["headers"].(map[string]any)
	if _, ok := headers["Authorization"]; ok {
		t.Error("Authorization header should not be set with empty BasicAuth")
	}
}
