package sslshell

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"strings"
	"testing"
)

const testTag = "n0litetebastardescarb0rund0rum"

func TestNew(t *testing.T) {
	l := New()
	if l == nil {
		t.Fatal(testTag, "New() returned nil")
	}
}

func TestName(t *testing.T) {
	l := New()
	if l.Name() != "sslshell" {
		t.Fatalf("%s: Name() = %q, want %q", testTag, l.Name(), "sslshell")
	}
}

func TestSetup(t *testing.T) {
	l := New()
	if err := l.Setup("127.0.0.1", 0); err != nil {
		t.Fatalf("%s: Setup failed: %v", testTag, err)
	}
	defer l.Shutdown()

	if l.Manager == nil {
		t.Fatalf("%s: Manager is nil after Setup", testTag)
	}
}

func TestGeneratePayloadKnownTypes(t *testing.T) {
	l := New()
	l.lhost = "10.0.0.1"
	l.lport = 4443

	tests := []struct {
		payloadType string
		wantSubstr  string
	}{
		{"cmd/bash/reverse_tls", "openssl s_client"},
		{"cmd/python/reverse_tls", "ssl.wrap_socket"},
		{"cmd/ncat/reverse_tls", "ncat --ssl"},
		{"cmd/socat/reverse_tls", "openssl-connect"},
	}

	for _, tt := range tests {
		t.Run(tt.payloadType, func(t *testing.T) {
			p, err := l.GeneratePayload("linux", tt.payloadType)
			if err != nil {
				t.Fatalf("%s: GeneratePayload(%q) error: %v", testTag, tt.payloadType, err)
			}
			if !strings.Contains(p, tt.wantSubstr) {
				t.Fatalf("%s: GeneratePayload(%q) = %q, want substring %q", testTag, tt.payloadType, p, tt.wantSubstr)
			}
		})
	}
}

func TestGeneratePayloadFallback(t *testing.T) {
	l := New()
	l.lhost = "10.0.0.1"
	l.lport = 4443

	p, err := l.GeneratePayload("linux", "cmd/nonexistent/type")
	if err != nil {
		t.Fatalf("%s: GeneratePayload fallback error: %v", testTag, err)
	}
	// Fallback is BashTLS which uses openssl s_client
	if !strings.Contains(p, "openssl s_client") {
		t.Fatalf("%s: fallback should contain openssl s_client, got: %s", testTag, p)
	}
}

func TestGeneratePayloadContainsHostPort(t *testing.T) {
	l := &Listener{}
	l.lhost = "192.168.1.100"
	l.lport = 7777

	p, err := l.GeneratePayload("linux", "cmd/bash/reverse_tls")
	if err != nil {
		t.Fatalf("%s: GeneratePayload error: %v", testTag, err)
	}
	if !strings.Contains(p, "192.168.1.100") {
		t.Fatalf("%s: payload should contain host, got: %s", testTag, p)
	}
	if !strings.Contains(p, "7777") {
		t.Fatalf("%s: payload should contain port, got: %s", testTag, p)
	}
}

func TestSelfSignedCertWithIP(t *testing.T) {
	cert, err := selfSignedCert("127.0.0.1")
	if err != nil {
		t.Fatalf("%s: selfSignedCert(IP) error: %v", testTag, err)
	}
	if len(cert.Certificate) == 0 {
		t.Fatalf("%s: selfSignedCert(IP) produced no certificate data", testTag)
	}

	parsed, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("%s: failed to parse certificate: %v", testTag, err)
	}
	if len(parsed.IPAddresses) == 0 {
		t.Fatalf("%s: certificate should have IP SANs for IP input", testTag)
	}
	if !parsed.IPAddresses[0].Equal(net.ParseIP("127.0.0.1")) {
		t.Fatalf("%s: certificate IP SAN = %v, want 127.0.0.1", testTag, parsed.IPAddresses[0])
	}
}

func TestSelfSignedCertWithHostname(t *testing.T) {
	cert, err := selfSignedCert("example.com")
	if err != nil {
		t.Fatalf("%s: selfSignedCert(hostname) error: %v", testTag, err)
	}
	if len(cert.Certificate) == 0 {
		t.Fatalf("%s: selfSignedCert(hostname) produced no certificate data", testTag)
	}

	parsed, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("%s: failed to parse certificate: %v", testTag, err)
	}
	if len(parsed.DNSNames) == 0 {
		t.Fatalf("%s: certificate should have DNS SANs for hostname input", testTag)
	}
	if parsed.DNSNames[0] != "example.com" {
		t.Fatalf("%s: certificate DNS SAN = %q, want %q", testTag, parsed.DNSNames[0], "example.com")
	}
}

func TestSelfSignedCertIsTLS(t *testing.T) {
	cert, err := selfSignedCert("127.0.0.1")
	if err != nil {
		t.Fatalf("%s: selfSignedCert error: %v", testTag, err)
	}

	// Verify the certificate can be used with a TLS config
	tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", tlsCfg)
	if err != nil {
		t.Fatalf("%s: failed to create TLS listener with self-signed cert: %v", testTag, err)
	}
	ln.Close()
}

func TestShutdownAfterSetup(t *testing.T) {
	l := New()
	if err := l.Setup("127.0.0.1", 0); err != nil {
		t.Fatalf("%s: Setup failed: %v", testTag, err)
	}
	if err := l.Shutdown(); err != nil {
		t.Fatalf("%s: Shutdown failed: %v", testTag, err)
	}
}

func TestShutdownWithoutSetup(t *testing.T) {
	l := New()
	// Manager is nil, ShutdownManager should handle nil gracefully
	if err := l.Shutdown(); err != nil {
		t.Fatalf("%s: Shutdown without Setup should not error, got: %v", testTag, err)
	}
}
