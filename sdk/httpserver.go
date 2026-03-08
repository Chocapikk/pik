package sdk

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// HTTPServerModule is a marker interface for modules that need an exploit
// HTTP server during exploitation (e.g. supply chain attacks, staging).
// The runner detects this interface and starts an HTTP server automatically.
// Modules register routes via Context.ServeRoute() and wait for hits via
// Context.WaitRoutes(). The server URL is available via Context.ExploitURL().
type HTTPServerModule interface {
	HTTPServer() // marker
}

// WithHTTPServer is an embeddable type that satisfies HTTPServerModule.
// Modules embed this in their struct to signal they need an exploit HTTP server.
type WithHTTPServer struct{}

func (WithHTTPServer) HTTPServer() {}

// serverRoute is a route registered by a module via Context.ServeRoute.
type serverRoute struct {
	pattern     string
	contentType string
	body        []byte
	hit         atomic.Bool
}

// ServerMux is an internal route table for the exploit HTTP server.
type ServerMux struct {
	mu     sync.RWMutex
	routes []*serverRoute
}

// ServeRoute registers a route on the exploit HTTP server.
// Requests whose path contains the pattern are served with the given body.
func (m *ServerMux) ServeRoute(pattern, contentType string, body []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.routes = append(m.routes, &serverRoute{
		pattern:     pattern,
		contentType: contentType,
		body:        body,
	})
}

// WaitRoutes blocks until all patterns have been hit or timeout expires.
func (m *ServerMux) WaitRoutes(timeoutSec int, patterns ...string) error {
	deadline := time.After(time.Duration(timeoutSec) * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			return Errorf("timeout waiting for HTTP requests")
		case <-ticker.C:
			if m.allHit(patterns) {
				return nil
			}
		}
	}
}

func (m *ServerMux) allHit(patterns []string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range patterns {
		hit := false
		for _, r := range m.routes {
			if r.pattern == p && r.hit.Load() {
				hit = true
				break
			}
		}
		if !hit {
			return false
		}
	}
	return true
}

// Match finds the first route matching the path.
// Pattern syntax: no wildcard = exact, *suffix = ends-with,
// prefix* = starts-with, *contains* = substring.
// Used internally by the runner's HTTP handler.
func (m *ServerMux) Match(path string) (string, []byte, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, r := range m.routes {
		if matchPattern(path, r.pattern) {
			r.hit.Store(true)
			return r.contentType, r.body, true
		}
	}
	return "", nil, false
}

func matchPattern(path, pattern string) bool {
	star := strings.Contains(pattern, "*")
	if !star {
		return path == pattern
	}
	p := strings.Trim(pattern, "*")
	prefix := strings.HasPrefix(pattern, "*")
	suffix := strings.HasSuffix(pattern, "*")
	if prefix && suffix {
		return strings.Contains(path, p)
	}
	if prefix {
		return strings.HasSuffix(path, p)
	}
	return strings.HasPrefix(path, p)
}
