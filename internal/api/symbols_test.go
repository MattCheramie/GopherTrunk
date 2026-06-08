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
				SymI:         []float32{0.7, -0.7, -0.7, 0.7},
				SymQ:         []float32{0.7, 0.7, -0.7, -0.7},
				Dibits:       DibitArray{1, 0, 3, 2},
				BaseIdx:      0,
			},
			{
				TimestampNs:  2,
				SymbolRateHz: 4800,
				CenterHz:     851_012_500,
				Dibits:       DibitArray{2, 2},
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
		// Regression guard for the "waiting for symbols" bug: dibits must be
		// a JSON *number array* on the wire, not Go's default base64 string
		// for []byte/[]uint8 — the web client drops any frame whose dibits
		// isn't an array (Array.isArray). Inspect the raw JSON, not the
		// re-unmarshalled struct (Go unmarshal would accept either form).
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(msg, &raw); err != nil {
			t.Fatalf("raw unmarshal #%d: %v", i, err)
		}
		if got := string(raw["dibits"]); len(got) == 0 || got[0] != '[' {
			t.Errorf("frame #%d dibits wire form = %s, want a JSON array", i, got)
		}
		if i == 0 {
			var dibits []int
			if err := json.Unmarshal(raw["dibits"], &dibits); err != nil {
				t.Fatalf("dibits not an int array: %v (raw=%s)", err, raw["dibits"])
			}
			want := []int{1, 0, 3, 2}
			for k, v := range want {
				if dibits[k] != v {
					t.Errorf("dibits = %v, want %v", dibits, want)
					break
				}
			}
		}
		if i == 0 {
			if len(f.SymI) != 4 || len(f.SymQ) != 4 {
				t.Errorf("frame #0 symbol track len = (%d,%d), want (4,4)", len(f.SymI), len(f.SymQ))
			} else if f.SymI[0] != 0.7 || f.SymQ[2] != -0.7 {
				t.Errorf("frame #0 symbol track not round-tripped: SymI[0]=%v SymQ[2]=%v", f.SymI[0], f.SymQ[2])
			}
		}
	}

	if prov.gotPro != "p25-cqpsk" {
		t.Errorf("provider got proto %q, want p25-cqpsk", prov.gotPro)
	}
	if prov.gotOff != 12_500 {
		t.Errorf("provider got offset %d, want 12500", prov.gotOff)
	}
}
