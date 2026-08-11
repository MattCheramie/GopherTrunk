package voice

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// statStubVocoder is a Vocoder + StatProvider test double. Decode returns
// silence and counts calls; VoiceStats reports a configurable frame
// classification so tests can drive the recorder's empty-recording
// suppression (all idle/silent ⇒ Voiced+Unvoiced == 0).
type statStubVocoder struct {
	mu             sync.Mutex
	decodes        int
	voicedPerFrame bool // when true, each decoded frame counts as voiced
}

func (v *statStubVocoder) Name() string   { return "stat-stub" }
func (v *statStubVocoder) FrameSize() int { return 11 }
func (v *statStubVocoder) Reset()         {}
func (v *statStubVocoder) Close() error   { return nil }
func (v *statStubVocoder) ResetStats()    {}

func (v *statStubVocoder) Decode(frame []byte) ([]int16, error) {
	v.mu.Lock()
	v.decodes++
	v.mu.Unlock()
	return make([]int16, 160), nil
}

func (v *statStubVocoder) decodeCount() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.decodes
}

func (v *statStubVocoder) VoiceStats() (VoiceStats, bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.decodes == 0 {
		return VoiceStats{}, false
	}
	vs := VoiceStats{Frames: v.decodes, MinB0: -1, MaxB0: -1}
	if v.voicedPerFrame {
		vs.Voiced = v.decodes
	} else {
		// All frames idle-muted: no real speech.
		vs.IdleMuted = v.decodes
	}
	return vs, true
}

// vocoderCapture safely hands the recorder-constructed stub back to the
// test goroutine (the factory runs on the recorder's Run goroutine).
type vocoderCapture struct {
	mu sync.Mutex
	v  *statStubVocoder
}

func (c *vocoderCapture) set(v *statStubVocoder) { c.mu.Lock(); c.v = v; c.mu.Unlock() }
func (c *vocoderCapture) get() *statStubVocoder  { c.mu.Lock(); defer c.mu.Unlock(); return c.v }

// TestRecorderSuppressesAllIdleRecording: a vocoder call that decodes no
// real speech (every frame idle-muted ⇒ Voiced+Unvoiced == 0) must have its
// WAV removed and publish no CallComplete — the tiny-file-spam fix.
func TestRecorderSuppressesAllIdleRecording(t *testing.T) {
	vc := &vocoderCapture{}
	DefaultRegistry.Register("stat-stub-idle", func() (Vocoder, error) {
		v := &statStubVocoder{voicedPerFrame: false}
		vc.set(v)
		return v, nil
	})

	bus := events.NewBus(16)
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus:                bus,
		OutDir:             dir,
		SampleRate:         8000,
		VocoderForProtocol: map[string]string{"test-idle": "stat-stub-idle"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer bus.Close()

	// Watch for CallComplete; an all-idle call must not publish one.
	sub := bus.Subscribe()
	defer sub.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant:        trunking.Grant{System: "S", Protocol: "test-idle", GroupID: 7, SourceID: 9},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	waitSession(t, r, "VOICE-1", true)

	for i := 0; i < 5; i++ {
		if err := r.WriteRawFrame("VOICE-1", make([]byte, 11)); err != nil {
			t.Fatalf("WriteRawFrame: %v", err)
		}
	}

	bus.Publish(events.Event{Kind: events.KindCallEnd, Payload: trunking.CallEnd{
		Grant: cs.Grant, DeviceSerial: "VOICE-1",
		StartedAt: cs.StartedAt, EndedAt: cs.StartedAt.Add(time.Second),
		Reason: trunking.EndReasonNormal,
	}})
	waitSession(t, r, "VOICE-1", false)

	wavPath := filepath.Join(dir, "S", "7", "20260505_000000_7.wav")
	if _, err := os.Stat(wavPath); !os.IsNotExist(err) {
		t.Errorf("all-idle WAV %s should have been removed; stat err = %v", wavPath, err)
	}

	// Drain the bus briefly and assert no CallComplete fired.
	timeout := time.After(200 * time.Millisecond)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCallComplete {
				t.Fatal("all-idle call must not publish CallComplete")
			}
		case <-timeout:
			return
		}
	}
}

// TestRecorderNoAudioLeavesNoFiles: a call that starts and ends without a
// single frame must leave nothing on disk — no 44-byte header-only .wav and
// no 0-byte .raw — and publish no CallComplete. Capture files are opened
// lazily on the first write, so a no-audio call never touches the filesystem.
func TestRecorderNoAudioLeavesNoFiles(t *testing.T) {
	vc := &vocoderCapture{}
	DefaultRegistry.Register("stat-stub-noaudio", func() (Vocoder, error) {
		v := &statStubVocoder{voicedPerFrame: true}
		vc.set(v)
		return v, nil
	})

	bus := events.NewBus(16)
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus:                bus,
		OutDir:             dir,
		SampleRate:         8000,
		WriteRaw:           true, // exercise the .raw sidecar path too
		VocoderForProtocol: map[string]string{"test-noaudio": "stat-stub-noaudio"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer bus.Close()

	sub := bus.Subscribe()
	defer sub.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant:        trunking.Grant{System: "S", Protocol: "test-noaudio", GroupID: 7, SourceID: 9},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	waitSession(t, r, "VOICE-1", true)

	// No frames written at all, then the call ends.
	wavPath := filepath.Join(dir, "S", "7", "20260505_000000_7.wav")
	rawPath := filepath.Join(dir, "S", "7", "20260505_000000_7.raw")
	if _, err := os.Stat(wavPath); !os.IsNotExist(err) {
		t.Errorf("WAV %s must not exist before any audio is written; stat err = %v", wavPath, err)
	}
	if _, err := os.Stat(rawPath); !os.IsNotExist(err) {
		t.Errorf("raw %s must not exist before any audio is written; stat err = %v", rawPath, err)
	}

	bus.Publish(events.Event{Kind: events.KindCallEnd, Payload: trunking.CallEnd{
		Grant: cs.Grant, DeviceSerial: "VOICE-1",
		StartedAt: cs.StartedAt, EndedAt: cs.StartedAt.Add(time.Second),
		Reason: trunking.EndReasonNormal,
	}})
	waitSession(t, r, "VOICE-1", false)

	if _, err := os.Stat(wavPath); !os.IsNotExist(err) {
		t.Errorf("no-audio WAV %s must not be created; stat err = %v", wavPath, err)
	}
	if _, err := os.Stat(rawPath); !os.IsNotExist(err) {
		t.Errorf("no-audio raw %s must not be created; stat err = %v", rawPath, err)
	}

	timeout := time.After(200 * time.Millisecond)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCallComplete {
				t.Fatal("no-audio call must not publish CallComplete")
			}
		case <-timeout:
			return
		}
	}
}

// TestRecorderCallIDFenceDropsStaleFrames: a frame tagged with a CallID that
// doesn't match the open session is dropped before it reaches the vocoder —
// the voice-tap mix-up fence.
func TestRecorderCallIDFenceDropsStaleFrames(t *testing.T) {
	vc := &vocoderCapture{}
	DefaultRegistry.Register("stat-stub-voiced", func() (Vocoder, error) {
		v := &statStubVocoder{voicedPerFrame: true}
		vc.set(v)
		return v, nil
	})

	bus := events.NewBus(16)
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus:                bus,
		OutDir:             dir,
		SampleRate:         8000,
		VocoderForProtocol: map[string]string{"test-voiced": "stat-stub-voiced"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer bus.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	const wantCallID = 42
	cs := trunking.CallStart{
		Grant: trunking.Grant{
			System: "S", Protocol: "test-voiced", GroupID: 7, SourceID: 9, CallID: wantCallID,
		},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	waitSession(t, r, "VOICE-1", true)

	// A stale frame from a different call must be dropped.
	if err := r.WriteRawFrameForCall("VOICE-1", wantCallID+1, make([]byte, 11), 0); err != nil {
		t.Fatalf("stale WriteRawFrameForCall: %v", err)
	}
	// The matching frame must be decoded.
	if err := r.WriteRawFrameForCall("VOICE-1", wantCallID, make([]byte, 11), 0); err != nil {
		t.Fatalf("matching WriteRawFrameForCall: %v", err)
	}

	v := vc.get()
	if v == nil {
		t.Fatal("vocoder was never constructed")
	}
	if got := v.decodeCount(); got != 1 {
		t.Errorf("vocoder decoded %d frames; want 1 (stale frame must be fenced)", got)
	}
}

// TestRecorderNoAudioLeavesNoDir: a call that starts and ends without a single
// frame must not even create its talkgroup directory. The empty <system>/<tg>
// folder is otherwise left behind when a grant is followed but the voice tap
// never yields audio (e.g. the SDR is off-frequency during a borrowed hunt),
// littering the recordings tree with empty TG folders.
func TestRecorderNoAudioLeavesNoDir(t *testing.T) {
	DefaultRegistry.Register("stat-stub-nodir", func() (Vocoder, error) {
		return &statStubVocoder{voicedPerFrame: true}, nil
	})

	bus := events.NewBus(16)
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus:                bus,
		OutDir:             dir,
		SampleRate:         8000,
		WriteRaw:           true,
		VocoderForProtocol: map[string]string{"test-nodir": "stat-stub-nodir"},
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
		Grant:        trunking.Grant{System: "S", Protocol: "test-nodir", GroupID: 1421, SourceID: 0},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	waitSession(t, r, "VOICE-1", true)

	bus.Publish(events.Event{Kind: events.KindCallEnd, Payload: trunking.CallEnd{
		Grant: cs.Grant, DeviceSerial: "VOICE-1",
		StartedAt: cs.StartedAt, EndedAt: cs.StartedAt.Add(time.Second),
		Reason: trunking.EndReasonNormal,
	}})
	waitSession(t, r, "VOICE-1", false)

	tgDir := filepath.Join(dir, "S", "1421")
	if _, err := os.Stat(tgDir); !os.IsNotExist(err) {
		t.Errorf("no-audio call must not create talkgroup dir %s; stat err = %v", tgDir, err)
	}
}
