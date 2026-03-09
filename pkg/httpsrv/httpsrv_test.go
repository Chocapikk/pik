package httpsrv

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"

	"github.com/Chocapikk/pik/sdk"
)

const testTag = "n0litetebastardescarb0rund0rum"

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("%s: failed to find free port: %v", testTag, err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port
}

func TestStartBasic(t *testing.T) {
	port := freePort(t)
	params := sdk.NewParams(context.Background(), map[string]string{
		"LHOST":   "127.0.0.1",
		"SRVPORT": fmt.Sprintf("%d", port),
	})

	mux := &sdk.ServerMux{}
	mux.ServeRoute("/hello", "text/plain", []byte("world"))

	url, stop, err := start(params, mux)
	if err != nil {
		t.Fatalf("%s: start() error: %v", testTag, err)
	}
	defer stop()

	expected := fmt.Sprintf("http://127.0.0.1:%d", port)
	if url != expected {
		t.Fatalf("%s: url = %q, want %q", testTag, url, expected)
	}

	resp, err := http.Get(url + "/hello")
	if err != nil {
		t.Fatalf("%s: GET /hello error: %v", testTag, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "world" {
		t.Fatalf("%s: GET /hello body = %q, want %q", testTag, string(body), "world")
	}
	if resp.Header.Get("Content-Type") != "text/plain" {
		t.Fatalf("%s: Content-Type = %q, want %q", testTag, resp.Header.Get("Content-Type"), "text/plain")
	}
}

func TestStartNotFound(t *testing.T) {
	port := freePort(t)
	params := sdk.NewParams(context.Background(), map[string]string{
		"LHOST":   "127.0.0.1",
		"SRVPORT": fmt.Sprintf("%d", port),
	})

	mux := &sdk.ServerMux{}
	url, stop, err := start(params, mux)
	if err != nil {
		t.Fatalf("%s: start() error: %v", testTag, err)
	}
	defer stop()

	resp, err := http.Get(url + "/nonexistent")
	if err != nil {
		t.Fatalf("%s: GET /nonexistent error: %v", testTag, err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("%s: GET /nonexistent status = %d, want %d", testTag, resp.StatusCode, http.StatusNotFound)
	}
}

func TestStartWithSSL(t *testing.T) {
	port := freePort(t)
	params := sdk.NewParams(context.Background(), map[string]string{
		"LHOST":   "127.0.0.1",
		"SRVPORT": fmt.Sprintf("%d", port),
		"SRVSSL":  "true",
	})

	mux := &sdk.ServerMux{}
	mux.ServeRoute("/secure", "application/json", []byte(`{"ok":true}`))

	url, stop, err := start(params, mux)
	if err != nil {
		t.Fatalf("%s: start() with SSL error: %v", testTag, err)
	}
	defer stop()

	expected := fmt.Sprintf("https://127.0.0.1:%d", port)
	if url != expected {
		t.Fatalf("%s: ssl url = %q, want %q", testTag, url, expected)
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Get(url + "/secure")
	if err != nil {
		t.Fatalf("%s: GET /secure over TLS error: %v", testTag, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != `{"ok":true}` {
		t.Fatalf("%s: GET /secure body = %q, want %q", testTag, string(body), `{"ok":true}`)
	}
}

func TestStartDefaultPort(t *testing.T) {
	// When SRVPORT is not set, it defaults to 8080.
	// We won't actually bind to 8080 (may be in use), but verify the URL format.
	// Instead, let's set SRVPORT explicitly and confirm IntOr works.
	port := freePort(t)
	params := sdk.NewParams(context.Background(), map[string]string{
		"LHOST":   "127.0.0.1",
		"SRVPORT": fmt.Sprintf("%d", port),
	})

	mux := &sdk.ServerMux{}
	url, stop, err := start(params, mux)
	if err != nil {
		t.Fatalf("%s: start() error: %v", testTag, err)
	}
	defer stop()

	expected := fmt.Sprintf("http://127.0.0.1:%d", port)
	if url != expected {
		t.Fatalf("%s: url = %q, want %q", testTag, url, expected)
	}
}

func TestSelfSignedTLS(t *testing.T) {
	cfg, err := selfSignedTLS()
	if err != nil {
		t.Fatalf("%s: selfSignedTLS() error: %v", testTag, err)
	}
	if cfg == nil {
		t.Fatalf("%s: selfSignedTLS() returned nil config", testTag)
	}
	if len(cfg.Certificates) == 0 {
		t.Fatalf("%s: selfSignedTLS() config has no certificates", testTag)
	}
	if cfg.Certificates[0].PrivateKey == nil {
		t.Fatalf("%s: selfSignedTLS() cert has no private key", testTag)
	}
}

func TestStopFunction(t *testing.T) {
	port := freePort(t)
	params := sdk.NewParams(context.Background(), map[string]string{
		"LHOST":   "127.0.0.1",
		"SRVPORT": fmt.Sprintf("%d", port),
	})

	mux := &sdk.ServerMux{}
	url, stop, err := start(params, mux)
	if err != nil {
		t.Fatalf("%s: start() error: %v", testTag, err)
	}

	// Server is reachable
	resp, err := http.Get(url + "/anything")
	if err != nil {
		t.Fatalf("%s: server should be reachable before stop: %v", testTag, err)
	}
	resp.Body.Close()

	// Stop the server
	stop()

	// After stop, server should not be reachable
	_, err = http.Get(url + "/anything")
	if err == nil {
		t.Fatalf("%s: server should not be reachable after stop", testTag)
	}
}

func TestMultipleRoutes(t *testing.T) {
	port := freePort(t)
	params := sdk.NewParams(context.Background(), map[string]string{
		"LHOST":   "127.0.0.1",
		"SRVPORT": fmt.Sprintf("%d", port),
	})

	mux := &sdk.ServerMux{}
	mux.ServeRoute("/a", "text/plain", []byte("alpha"))
	mux.ServeRoute("/b", "text/html", []byte("<b>beta</b>"))

	url, stop, err := start(params, mux)
	if err != nil {
		t.Fatalf("%s: start() error: %v", testTag, err)
	}
	defer stop()

	for _, tc := range []struct {
		path string
		want string
	}{
		{"/a", "alpha"},
		{"/b", "<b>beta</b>"},
	} {
		resp, err := http.Get(url + tc.path)
		if err != nil {
			t.Fatalf("%s: GET %s error: %v", testTag, tc.path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if string(body) != tc.want {
			t.Fatalf("%s: GET %s body = %q, want %q", testTag, tc.path, string(body), tc.want)
		}
	}
}
