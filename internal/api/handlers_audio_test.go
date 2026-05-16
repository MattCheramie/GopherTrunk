package api

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// fakeAudio implements AudioController for tests. Records SetVolume /
// SetMuted / SetRecordingEnabled calls so assertions can confirm
// the PATCH handler routed each knob correctly.
type fakeAudio struct {
	volume   atomic.Uint32 // float32 bits
	muted    atomic.Bool
	rec      atomic.Bool
	drops    uint64
	rate     uint32
	backend  bool
	setVolN  atomic.Int32
	setMuteN atomic.Int32
	setRecN  atomic.Int32
}

func newFakeAudio() *fakeAudio {
	f := &fakeAudio{rate: 8000, backend: true}
	f.volume.Store(0x3F400000) // 0.75 in float32 bits
	f.rec.Store(true)
	return f
}

func (f *fakeAudio) Volume() float32 {
	return float32frombits32(f.volume.Load())
}

func (f *fakeAudio) SetVolume(v float32) {
	f.volume.Store(float32bits32(v))
	f.setVolN.Add(1)
}

func (f *fakeAudio) Muted() bool { return f.muted.Load() }

func (f *fakeAudio) SetMuted(m bool) {
	f.muted.Store(m)
	f.setMuteN.Add(1)
}

func (f *fakeAudio) RecordingEnabled() bool { return f.rec.Load() }

func (f *fakeAudio) SetRecordingEnabled(enabled bool) {
	f.rec.Store(enabled)
	f.setRecN.Add(1)
}

func (f *fakeAudio) DropsTotal() uint64   { return f.drops }
func (f *fakeAudio) SampleRate() uint32   { return f.rate }
func (f *fakeAudio) BackendEnabled() bool { return f.backend }

func float32bits32(f float32) uint32     { return math.Float32bits(f) }
func float32frombits32(u uint32) float32 { return math.Float32frombits(u) }

func TestAudioStatus_OK(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	fa := newFakeAudio()
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Audio: fa})
	defer teardown()

	resp, err := http.Get(base + "/api/v1/audio")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var body AudioStatusDTO
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !body.BackendEnabled {
		t.Errorf("backend_enabled = false")
	}
	if body.SampleRate != 8000 {
		t.Errorf("sample_rate=%d", body.SampleRate)
	}
	if body.Volume < 0.74 || body.Volume > 0.76 {
		t.Errorf("volume=%f, want ~0.75", body.Volume)
	}
	if !body.RecordingEnabled {
		t.Errorf("recording_enabled = false")
	}
}

func TestAudioStatus_NotWired(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{Bus: bus})
	defer teardown()

	resp, err := http.Get(base + "/api/v1/audio")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 503 {
		t.Errorf("status=%d, want 503", resp.StatusCode)
	}
}

func TestAudioPatch_AppliesAllFields(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	fa := newFakeAudio()
	base, teardown := mkServer(t, ServerOptions{
		Bus:            bus,
		AllowMutations: true,
		Audio:          fa,
	})
	defer teardown()

	body := `{"volume":0.5,"muted":true,"recording_enabled":false}`
	req, _ := http.NewRequest(http.MethodPatch, base+"/api/v1/audio", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if fa.setVolN.Load() != 1 || fa.setMuteN.Load() != 1 || fa.setRecN.Load() != 1 {
		t.Errorf("expected one SetVolume / SetMuted / SetRecordingEnabled call each, got %d/%d/%d",
			fa.setVolN.Load(), fa.setMuteN.Load(), fa.setRecN.Load())
	}
	if !fa.Muted() {
		t.Errorf("muted not applied")
	}
	if fa.RecordingEnabled() {
		t.Errorf("recording not disabled")
	}
}

// TestAudioPatch_PublishesSSEEvent verifies the PATCH handler emits
// an events.KindAudioState event on the bus so SSE subscribers
// converge instantly instead of waiting for the next poll. The
// emitted payload is the new state (same shape as the HTTP
// response).
func TestAudioPatch_PublishesSSEEvent(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	fa := newFakeAudio()
	base, teardown := mkServer(t, ServerOptions{
		Bus:            bus,
		AllowMutations: true,
		Audio:          fa,
	})
	defer teardown()

	body := `{"muted":true}`
	req, _ := http.NewRequest(http.MethodPatch, base+"/api/v1/audio", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}

	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindAudioState {
			t.Errorf("event kind = %q, want audio.state", ev.Kind)
		}
		state, ok := ev.Payload.(AudioStatusDTO)
		if !ok {
			t.Fatalf("payload type = %T, want AudioStatusDTO", ev.Payload)
		}
		if !state.Muted {
			t.Errorf("event payload muted = false, want true")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("no audio.state event published within 500 ms")
	}
}

func TestAudioPatch_RejectsOutOfRangeVolume(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	fa := newFakeAudio()
	base, teardown := mkServer(t, ServerOptions{
		Bus:            bus,
		AllowMutations: true,
		Audio:          fa,
	})
	defer teardown()

	body := `{"volume":2.0}`
	req, _ := http.NewRequest(http.MethodPatch, base+"/api/v1/audio", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Errorf("status=%d, want 400", resp.StatusCode)
	}
	if fa.setVolN.Load() != 0 {
		t.Errorf("SetVolume should not be called on rejected request")
	}
}

func TestAudioPatch_RejectsEmptyBody(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	fa := newFakeAudio()
	base, teardown := mkServer(t, ServerOptions{
		Bus:            bus,
		AllowMutations: true,
		Audio:          fa,
	})
	defer teardown()

	body := `{}`
	req, _ := http.NewRequest(http.MethodPatch, base+"/api/v1/audio", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Errorf("status=%d, want 400", resp.StatusCode)
	}
}

func TestAudioPatch_GatedByAuth(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	fa := newFakeAudio()
	base, teardown := mkServer(t, ServerOptions{
		Bus:   bus,
		Audio: fa,
		// Force token-required auth; loopback bypass doesn't apply.
		Auth: AuthConfig{Mode: AuthModeRequired, Token: "test-token"},
	})
	defer teardown()

	body := `{"muted":true}`
	req, _ := http.NewRequest(http.MethodPatch, base+"/api/v1/audio", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status=%d, want 401", resp.StatusCode)
	}
	if fa.setMuteN.Load() != 0 {
		t.Errorf("SetMuted called despite gate")
	}
}
