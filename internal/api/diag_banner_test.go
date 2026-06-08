package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	gtdiag "github.com/MattCheramie/GopherTrunk/internal/diag"
	"github.com/MattCheramie/GopherTrunk/internal/events"
)

func newDiagBannerTestServer(t *testing.T, verbose bool) *httptest.Server {
	t.Helper()
	t.Setenv(gtdiag.QuietBannerEnv, "") // banner must render in these tests
	bus := events.NewBus(8)
	t.Cleanup(bus.Close)
	srv, err := NewServer(ServerOptions{
		Addr:          "127.0.0.1:0",
		Bus:           bus,
		Diagnostics:   gtdiag.NewCollectorWithDongles(nil, nil),
		VerboseErrors: verbose,
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	ts := httptest.NewServer(srv.routes())
	t.Cleanup(ts.Close)
	return ts
}

// decodeErrEnvelope reads a JSON error body into its fields.
func decodeErrEnvelope(t *testing.T, r io.Reader) (msg string, diag map[string]any) {
	t.Helper()
	var body struct {
		Error string         `json:"error"`
		Diag  map[string]any `json:"diag"`
	}
	if err := json.NewDecoder(r).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return body.Error, body.Diag
}

func TestErrorEnvelopeTerseWhenVerboseOff(t *testing.T) {
	ts := newDiagBannerTestServer(t, false)
	// An unknown path under a known prefix; pick a 404-producing route.
	resp, err := http.Get(ts.URL + "/api/v1/systems/does-not-exist")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	msg, diag := decodeErrEnvelope(t, resp.Body)
	if msg == "" {
		t.Errorf("expected an error message")
	}
	if diag != nil {
		t.Errorf("verbose off should omit diag block, got %v", diag)
	}
}

func TestErrorEnvelopeIncludesBannerWhenVerboseOn(t *testing.T) {
	ts := newDiagBannerTestServer(t, true)
	resp, err := http.Get(ts.URL + "/api/v1/systems/does-not-exist")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	_, diag := decodeErrEnvelope(t, resp.Body)
	if diag == nil {
		t.Fatalf("verbose on should include diag block")
	}
	if banner, _ := diag["banner"].(string); banner == "" {
		t.Errorf("diag block missing banner: %v", diag)
	}
}

func TestDiagBannerEndpointGating(t *testing.T) {
	// Verbose off: bare GET is forbidden, ?verbose=1 (loopback, auth
	// auto) is allowed.
	ts := newDiagBannerTestServer(t, false)
	resp, err := http.Get(ts.URL + "/api/v1/diag/banner")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("bare banner GET status = %d, want 403", resp.StatusCode)
	}

	resp2, err := http.Get(ts.URL + "/api/v1/diag/banner?verbose=1")
	if err != nil {
		t.Fatalf("GET verbose: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("?verbose=1 banner status = %d, want 200", resp2.StatusCode)
	}
	var body struct {
		Banner string `json:"banner"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Banner == "" {
		t.Errorf("expected a banner string")
	}
}
