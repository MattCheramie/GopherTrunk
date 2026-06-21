package voice

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/loudness"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func mkRecorder(t *testing.T, writeRaw bool) (*Recorder, *events.Bus, string) {
	t.Helper()
	bus := events.NewBus(8)
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus:        bus,
		OutDir:     dir,
		SampleRate: 8000,
		WriteRaw:   writeRaw,
	})
	if err != nil {
		t.Fatal(err)
	}
	return r, bus, dir
}

func TestRecorderWritesPerCallWav(t *testing.T) {
	r, bus, dir := mkRecorder(t, false)
	defer r.Close()
	defer bus.Close()

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

	// Wait for the recorder to open the session.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if r.HasSession("VOICE-1") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	if err := r.WritePCM("VOICE-1", make([]int16, 1600)); err != nil {
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

	// Wait for session to drain.
	deadline = time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if !r.HasSession("VOICE-1") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	want := filepath.Join(dir, "TestSystem", "FIRE-DISP", "20260505T123045Z_src56789.wav")
	st, err := os.Stat(want)
	if err != nil {
		t.Fatalf("expected wav at %s: %v", want, err)
	}
	if st.Size() < int64(wavHeaderSize+1600*2) {
		t.Errorf("wav size = %d, want at least %d", st.Size(), wavHeaderSize+1600*2)
	}
}

// TestRecorderTimeslotInWavName confirms a DMR Tier III carrier's two
// concurrent calls (same system + talkgroup, one per TDMA slot, served
// by two voice devices) land in the same directory as distinct WAVs
// tagged by slot — so the stamp+src basenames don't collide and each
// file is self-labelling.
func TestRecorderTimeslotInWavName(t *testing.T) {
	r, bus, dir := mkRecorder(t, false)
	defer r.Close()
	defer bus.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	started := time.Date(2026, 5, 5, 12, 30, 45, 0, time.UTC)
	tg := &trunking.TalkGroup{ID: 1234, AlphaTag: "FIRE-DISP", Record: true}
	// Same TG, same start second, same source — only the slot (and the
	// serving device) differ. Without the slot tag the basenames collide.
	for _, c := range []struct {
		serial string
		slot   uint8
	}{{"VOICE-1", 1}, {"VOICE-2", 2}} {
		cs := trunking.CallStart{
			Grant: trunking.Grant{
				System: "TestSystem", Protocol: "dmr-tier3",
				GroupID: 1234, SourceID: 56789, FrequencyHz: 460_000_000, Timeslot: c.slot,
			},
			Talkgroup:    tg,
			DeviceSerial: c.serial,
			StartedAt:    started,
		}
		bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
		deadline := time.Now().Add(500 * time.Millisecond)
		for time.Now().Before(deadline) {
			if r.HasSession(c.serial) {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		if err := r.WritePCM(c.serial, make([]int16, 1600)); err != nil {
			t.Fatal(err)
		}
		bus.Publish(events.Event{Kind: events.KindCallEnd, Payload: trunking.CallEnd{
			Grant: cs.Grant, Talkgroup: tg, DeviceSerial: c.serial,
			StartedAt: started, EndedAt: started.Add(time.Second), Reason: trunking.EndReasonNormal,
		}})
		deadline = time.Now().Add(500 * time.Millisecond)
		for time.Now().Before(deadline) {
			if !r.HasSession(c.serial) {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
	}

	for _, slot := range []int{1, 2} {
		want := filepath.Join(dir, "TestSystem", "FIRE-DISP",
			fmt.Sprintf("20260505T123045Z_freq460000000_src56789_ts%d.wav", slot))
		if _, err := os.Stat(want); err != nil {
			t.Errorf("expected per-slot wav at %s: %v", want, err)
		}
	}
}

// TestRecorderSkipsRecordFalseTalkgroup confirms a talkgroup flagged
// Record=false is followed (the CallStart is consumed) but no session
// — and therefore no .wav — is opened for it.
func TestRecorderSkipsRecordFalseTalkgroup(t *testing.T) {
	r, bus, _ := mkRecorder(t, false)
	defer r.Close()
	defer bus.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant:        trunking.Grant{System: "S", Protocol: "p25", GroupID: 500, SourceID: 1},
		Talkgroup:    &trunking.TalkGroup{ID: 500, AlphaTag: "PRIVATE", Record: false},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Now(),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})

	// Give the recorder time to (correctly) drop the CallStart.
	deadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		if r.HasSession("VOICE-1") {
			t.Fatal("recorder opened a session for a Record=false talkgroup")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// TestRecorderGateBlocksNewSessions confirms the runtime
// SetRecordingEnabled(false) gate stops handleStart from opening
// new .wav files while leaving in-flight sessions alone. Matches the
// TUI / API "press R to toggle recording" semantics.
func TestRecorderGateBlocksNewSessions(t *testing.T) {
	r, bus, dir := mkRecorder(t, false)
	defer r.Close()
	defer bus.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	r.SetRecordingEnabled(false)
	if r.RecordingEnabled() {
		t.Fatal("RecordingEnabled should report false after SetRecordingEnabled(false)")
	}

	cs := trunking.CallStart{
		Grant: trunking.Grant{
			System:   "Sys",
			Protocol: "p25",
			GroupID:  7,
			SourceID: 8,
		},
		Talkgroup:    &trunking.TalkGroup{ID: 7, AlphaTag: "GATE", Record: true},
		DeviceSerial: "VOICE-G",
		StartedAt:    time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	time.Sleep(50 * time.Millisecond)
	if r.HasSession("VOICE-G") {
		t.Fatal("recorder opened a session despite gate")
	}

	// Re-enable and confirm the next CallStart opens a session.
	r.SetRecordingEnabled(true)
	cs2 := cs
	cs2.DeviceSerial = "VOICE-G2"
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs2})
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if r.HasSession("VOICE-G2") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !r.HasSession("VOICE-G2") {
		t.Fatal("session not opened after gate re-enabled")
	}
	// dir is unused here but the helper drops a tempdir we should
	// not crash on.
	_ = dir
}

func TestRecorderWritePCMDropsWithoutSession(t *testing.T) {
	r, bus, _ := mkRecorder(t, false)
	defer r.Close()
	defer bus.Close()
	if err := r.WritePCM("UNKNOWN", []int16{1, 2, 3}); err != nil {
		t.Errorf("WritePCM with no session should drop silently, got %v", err)
	}
}

func TestRecorderRawFrameSidecar(t *testing.T) {
	r, bus, dir := mkRecorder(t, true)
	defer r.Close()
	defer bus.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant:        trunking.Grant{System: "S", GroupID: 99, SourceID: 7},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if r.HasSession("VOICE-1") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	frame := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42}
	if err := r.WriteRawFrame("VOICE-1", frame); err != nil {
		t.Fatal(err)
	}
	if err := r.WriteRawFrame("VOICE-1", frame); err != nil {
		t.Fatal(err)
	}
	bus.Publish(events.Event{
		Kind: events.KindCallEnd,
		Payload: trunking.CallEnd{
			Grant: cs.Grant, DeviceSerial: "VOICE-1",
			StartedAt: cs.StartedAt, EndedAt: cs.StartedAt.Add(time.Second),
			Reason: trunking.EndReasonNormal,
		},
	})

	deadline = time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if !r.HasSession("VOICE-1") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	rawPath := filepath.Join(dir, "S", "99", "20260505T000000Z_src7.raw")
	data, err := os.ReadFile(rawPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) != 2*len(frame) {
		t.Errorf("raw size = %d, want %d", len(data), 2*len(frame))
	}
}

func TestRecorderProVoiceForcesRawSidecar(t *testing.T) {
	// Even with the recorder's global WriteRaw flag off, an EDACS
	// ProVoice grant must open a .raw sidecar — that's the only useful
	// output for an unstreamable vocoder.
	r, bus, dir := mkRecorder(t, false)
	defer r.Close()
	defer bus.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant: trunking.Grant{
			System: "EDACS-Site", Protocol: "edacs", GroupID: 0x4321,
			SourceID: 12, ProVoice: true,
		},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 5, 5, 1, 2, 3, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if r.HasSession("VOICE-1") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	frame := []byte{0x11, 0x22, 0x33, 0x44}
	if err := r.WriteRawFrame("VOICE-1", frame); err != nil {
		t.Fatal(err)
	}

	bus.Publish(events.Event{
		Kind: events.KindCallEnd,
		Payload: trunking.CallEnd{
			Grant: cs.Grant, DeviceSerial: "VOICE-1",
			StartedAt: cs.StartedAt, EndedAt: cs.StartedAt.Add(time.Second),
			Reason: trunking.EndReasonNormal,
		},
	})
	deadline = time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if !r.HasSession("VOICE-1") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	rawPath := filepath.Join(dir, "EDACS-Site", "17185", "20260505T010203Z_src12.raw")
	data, err := os.ReadFile(rawPath)
	if err != nil {
		t.Fatalf("expected .raw sidecar at %s: %v", rawPath, err)
	}
	if len(data) != len(frame) {
		t.Errorf("raw size = %d, want %d", len(data), len(frame))
	}
}

func TestRecorderNonProVoiceSkipsRawWhenDisabled(t *testing.T) {
	// Sanity check: with WriteRaw=false and a non-ProVoice grant, the
	// recorder must not create a .raw sidecar.
	r, bus, dir := mkRecorder(t, false)
	defer r.Close()
	defer bus.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant:        trunking.Grant{System: "S", Protocol: "p25", GroupID: 7, SourceID: 1},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if r.HasSession("VOICE-1") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	// WriteRawFrame should be a silent drop with no sidecar open.
	if err := r.WriteRawFrame("VOICE-1", []byte{0xAA}); err != nil {
		t.Fatal(err)
	}
	bus.Publish(events.Event{
		Kind: events.KindCallEnd,
		Payload: trunking.CallEnd{
			Grant: cs.Grant, DeviceSerial: "VOICE-1",
			StartedAt: cs.StartedAt, EndedAt: cs.StartedAt.Add(time.Second),
			Reason: trunking.EndReasonNormal,
		},
	})
	deadline = time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if !r.HasSession("VOICE-1") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	rawPath := filepath.Join(dir, "S", "7", "20260505T000000Z_src1.raw")
	if _, err := os.Stat(rawPath); !os.IsNotExist(err) {
		t.Errorf(".raw sidecar should not exist (WriteRaw=false, non-ProVoice): err=%v", err)
	}
}

func TestRecorderClosesOpenSessions(t *testing.T) {
	r, bus, _ := mkRecorder(t, false)
	defer bus.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: trunking.CallStart{
			Grant: trunking.Grant{System: "S", GroupID: 1}, DeviceSerial: "X",
			StartedAt: time.Now().UTC(),
		},
	})
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if r.HasSession("X") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if got := r.SessionCount(); got != 0 {
		t.Errorf("sessions not drained: %d", got)
	}
}

func TestSanitize(t *testing.T) {
	// Dots are intentionally preserved (talkgroup names like "Site 1.2"
	// are common); slashes / spaces / shell metacharacters are mapped to
	// underscores. Empty input stays empty.
	cases := map[string]string{
		"FIRE-DISP":         "FIRE-DISP",
		"Fire / EMS":        "Fire___EMS",
		"  spaces  ":        "spaces",
		"path/../traversal": "path_.._traversal",
		"":                  "",
	}
	for in, want := range cases {
		if got := sanitize(in); got != want {
			t.Errorf("sanitize(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestRecorderInstantiatesVocoderForMappedProtocol: a CallStart
// with Grant.Protocol = "test-null" + a custom map mapping that
// protocol to the always-registered "null" vocoder must produce
// a session with an active vocoder. We use "null" (instead of
// "imbe" or "ambe2") because the voice package can't import the
// real decoder packages without creating a cycle — those
// integration tests live in internal/voice/imbe and
// internal/voice/ambe2.
func TestRecorderInstantiatesVocoderForMappedProtocol(t *testing.T) {
	bus := events.NewBus(8)
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus:                bus,
		OutDir:             dir,
		SampleRate:         8000,
		WriteRaw:           true,
		VocoderForProtocol: map[string]string{"test-null": "null"},
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
		Grant:        trunking.Grant{System: "S", Protocol: "test-null", GroupID: 1, SourceID: 1},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if r.HasSession("VOICE-1") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Inspect the session under the recorder's lock.
	r.mu.Lock()
	s, ok := r.sessions["VOICE-1"]
	r.mu.Unlock()
	if !ok {
		t.Fatal("session not opened")
	}
	if s.vocoder == nil {
		t.Errorf("vocoder = nil, want non-nil for protocol=test-null → null")
	}
	if s.vocoderName != "null" {
		t.Errorf("vocoderName = %q, want %q", s.vocoderName, "null")
	}
}

// TestRecorderWriteRawFrameDecodesIntoWav: feeding raw frames to
// WriteRawFrame on a session with a NullVocoder writes silence
// samples into the WAV. Pins that the decode → WAV path is wired.
func TestRecorderWriteRawFrameDecodesIntoWav(t *testing.T) {
	bus := events.NewBus(8)
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus:                bus,
		OutDir:             dir,
		SampleRate:         8000,
		VocoderForProtocol: map[string]string{"test-null": "null"},
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
		Grant:        trunking.Grant{System: "S", Protocol: "test-null", GroupID: 1, SourceID: 2},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if r.HasSession("VOICE-1") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	// NullVocoder.FrameSize is 11 by default (set by the
	// package's init); send 3 frames worth of zero bytes.
	for i := 0; i < 3; i++ {
		if err := r.WriteRawFrame("VOICE-1", make([]byte, 11)); err != nil {
			t.Fatalf("WriteRawFrame: %v", err)
		}
	}

	bus.Publish(events.Event{
		Kind: events.KindCallEnd,
		Payload: trunking.CallEnd{
			Grant: cs.Grant, DeviceSerial: "VOICE-1",
			StartedAt: cs.StartedAt, EndedAt: cs.StartedAt.Add(time.Second),
			Reason: trunking.EndReasonNormal,
		},
	})
	deadline = time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if !r.HasSession("VOICE-1") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	wavPath := filepath.Join(dir, "S", "1", "20260505T000000Z_src2.wav")
	st, err := os.Stat(wavPath)
	if err != nil {
		t.Fatalf("wav at %s: %v", wavPath, err)
	}
	// 3 frames × 160 samples × 2 bytes = 960 bytes payload + 44 header.
	wantSize := int64(wavHeaderSize + 3*160*2)
	if st.Size() != wantSize {
		t.Errorf("wav size = %d, want %d", st.Size(), wantSize)
	}
}

// fakeDecodedTap records the PCM the recorder forwards from its
// digital decode path so a test can confirm live consumers are fed.
type fakeDecodedTap struct {
	mu    sync.Mutex
	calls int
	total int // total samples forwarded
	last  string
}

func (f *fakeDecodedTap) WritePCM(serial string, samples []int16) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.total += len(samples)
	f.last = serial
	return nil
}

func (f *fakeDecodedTap) snapshot() (calls, total int, last string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls, f.total, f.last
}

// fakeCallAwareTap implements DecodedPCMCallSink, recording the CallID the
// recorder hands to each forwarded chunk so a test can confirm the live fan-out
// is fenced by the session's call identity (voice-tap audio-bleed guard).
type fakeCallAwareTap struct {
	mu      sync.Mutex
	callIDs []uint64
}

func (f *fakeCallAwareTap) WritePCM(string, []int16) error { return nil }
func (f *fakeCallAwareTap) WritePCMForCall(_ string, callID uint64, _ []int16) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.callIDs = append(f.callIDs, callID)
	return nil
}
func (f *fakeCallAwareTap) snapshot() []uint64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]uint64(nil), f.callIDs...)
}

// TestRecorderForwardsDecodedPCMToCallAwareTap: when the live tap implements
// DecodedPCMCallSink, the recorder must forward the decoding session's CallID
// so the tap can fence a stale frame from a previous call on a reused voice-tap
// serial (the audio-bleed guard). Each forwarded chunk carries the call's CallID.
func TestRecorderForwardsDecodedPCMToCallAwareTap(t *testing.T) {
	bus := events.NewBus(8)
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus:                bus,
		OutDir:             dir,
		SampleRate:         8000,
		VocoderForProtocol: map[string]string{"test-null": "null"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer bus.Close()

	tap := &fakeCallAwareTap{}
	r.SetDecodedPCMSink(tap)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant:        trunking.Grant{System: "S", Protocol: "test-null", GroupID: 1, CallID: 77},
		DeviceSerial: "tap-0",
		StartedAt:    time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if r.HasSession("tap-0") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	// A matching call-aware frame forwards the session's CallID.
	if err := r.WriteRawFrameForCall("tap-0", 77, make([]byte, 11), 0); err != nil {
		t.Fatalf("WriteRawFrameForCall: %v", err)
	}
	// A frame from a different (stale) call is fenced by sessionForWrite before
	// decode, so it must never reach the tap.
	if err := r.WriteRawFrameForCall("tap-0", 999, make([]byte, 11), 0); err != nil {
		t.Fatalf("WriteRawFrameForCall stale: %v", err)
	}

	got := tap.snapshot()
	if len(got) != 1 || got[0] != 77 {
		t.Errorf("forwarded callIDs = %v, want exactly [77] (stale 999 fenced)", got)
	}
}

// fakeRawTap implements RawFrameCallSink, recording the verbatim bytes, vocoder
// name, and CallID the recorder forwards for each raw frame so a test can
// confirm the raw fan-out carries the codec label and is fenced by the
// session's call identity.
type fakeRawTap struct {
	mu       sync.Mutex
	callIDs  []uint64
	vocoders []string
	frames   [][]byte
}

func (f *fakeRawTap) WriteRawFrame(_, vocoder string, frame []byte) error {
	return f.WriteRawFrameForCall("", 0, vocoder, frame)
}
func (f *fakeRawTap) WriteRawFrameForCall(_ string, callID uint64, vocoder string, frame []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.callIDs = append(f.callIDs, callID)
	f.vocoders = append(f.vocoders, vocoder)
	f.frames = append(f.frames, append([]byte(nil), frame...))
	return nil
}
func (f *fakeRawTap) snapshot() (callIDs []uint64, vocoders []string, frames [][]byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]uint64(nil), f.callIDs...), append([]string(nil), f.vocoders...), f.frames
}

// TestRecorderForwardsRawFramesToTap: the recorder must forward verbatim raw
// vocoder frames to the raw tap (the gRPC publisher's include_raw subscribers),
// carrying the session's vocoder name and CallID, and a stale-call frame must be
// fenced by sessionForWrite before reaching the tap — the raw complement of the
// decoded-PCM bleed guard.
func TestRecorderForwardsRawFramesToTap(t *testing.T) {
	bus := events.NewBus(8)
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus:                bus,
		OutDir:             dir,
		SampleRate:         8000,
		VocoderForProtocol: map[string]string{"test-null": "null"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer bus.Close()

	tap := &fakeRawTap{}
	r.SetRawFrameSink(tap)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant:        trunking.Grant{System: "S", Protocol: "test-null", GroupID: 1, CallID: 77},
		DeviceSerial: "tap-0",
		StartedAt:    time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if r.HasSession("tap-0") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	// A matching frame forwards the verbatim bytes, the session vocoder name,
	// and the session CallID.
	want := []byte{0xAA, 0xBB, 0xCC, 0, 0, 0, 0, 0, 0, 0, 0} // NullVocoder.FrameSize = 11
	if err := r.WriteRawFrameForCall("tap-0", 77, want, 0); err != nil {
		t.Fatalf("WriteRawFrameForCall: %v", err)
	}
	// A stale-call frame is fenced before the tap.
	if err := r.WriteRawFrameForCall("tap-0", 999, make([]byte, 11), 0); err != nil {
		t.Fatalf("WriteRawFrameForCall stale: %v", err)
	}

	callIDs, vocoders, frames := tap.snapshot()
	if len(callIDs) != 1 || callIDs[0] != 77 {
		t.Fatalf("forwarded callIDs = %v, want exactly [77] (stale 999 fenced)", callIDs)
	}
	if vocoders[0] != "null" {
		t.Errorf("vocoder = %q, want null", vocoders[0])
	}
	if string(frames[0]) != string(want) {
		t.Errorf("frame bytes = %v, want %v", frames[0], want)
	}
}

// TestRecorderForwardsDecodedPCMToTap: PCM the recorder decodes from
// digital vocoder frames in WriteRawFrame must be forwarded to the
// live-audio tap (web stream / player / tone-out). Without this the
// live audio path is silent for digital calls while recordings play
// fine — issue #598. Also pins that the analog WritePCM path does NOT
// re-invoke the tap (those sinks already get analog PCM via the
// composer fanout; tapping there would double-send).
func TestRecorderForwardsDecodedPCMToTap(t *testing.T) {
	bus := events.NewBus(8)
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus:                bus,
		OutDir:             dir,
		SampleRate:         8000,
		VocoderForProtocol: map[string]string{"test-null": "null"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer bus.Close()

	tap := &fakeDecodedTap{}
	r.SetDecodedPCMSink(tap)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant:        trunking.Grant{System: "S", Protocol: "test-null", GroupID: 1, SourceID: 2},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if r.HasSession("VOICE-1") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	// NullVocoder.FrameSize is 11 and decodes to 160 samples/frame.
	for i := 0; i < 3; i++ {
		if err := r.WriteRawFrame("VOICE-1", make([]byte, 11)); err != nil {
			t.Fatalf("WriteRawFrame: %v", err)
		}
	}

	calls, total, last := tap.snapshot()
	if calls != 3 {
		t.Errorf("tap WritePCM calls = %d, want 3", calls)
	}
	if total != 3*160 {
		t.Errorf("tap forwarded %d samples, want %d", total, 3*160)
	}
	if last != "VOICE-1" {
		t.Errorf("tap serial = %q, want VOICE-1", last)
	}

	// The analog WritePCM path must not touch the tap (avoids
	// double-sending: those sinks get analog PCM via the fanout).
	if err := r.WritePCM("VOICE-1", []int16{1, 2, 3, 4}); err != nil {
		t.Fatalf("WritePCM: %v", err)
	}
	if calls2, _, _ := tap.snapshot(); calls2 != 3 {
		t.Errorf("tap calls after analog WritePCM = %d, want 3 (unchanged)", calls2)
	}
}

// TestRecorderEmptyVocoderMapDisablesAutoDecode: passing a
// non-nil but empty VocoderForProtocol disables auto-decode for
// every protocol — the .raw sidecar (when enabled) is still the
// only audio output for digital calls.
func TestRecorderEmptyVocoderMapDisablesAutoDecode(t *testing.T) {
	bus := events.NewBus(8)
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus:                bus,
		OutDir:             dir,
		SampleRate:         8000,
		VocoderForProtocol: map[string]string{}, // explicit empty
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
		Grant:        trunking.Grant{System: "S", Protocol: "p25", GroupID: 1, SourceID: 1},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if r.HasSession("VOICE-1") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	r.mu.Lock()
	s, ok := r.sessions["VOICE-1"]
	r.mu.Unlock()
	if !ok {
		t.Fatal("session not opened")
	}
	if s.vocoder != nil {
		t.Errorf("vocoder = %v, want nil with empty VocoderForProtocol map", s.vocoderName)
	}
}

// TestRecorderUnmappedProtocolSkipsVocoder: protocols not in the
// map (typically analog protocols like "motorola" / "ltr") should
// produce no vocoder. The composer's FM chain feeds WritePCM
// directly for those calls.
func TestRecorderUnmappedProtocolSkipsVocoder(t *testing.T) {
	r, bus, _ := mkRecorder(t, false)
	defer r.Close()
	defer bus.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	// "motorola" is intentionally absent from
	// DefaultVocoderForProtocol — the trunking decoder publishes
	// it for analog-passthrough calls.
	cs := trunking.CallStart{
		Grant:        trunking.Grant{System: "S", Protocol: "motorola", GroupID: 1, SourceID: 1},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if r.HasSession("VOICE-1") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	r.mu.Lock()
	s, ok := r.sessions["VOICE-1"]
	r.mu.Unlock()
	if !ok {
		t.Fatal("session not opened")
	}
	if s.vocoder != nil {
		t.Errorf("vocoder = %q, want nil for unmapped protocol", s.vocoderName)
	}
}

// TestRecorderUnknownVocoderNameLogsAndProceeds: a Protocol mapping
// pointing at a registry name no factory has registered must not
// panic — the recorder logs and continues with vocoder = nil.
func TestRecorderUnknownVocoderNameLogsAndProceeds(t *testing.T) {
	bus := events.NewBus(8)
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus:                bus,
		OutDir:             dir,
		SampleRate:         8000,
		VocoderForProtocol: map[string]string{"x": "no-such-vocoder"},
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
		Grant:        trunking.Grant{System: "S", Protocol: "x", GroupID: 1, SourceID: 1},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if r.HasSession("VOICE-1") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	r.mu.Lock()
	s, ok := r.sessions["VOICE-1"]
	r.mu.Unlock()
	if !ok {
		t.Fatal("session not opened")
	}
	if s.vocoder != nil {
		t.Errorf("vocoder should be nil after registry miss, got %q", s.vocoderName)
	}
}

// TestDefaultVocoderForProtocolMappings: pin the default
// Protocol → vocoder mapping so a future refactor doesn't
// silently drop a digital protocol from the auto-decode path.
func TestDefaultVocoderForProtocolMappings(t *testing.T) {
	got := DefaultVocoderForProtocol()
	want := map[string]string{
		"p25":        "imbe",
		"p25-phase2": "ambe2",
		"dmr-tier1":  "ambe2-dmr",
		"dmr-tier2":  "ambe2-dmr",
		"dmr-tier3":  "ambe2-dmr",
		"nxdn":       "ambe2",
		"dpmr":       "ambe2",
		"tetra":      "ambe2",
	}
	if len(got) != len(want) {
		t.Errorf("len = %d, want %d", len(got), len(want))
	}
	for k, v := range want {
		if g := got[k]; g != v {
			t.Errorf("Default[%q] = %q, want %q", k, g, v)
		}
	}
}

// mkSkipEncRecorder builds a recorder with SkipEncrypted enabled.
func mkSkipEncRecorder(t *testing.T) (*Recorder, *events.Bus, string) {
	t.Helper()
	bus := events.NewBus(8)
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus:           bus,
		OutDir:        dir,
		SampleRate:    8000,
		SkipEncrypted: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return r, bus, dir
}

// TestRecorderSkipsEncryptedCallAtStart confirms that with SkipEncrypted
// set, a CallStart whose grant already flags encryption opens no session
// and writes no files.
func TestRecorderSkipsEncryptedCallAtStart(t *testing.T) {
	r, bus, dir := mkSkipEncRecorder(t)
	defer r.Close()
	defer bus.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant: trunking.Grant{
			System: "S", Protocol: "p25", GroupID: 700, SourceID: 9,
			Encrypted: true, AlgorithmID: 0x84, KeyID: 1,
		},
		Talkgroup:    &trunking.TalkGroup{ID: 700, AlphaTag: "SECURE", Record: true},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Now(),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})

	// The recorder should (correctly) drop the CallStart — no session ever.
	deadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		if r.HasSession("VOICE-1") {
			t.Fatal("recorder opened a session for an encrypted call")
		}
		time.Sleep(5 * time.Millisecond)
	}
	assertNoWav(t, dir)
}

// TestRecorderAbortsMidCallEncryptedSourceUpdate confirms that a call
// which only reveals its encryption mid-stream (via KindCallSourceUpdate)
// has its in-progress files closed and deleted and no KindCallComplete
// is published.
func TestRecorderAbortsMidCallEncryptedSourceUpdate(t *testing.T) {
	r, bus, dir := mkSkipEncRecorder(t)
	defer r.Close()
	defer bus.Close()

	// Independent subscription to assert no CallComplete escapes.
	watch := bus.Subscribe()
	defer watch.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant:        trunking.Grant{System: "S", Protocol: "p25-phase2", GroupID: 701, SourceID: 0},
		Talkgroup:    &trunking.TalkGroup{ID: 701, AlphaTag: "TAC", Record: true},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Now(),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	waitSession(t, r, "VOICE-1", true)

	// Lay down some audio so a real file exists on disk.
	if err := r.WritePCM("VOICE-1", make([]int16, 1600)); err != nil {
		t.Fatal(err)
	}
	if got := wavFiles(t, dir); len(got) != 1 {
		t.Fatalf("expected 1 wav before abort, got %d", len(got))
	}

	// Encryption surfaces in-call on the traffic channel.
	bus.Publish(events.Event{Kind: events.KindCallSourceUpdate, Payload: trunking.CallSourceUpdate{
		DeviceSerial: "VOICE-1", SourceID: 42, Encrypted: true, At: time.Now(),
	}})
	waitSession(t, r, "VOICE-1", false)

	assertNoWav(t, dir)

	// Drain a short window and confirm no CallComplete was published.
	deadline := time.After(150 * time.Millisecond)
	for {
		select {
		case ev := <-watch.C:
			if ev.Kind == events.KindCallComplete {
				t.Fatal("CallComplete published for an aborted encrypted call")
			}
		case <-deadline:
			return
		}
	}
}

// TestRecorderAbortsOnEncryptionSync confirms the KindCallEncryption path
// (P25 Phase 1 LDU2) also aborts when the recovered Algorithm ID is not
// CLEAR (0x80).
func TestRecorderAbortsOnEncryptionSync(t *testing.T) {
	r, bus, dir := mkSkipEncRecorder(t)
	defer r.Close()
	defer bus.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant:        trunking.Grant{System: "S", Protocol: "p25", GroupID: 702, SourceID: 5},
		Talkgroup:    &trunking.TalkGroup{ID: 702, AlphaTag: "OPS", Record: true},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Now(),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	waitSession(t, r, "VOICE-1", true)
	if err := r.WritePCM("VOICE-1", make([]int16, 800)); err != nil {
		t.Fatal(err)
	}

	// A CLEAR (0x80) sync must NOT abort.
	bus.Publish(events.Event{Kind: events.KindCallEncryption, Payload: trunking.CallEncryption{
		DeviceSerial: "VOICE-1", AlgorithmID: 0x80, At: time.Now(),
	}})
	time.Sleep(50 * time.Millisecond)
	if !r.HasSession("VOICE-1") {
		t.Fatal("CLEAR encryption sync wrongly aborted the recording")
	}

	// A real algorithm (AES, 0x84) must abort.
	bus.Publish(events.Event{Kind: events.KindCallEncryption, Payload: trunking.CallEncryption{
		DeviceSerial: "VOICE-1", AlgorithmID: 0x84, At: time.Now(),
	}})
	waitSession(t, r, "VOICE-1", false)
	assertNoWav(t, dir)
}

// TestRecorderRecordsEncryptedWhenSkipDisabled is the regression guard:
// with SkipEncrypted off (the default), an encrypted call records as
// usual.
func TestRecorderRecordsEncryptedWhenSkipDisabled(t *testing.T) {
	r, bus, _ := mkRecorder(t, false) // SkipEncrypted defaults to false
	defer r.Close()
	defer bus.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant: trunking.Grant{
			System: "S", Protocol: "p25", GroupID: 703, SourceID: 7, Encrypted: true,
		},
		Talkgroup:    &trunking.TalkGroup{ID: 703, AlphaTag: "ENC", Record: true},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Now(),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	waitSession(t, r, "VOICE-1", true)

	// Mid-call encryption signalling must be ignored when SkipEncrypted is off.
	bus.Publish(events.Event{Kind: events.KindCallSourceUpdate, Payload: trunking.CallSourceUpdate{
		DeviceSerial: "VOICE-1", Encrypted: true, At: time.Now(),
	}})
	time.Sleep(50 * time.Millisecond)
	if !r.HasSession("VOICE-1") {
		t.Fatal("recorder aborted an encrypted call with SkipEncrypted disabled")
	}
}

// wavFiles returns every .wav path under root.
func wavFiles(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(p) == ".wav" {
			out = append(out, p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// assertNoWav fails if any .wav file exists under root.
func assertNoWav(t *testing.T, root string) {
	t.Helper()
	if got := wavFiles(t, root); len(got) != 0 {
		t.Fatalf("expected no wav files, found %v", got)
	}
}

// TestRecorderNormalizesFinishedWav confirms that, with Normalize enabled,
// a finished recording is loudness-normalized in place toward the target
// before its CallComplete is observable on disk.
func TestRecorderNormalizesFinishedWav(t *testing.T) {
	bus := events.NewBus(8)
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus:        bus,
		OutDir:     dir,
		SampleRate: 8000,
		Normalize:  NormalizeConfig{Enabled: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer bus.Close()

	// Subscribe so we can wait for CallComplete, which the recorder
	// publishes only after the finished WAV has been normalized in place.
	// Waiting on session removal instead would race the rewrite, since the
	// session is deleted before normalization runs.
	watch := bus.Subscribe()
	defer watch.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant: trunking.Grant{
			System: "TestSystem", Protocol: "motorola",
			GroupID: 1234, SourceID: 56789,
		},
		Talkgroup:    &trunking.TalkGroup{ID: 1234, AlphaTag: "FIRE-DISP", Record: true},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 5, 5, 12, 30, 45, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	waitSession(t, r, "VOICE-1", true)

	// ~2.5 s of a quiet 1 kHz tone (~-24 LUFS) that needs boosting toward
	// the -16 LUFS target.
	const n = 8000 * 5 / 2
	pcm := make([]int16, n)
	for i := range pcm {
		v := 0.1 * math.Sin(2*math.Pi*1000*float64(i)/8000)
		pcm[i] = int16(v * 32767)
	}
	if err := r.WritePCM("VOICE-1", pcm); err != nil {
		t.Fatal(err)
	}

	bus.Publish(events.Event{Kind: events.KindCallEnd, Payload: trunking.CallEnd{
		Grant: cs.Grant, Talkgroup: cs.Talkgroup, DeviceSerial: "VOICE-1",
		StartedAt: cs.StartedAt, EndedAt: cs.StartedAt.Add(3 * time.Second),
		Reason: trunking.EndReasonNormal,
	}})

	// Block until the call is finalized AND normalized.
	var donePath string
	deadline := time.After(2 * time.Second)
	for donePath == "" {
		select {
		case ev := <-watch.C:
			if ev.Kind == events.KindCallComplete {
				donePath = ev.Payload.(trunking.CallComplete).AudioPath
			}
		case <-deadline:
			t.Fatal("timed out waiting for CallComplete")
		}
	}

	samples, rate, err := ReadWAVSamples(donePath)
	if err != nil {
		t.Fatalf("read normalized wav: %v", err)
	}
	fs := make([]float64, len(samples))
	for i, s := range samples {
		fs[i] = float64(s) / 32768.0
	}
	lufs, ok := loudness.IntegratedLUFS(fs, int(rate))
	if !ok {
		t.Fatal("normalized wav not measurable")
	}
	if lufs < -17.0 || lufs > -15.0 {
		t.Fatalf("normalized loudness %.2f LUFS not near -16 target", lufs)
	}
}
