package api

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// TestServerServesTLSWhenConfigured spins a self-signed cert/key
// pair, hands the on-disk paths to ServerOptions.TLSCert/TLSKey,
// and confirms a TLS client can reach /api/v1/health while a plain
// HTTP client to the same port fails. Covers the
// http.Server.ServeTLS branch added by PR-J.
func TestServerServesTLSWhenConfigured(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := writeSelfSignedCert(t, dir, "127.0.0.1")

	bus := events.NewBus(8)
	defer bus.Close()

	port := freeTCPPort(t)
	addr := "127.0.0.1:" + port

	srv, err := NewServer(ServerOptions{
		Addr:    addr,
		Bus:     bus,
		Version: "tls-test",
		TLSCert: certPath,
		TLSKey:  keyPath,
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- srv.Run(t.Context()) }()
	defer srv.Close()

	// Probe loop — TLS server takes a moment to bind.
	tlsURL := "https://" + addr + "/api/v1/health"
	insecureClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // self-signed cert in test
		},
		Timeout: 2 * time.Second,
	}
	deadline := time.Now().Add(3 * time.Second)
	var resp *http.Response
	for {
		resp, err = insecureClient.Get(tlsURL)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("TLS probe never succeeded: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	defer resp.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode TLS body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("TLS health body status = %v, want ok", body["status"])
	}
	if body["version"] != "tls-test" {
		t.Errorf("TLS health body version = %v, want tls-test", body["version"])
	}

	// A plain HTTP GET to the same port should fail at the TLS
	// handshake (Go's net/http server sends back a "Bad Request"
	// response with `Client sent an HTTP request to an HTTPS
	// server`). The 200 OK from the TLS path above confirmed the
	// real handler works; here we just confirm we don't get the
	// same 200 from a plain GET.
	plainClient := &http.Client{Timeout: 500 * time.Millisecond}
	if resp2, err := plainClient.Get("http://" + addr + "/api/v1/health"); err == nil {
		defer resp2.Body.Close()
		if resp2.StatusCode == http.StatusOK {
			t.Errorf("plain HTTP probe to TLS port returned 200; expected non-200 or handshake error")
		}
	}
}

// TestNewServerRejectsHalfTLS asserts the all-or-nothing validation:
// setting TLSCert without TLSKey (or vice versa) is a config error,
// not a silent fall-back to plain HTTP.
func TestNewServerRejectsHalfTLS(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()

	if _, err := NewServer(ServerOptions{
		Addr: "127.0.0.1:0", Bus: bus, TLSCert: "/tmp/cert.pem",
	}); err == nil {
		t.Error("NewServer accepted TLSCert without TLSKey")
	}
	if _, err := NewServer(ServerOptions{
		Addr: "127.0.0.1:0", Bus: bus, TLSKey: "/tmp/key.pem",
	}); err == nil {
		t.Error("NewServer accepted TLSKey without TLSCert")
	}
}

// writeSelfSignedCert produces a fresh ECDSA P-256 cert + key pair
// valid for `host` and writes them as PEM files under dir. Returns
// the (certPath, keyPath) pair.
func writeSelfSignedCert(t *testing.T, dir, host string) (string, string) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: host},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	if ip := net.ParseIP(host); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{host}
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")
	cf, err := os.Create(certPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := pem.Encode(cf, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		t.Fatal(err)
	}
	cf.Close()
	kf, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	if err := pem.Encode(kf, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}); err != nil {
		t.Fatal(err)
	}
	kf.Close()
	return certPath, keyPath
}

// freeTCPPort returns a port that's available at call time. Inherent
// TOCTOU vs. the test binding — caller should bind immediately.
func freeTCPPort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().(*net.TCPAddr)
	l.Close()
	return strconv.Itoa(addr.Port)
}
