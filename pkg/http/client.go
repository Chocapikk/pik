package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	nethttp "net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/Chocapikk/pik/sdk"
	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/pkg/text"
)

// --- Types ---

const defaultMaxBody = 10 * 1024 * 1024

// Option configures a Session.
type Option func(*Session)

// Session is a persistent HTTP client shared across requests.
type Session struct {
	client  *nethttp.Client
	headers map[string]string
	maxBody int64
}

// Request describes a single HTTP request.
type Request struct {
	Method      string
	Path        string
	Query       url.Values
	Form        url.Values
	Body        io.Reader
	ContentType string
	Headers     map[string]string
	Ctx         context.Context
	Timeout     time.Duration
	NoRedirect  bool
	NoCookies   bool
	BasicAuth   [2]string
	MaxBody     int64
}

// Run binds a Session to a target for the duration of an exploit.
type Run struct {
	Session   *Session
	Target    string
	TargetURI string
	Ctx       context.Context
}

// --- Constructors ---

// WithTransport sets a shared transport for connection pooling.
func WithTransport(t *nethttp.Transport) Option {
	return func(s *Session) { s.client.Transport = t }
}

// NewSession creates a new HTTP session. TLS verification is disabled by default.
func NewSession(opts ...Option) *Session {
	jar, _ := cookiejar.New(nil)
	s := &Session{
		client: &nethttp.Client{
			Jar: jar,
			Transport: &nethttp.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			Timeout: 10 * time.Second,
		},
		headers: make(map[string]string),
		maxBody: defaultMaxBody,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// NewRun creates a Run with a fresh session. Auto-detects scheme if missing.
func NewRun(ctx context.Context, target string, opts ...Option) *Run {
	return &Run{Session: NewSession(opts...), Target: AutoScheme(target), TargetURI: "/", Ctx: ctx}
}

// WithProxy sets an HTTP/SOCKS5 proxy on the session transport.
func WithProxy(proxyURL string) Option {
	return func(s *Session) {
		parsed, err := url.Parse(proxyURL)
		if err != nil {
			return
		}
		if t, ok := s.client.Transport.(*nethttp.Transport); ok {
			t.Proxy = nethttp.ProxyURL(parsed)
		}
	}
}

// FromModule creates a Run from module params.
func FromModule(params sdk.Params, opts ...Option) *Run {
	ctx := params.Ctx
	if t := poolTransport(ctx); t != nil {
		opts = append([]Option{WithTransport(t)}, opts...)
	}
	if proxy := params.Get("PROXIES"); proxy != "" {
		opts = append(opts, WithProxy(proxy))
	}
	target := applyRPORT(params.Target(), params.Get("RPORT"))
	run := NewRun(ctx, target, opts...)
	run.TargetURI = params.GetOr("TARGETURI", "/")
	return run
}

// applyRPORT appends the port to the target if RPORT is set and the target
// doesn't already include a port.
func applyRPORT(target, rport string) string {
	if rport == "" || rport == "80" || rport == "443" {
		return target
	}
	// If target already has a scheme, parse it properly.
	if strings.Contains(target, "://") {
		u, err := url.Parse(target)
		if err != nil || u.Port() != "" {
			return target
		}
		u.Host = u.Hostname() + ":" + rport
		return u.String()
	}
	// Bare host - only add port if not already present.
	if strings.Contains(target, ":") {
		return target
	}
	return target + ":" + rport
}

// --- Connection pooling ---

type transportKey struct{}

// WithPool returns a context carrying a shared HTTP transport.
func WithPool(ctx context.Context, threads int, proxyURL string) context.Context {
	transport := &nethttp.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:        threads * 2,
		MaxIdleConnsPerHost: threads,
		MaxConnsPerHost:     threads,
		IdleConnTimeout:     30 * time.Second,
	}
	if proxyURL != "" {
		if parsed, err := url.Parse(proxyURL); err == nil {
			transport.Proxy = nethttp.ProxyURL(parsed)
		}
	}
	return context.WithValue(ctx, transportKey{}, transport)
}

func poolTransport(ctx context.Context) *nethttp.Transport {
	t, _ := ctx.Value(transportKey{}).(*nethttp.Transport)
	return t
}

// --- Session methods ---

// Send builds and sends an HTTP request.
func (s *Session) Send(target string, req Request) (*Response, error) {
	if req.Method == "" {
		req.Method = "GET"
	}

	body, bodyBytes, ct := s.resolveBody(req)

	ctx := req.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	rawURL := target + req.Path
	if len(req.Query) > 0 {
		rawURL += "?" + req.Query.Encode()
	}

	httpReq, err := nethttp.NewRequestWithContext(ctx, req.Method, rawURL, body)
	if err != nil {
		return nil, err
	}

	s.applyHeaders(httpReq, req, ct)

	if output.IsDebug() {
		debugRequest(httpReq, bodyBytes)
	}

	resp, err := s.doRequest(httpReq, req)
	if err != nil {
		output.Debug("HTTP error: %v", err)
		return nil, err
	}

	if output.IsDebug() {
		debugResponse(resp)
	}

	maxBody := s.maxBody
	if req.MaxBody > 0 {
		maxBody = req.MaxBody
	}
	if maxBody > 0 {
		resp.Body = io.NopCloser(io.LimitReader(resp.Body, maxBody))
	}

	return WrapResponse(resp), nil
}

// Get is a shortcut for GET requests.
func (s *Session) Get(target string) (*Response, error) {
	return s.Send(target, Request{})
}

// PostForm is a shortcut for POST with form data.
func (s *Session) PostForm(target string, data url.Values) (*Response, error) {
	return s.Send(target, Request{Method: "POST", Form: data})
}

// Run binds this Session to a target.
func (s *Session) Run(ctx context.Context, target string) *Run {
	return &Run{Session: s, Target: target, TargetURI: "/", Ctx: ctx}
}

// --- Run methods ---

// Send dispatches a request through the bound session. Path is joined to TargetURI.
func (r *Run) Send(req Request) (*Response, error) {
	if req.Ctx == nil {
		req.Ctx = r.Ctx
	}
	req.Path = NormalizeURI(r.TargetURI, req.Path)
	return r.Session.Send(r.Target, req)
}

// --- Target helpers ---

// AutoScheme probes HTTPS then falls back to HTTP if the target has no scheme.
func AutoScheme(target string) string {
	if strings.Contains(target, "://") {
		return target
	}

	https := "https://" + target
	client := &nethttp.Client{
		Timeout:   3 * time.Second,
		Transport: &nethttp.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		CheckRedirect: func(*nethttp.Request, []*nethttp.Request) error {
			return nethttp.ErrUseLastResponse
		},
	}
	resp, err := client.Get(https)
	if err == nil {
		resp.Body.Close()
		return https
	}
	return "http://" + target
}

// --- URI helpers ---

// NormalizeURI joins path segments cleanly.
func NormalizeURI(parts ...string) string {
	joined := ""
	for _, seg := range parts {
		seg = strings.TrimRight(seg, "/")
		if seg == "" {
			continue
		}
		if !strings.HasPrefix(seg, "/") {
			seg = "/" + seg
		}
		joined += seg
	}
	if joined == "" {
		return "/"
	}
	last := parts[len(parts)-1]
	if len(last) > 1 && strings.HasSuffix(last, "/") {
		joined += "/"
	}
	return joined
}

// --- Internal helpers ---

func (s *Session) resolveBody(req Request) (io.Reader, []byte, string) {
	ct := req.ContentType
	if req.Body != nil {
		if output.IsDebug() {
			raw, _ := io.ReadAll(req.Body)
			return bytes.NewReader(raw), raw, ct
		}
		return req.Body, nil, ct
	}
	if len(req.Form) > 0 {
		encoded := req.Form.Encode()
		if ct == "" {
			ct = "application/x-www-form-urlencoded"
		}
		return strings.NewReader(encoded), []byte(encoded), ct
	}
	return nil, nil, ct
}

func (s *Session) applyHeaders(httpReq *nethttp.Request, req Request, ct string) {
	if ct != "" {
		httpReq.Header.Set("Content-Type", ct)
	}
	httpReq.Header.Set("User-Agent", text.RandUserAgent())
	// Direct map assignment instead of Header.Set() to preserve original
	// casing. Some servers (e.g. Spring Cloud Function) are case-sensitive
	// on header names and Header.Set() canonicalizes them.
	for k, v := range s.headers {
		httpReq.Header[k] = []string{v}
	}
	for k, v := range req.Headers {
		httpReq.Header[k] = []string{v}
	}
	if req.BasicAuth[0] != "" {
		httpReq.SetBasicAuth(req.BasicAuth[0], req.BasicAuth[1])
	}
}

func (s *Session) doRequest(httpReq *nethttp.Request, req Request) (*nethttp.Response, error) {
	custom := req.Timeout > 0 && req.Timeout != s.client.Timeout
	if !req.NoRedirect && !req.NoCookies && !custom {
		return s.client.Do(httpReq)
	}
	clone := *s.client
	if custom {
		clone.Timeout = req.Timeout
	}
	if req.NoRedirect {
		clone.CheckRedirect = func(*nethttp.Request, []*nethttp.Request) error {
			return nethttp.ErrUseLastResponse
		}
	}
	if req.NoCookies {
		clone.Jar = nil
	}
	return clone.Do(httpReq)
}
