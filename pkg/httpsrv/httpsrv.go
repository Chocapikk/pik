package httpsrv

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Chocapikk/pik/sdk"
)

func init() { sdk.SetHTTPServerFunc(start) }

// start creates an HTTP server backed by the context's ServerMux.
// Binds on LHOST:SRVPORT. Supports optional self-signed TLS via SRVSSL.
func start(params sdk.Params, mux *sdk.ServerMux) (string, func(), error) {
	lhost := params.Lhost()
	port := params.IntOr("SRVPORT", 8080)
	useSSL := strings.EqualFold(params.GetOr("SRVSSL", "false"), "true")

	addr := fmt.Sprintf("%s:%d", lhost, port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return "", nil, err
	}

	if useSSL {
		tlsCfg, err := selfSignedTLS()
		if err != nil {
			ln.Close()
			return "", nil, fmt.Errorf("tls: %w", err)
		}
		ln = tls.NewListener(ln, tlsCfg)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct, body, ok := mux.Match(r.URL.Path)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", ct)
		w.Write(body) //nolint:errcheck
	})

	srv := &http.Server{
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}
	go srv.Serve(ln) //nolint:errcheck

	scheme := "http"
	if useSSL {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s:%d", scheme, lhost, port)
	return url, func() { srv.Close() }, nil
}

func selfSignedTLS() (*tls.Config, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	cert := tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}
	return &tls.Config{Certificates: []tls.Certificate{cert}}, nil
}
