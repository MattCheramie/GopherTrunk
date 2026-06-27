package voice

import (
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// loudVocoder decodes every frame to a constant, full-level tone so a call
// that ends mid-frame leaves a non-zero final sample — the abrupt cut that
// the end-of-call fade is meant to smooth.
type loudVocoder struct{}

func (loudVocoder) Name() string   { return "loud" }
func (loudVocoder) FrameSize() int { return 11 }
func (loudVocoder) Reset()         {}
func (loudVocoder) Close() error   { return nil }
func (loudVocoder) Decode([]byte) ([]int16, error) {
	s := make([]int16, 160)
	for i := range s {
		s[i] = 12000
	}
	return s, nil
}

func absI16(v int16) int16 {
	if v < 0 {
		return -v
	}
	return v
}

func readWAVSamples(t *testing.T, path string) []int16 {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read wav %s: %v", path, err)
	}
	if len(raw) < wavHeaderSize {
		t.Fatalf("wav %s shorter than header (%d bytes)", path, len(raw))
	}
	body := raw[wavHeaderSize:]
	out := make([]int16, len(body)/2)
	for i := range out {
		out[i] = int16(binary.LittleEndian.Uint16(body[i*2:]))
	}
	return out
}

// TestRecorderFadesDigitalTailToZero is the regression for the "trailing
// DJ" scratch on short digital transmissions: a call that stops at a
// non-zero decoded level must not end abruptly. finalizeLocked appends a
// short fade from the last decoded sample down toward zero. Fails before
// the fade is added (the WAV ends at the full 12000 level).
func TestRecorderFadesDigitalTailToZero(t *testing.T) {
	DefaultRegistry.Register("loud-voc", func() (Vocoder, error) { return loudVocoder{}, nil })

	bus := events.NewBus(8)
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus:                bus,
		OutDir:             dir,
		SampleRate:         8000,
		VocoderForProtocol: map[string]string{"test-loud": "loud-voc"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer bus.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant:        trunking.Grant{System: "S", Protocol: "test-loud", GroupID: 1, SourceID: 3},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) && !r.HasSession("VOICE-1") {
		time.Sleep(5 * time.Millisecond)
	}

	for i := 0; i < 3; i++ {
		if err := r.WriteRawFrame("VOICE-1", make([]byte, 11)); err != nil {
			t.Fatalf("WriteRawFrame: %v", err)
		}
	}

	bus.Publish(events.Event{Kind: events.KindCallEnd, Payload: trunking.CallEnd{
		Grant: cs.Grant, DeviceSerial: "VOICE-1",
		StartedAt: cs.StartedAt, EndedAt: cs.StartedAt.Add(time.Second),
		Reason: trunking.EndReasonNormal,
	}})
	deadline = time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) && r.HasSession("VOICE-1") {
		time.Sleep(5 * time.Millisecond)
	}

	samples := readWAVSamples(t, filepath.Join(dir, "S", "1", "20260505T000000Z_src3.wav"))
	if len(samples) < 200 {
		t.Fatalf("too few samples decoded: %d", len(samples))
	}
	// Body holds the constant decoded level.
	if got := samples[100]; got != 12000 {
		t.Fatalf("body sample = %d, want 12000", got)
	}
	// The recording must not end abruptly at the decoded level.
	if last := samples[len(samples)-1]; absI16(last) > 1000 {
		t.Fatalf("final sample = %d, want a faded value near 0 (no abrupt cut)", last)
	}
	// The appended tail descends monotonically from the body level.
	tail := samples[len(samples)-80:]
	for i := 1; i < len(tail); i++ {
		if absI16(tail[i]) > absI16(tail[i-1]) {
			t.Fatalf("tail not monotonically decreasing at %d: %d then %d",
				i, tail[i-1], tail[i])
		}
	}
}

// TestFadeTail covers the pure ramp helper directly.
func TestFadeTail(t *testing.T) {
	if fadeTail(0, 80) != nil {
		t.Error("fadeTail(0, …) should be nil — silence needs no fade")
	}
	if fadeTail(1000, 0) != nil {
		t.Error("fadeTail(…, 0) should be nil")
	}
	tail := fadeTail(8000, 80)
	if len(tail) != 80 {
		t.Fatalf("len = %d, want 80", len(tail))
	}
	if tail[0] != 8000 {
		t.Errorf("tail[0] = %d, want 8000 (continues from last sample)", tail[0])
	}
	for i := 1; i < len(tail); i++ {
		if absI16(tail[i]) > absI16(tail[i-1]) {
			t.Fatalf("not monotonically decreasing at %d", i)
		}
	}
	if absI16(tail[len(tail)-1]) > 200 {
		t.Errorf("tail end = %d, want near 0", tail[len(tail)-1])
	}
}
