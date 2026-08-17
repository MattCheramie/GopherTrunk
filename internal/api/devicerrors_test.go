package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// staleSymbolProvider stands in for the daemon after a restart with different
// hardware: the serial the browser still holds is not in the broker map.
type staleSymbolProvider struct{ opened int }

func (p *staleSymbolProvider) OpenSymbolStream(context.Context, string, string, int32) (<-chan SymbolFrame, func(), error) {
	p.opened++
	return nil, nil, ErrUnknownDevice
}

// TestDiagnosticStreamsRejectUnknownDeviceBeforeUpgrade is the failing-first
// regression for the reconnect storm an operator hit by leaving a browser tab
// open across a daemon restart that changed SDRs.
//
// The handlers used to upgrade the WebSocket first and only then resolve the
// serial, so an unknown device produced a 101 handshake followed immediately by
// a 1011 close. Browser clients reset their reconnect backoff in `onopen`, so a
// handshake that always succeeds means the backoff never grows — the tab
// reconnected two to four times a second indefinitely and the daemon logged
// "OpenSymbolStream failed" each time. An HTTP status is something a client can
// actually back off from, and 404 is what it is.
func TestDiagnosticStreamsRejectUnknownDeviceBeforeUpgrade(t *testing.T) {
	bus := events.NewBus(8)
	t.Cleanup(bus.Close)
	symbols := &staleSymbolProvider{}
	srv, err := NewServer(ServerOptions{
		Addr:     "127.0.0.1:0",
		Bus:      bus,
		Symbols:  symbols,
		Spectrum: &fakeSpectrumProvider{devices: []SpectrumDevice{{Serial: "live-sdr"}}},
		Diag:     &fakeDiagProvider{},
		Mixer:    &fakeMixerProvider{},
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	ts := httptest.NewServer(srv.routes())
	t.Cleanup(ts.Close)

	// The real stale serial from the field report, spaces and colon included.
	const stale = "AIRSPY SN:256863C82B25748B"
	q := url.Values{"device": {stale}}.Encode()
	for _, path := range []string{
		"/api/v1/diag/symbols?" + q,
		"/api/v1/spectrum/stream?" + q,
		"/api/v1/diag/iq?" + q,
		"/api/v1/diag/mixer?" + q,
	} {
		// A plain GET (no Upgrade headers) must be answered, not upgraded.
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		body := make([]byte, 512)
		n, _ := resp.Body.Read(body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s: status = %d, want 404", path, resp.StatusCode)
		}
		if !strings.Contains(string(body[:n]), stale) {
			t.Errorf("%s: body %q does not name the unknown serial", path, string(body[:n]))
		}
	}

	// And the provider is never reached: the request is rejected on the serial
	// alone, so no per-request DSP engine is built for a device that is gone.
	if symbols.opened != 0 {
		t.Errorf("symbol provider opened %d times for an unknown device, want 0", symbols.opened)
	}
}

// TestKnownDeviceStillUpgrades guards the check against over-reach: a serial the
// daemon really has must pass through to the provider as before.
func TestKnownDeviceStillUpgrades(t *testing.T) {
	bus := events.NewBus(8)
	t.Cleanup(bus.Close)
	symbols := &staleSymbolProvider{}
	srv, err := NewServer(ServerOptions{
		Addr:     "127.0.0.1:0",
		Bus:      bus,
		Symbols:  symbols,
		Spectrum: &fakeSpectrumProvider{devices: []SpectrumDevice{{Serial: "live-sdr"}}},
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	ts := httptest.NewServer(srv.routes())
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/api/v1/diag/symbols?device=live-sdr")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		t.Error("a known serial was rejected as unknown")
	}
}
