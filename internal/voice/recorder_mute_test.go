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

// loudFakeVocoder decodes every frame to a constant non-zero level so a
// muted (zeroed) recording is distinguishable from a decoded one.
type loudFakeVocoder struct{}

func (loudFakeVocoder) Name() string   { return "loud-fake-voc" }
func (loudFakeVocoder) FrameSize() int { return 11 }
func (loudFakeVocoder) Decode([]byte) ([]int16, error) {
	out := make([]int16, 160)
	for i := range out {
		out[i] = 9000
	}
	return out, nil
}
func (loudFakeVocoder) Reset()       {}
func (loudFakeVocoder) Close() error { return nil }

func runMuteCase(t *testing.T, mute bool, encrypted bool, keyConfigured func(string, uint8, uint16, uint32) bool, midCall bool) (wav []int16, raw []byte) {
	t.Helper()
	DefaultRegistry.Register("loud-fake-voc", func() (Vocoder, error) { return loudFakeVocoder{}, nil })
	bus := events.NewBus(8)
	defer bus.Close()
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus:                bus,
		OutDir:             dir,
		SampleRate:         8000,
		WriteRaw:           true,
		MuteEncrypted:      mute,
		KeyConfigured:      keyConfigured,
		VocoderForProtocol: map[string]string{"p25-fake": "loud-fake-voc"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant:        trunking.Grant{System: "S", Protocol: "p25-fake", GroupID: 7, Encrypted: encrypted && !midCall, AlgorithmID: 0x84, KeyID: 3},
		Talkgroup:    &trunking.TalkGroup{ID: 7, AlphaTag: "PD", Record: true},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 5, 29, 17, 25, 2, 0, time.UTC),
	}
	if midCall {
		cs.Grant.AlgorithmID, cs.Grant.KeyID = 0, 0
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	waitForSession(t, r, "VOICE-1", true)
	frame := make([]byte, 11)
	for i := range frame {
		frame[i] = byte(0xA5 + i)
	}
	// Two clear-looking frames, then (mid-call case) the Encryption Sync
	// arrives, then two more frames.
	for i := 0; i < 2; i++ {
		if err := r.WriteRawFrame("VOICE-1", frame); err != nil {
			t.Fatal(err)
		}
	}
	if midCall && encrypted {
		bus.Publish(events.Event{Kind: events.KindCallEncryption, Payload: trunking.CallEncryption{DeviceSerial: "VOICE-1", System: "S", GroupID: 7, AlgorithmID: 0x84, KeyID: 3}})
		// Let the recorder's Run loop apply the backfill before the next frame.
		deadline := time.Now().Add(time.Second)
		for time.Now().Before(deadline) {
			r.mu.Lock()
			s := r.sessions["VOICE-1"]
			enc := s != nil && s.cs.Grant.Encrypted
			r.mu.Unlock()
			if enc {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
	for i := 0; i < 2; i++ {
		if err := r.WriteRawFrame("VOICE-1", frame); err != nil {
			t.Fatal(err)
		}
	}
	bus.Publish(events.Event{Kind: events.KindCallEnd, Payload: trunking.CallEnd{
		Grant: cs.Grant, Talkgroup: cs.Talkgroup, DeviceSerial: "VOICE-1",
		StartedAt: cs.StartedAt, EndedAt: cs.StartedAt.Add(time.Second), Reason: trunking.EndReasonNormal,
	}})
	waitForSession(t, r, "VOICE-1", false)
	var wavPath, rawPath string
	filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		switch filepath.Ext(p) {
		case ".wav":
			wavPath = p
		case ".raw":
			rawPath = p
		}
		return nil
	})
	if wavPath == "" || rawPath == "" {
		t.Fatalf("recording files missing: wav=%q raw=%q", wavPath, rawPath)
	}
	raw, _ = os.ReadFile(rawPath)
	return readWAVSamples(t, wavPath), raw
}

func countLoud(samples []int16) int {
	n := 0
	for _, v := range samples {
		if v == 9000 {
			n++
		}
	}
	return n
}

// TestRecorderMutesUndecryptableEncryptedAudio is the failing-first
// regression for recordings.mute_encrypted: an encrypted call with no key
// records silence (the vocoder's ciphertext rendering never reaches the
// WAV) while the .raw sidecar still holds every ciphertext frame; a call
// whose key IS configured, a clear call, and the default (mute off) all
// record the decoded audio unchanged.
func TestRecorderMutesUndecryptableEncryptedAudio(t *testing.T) {
	noKey := func(string, uint8, uint16, uint32) bool { return false }
	hasKey := func(sys string, _ uint8, kid uint16, _ uint32) bool { return sys == "S" && kid == 3 }

	wav, raw := runMuteCase(t, true, true, noKey, false)
	if countLoud(wav) != 0 {
		t.Fatalf("muted encrypted call: %d loud samples in WAV, want 0", countLoud(wav))
	}
	if len(wav) < 4*160 {
		t.Fatalf("muted WAV has %d samples — silence must keep the timeline, not drop frames", len(wav))
	}
	if len(raw) != 4*11 {
		t.Fatalf(".raw sidecar has %d bytes, want 44 (ciphertext frames must be kept)", len(raw))
	}

	// Default (mute off): the legacy behaviour, ciphertext through the vocoder.
	wav, _ = runMuteCase(t, false, true, noKey, false)
	if countLoud(wav) < 4*160 {
		t.Fatalf("mute off: %d loud samples, want ≥ %d", countLoud(wav), 4*160)
	}
	// Key configured: decrypted in-process, never muted.
	wav, _ = runMuteCase(t, true, true, hasKey, false)
	if countLoud(wav) < 4*160 {
		t.Fatalf("key configured: %d loud samples, want ≥ %d", countLoud(wav), 4*160)
	}
	// Clear call with mute on: untouched.
	wav, _ = runMuteCase(t, true, false, noKey, false)
	if countLoud(wav) < 4*160 {
		t.Fatalf("clear call: %d loud samples, want ≥ %d", countLoud(wav), 4*160)
	}
	// Encryption discovered mid-call (P25 LDU2 ES): the first two frames
	// decoded, the frames after the Encryption Sync are silent.
	wav, _ = runMuteCase(t, true, true, noKey, true)
	if got := countLoud(wav); got != 2*160 {
		t.Fatalf("mid-call encryption: %d loud samples, want exactly %d (two frames before the ES)", got, 2*160)
	}
}
