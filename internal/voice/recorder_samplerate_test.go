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

// fakeVocoder is a minimal Vocoder that emits one 20 ms / 160-sample
// frame at the vocoder-native 8 kHz rate, regardless of input. It lets
// the recorder tests exercise the decoded-WAV path without linking the
// patented IMBE/AMBE decoders into the test binary.
type fakeVocoder struct{}

func (fakeVocoder) Name() string                   { return "fake-voc" }
func (fakeVocoder) FrameSize() int                 { return 11 }
func (fakeVocoder) Decode([]byte) ([]int16, error) { return make([]int16, 160), nil }
func (fakeVocoder) Reset()                         {}
func (fakeVocoder) Close() error                   { return nil }

func waitForSession(t *testing.T, r *Recorder, serial string, want bool) {
	t.Helper()
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if r.HasSession(serial) == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("session %q HasSession != %v after timeout", serial, want)
}

func wavHeaderRate(t *testing.T, path string) uint32 {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read wav %s: %v", path, err)
	}
	if len(b) < wavHeaderSize {
		t.Fatalf("wav %s too short: %d bytes", path, len(b))
	}
	return binary.LittleEndian.Uint32(b[24:28])
}

// TestRecorderForcesVocoderNativeRate verifies that a call whose protocol
// maps to a vocoder gets a WAV header at the vocoder-native 8 kHz even
// when recordings.sample_rate is configured to something else. Without
// this guard the recorder wrote the vocoder's 8 kHz PCM under a 48 kHz
// header, so clean audio played back 6x too fast (garbled) — issue #356.
func TestRecorderForcesVocoderNativeRate(t *testing.T) {
	DefaultRegistry.Register("fake-voc", func() (Vocoder, error) { return fakeVocoder{}, nil })

	bus := events.NewBus(8)
	defer bus.Close()
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus:                bus,
		OutDir:             dir,
		SampleRate:         48000, // intentionally != 8000
		VocoderForProtocol: map[string]string{"p25-fake": "fake-voc"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant: trunking.Grant{
			System:   "S",
			Protocol: "p25-fake",
			GroupID:  7,
			SourceID: 42,
		},
		Talkgroup:    &trunking.TalkGroup{ID: 7, AlphaTag: "FIRE", Record: true},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 5, 29, 17, 25, 2, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	waitForSession(t, r, "VOICE-1", true)

	if err := r.WriteRawFrame("VOICE-1", make([]byte, 11)); err != nil {
		t.Fatal(err)
	}

	bus.Publish(events.Event{Kind: events.KindCallEnd, Payload: trunking.CallEnd{
		Grant:        cs.Grant,
		Talkgroup:    cs.Talkgroup,
		DeviceSerial: "VOICE-1",
		StartedAt:    cs.StartedAt,
		EndedAt:      cs.StartedAt.Add(time.Second),
		Reason:       trunking.EndReasonNormal,
	}})
	waitForSession(t, r, "VOICE-1", false)

	wav := filepath.Join(dir, "S", "FIRE", "20260529T172502Z_src42.wav")
	if got := wavHeaderRate(t, wav); got != pcmHzDefault {
		t.Errorf("decoded-call WAV header rate = %d, want %d", got, pcmHzDefault)
	}
}

// TestRecorderAnalogKeepsConfiguredRate verifies the complement: a call
// with no vocoder mapping (analog/NBFM fed via WritePCM) keeps the
// configured recordings.sample_rate in its WAV header.
func TestRecorderAnalogKeepsConfiguredRate(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus:                bus,
		OutDir:             dir,
		SampleRate:         16000,
		VocoderForProtocol: map[string]string{}, // no vocoder for any protocol
	})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant:        trunking.Grant{System: "S", Protocol: "nbfm", GroupID: 9, SourceID: 1},
		Talkgroup:    &trunking.TalkGroup{ID: 9, AlphaTag: "ANALOG", Record: true},
		DeviceSerial: "VOICE-2",
		StartedAt:    time.Date(2026, 5, 29, 17, 30, 12, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	waitForSession(t, r, "VOICE-2", true)

	if err := r.WritePCM("VOICE-2", make([]int16, 320)); err != nil {
		t.Fatal(err)
	}

	bus.Publish(events.Event{Kind: events.KindCallEnd, Payload: trunking.CallEnd{
		Grant:        cs.Grant,
		Talkgroup:    cs.Talkgroup,
		DeviceSerial: "VOICE-2",
		StartedAt:    cs.StartedAt,
		EndedAt:      cs.StartedAt.Add(time.Second),
		Reason:       trunking.EndReasonNormal,
	}})
	waitForSession(t, r, "VOICE-2", false)

	wav := filepath.Join(dir, "S", "ANALOG", "20260529T173012Z_src1.wav")
	if got := wavHeaderRate(t, wav); got != 16000 {
		t.Errorf("analog WAV header rate = %d, want 16000", got)
	}
}
