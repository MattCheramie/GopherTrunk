package api

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

type fakeAutoRecord struct{ fired atomic.Int64 }

func (f *fakeAutoRecord) Trigger() { f.fired.Add(1) }

func newAutoRecordTestServer(t *testing.T, prov AutoRecordProvider) *httptest.Server {
	t.Helper()
	bus := events.NewBus(8)
	t.Cleanup(bus.Close)
	srv, err := NewServer(ServerOptions{
		Addr:           "127.0.0.1:0",
		Bus:            bus,
		AllowMutations: true,
		Siglab:         SiglabOptions{Enabled: true, TempDir: t.TempDir()},
		AutoRecord:     prov,
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	t.Cleanup(func() { _ = srv.Close() })
	ts := httptest.NewServer(srv.routes())
	t.Cleanup(ts.Close)
	return ts
}

func TestAutoRecordTriggerUnavailableWhenNotWired(t *testing.T) {
	ts := newSiglabTestServer(t) // no AutoRecord provider
	resp, err := http.Post(ts.URL+"/api/v1/siglab/autorecord/trigger", "application/json", nil)
	if err != nil {
		t.Fatalf("POST trigger: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", resp.StatusCode)
	}
}

func TestAutoRecordTriggerFires(t *testing.T) {
	prov := &fakeAutoRecord{}
	ts := newAutoRecordTestServer(t, prov)
	resp, err := http.Post(ts.URL+"/api/v1/siglab/autorecord/trigger", "application/json", nil)
	if err != nil {
		t.Fatalf("POST trigger: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", resp.StatusCode)
	}
	if prov.fired.Load() != 1 {
		t.Fatalf("Trigger called %d times, want 1", prov.fired.Load())
	}
}
