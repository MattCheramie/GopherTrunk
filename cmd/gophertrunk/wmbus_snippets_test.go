package main

import (
	"encoding/hex"
	"math/rand"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/wmbus"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// `wmbus -save-frames` (#1256): from a long capture of mostly empty air with
// two meters in it, the saver writes one small IQ WAV + sidecar per
// telegram, and each snippet replays through `wmbus -in` to exactly the
// telegram it was cut around. That is what lets an operator in the field
// share a few hundred KB instead of a ~100 MB capture.
func TestWMBusSaveFramesWritesReplayableSnippets(t *testing.T) {
	const (
		rate   = 1_600_000.0
		offset = 25_000.0
		iperl  = "1844ae4c4455223368077a55000000041389e20100023b0000"
	)
	telegrams := []string{wmbusQcaloric, iperl}
	r := rand.New(rand.NewSource(1))
	noise := func(n int) []complex64 {
		out := make([]complex64, n)
		for i := range out {
			out[i] = complex(float32(r.NormFloat64()*0.03), float32(r.NormFloat64()*0.03))
		}
		return out
	}
	// 0.6 s of noise, telegram, 0.9 s of noise, telegram, 0.5 s of noise.
	iq := noise(int(0.6 * rate))
	for i, h := range telegrams {
		data, _ := hex.DecodeString(h)
		iq = append(iq, wmbus.Synthesize(data, wmbus.SynthOptions{Mode: wmbus.ModeT1, SampleRateHz: rate, OffsetHz: offset, NoiseRMS: 0.03, Seed: int64(i + 2)})...)
		iq = append(iq, noise(int([]float64{0.9, 0.5}[i]*rate))...)
	}
	dir := t.TempDir()
	capPath := filepath.Join(dir, "street.cu8")
	if err := siglab.WriteCapture(capPath, iq, siglab.FormatU8); err != nil {
		t.Fatal(err)
	}

	snipDir := filepath.Join(dir, "meters")
	var saver *wmbusSnippetSaver
	rateHz := rate
	_, err := decodeWMBusFile(capPath, "u8", &rateHz, func(rr float64) *wmbus.Receiver {
		var err error
		if saver, err = newWMBusSnippetSaver(snipDir, rr, 0, offset); err != nil {
			t.Fatal(err)
		}
		rx, err := wmbus.NewReceiver(wmbus.ReceiverOptions{SampleRateHz: rr, OffsetHz: offset, OnFrame: func(f wmbus.Frame, info wmbus.FrameInfo) {
			tg, _ := wmbus.ParseTelegram(f)
			saver.onFrame("t", f, tg, info)
		}})
		if err != nil {
			t.Fatal(err)
		}
		return rx
	}, func() *wmbusSnippetSaver { return saver })
	if err != nil {
		t.Fatal(err)
	}
	if err := saver.flush(true); err != nil {
		t.Fatal(err)
	}

	wavs, _ := filepath.Glob(filepath.Join(snipDir, "*.wav"))
	if len(wavs) != 2 || saver.saved != 2 {
		t.Fatalf("snippets = %v (saved %d), want one per telegram (2)", wavs, saver.saved)
	}
	got := map[string]bool{}
	for _, w := range wavs {
		meta, err := siglab.LoadMetadata(strings.TrimSuffix(w, ".wav") + ".metadata.json")
		if err != nil {
			t.Fatalf("%s: sidecar: %v", w, err)
		}
		if meta.SampleRateHz != rate || meta.System["wmbus_offset_hz"] != "25000" || meta.System["crc_ok"] != "true" {
			t.Errorf("%s: sidecar = %+v", w, meta)
		}
		snip, snipRate, err := siglab.DecodeContainerFile(w)
		if err != nil {
			t.Fatalf("%s: %v", w, err)
		}
		if snipRate != uint32(rate) {
			t.Errorf("%s: WAV rate %d, want %d", w, snipRate, uint32(rate))
		}
		if frac := float64(len(snip)) / float64(len(iq)); frac > 0.05 {
			t.Errorf("%s: snippet is %.1f%% of the capture; it should hold one telegram, not the air around it", w, 100*frac)
		}
		// Replay the snippet as `wmbus -in` does: it must decode to the
		// very telegram its sidecar names.
		var frames []wmbus.Frame
		r := 1.0
		if _, err := decodeWMBusFile(w, "u8", &r, func(rr float64) *wmbus.Receiver {
			rx, err := wmbus.NewReceiver(wmbus.ReceiverOptions{SampleRateHz: rr, OffsetHz: offset, OnFrame: func(f wmbus.Frame, _ wmbus.FrameInfo) {
				if f.CRCOK {
					frames = append(frames, f)
				}
			}})
			if err != nil {
				t.Fatal(err)
			}
			return rx
		}, nil); err != nil {
			t.Fatalf("%s: replay: %v", w, err)
		}
		if len(frames) != 1 {
			t.Fatalf("%s: replay decoded %d CRC-valid frames, want 1", w, len(frames))
		}
		h := hex.EncodeToString(frames[0].Data)
		if h != meta.System["frame_hex"] {
			t.Errorf("%s: replay decoded %s, sidecar names %s", w, h, meta.System["frame_hex"])
		}
		got[h] = true
	}
	for _, h := range telegrams {
		if !got[h] {
			t.Errorf("no snippet replays to telegram %s", h)
		}
	}
}
