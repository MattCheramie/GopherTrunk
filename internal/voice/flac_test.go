package voice

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// speechLikeSamples builds a deterministic multi-tone signal at the given
// amplitude — long enough to span multiple FLAC blocks and loud enough for
// the loudness meter to measure.
func speechLikeSamples(n int, amp float64) []int16 {
	out := make([]int16, n)
	for i := range out {
		v := amp * (0.6*math.Sin(2*math.Pi*300*float64(i)/8000) +
			0.4*math.Sin(2*math.Pi*1100*float64(i)/8000))
		out[i] = int16(math.Round(v * 32767))
	}
	return out
}

// TestFlacWriterRoundTrip pins the mono voice FLAC path: WriteSamples →
// Close → ReadFLACSamples must return bit-identical PCM at the header rate,
// with DataBytes reporting the uncompressed payload like WavWriter does.
func TestFlacWriterRoundTrip(t *testing.T) {
	samples := speechLikeSamples(10_000, 0.4) // > 2 blocks of 4096
	path := filepath.Join(t.TempDir(), "call.flac")
	w, err := NewFlacFile(path, 8000)
	if err != nil {
		t.Fatalf("NewFlacFile: %v", err)
	}
	// Two writes to exercise the block boundary carry.
	if err := w.WriteSamples(samples[:3000]); err != nil {
		t.Fatalf("WriteSamples: %v", err)
	}
	if err := w.WriteSamples(samples[3000:]); err != nil {
		t.Fatalf("WriteSamples: %v", err)
	}
	if got, want := w.DataBytes(), uint32(2*len(samples)); got != want {
		t.Errorf("DataBytes = %d, want %d", got, want)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if got := w.DataBytes(); got != uint32(2*len(samples)) {
		t.Errorf("DataBytes after Close = %d, want %d", got, 2*len(samples))
	}

	got, rate, err := ReadFLACSamples(path)
	if err != nil {
		t.Fatalf("ReadFLACSamples: %v", err)
	}
	if rate != 8000 {
		t.Errorf("rate = %d, want 8000", rate)
	}
	if len(got) != len(samples) {
		t.Fatalf("decoded %d samples, want %d", len(got), len(samples))
	}
	for i := range samples {
		if got[i] != samples[i] {
			t.Fatalf("sample %d = %d, want %d", i, got[i], samples[i])
		}
	}

	// A speech-band signal must actually compress vs the WAV equivalent.
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Size() >= int64(44+2*len(samples)) {
		t.Errorf("flac size %d is not smaller than the %d-byte wav equivalent", fi.Size(), 44+2*len(samples))
	}
}

// TestReadAudioSamplesSniffsContainer proves the format-agnostic reader picks
// the decoder from the file content, not the extension.
func TestReadAudioSamplesSniffsContainer(t *testing.T) {
	dir := t.TempDir()
	samples := speechLikeSamples(4000, 0.3)

	// A flac body under a lying .wav name still decodes as flac.
	lyingPath := filepath.Join(dir, "call.wav")
	fw, err := NewFlacFile(lyingPath, 8000)
	if err != nil {
		t.Fatal(err)
	}
	if err := fw.WriteSamples(samples); err != nil {
		t.Fatal(err)
	}
	if err := fw.Close(); err != nil {
		t.Fatal(err)
	}
	got, rate, err := ReadAudioSamples(lyingPath)
	if err != nil {
		t.Fatalf("ReadAudioSamples(flac body): %v", err)
	}
	if rate != 8000 || len(got) != len(samples) {
		t.Fatalf("flac body: %d samples @ %d Hz, want %d @ 8000", len(got), rate, len(samples))
	}

	wavPath := filepath.Join(dir, "call2.wav")
	ww, err := NewWavFile(wavPath, 8000)
	if err != nil {
		t.Fatal(err)
	}
	if err := ww.WriteSamples(samples); err != nil {
		t.Fatal(err)
	}
	if err := ww.Close(); err != nil {
		t.Fatal(err)
	}
	got, rate, err = ReadAudioSamples(wavPath)
	if err != nil {
		t.Fatalf("ReadAudioSamples(wav): %v", err)
	}
	if rate != 8000 || len(got) != len(samples) {
		t.Fatalf("wav: %d samples @ %d Hz, want %d @ 8000", len(got), rate, len(samples))
	}
}

// TestNormalizeFLACFileInPlace pins normalization on a flac recording: the
// quiet call is boosted and rewritten in place AS FLAC (content-sniffed), so
// recordings.format: flac keeps the whole normalize → web-playback chain
// intact.
func TestNormalizeFLACFileInPlace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quiet.flac")
	w, err := NewFlacFile(path, 8000)
	if err != nil {
		t.Fatal(err)
	}
	orig := speechLikeSamples(3*8000, 0.05) // quiet: needs boost
	if err := w.WriteSamples(orig); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	if err := NormalizeWAVFile(path, NormalizeConfig{Enabled: true}); err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if !isFLACAudioFile(path) {
		t.Fatal("normalized file is no longer FLAC — the rewrite changed container")
	}
	got, rate, err := ReadFLACSamples(path)
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	if rate != 8000 || len(got) != len(orig) {
		t.Fatalf("normalized: %d samples @ %d Hz, want %d @ 8000", len(got), rate, len(orig))
	}
	// The boost must have actually been applied.
	var origPeak, gotPeak int16
	for i := range orig {
		if orig[i] > origPeak {
			origPeak = orig[i]
		}
		if got[i] > gotPeak {
			gotPeak = got[i]
		}
	}
	if gotPeak <= origPeak {
		t.Errorf("peak after normalize = %d, want > %d (gain applied)", gotPeak, origPeak)
	}
}

// TestRecorderWritesPerCallFLAC is the recorder end-to-end pin for
// recordings.format: flac — the per-call file lands with a .flac extension,
// is a real FLAC stream, and decodes to the written PCM.
func TestRecorderWritesPerCallFLAC(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus:        bus,
		OutDir:     dir,
		Format:     "flac",
		SampleRate: 8000,
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
			System:   "TestSystem",
			Protocol: "p25",
			GroupID:  1234,
			SourceID: 56789,
		},
		Talkgroup:    &trunking.TalkGroup{ID: 1234, AlphaTag: "FIRE-DISP", Record: true},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 5, 5, 12, 30, 45, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if r.HasSession("VOICE-1") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	pcm := speechLikeSamples(1600, 0.3)
	if err := r.WritePCM("VOICE-1", pcm); err != nil {
		t.Fatal(err)
	}

	end := trunking.CallEnd{
		Grant:        cs.Grant,
		Talkgroup:    cs.Talkgroup,
		DeviceSerial: "VOICE-1",
		StartedAt:    cs.StartedAt,
		EndedAt:      cs.StartedAt.Add(2 * time.Second),
		Reason:       trunking.EndReasonNormal,
	}
	bus.Publish(events.Event{Kind: events.KindCallEnd, Payload: end})

	deadline = time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if !r.HasSession("VOICE-1") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	want := filepath.Join(dir, "TestSystem", "FIRE-DISP", "20260505_123045_1234.flac")
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("expected flac at %s: %v", want, err)
	}
	got, rate, err := ReadFLACSamples(want)
	if err != nil {
		t.Fatalf("decode recording: %v", err)
	}
	if rate != 8000 {
		t.Errorf("recording rate = %d, want 8000", rate)
	}
	if len(got) != len(pcm) {
		t.Fatalf("recording holds %d samples, want %d", len(got), len(pcm))
	}
	for i := range pcm {
		if got[i] != pcm[i] {
			t.Fatalf("sample %d = %d, want %d", i, got[i], pcm[i])
		}
	}
}
