package ambe2

import (
	"encoding/hex"
	"math"
	"os"
	"testing"
)

// Literal frames from the committed #644 fixture (testdata/dmr-voice.raw,
// 7 bytes per frame, MSB-first) and reference values from an independent
// decoder: szechyjs/mbelib 1.3.0 mbe_decodeAmbe2450Parms (ambe3600x2450.c)
// run over the same bytes; mbelib-neo (the dsd-neo vocoder, JMBE-derived)
// reproduces every gamma value below to the printed precision. Literal
// vectors cross-checked against an independent decoder are the only thing
// that catches constant/table drift — a round-trip test cannot.
const (
	dmrSampleFrame0  = "b0b9243e0bc180" // voice frame, b0=91, L=37
	dmrSampleFrame19 = "f84902a06c3d80" // first AMBE+2 silence frame, b0=124
)

// mbelibFrame0Tl is Tl[1..37] for dmrSampleFrame0 straight out of
// mbe_decodeAmbe2450Parms (before the log2Ml prediction), the stateless
// part of the unpack: b0..b8 → tables → PRBA/HOC inverse DCTs.
var mbelibFrame0Tl = []float64{
	-0.28504, 0.04614, -0.06424, 0.02870, 0.85792, 1.17561, 0.30392, -0.66631,
	-0.15903, -0.09478, -0.04798, 0.06767, 0.25219, 0.35697, 0.44166, 0.57037,
	1.32356, 0.83144, 0.19813, -0.21465, -0.23297, 0.13734, 0.64194, 0.81750,
	0.45392, -0.03082, -0.05036, -0.34765, -0.56400, -0.54765, -0.53555, -0.63647,
	-0.62833, -0.48978, -0.67395, -1.44068, -2.23123,
}

func mustFrame(t *testing.T, h string) []byte {
	t.Helper()
	b, err := hex.DecodeString(h)
	if err != nil || len(b) != FrameBytes {
		t.Fatalf("bad literal frame %q: %v", h, err)
	}
	return b
}

// TestUnpack2450TlMatchesMbelibFrame0 pins the stateless spectral-residual
// reconstruction of the 3600x2450 unpack against the mbelib reference on a
// real over-the-air frame: same tables, same DCTs, to 1e-5.
func TestUnpack2450TlMatchesMbelibFrame0(t *testing.T) {
	p, err := unpackParams2450(unpackInfoBits(mustFrame(t, dmrSampleFrame0)))
	if err != nil {
		t.Fatal(err)
	}
	if p.B0 != 91 || p.L != len(mbelibFrame0Tl) || p.SilenceFrame {
		t.Fatalf("b0=%d L=%d silence=%v, want b0=91 L=%d voice", p.B0, p.L, p.SilenceFrame, len(mbelibFrame0Tl))
	}
	for l, want := range mbelibFrame0Tl {
		if got := p.Tl[l+1]; math.Abs(got-want) > 2e-5 {
			t.Errorf("Tl[%d] = %.5f, mbelib %.5f", l+1, got, want)
		}
	}
}

// TestUnpack2450SilenceFrameIsUnvoicedNoiseFrame pins how an AMBE+2 silence
// frame (b0 124/125) unpacks: as mbelib's fixed all-unvoiced model
// (w0 = 2π/32, L = 14) carrying the frame's own gain delta and spectral
// residuals — not as a Silent short-circuit. Fails against the old unpack,
// which returned Params{Silent: true} with no gain and no envelope.
func TestUnpack2450SilenceFrameIsUnvoicedNoiseFrame(t *testing.T) {
	info := unpackInfoBits(mustFrame(t, dmrSampleFrame19))
	p, err := unpackParams2450(info)
	if err != nil {
		t.Fatal(err)
	}
	if p.B0 != 124 || !p.SilenceFrame {
		t.Fatalf("b0=%d silence=%v, want a b0=124 silence frame", p.B0, p.SilenceFrame)
	}
	if p.Silent || p.Tone {
		t.Fatalf("silence frame flagged Silent=%v Tone=%v; the references decode it as a voice-path frame", p.Silent, p.Tone)
	}
	if p.L != 14 || math.Abs(p.W0-2*math.Pi/32) > 1e-12 {
		t.Errorf("model L=%d w0=%.6f, want L=14 w0=2π/32=%.6f (mbelib silence model)", p.L, p.W0, 2*math.Pi/32)
	}
	for l := 1; l <= p.L; l++ {
		if p.Vl[l] != 0 {
			t.Errorf("Vl[%d]=%d, want 0 (all-unvoiced)", l, p.Vl[l])
		}
	}
	// The gain delta must come from the frame's b2 bits, exactly as for a
	// voice frame.
	b2 := int(info[8])<<4 | int(info[9])<<3 | int(info[10])<<2 | int(info[11])<<1 | int(info[36])
	if p.B2 != b2 || p.DeltaGamma != dmrDg[b2] {
		t.Errorf("DeltaGamma=%.4f (b2=%d), want dmrDg[%d]=%.4f", p.DeltaGamma, p.B2, b2, dmrDg[b2])
	}
	var nonZero int
	for l := 1; l <= p.L; l++ {
		if p.Tl[l] != 0 {
			nonZero++
		}
	}
	if nonZero == 0 {
		t.Errorf("no spectral residuals decoded for the silence frame")
	}
}

// TestDecodeDMRSampleGammaTracksReference is the cross-frame regression for
// the #644 onset defect. The fixture is 47% silence frames in runs of up to
// 63 (~1.3 s). Both reference decoders carry the gain predictor through
// them (gamma = ΔΓ + 0.5·γ_prev on every frame, silence included); the old
// decoder reset gamma to 0 on each silence frame, so the first voice frame
// after every pause landed up to 4 log2 units (≈24 dB) below the
// references (frame 95: 4.37 vs 8.40; frame 308: 5.78 vs 10.03). The
// expected values are cur_mp->gamma from mbelib after decoding the named
// frame; mbelib-neo agrees on every one.
func TestDecodeDMRSampleGammaTracksReference(t *testing.T) {
	raw, err := os.ReadFile("testdata/dmr-voice.raw")
	if err != nil {
		t.Skipf("no #644 sample at testdata/dmr-voice.raw: %v", err)
	}
	want := map[int]float64{
		0:   4.22829,  // first voice frame (no history)
		19:  3.78742,  // first silence frame: its own gamma
		63:  -2.37824, // onset after a 44-frame silence run
		95:  8.40044,  // onset after a 20-frame run (old decoder: 4.37)
		103: 7.36772,
		162: -2.62281,
		169: 2.68536,
		279: -2.64333,
		308: 10.03160, // onset after a 63-frame run (old decoder: 5.78)
	}
	d := NewDMR()
	for i := 0; i+FrameBytes <= len(raw); i += FrameBytes {
		frame := i / FrameBytes
		if _, err := d.Decode(raw[i : i+FrameBytes]); err != nil {
			t.Fatalf("decode frame %d: %v", frame, err)
		}
		if w, ok := want[frame]; ok {
			if math.Abs(d.prevGamma-w) > 2e-3 {
				t.Errorf("frame %d: gamma %.5f, reference %.5f", frame, d.prevGamma, w)
			}
		}
	}
}
