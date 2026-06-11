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

type fakeMixerProvider struct {
	openErr error
	frames  []MixerFrame
	gotPro  string
	gotOff  int32
}

func (f *fakeMixerProvider) OpenMixerStream(ctx context.Context, _ string, proto string, offset int32) (<-chan MixerFrame, func(), error) {
	f.gotPro = proto
	f.gotOff = offset
	if f.openErr != nil {
		return nil, nil, f.openErr
	}
	out := make(chan MixerFrame, 4)
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

func newMixerTestServer(t *testing.T, prov MixerProvider) *httptest.Server {
	t.Helper()
	bus := events.NewBus(8)
	t.Cleanup(bus.Close)
	srv, err := NewServer(ServerOptions{
		Addr:  "127.0.0.1:0",
		Bus:   bus,
		Mixer: prov,
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	ts := httptest.NewServer(srv.routes())
	t.Cleanup(ts.Close)
	return ts
}

func TestMixerStreamReturns503WhenNotWired(t *testing.T) {
	ts := newMixerTestServer(t, nil)
	resp, err := http.Get(ts.URL + "/api/v1/diag/mixer?device=foo")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", resp.StatusCode)
	}
}

func TestMixerStreamRejectsMissingDevice(t *testing.T) {
	ts := newMixerTestServer(t, &fakeMixerProvider{})
	resp, err := http.Get(ts.URL + "/api/v1/diag/mixer")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestMixerStreamRejectsUnknownProto(t *testing.T) {
	ts := newMixerTestServer(t, &fakeMixerProvider{})
	resp, err := http.Get(ts.URL + "/api/v1/diag/mixer?device=foo&proto=bogus")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for unknown proto", resp.StatusCode)
	}
}

func TestMixerStreamDeliversFrames(t *testing.T) {
	prov := &fakeMixerProvider{
		frames: []MixerFrame{
			{
				TimestampNs:     1,
				CenterHz:        851_012_500,
				SampleRateHz:    48_000,
				OffsetHz:        12_500,
				CarrierOffsetHz: -120.5,
				RawBins:         []float32{-90, -40, -10, -45},
				TunedBins:       []float32{-92, -50, -8, -51},
			},
			{
				TimestampNs:  2,
				CenterHz:     851_012_500,
				SampleRateHz: 48_000,
				RawBins:      []float32{-88, -42},
				TunedBins:    []float32{-89, -43},
			},
		},
	}
	ts := newMixerTestServer(t, prov)

	u, _ := url.Parse(ts.URL)
	wsURL := "ws://" + u.Host + "/api/v1/diag/mixer?device=any&proto=p25-cqpsk&offset=12500"
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
		var f MixerFrame
		if err := json.Unmarshal(msg, &f); err != nil {
			t.Fatalf("unmarshal #%d: %v (raw=%s)", i, err, string(msg))
		}
		if f.SampleRateHz != 48_000 {
			t.Errorf("frame #%d SampleRateHz = %d, want 48000", i, f.SampleRateHz)
		}
		if len(f.RawBins) == 0 || len(f.TunedBins) == 0 {
			t.Errorf("frame #%d empty bins: raw=%d tuned=%d", i, len(f.RawBins), len(f.TunedBins))
		}
		if i == 0 {
			if len(f.RawBins) != 4 || f.RawBins[2] != -10 {
				t.Errorf("frame #0 raw bins not round-tripped: %v", f.RawBins)
			}
			if f.CarrierOffsetHz != -120.5 {
				t.Errorf("frame #0 carrier offset = %v, want -120.5", f.CarrierOffsetHz)
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
