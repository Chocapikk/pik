package sliver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const testTag = "n0litetebastardescarb0rund0rum"

// ---------------------------------------------------------------------------
// ParseConfig
// ---------------------------------------------------------------------------

func TestParseConfigValid(t *testing.T) {
	cfg := OperatorConfig{
		Operator:      "testop",
		LHost:         "10.0.0.1",
		LPort:         31337,
		Token:         "tok-abc-123",
		CACertificate: "-----BEGIN CERTIFICATE-----\nfake-ca\n-----END CERTIFICATE-----",
		PrivateKey:    "-----BEGIN PRIVATE KEY-----\nfake-key\n-----END PRIVATE KEY-----",
		Certificate:   "-----BEGIN CERTIFICATE-----\nfake-cert\n-----END CERTIFICATE-----",
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("%s: marshal: %v", testTag, err)
	}

	path := filepath.Join(t.TempDir(), "operator.cfg")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("%s: write: %v", testTag, err)
	}

	got, err := ParseConfig(path)
	if err != nil {
		t.Fatalf("%s: ParseConfig: %v", testTag, err)
	}

	if got.Operator != "testop" {
		t.Errorf("%s: Operator = %q, want %q", testTag, got.Operator, "testop")
	}
	if got.LHost != "10.0.0.1" {
		t.Errorf("%s: LHost = %q, want %q", testTag, got.LHost, "10.0.0.1")
	}
	if got.LPort != 31337 {
		t.Errorf("%s: LPort = %d, want %d", testTag, got.LPort, 31337)
	}
	if got.Token != "tok-abc-123" {
		t.Errorf("%s: Token = %q, want %q", testTag, got.Token, "tok-abc-123")
	}
	if got.CACertificate != cfg.CACertificate {
		t.Errorf("%s: CACertificate mismatch", testTag)
	}
	if got.PrivateKey != cfg.PrivateKey {
		t.Errorf("%s: PrivateKey mismatch", testTag)
	}
	if got.Certificate != cfg.Certificate {
		t.Errorf("%s: Certificate mismatch", testTag)
	}
}

func TestParseConfigAllFields(t *testing.T) {
	tests := []struct {
		name string
		cfg  OperatorConfig
	}{
		{
			name: "minimal",
			cfg:  OperatorConfig{Operator: "op1"},
		},
		{
			name: "full",
			cfg: OperatorConfig{
				Operator:      "op2",
				LHost:         "192.168.1.100",
				LPort:         443,
				Token:         "my-token",
				CACertificate: "ca-data",
				PrivateKey:    "key-data",
				Certificate:   "cert-data",
			},
		},
		{
			name: "zero port",
			cfg:  OperatorConfig{LPort: 0, LHost: "localhost"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.cfg)
			if err != nil {
				t.Fatalf("%s: marshal: %v", testTag, err)
			}

			path := filepath.Join(t.TempDir(), "operator.cfg")
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatalf("%s: write: %v", testTag, err)
			}

			got, err := ParseConfig(path)
			if err != nil {
				t.Fatalf("%s: ParseConfig: %v", testTag, err)
			}
			if got.Operator != tt.cfg.Operator {
				t.Errorf("%s: Operator = %q, want %q", testTag, got.Operator, tt.cfg.Operator)
			}
			if got.LHost != tt.cfg.LHost {
				t.Errorf("%s: LHost = %q, want %q", testTag, got.LHost, tt.cfg.LHost)
			}
			if got.LPort != tt.cfg.LPort {
				t.Errorf("%s: LPort = %d, want %d", testTag, got.LPort, tt.cfg.LPort)
			}
		})
	}
}

func TestParseConfigInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.cfg")
	if err := os.WriteFile(path, []byte("{not valid json!!!}"), 0o600); err != nil {
		t.Fatalf("%s: write: %v", testTag, err)
	}

	_, err := ParseConfig(path)
	if err == nil {
		t.Fatalf("%s: ParseConfig should fail on invalid JSON", testTag)
	}
}

func TestParseConfigMissingFile(t *testing.T) {
	_, err := ParseConfig(filepath.Join(t.TempDir(), "nonexistent.cfg"))
	if err == nil {
		t.Fatalf("%s: ParseConfig should fail on missing file", testTag)
	}
}

func TestParseConfigEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.cfg")
	if err := os.WriteFile(path, []byte{}, 0o600); err != nil {
		t.Fatalf("%s: write: %v", testTag, err)
	}

	_, err := ParseConfig(path)
	if err == nil {
		t.Fatalf("%s: ParseConfig should fail on empty file", testTag)
	}
}

func TestParseConfigExtraFields(t *testing.T) {
	raw := `{"operator":"op","lhost":"1.2.3.4","lport":9999,"token":"t","unknown_field":"ignored"}`
	path := filepath.Join(t.TempDir(), "extra.cfg")
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatalf("%s: write: %v", testTag, err)
	}

	got, err := ParseConfig(path)
	if err != nil {
		t.Fatalf("%s: ParseConfig should not fail on extra fields: %v", testTag, err)
	}
	if got.Operator != "op" {
		t.Errorf("%s: Operator = %q, want %q", testTag, got.Operator, "op")
	}
	if got.LHost != "1.2.3.4" {
		t.Errorf("%s: LHost = %q, want %q", testTag, got.LHost, "1.2.3.4")
	}
}

// ---------------------------------------------------------------------------
// tokenAuth
// ---------------------------------------------------------------------------

func TestTokenAuthGetRequestMetadata(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  string
	}{
		{"simple", "abc123", "Bearer abc123"},
		{"empty", "", "Bearer "},
		{"long token", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.payload.sig", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.payload.sig"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := tokenAuth{token: tt.token}
			md, err := auth.GetRequestMetadata(context.Background())
			if err != nil {
				t.Fatalf("%s: GetRequestMetadata: %v", testTag, err)
			}
			got, ok := md["Authorization"]
			if !ok {
				t.Fatalf("%s: missing Authorization key in metadata", testTag)
			}
			if got != tt.want {
				t.Errorf("%s: Authorization = %q, want %q", testTag, got, tt.want)
			}
		})
	}
}

func TestTokenAuthGetRequestMetadataMapSize(t *testing.T) {
	auth := tokenAuth{token: "tok"}
	md, err := auth.GetRequestMetadata(context.Background(), "extra", "args")
	if err != nil {
		t.Fatalf("%s: GetRequestMetadata: %v", testTag, err)
	}
	if len(md) != 1 {
		t.Errorf("%s: metadata should have exactly 1 key, got %d", testTag, len(md))
	}
}

func TestTokenAuthRequireTransportSecurity(t *testing.T) {
	auth := tokenAuth{token: "any"}
	if !auth.RequireTransportSecurity() {
		t.Errorf("%s: RequireTransportSecurity() = false, want true", testTag)
	}
}

// ---------------------------------------------------------------------------
// Backend.Name
// ---------------------------------------------------------------------------

func TestBackendName(t *testing.T) {
	b := New()
	if b.Name() != "sliver" {
		t.Errorf("%s: Name() = %q, want %q", testTag, b.Name(), "sliver")
	}
}

// ---------------------------------------------------------------------------
// New
// ---------------------------------------------------------------------------

func TestNew(t *testing.T) {
	b := New()
	if b == nil {
		t.Fatalf("%s: New() returned nil", testTag)
	}
}

func TestNewZeroState(t *testing.T) {
	b := New()
	if b.client != nil {
		t.Errorf("%s: new backend should have nil client", testTag)
	}
	if b.conn != nil {
		t.Errorf("%s: new backend should have nil conn", testTag)
	}
	if b.configPath != "" {
		t.Errorf("%s: new backend should have empty configPath", testTag)
	}
	if b.lhost != "" {
		t.Errorf("%s: new backend should have empty lhost", testTag)
	}
	if b.lport != 0 {
		t.Errorf("%s: new backend should have zero lport", testTag)
	}
	if b.listenerID != 0 {
		t.Errorf("%s: new backend should have zero listenerID", testTag)
	}
	if b.stageSrv != nil {
		t.Errorf("%s: new backend should have nil stageSrv", testTag)
	}
	if b.tcpStageLn != nil {
		t.Errorf("%s: new backend should have nil tcpStageLn", testTag)
	}
}

// ---------------------------------------------------------------------------
// Configure
// ---------------------------------------------------------------------------

func TestConfigure(t *testing.T) {
	b := New()
	b.Configure("/path/to/operator.cfg")
	if b.configPath != "/path/to/operator.cfg" {
		t.Errorf("%s: configPath = %q, want %q", testTag, b.configPath, "/path/to/operator.cfg")
	}
}

func TestConfigureOverwrite(t *testing.T) {
	b := New()
	b.Configure("/first.cfg")
	b.Configure("/second.cfg")
	if b.configPath != "/second.cfg" {
		t.Errorf("%s: configPath = %q, want %q", testTag, b.configPath, "/second.cfg")
	}
}

// ---------------------------------------------------------------------------
// Shutdown (no active connections)
// ---------------------------------------------------------------------------

func TestShutdownClean(t *testing.T) {
	b := New()
	if err := b.Shutdown(); err != nil {
		t.Fatalf("%s: Shutdown on fresh backend should not error: %v", testTag, err)
	}
}
