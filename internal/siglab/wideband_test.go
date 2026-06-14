package siglab

import (
	"math"
	"math/cmplx"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
)

// loadFixtureIQ reads one of the committed 48 kHz centred DMR fixtures.
func loadFixtureIQ(t *testing.T, name string) []complex64 {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "cmd", "gophertrunk", "testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	pairs := len(raw) / 8
	iq := make([]complex64, pairs)
	dec, _ := FormatF32.Decoder()
	dec(raw[:pairs*8], iq)
	return iq
}

// upshift resamples a 48 kHz baseband stream to outRateHz and mixes it to the
// given offset — the inverse of the survey's per-carrier DDC, used to plant a
// known carrier inside a synthetic wide-band buffer.
func upshift(in []complex64, l, m int, offsetHz, outRateHz float64) []complex64 {
	rs := dsp.NewResampler(l, m, 16, 9.0)
	up := rs.Process(nil, in)
	step := cmplx.Exp(complex(0, 2*math.Pi*offsetHz/outRateHz))
	ph := complex(1, 0)
	out := make([]complex64, len(up))
	for i, s := range up {
		out[i] = complex64(complex128(s) * ph)
		ph *= step
	}
	return out
}

// TestSurveyWidebandFindsAndSnapsTones checks the spectrum→peaks→grid-snap front
// end (detectCarriers) on a synthetic two-tone buffer placed ~400 Hz off the
// 12.5 kHz raster, over a realistic noise floor.
func TestSurveyWidebandFindsAndSnapsTones(t *testing.T) {
	const rate = 2_000_000.0
	const center = 441_000_000
	n := 1 << 18
	rng := newLCG(1)
	iq := make([]complex64, n)
	for i := range iq { // white noise floor
		iq[i] = complex(float32(rng.normal()*0.02), float32(rng.normal()*0.02))
	}
	tone := func(freqHz, amp float64) {
		step := cmplx.Exp(complex(0, 2*math.Pi*freqHz/rate))
		ph := complex(1, 0)
		for i := range iq {
			iq[i] += complex64(complex128(complex(amp, 0)) * ph)
			ph *= step
		}
	}
	tone(300_000+400, 1.0)  // ~400 Hz off a 12.5 kHz grid point
	tone(-250_000+400, 0.7) //

	_, cands := detectCarriers(iq, WidebandConfig{SampleRateHz: rate, CenterHz: center}, 4096, 0.05)
	if len(cands) < 2 {
		t.Fatalf("detected %d carriers, want >= 2", len(cands))
	}
	// The two strongest detections are the planted tones; both must snap onto
	// the 12.5 kHz raster (offset divisible by 12.5 kHz from the 441 MHz centre).
	for _, c := range cands[:2] {
		rel := int64(c.freqHz) - center
		if mod := ((rel % 12_500) + 12_500) % 12_500; mod > 100 && mod < 12_400 {
			t.Errorf("carrier %d Hz (offset %d) not snapped to 12.5 kHz grid (residual %d)", c.freqHz, rel, mod)
		}
	}
	// The strongest two should be near the two planted carriers.
	wantAbs := []int64{300_000, -250_000}
	for _, w := range wantAbs {
		found := false
		for _, c := range cands[:min2(4, len(cands))] {
			if absI64(int64(c.freqHz)-(center+w)) < 6_250 {
				found = true
			}
		}
		if !found {
			t.Errorf("no detected carrier near offset %d Hz; got %v", w, cands)
		}
	}
}

func absI64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
func min2(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// lcg is a tiny deterministic PRNG for test noise (no math/rand import churn).
type lcg struct{ s uint64 }

func newLCG(seed uint64) *lcg { return &lcg{s: seed*6364136223846793005 + 1} }
func (g *lcg) f64() float64 {
	g.s = g.s*6364136223846793005 + 1442695040888963407
	return float64(g.s>>11) / float64(1<<53)
}

// normal returns an approximately-Gaussian sample (sum of 3 uniforms, zero mean).
func (g *lcg) normal() float64 { return (g.f64() + g.f64() + g.f64() - 1.5) }

// TestSurveyWidebandRecognizesDMRTier3 is the end-to-end proof: it plants the
// real control-channel fixture (Aloha → Tier III lock + SystemID) and the real
// voice fixture into one synthetic 192 kHz wide-band buffer at distinct
// offsets, then asserts the survey recovers two carriers, classifies one as
// control and one as voice, and rolls them into a single DMR Tier III system
// with the verdict string.
func TestSurveyWidebandRecognizesDMRTier3(t *testing.T) {
	const wideRate = 192_000.0 // 4× the 48 kHz fixtures
	const center = 441_000_000
	cc := loadFixtureIQ(t, "dmr-t3-cc.cfile")
	voice := loadFixtureIQ(t, "dmr-voice-term.cfile")

	ctrl := upshift(cc, 4, 1, +48_000, wideRate)   // control at +48 kHz
	vox := upshift(voice, 4, 1, -48_000, wideRate) // voice at -48 kHz
	n := len(ctrl)
	if len(vox) < n {
		n = len(vox)
	}
	wide := make([]complex64, n)
	for i := 0; i < n; i++ {
		wide[i] = ctrl[i] + vox[i]
	}

	res, err := SurveyWidebandIQ(wide, "dmr-t3-wideband", WidebandConfig{
		SampleRateHz: wideRate, CenterHz: center,
	})
	if err != nil {
		t.Fatalf("survey: %v", err)
	}
	t.Logf("strategy=%s carriers=%d", res.Strategy, len(res.Carriers))
	for _, c := range res.Carriers {
		t.Logf("  %d Hz (off %+0.0f) proto=%s conf=%.2f locked=%v role=%s cc=%d sysid=%d",
			c.FreqHz, c.OffsetHz, c.Protocol, c.Confidence, c.Locked, c.Role, c.ColorCode, c.SystemID)
	}

	if len(res.Carriers) != 2 {
		t.Fatalf("found %d carriers, want 2", len(res.Carriers))
	}
	var control, voiceN int
	for _, c := range res.Carriers {
		switch c.Role {
		case RoleControl:
			control++
		case RoleVoice:
			voiceN++
		}
	}
	if control != 1 {
		t.Errorf("control carriers = %d, want 1", control)
	}
	if voiceN != 1 {
		t.Errorf("voice carriers = %d, want 1", voiceN)
	}

	if len(res.Systems) != 1 {
		t.Fatalf("rolled up %d systems, want 1: %+v", len(res.Systems), res.Systems)
	}
	sys := res.Systems[0]
	if !sys.Tier3 {
		t.Errorf("system not recognized as Tier III: %+v", sys)
	}
	if sys.ControlCount != 1 || sys.VoiceCount != 1 {
		t.Errorf("rollup control=%d voice=%d, want 1/1", sys.ControlCount, sys.VoiceCount)
	}
	if sys.SystemID != 61760 {
		t.Errorf("SystemID = %d, want 61760 (Aloha)", sys.SystemID)
	}
	if !strings.Contains(sys.Verdict, "DMR Tier III trunked system") {
		t.Errorf("verdict = %q, want it to name the Tier III system", sys.Verdict)
	}
	t.Logf("verdict: %s", sys.Verdict)
}
