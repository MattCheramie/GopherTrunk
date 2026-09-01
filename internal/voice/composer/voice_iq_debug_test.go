package composer

import (
	"context"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"log/slog"
)

// TestTeeVoiceIQWritesPerCallCapture pins the diagnostic-container voice
// capture: the tee must (1) pass every chunk through to the chain unchanged
// and in order, (2) write those exact samples as cs16 to a per-call file,
// and (3) leave a siglab metadata sidecar carrying the grant's parameters
// (system, talkgroup, source, frequency, rate) so the file drops straight
// into replay/siglab.
func TestTeeVoiceIQWritesPerCallCapture(t *testing.T) {
	dir := t.TempDir()
	c := &Composer{
		log:          slog.Default(),
		voiceIQDebug: VoiceIQDebugConfig{Enabled: true, Dir: dir},
	}
	cs := trunking.CallStart{
		Grant: trunking.Grant{
			System: "Ost", Protocol: "p25", GroupID: 204, SourceID: 1234,
			FrequencyHz: 450_962_500, RFSSID: 1, SiteID: 7, NAC: 0x2C1,
		},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Unix(1_700_000_000, 0).UTC(),
	}

	in := make(chan []complex64, 4)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := c.teeVoiceIQ(ctx, in, cs, 48_000)

	want := []complex64{0.5 + 0i, 0 - 0.25i, -0.125 + 0.75i, 1 + 0i}
	in <- want[:2]
	in <- want[2:]
	close(in)

	var got []complex64
	for chunk := range out {
		got = append(got, chunk...)
	}
	if len(got) != len(want) {
		t.Fatalf("chain received %d samples, want %d (the tee must be lossless toward the chain)", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("chain sample %d = %v, want %v", i, got[i], want[i])
		}
	}

	// The writer closes asynchronously after out closes; poll briefly for the
	// final sidecar rewrite (it carries the samples count).
	files, _ := filepath.Glob(filepath.Join(dir, "*_voice.cs16"))
	if len(files) != 1 {
		t.Fatalf("capture files = %v, want exactly one *_voice.cs16", files)
	}
	metaPath := strings.TrimSuffix(files[0], ".cs16") + ".metadata.json"
	deadline := time.Now().Add(2 * time.Second)
	var meta *siglab.Metadata
	for {
		m, err := siglab.LoadMetadata(metaPath)
		if err == nil && m.System["samples"] != "" {
			meta = m
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("metadata sidecar never finalised (err=%v, meta=%+v)", err, m)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if meta.Protocol != "p25" || meta.SampleRateHz != 48_000 || meta.CenterFreqHz != 450_962_500 {
		t.Errorf("metadata = proto %q rate %v center %d, want p25/48000/450962500", meta.Protocol, meta.SampleRateHz, meta.CenterFreqHz)
	}
	if meta.System["system"] != "Ost" || meta.System["talkgroup"] != "204" || meta.System["source_id"] != "1234" {
		t.Errorf("metadata system fields = %+v", meta.System)
	}
	if meta.System["samples"] != "4" {
		t.Errorf("metadata samples = %q, want 4", meta.System["samples"])
	}
	if meta.System["truncated"] == "true" {
		t.Error("capture marked truncated on a trivially small stream")
	}

	// The cs16 body must be the exact samples, int16-scaled.
	raw, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != len(want)*4 {
		t.Fatalf("capture bytes = %d, want %d (4 per sample)", len(raw), len(want)*4)
	}
	for i, s := range want {
		re := int16(binary.LittleEndian.Uint16(raw[4*i:]))
		im := int16(binary.LittleEndian.Uint16(raw[4*i+2:]))
		if math.Abs(float64(re)-float64(real(s))*32767) > 1 ||
			math.Abs(float64(im)-float64(imag(s))*32767) > 1 {
			t.Errorf("sample %d = (%d,%d), want ~(%v,%v)", i, re, im, real(s)*32767, imag(s)*32767)
		}
	}
}

// TestTeeVoiceIQFormats pins the wav/flac container options: the per-call file
// gets the right extension and signature, and the metadata Format matches.
func TestTeeVoiceIQFormats(t *testing.T) {
	cases := []struct {
		format siglab.SampleFormat
		ext    string
		sig    string // leading bytes the container file must start with
	}{
		{siglab.FormatWAV, "wav", "RIFF"},
		{siglab.FormatFLAC, "flac", "fLaC"},
	}
	for _, tc := range cases {
		t.Run(tc.ext, func(t *testing.T) {
			dir := t.TempDir()
			c := &Composer{
				log:          slog.Default(),
				voiceIQDebug: VoiceIQDebugConfig{Enabled: true, Dir: dir, Format: tc.format},
			}
			cs := trunking.CallStart{
				Grant:        trunking.Grant{System: "Ost", Protocol: "p25", GroupID: 5, FrequencyHz: 450_000_000},
				DeviceSerial: "V",
				StartedAt:    time.Unix(1_700_000_000, 0).UTC(),
			}
			in := make(chan []complex64, 4)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			out := c.teeVoiceIQ(ctx, in, cs, 48_000)
			// Enough samples that flac emits at least one frame.
			block := make([]complex64, 6000)
			for i := range block {
				block[i] = complex(float32(math.Cos(float64(i)*0.03)), float32(math.Sin(float64(i)*0.03)))
			}
			in <- block
			close(in)
			for range out { //nolint:revive // drain the passthrough
			}

			var files []string
			deadline := time.Now().Add(2 * time.Second)
			for {
				files, _ = filepath.Glob(filepath.Join(dir, "*_voice."+tc.ext))
				if len(files) == 1 {
					if m, err := siglab.LoadMetadata(strings.TrimSuffix(files[0], "."+tc.ext) + ".metadata.json"); err == nil && m.System["samples"] != "" {
						if m.Format != tc.ext {
							t.Errorf("metadata Format = %q, want %q", m.Format, tc.ext)
						}
						break
					}
				}
				if time.Now().After(deadline) {
					t.Fatalf("no finalised *_voice.%s capture (files=%v)", tc.ext, files)
				}
				time.Sleep(10 * time.Millisecond)
			}
			raw, err := os.ReadFile(files[0])
			if err != nil {
				t.Fatal(err)
			}
			if len(raw) < len(tc.sig) || string(raw[:len(tc.sig)]) != tc.sig {
				t.Errorf("%s file does not start with %q signature", tc.ext, tc.sig)
			}
		})
	}
}
