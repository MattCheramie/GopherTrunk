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

type fakeDiagProvider struct {
	openErr error
	frames  []IQFrame
}

func (f *fakeDiagProvider) OpenIQStream(ctx context.Context, _ string, _ uint32, _ int32) (<-chan IQFrame, func(), error) {
	if f.openErr != nil {
		return nil, nil, f.openErr
	}
	out := make(chan IQFrame, 4)
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

func newDiagTestServer(t *testing.T, prov DiagProvider) *httptest.Server {
	t.Helper()
	bus := events.NewBus(8)
	t.Cleanup(bus.Close)
	srv, err := NewServer(ServerOptions{
		Addr: "127.0.0.1:0",
		Bus:  bus,
		Diag: prov,
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	ts := httptest.NewServer(srv.routes())
	t.Cleanup(ts.Close)
	return ts
}

func TestDiagStreamReturns503WhenNotWired(t *testing.T) {
	ts := newDiagTestServer(t, nil)
	resp, err := http.Get(ts.URL + "/api/v1/diag/iq?device=foo")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", resp.StatusCode)
	}
}

func TestDiagStreamRejectsMissingDevice(t *testing.T) {
	ts := newDiagTestServer(t, &fakeDiagProvider{})
	resp, err := http.Get(ts.URL + "/api/v1/diag/iq")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestDiagStreamDeliversFrames(t *testing.T) {
	prov := &fakeDiagProvider{
		frames: []IQFrame{
			{
				TimestampNs:  1,
				SampleRateHz: 2000,
				CenterHz:     851_012_500,
				Points:       []IQPoint{{I: 0.5, Q: 0.25}, {I: -0.3, Q: 0.7}},
				EnergyDBFS:   -10,
			},
			{
				TimestampNs:  2,
				SampleRateHz: 2000,
				CenterHz:     851_012_500,
				Points:       []IQPoint{{I: 0.1, Q: 0.1}},
				EnergyDBFS:   -25,
			},
		},
	}
	ts := newDiagTestServer(t, prov)

	u, _ := url.Parse(ts.URL)
	wsURL := "ws://" + u.Host + "/api/v1/diag/iq?device=any&rate=2000"
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
		var f IQFrame
		if err := json.Unmarshal(msg, &f); err != nil {
			t.Fatalf("unmarshal #%d: %v (raw=%s)", i, err, string(msg))
		}
		if f.CenterHz != 851_012_500 {
			t.Errorf("frame #%d CenterHz = %d", i, f.CenterHz)
		}
		if i == 0 && len(f.Points) != 2 {
			t.Errorf("frame #%d points len = %d, want 2", i, len(f.Points))
		}
	}
}
