package voice

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

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
		Talkgroup:    &trunking.TalkGroup{ID: 1234, AlphaTag: "FIRE-DISP"},
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
		Talkgroup:    &trunking.TalkGroup{ID: 7, AlphaTag: "GATE"},
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
		"FIRE-DISP":          "FIRE-DISP",
		"Fire / EMS":         "Fire___EMS",
		"  spaces  ":         "spaces",
		"path/../traversal":  "path_.._traversal",
		"":                   "",
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
		"dmr-tier2":  "ambe2",
		"dmr-tier3":  "ambe2",
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
