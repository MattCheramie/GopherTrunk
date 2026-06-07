package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/gorilla/websocket"
)

type fakeSymbolProvider struct {
	openErr error
	frames  []SymbolFrame
	gotPro  string
	gotOff  int32
}

func (f *fakeSymbolProvider) OpenSymbolStream(ctx context.Context, _ string, proto string, offset int32) (<-chan SymbolFrame, func(), error) {
	f.gotPro = proto
	f.gotOff = offset
	if f.openErr != nil {
		return nil, nil, f.openErr
	}
	out := make(chan SymbolFrame, 4)
	streamCtx, cancel := context.WithCancel(ctx)
	go func() {
		defer close(out)
		for _, fr := range f.frames {
			select {
			case <-streamCtx.Done():
				return
			case out <- fr:
			}
			time.Sleep(5 * time.Millisecond)
		}
		<-streamCtx.Done()
	}()
	return out, cancel, nil
}

func newSymbolTestServer(t *testing.T, prov SymbolProvider) *httptest.Server {
	t.Helper()
	bus := events.NewBus(8)
	t.Cleanup(bus.Close)
	srv, err := NewServer(ServerOptions{
		Addr:    "127.0.0.1:0",
		Bus:     bus,
		Symbols: prov,
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	ts := httptest.NewServer(srv.routes())
	t.Cleanup(ts.Close)
	return ts
}

func TestSymbolStreamReturns503WhenNotWired(t *testing.T) {
	ts := newSymbolTestServer(t, nil)
	resp, err := http.Get(ts.URL + "/api/v1/diag/symbols?device=foo")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", resp.StatusCode)
	}
}

func TestSymbolStreamRejectsMissingDevice(t *testing.T) {
	ts := newSymbolTestServer(t, &fakeSymbolProvider{})
	resp, err := http.Get(ts.URL + "/api/v1/diag/symbols")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestSymbolStreamRejectsUnknownProto(t *testing.T) {
	ts := newSymbolTestServer(t, &fakeSymbolProvider{})
	resp, err := http.Get(ts.URL + "/api/v1/diag/symbols?device=foo&proto=bogus")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for unknown proto", resp.StatusCode)
	}
}

func TestSymbolStreamDeliversFrames(t *testing.T) {
	prov := &fakeSymbolProvider{
		frames: []SymbolFrame{
			{
				TimestampNs:  1,
				SymbolRateHz: 4800,
				CenterHz:     851_012_500,
				OffsetHz:     12_500,
				Soft:         []float32{0.9, -0.3, -0.95, 0.31},
				Dibits:       []uint8{1, 0, 3, 2},
				BaseIdx:      0,
			},
			{
				TimestampNs:  2,
				SymbolRateHz: 4800,
				CenterHz:     851_012_500,
				Dibits:       []uint8{2, 2},
				BaseIdx:      4,
			},
		},
	}
	ts := newSymbolTestServer(t, prov)

	u, _ := url.Parse(ts.URL)
	wsURL := "ws://" + u.Host + "/api/v1/diag/symbols?device=any&proto=p25-cqpsk&offset=12500"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	for i := 0; i < 2; i++ {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("ReadMessage #%d: %v", i, err)
		}
		var f SymbolFrame
		if err := json.Unmarshal(msg, &f); err != nil {
			t.Fatalf("unmarshal #%d: %v (raw=%s)", i, err, string(msg))
		}
		if f.SymbolRateHz != 4800 {
			t.Errorf("frame #%d SymbolRateHz = %v", i, f.SymbolRateHz)
		}
		if i == 0 && len(f.Dibits) != 4 {
			t.Errorf("frame #%d dibits len = %d, want 4", i, len(f.Dibits))
		}
	}

	if prov.gotPro != "p25-cqpsk" {
		t.Errorf("provider got proto %q, want p25-cqpsk", prov.gotPro)
	}
	if prov.gotOff != 12_500 {
		t.Errorf("provider got offset %d, want 12500", prov.gotOff)
	}
}
