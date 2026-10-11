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

// pcmCapture is a DecodedPCMSink that keeps every sample it is handed.
type pcmCapture struct {
	mu      sync.Mutex
	samples []int16
}

func (c *pcmCapture) WritePCM(_ string, s []int16) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.samples = append(c.samples, s...)
	return nil
}

// recordDigitalWithGain runs one four-frame digital call through the
// recorder (loudFakeVocoder decodes every frame to 9000) with the grant's
// AudioGainDB, and returns the WAV's samples and the live tap's.
func recordDigitalWithGain(t *testing.T, gainDB float64) (wav, live []int16) {
	t.Helper()
	DefaultRegistry.Register("loud-fake-voc", func() (Vocoder, error) { return loudFakeVocoder{}, nil })
	bus := events.NewBus(8)
	defer bus.Close()
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus: bus, OutDir: dir, SampleRate: 8000,
		VocoderForProtocol: map[string]string{"p25-fake": "loud-fake-voc"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	tap := &pcmCapture{}
	r.SetDecodedPCMSink(tap)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant: trunking.Grant{
			System: "scanner", Protocol: "p25-fake", GroupID: 0x80000001,
			Conventional: true, AudioGainDB: gainDB,
		},
		Talkgroup:    &trunking.TalkGroup{ID: 0x80000001, AlphaTag: "VCALL10", Record: true},
		DeviceSerial: "CONV",
		StartedAt:    time.Date(2026, 10, 11, 0, 27, 31, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	waitForSession(t, r, "CONV", true)
	frame := make([]byte, 11)
	for i := 0; i < 4; i++ {
		if err := r.WriteRawFrame("CONV", frame); err != nil {
			t.Fatal(err)
		}
	}
	bus.Publish(events.Event{Kind: events.KindCallEnd, Payload: trunking.CallEnd{
		Grant: cs.Grant, Talkgroup: cs.Talkgroup, DeviceSerial: "CONV",
		StartedAt: cs.StartedAt, EndedAt: cs.StartedAt.Add(time.Second), Reason: trunking.EndReasonNormal,
	}})
	waitForSession(t, r, "CONV", false)
	var wavPath string
	filepath.Walk(dir, func(p string, _ os.FileInfo, _ error) error {
		if filepath.Ext(p) == ".wav" {
			wavPath = p
		}
		return nil
	})
	if wavPath == "" {
		t.Fatal("no WAV written")
	}
	tap.mu.Lock()
	defer tap.mu.Unlock()
	return readWAVSamples(t, wavPath), append([]int16(nil), tap.samples...)
}

// TestRecorderAppliesConventionalAudioGain pins scanner.conventional[]
// .audio_gain_db on a digital channel (#1239): digital PCM exists only in
// the recorder, so the grant's AudioGainDB must scale the vocoder output
// there, in the WAV and the live stream alike, and clamp at full scale.
func TestRecorderAppliesConventionalAudioGain(t *testing.T) {
	for _, tc := range []struct {
		name   string
		gainDB float64
		want   int16
	}{
		{"unity", 0, 9000},
		{"+6 dB", 6, 17957}, // 9000 × 10^(6/20)
		{"-6 dB", -6, 4511},
		{"clamped", 12, 32767}, // 9000 × 3.98 overshoots full scale
	} {
		t.Run(tc.name, func(t *testing.T) {
			wav, live := recordDigitalWithGain(t, tc.gainDB)
			const n = 4 * 160 // the decoded frames; a short fade follows
			if len(wav) < n || len(live) < n {
				t.Fatalf("got %d WAV / %d live samples, want ≥ %d", len(wav), len(live), n)
			}
			for i := 0; i < n; i++ {
				if wav[i] != tc.want || live[i] != tc.want {
					t.Fatalf("sample %d: WAV %d, live %d, want %d", i, wav[i], live[i], tc.want)
				}
			}
		})
	}
}
