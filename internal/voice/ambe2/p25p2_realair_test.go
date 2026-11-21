package ambe2

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/voice"
)

// The first P25 Phase 2 voice GopherTrunk decoded on air (18 Nov 2026, a
// clear group call on a 867 MHz Phase 2 site, sent by a Discord reporter
// who heard "garbled noise, as if it is encrypted" where DSD+ played the
// same system fine). testdata/p25p2-voice.raw is that call's .raw sidecar:
// 118 frames, 7 bytes each, exactly what the composer's Phase 2 chain
// hands the recorder (PN44 descrambled, Golay decoded, 49 bits MSB first).
//
// testdata/p25p2-voice.mbelib2450.txt is an independent decoder's reading
// of the same bytes: szechyjs/mbelib 1.3.0 mbe_processAmbe2450Data, one
// line per frame — w0, L, voiced band count, gamma, PCM RMS in dB.
//
// The frames are AMBE+2 3600x2450, the P25 half-rate vocoder
// (TIA-102.BABA-1), the same one DMR and NXDN carry. The recorder used to
// hand "p25-phase2" to the base "ambe2" decoder, which is the 3600x2400
// table set D-STAR uses: the two layouts share the top four pitch bits and
// nothing else, so every call came out as speech-shaped noise.

type mbelibFrame struct {
	w0     float64
	l      int
	voiced int
	gamma  float64
	rmsDb  float64
}

func loadP25P2RealAir(t *testing.T) (frames [][]byte, ref []mbelibFrame) {
	t.Helper()
	raw, err := os.ReadFile("testdata/p25p2-voice.raw")
	if err != nil {
		t.Fatalf("real-air fixture: %v", err)
	}
	for i := 0; i+FrameBytes <= len(raw); i += FrameBytes {
		frames = append(frames, raw[i:i+FrameBytes])
	}
	f, err := os.Open("testdata/p25p2-voice.mbelib2450.txt")
	if err != nil {
		t.Fatalf("mbelib reference: %v", err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var m mbelibFrame
		if _, err := fmt.Sscan(line, &m.w0, &m.l, &m.voiced, &m.gamma, &m.rmsDb); err != nil {
			t.Fatalf("mbelib reference line %q: %v", line, err)
		}
		ref = append(ref, m)
	}
	if len(frames) == 0 || len(frames) != len(ref) {
		t.Fatalf("fixture has %d frames, reference %d lines", len(frames), len(ref))
	}
	return frames, ref
}

// frameRMSDb is the reference file's level measure: 10·log10(mean square + 1)
// over one 160-sample frame.
func frameRMSDb(pcm []int16) float64 {
	var s float64
	for _, v := range pcm {
		s += float64(v) * float64(v)
	}
	return 10 * math.Log10(s/float64(len(pcm))+1)
}

func pearson(a, b []float64) float64 {
	n := float64(len(a))
	var ma, mb float64
	for i := range a {
		ma += a[i]
		mb += b[i]
	}
	ma /= n
	mb /= n
	var num, da, db float64
	for i := range a {
		num += (a[i] - ma) * (b[i] - mb)
		da += (a[i] - ma) * (a[i] - ma)
		db += (b[i] - mb) * (b[i] - mb)
	}
	if da == 0 || db == 0 {
		return 0
	}
	return num / math.Sqrt(da*db)
}

// envelopeVsMbelib decodes the fixture through the named registry vocoder
// and correlates its per-frame level with the mbelib 2450 decode's.
func envelopeVsMbelib(t *testing.T, name string) float64 {
	t.Helper()
	frames, ref := loadP25P2RealAir(t)
	v, err := voice.DefaultRegistry.New(name)
	if err != nil {
		t.Fatalf("vocoder %q: %v", name, err)
	}
	defer v.Close()
	if v.FrameSize() != FrameBytes {
		t.Fatalf("vocoder %q takes %d-byte frames; the Phase 2 chain writes %d", name, v.FrameSize(), FrameBytes)
	}
	got := make([]float64, len(frames))
	want := make([]float64, len(frames))
	for i, fr := range frames {
		pcm, err := v.Decode(fr)
		if err != nil {
			t.Fatalf("%s frame %d: %v", name, i, err)
		}
		got[i] = frameRMSDb(pcm)
		want[i] = ref[i].rmsDb
	}
	return pearson(got, want)
}

// TestP25Phase2DefaultVocoderDecodesRealAirSpeech is the regression for the
// "garbled, as if encrypted" report: whatever the recorder picks for
// "p25-phase2" has to render real on-air Phase 2 frames the way the
// reference half-rate decoder does. With the 3600x2400 tables the level
// track is unrelated to the reference's (the gain index is read from the
// wrong bits); with the 3600x2450 tables it follows it frame for frame.
func TestP25Phase2DefaultVocoderDecodesRealAirSpeech(t *testing.T) {
	name, ok := voice.DefaultVocoderForProtocol()["p25-phase2"]
	if !ok {
		t.Fatal(`no default vocoder for "p25-phase2"`)
	}
	r := envelopeVsMbelib(t, name)
	t.Logf("p25-phase2 → %q: level-track correlation with mbelib 3600x2450 = %.3f", name, r)
	if r < 0.85 {
		t.Errorf("p25-phase2 decodes through %q, whose audio does not follow the reference half-rate decode of the same on-air frames (correlation %.3f, want ≥ 0.85)", name, r)
	}
	// The 3600x2400 decoder on the same frames is the failure the reporter
	// heard; keep it measurably apart so this test cannot pass by accident.
	if wrong := envelopeVsMbelib(t, VocoderName); wrong > 0.5 {
		t.Errorf("3600x2400 decode of 3600x2450 frames correlates %.3f with the reference; the fixture no longer separates the two table sets", wrong)
	}
}

// TestP25Phase2RealAirFramesUnpackLikeMbelib2450 pins the model parameters
// of every frame of the call against mbelib: pitch, harmonic count, voicing
// and the gain recursion gamma = ΔΓ + ½·gamma_prev. A second real-air source
// for the 2450 tables beside the DMR fixture (testdata/dmr-voice.raw).
func TestP25Phase2RealAirFramesUnpackLikeMbelib2450(t *testing.T) {
	frames, ref := loadP25P2RealAir(t)
	var gamma float64
	voicedFrames := 0
	for i, fr := range frames {
		p, err := unpackParams2450(unpackInfoBits(fr))
		if err != nil {
			t.Fatalf("frame %d: %v", i, err)
		}
		if p.Tone || p.Silent {
			t.Fatalf("frame %d: tone=%v silent=%v, mbelib decodes a voice-path frame", i, p.Tone, p.Silent)
		}
		gamma = p.DeltaGamma + 0.5*gamma
		voiced := 0
		for l := 1; l <= p.L; l++ {
			voiced += p.Vl[l]
		}
		if voiced > 0 {
			voicedFrames++
		}
		m := ref[i]
		if p.L != m.l || voiced != m.voiced || math.Abs(p.W0-m.w0) > 2e-6 || math.Abs(gamma-m.gamma) > 2e-4 {
			t.Errorf("frame %d: w0=%.6f L=%d voiced=%d gamma=%.4f, mbelib w0=%.6f L=%d voiced=%d gamma=%.4f",
				i, p.W0, p.L, voiced, gamma, m.w0, m.l, m.voiced, m.gamma)
		}
	}
	// A speech call: most frames carry voiced bands (88 of 118 here).
	if voicedFrames < len(frames)/2 {
		t.Errorf("%d of %d frames have a voiced band; the fixture should be a speech call", voicedFrames, len(frames))
	}
}
