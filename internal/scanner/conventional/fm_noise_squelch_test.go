package conventional

import (
	"encoding/binary"
	"math"
	"math/rand"
	"os"
	"sort"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

// The FM noise squelch (issue #1329), measured on real air first: the
// receiver noise and the FM transmissions in the repository's captures, at
// the rates the scanner really runs at. Synthetic signals fill in what no
// capture covers (a set carrier-to-noise ratio, other sample rates).

const (
	// mmrCapture is a 1.09 s RTL-SDR capture at 2.4 MS/s (complex float32)
	// holding a P25 Phase 1 control channel 37.5 kHz above its centre, a
	// Phase 2 (H-DQPSK) carrier 637.5 kHz above it, and nothing else.
	mmrCapture     = "../../../samples/mmr/mmr-s9-1.cfile"
	mmrRate        = 2_400_000
	mmrP25Offset   = 37_500
	mmrPhase2Offs  = 637_500
	mdc1200Slice   = "../../radio/mdc1200/afsk/testdata/mdc1200_unit1777_pttid_end_48k.cs16"
	radtelTone     = "testdata/ctcss100_radtel_447100k_48k.cs16"
	radtelNoTone   = "testdata/ctcss_none_radtel_447100k_48k.cs16"
	kricNoiseSlice = "testdata/am_kric_126400k_noise_48k.cs16"
	kricCarrier    = "testdata/am_kric_126400k_carrier_48k.cs16"
)

// mmrEmptyOffsets are offsets of the capture with no signal on them.
var mmrEmptyOffsets = []float64{-900_000, -500_000, -300_000, 150_000, 300_000, 900_000}

func loadCF32(t testing.TB, path string) []complex64 {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	iq := make([]complex64, len(b)/8)
	for i := range iq {
		iq[i] = complex(
			math.Float32frombits(binary.LittleEndian.Uint32(b[8*i:])),
			math.Float32frombits(binary.LittleEndian.Uint32(b[8*i+4:])))
	}
	return iq
}

// shiftHz moves the signal offsetHz above the stream's centre to its centre,
// as tuning the SDR there would, and scales it by gain.
func shiftHz(iq []complex64, offsetHz, rate, gain float64) []complex64 {
	out := make([]complex64, len(iq))
	for i, s := range iq {
		ph := -2 * math.Pi * offsetHz * float64(i) / rate
		c, sn := math.Cos(ph)*gain, math.Sin(ph)*gain
		out[i] = complex(
			float32(float64(real(s))*c-float64(imag(s))*sn),
			float32(float64(real(s))*sn+float64(imag(s))*c))
	}
	return out
}

func whiteNoise(rng *rand.Rand, n int, sigma float64) []complex64 {
	out := make([]complex64, n)
	for i := range out {
		out[i] = complex(float32(rng.NormFloat64()*sigma), float32(rng.NormFloat64()*sigma))
	}
	return out
}

// noiseRatios runs an fm channel's meter over iq in chunks, as the scanner
// does, and returns every window's ratio in order plus the last level.
func noiseRatios(t testing.TB, iq []complex64, rate float64, chunk int) (ratios []float64, levelDb float64) {
	t.Helper()
	pm := buildChannelPowerMeter(Channel{Mode: "fm"}, rate)
	if pm == nil || pm.noise == nil {
		t.Fatalf("no noise squelch built for an fm channel at %v Hz", rate)
	}
	last := math.Inf(1)
	for s := 0; s+chunk <= len(iq); s += chunk {
		levelDb = pm.process(iq[s : s+chunk])
		if r := pm.noise.ratio(); r != last && !math.IsInf(r, 1) {
			ratios = append(ratios, r)
			last = r
		}
	}
	if len(ratios) == 0 {
		t.Fatal("the meter never completed a window")
	}
	return ratios, levelDb
}

func ratioRange(r []float64) (lo, med, hi float64) {
	s := append([]float64(nil), r...)
	sort.Float64s(s)
	return s[0], s[len(s)/2], s[len(s)-1]
}

// Real receiver noise reads 1 whatever its level: the RTL-SDR's own at six
// empty offsets of a 2.4 MS/s capture (about -53 dBFS in the channel), the
// same raised 40 dB and lowered 40 dB, and the air-band receiver noise that
// follows the transmission in the #1219 capture. It never comes near the
// hold threshold, let alone the open one.
func TestFMNoiseSquelchReadsOneOnRealReceiverNoise(t *testing.T) {
	check := func(name string, iq []complex64, rate float64, chunk int) {
		r, lvl := noiseRatios(t, iq, rate, chunk)
		lo, med, hi := ratioRange(r)
		t.Logf("%-28s level %6.1f dBFS  ratio %.2f..%.2f, median %.2f (%d windows)", name, lvl, lo, hi, med, len(r))
		if med < 0.93 || med > 1.08 {
			t.Errorf("%s: median ratio %.3f, want about 1 on noise", name, med)
		}
		if lo <= fmNoiseHoldRatio+0.05 {
			t.Errorf("%s: ratio fell to %.3f on noise, want well above the hold threshold %.2f", name, lo, fmNoiseHoldRatio)
		}
	}
	// Half a second of the capture per offset: 48 windows each.
	mmr := loadCF32(t, mmrCapture)[:mmrRate/2]
	for _, off := range mmrEmptyOffsets {
		check("rtl-sdr noise", shiftHz(mmr, off, mmrRate, 1), mmrRate, 8192)
	}
	check("rtl-sdr noise +40 dB", shiftHz(mmr, -500_000, mmrRate, 100), mmrRate, 8192)
	check("rtl-sdr noise -40 dB", shiftHz(mmr, -500_000, mmrRate, 0.01), mmrRate, 8192)
	check("air-band noise, 48 kHz", readCS16Fixture(t, kricNoiseSlice), 48_000, 164)
	check("air-band noise, 2.4 MS/s", loadRealAirAt2400k(t, "am_kric_126400k_noise_48k.cs16")[:mmrRate/2], mmrRate, 16384)
}

// Every real transmission in the repository's captures quiets the
// discriminator far past the open threshold: two NFM handhelds (with and
// without CTCSS), an MDC1200 burst from a radio that clipped the ADC, an
// analog test tone with 57 % of its samples at the rail, an AM carrier, a
// P25 C4FM control channel, and the least FM-like of them, a linear
// H-DQPSK carrier. worst is the highest ratio allowed in any window while
// the carrier is up.
func TestFMNoiseSquelchOpensOnRealTransmissions(t *testing.T) {
	// Half a second of each 2.4 MS/s stream is 48 windows.
	const half = mmrRate / 2
	mmr := loadCF32(t, mmrCapture)[:half]
	for _, tc := range []struct {
		name  string
		iq    []complex64
		rate  float64
		chunk int
		worst float64
		// whole: the carrier is up for the whole slice. Otherwise the
		// slice ends in noise and only its quietest half is judged.
		whole bool
	}{
		{"radtel NFM + CTCSS, 48 kHz", readCS16Fixture(t, radtelTone), 48_000, 164, 0.01, true},
		{"radtel NFM + CTCSS, 2.4 MS/s", loadRealAirAt2400k(t, "ctcss100_radtel_447100k_48k.cs16")[:half], mmrRate, 16384, 0.01, true},
		{"radtel NFM, 2.4 MS/s", loadRealAirAt2400k(t, "ctcss_none_radtel_447100k_48k.cs16")[:half], mmrRate, 16384, 0.01, true},
		{"analog tone, 57 % clipped", readCS16Fixture(t, analogToneFixture), 48_000, 164, 0.15, true},
		{"AM carrier", readCS16Fixture(t, kricCarrier), 48_000, 164, 0.01, true},
		{"MDC1200 burst (clipped radio)", readCS16Fixture(t, mdc1200Slice), 48_000, 164, 0.01, false},
		{"P25 C4FM control channel", shiftHz(mmr, mmrP25Offset, mmrRate, 1), mmrRate, 8192, 0.01, true},
		{"P25 Phase 2 H-DQPSK", shiftHz(mmr, mmrPhase2Offs, mmrRate, 1), mmrRate, 8192, 0.25, true},
	} {
		r, lvl := noiseRatios(t, tc.iq, tc.rate, tc.chunk)
		if !tc.whole {
			sort.Float64s(r)
			r = r[:len(r)/2]
		}
		lo, med, hi := ratioRange(r)
		t.Logf("%-30s level %6.1f dBFS  ratio %.4f..%.4f, median %.4f", tc.name, lvl, lo, hi, med)
		if hi > tc.worst {
			t.Errorf("%s: ratio reached %.3f, want <= %.2f", tc.name, hi, tc.worst)
		}
		if hi >= fmNoiseOpenRatio {
			t.Errorf("%s: ratio %.3f would not open the squelch (threshold %.2f)", tc.name, hi, fmNoiseOpenRatio)
		}
	}
}

// The meter reports a carrier within fmNoiseLatency of its arrival, from any
// chunk alignment: real noise, then the real NFM transmission, at 2.4 MS/s.
func TestFMNoiseSquelchLatency(t *testing.T) {
	noise := shiftHz(loadCF32(t, mmrCapture)[:mmrRate/5], -500_000, mmrRate, 6)
	voice := loadRealAirAt2400k(t, "ctcss_none_radtel_447100k_48k.cs16")[:mmrRate/5]
	for _, chunk := range []int{4096, 8192, 16384, 10_000} {
		pm := buildChannelPowerMeter(Channel{Mode: "fm"}, mmrRate)
		iq := append(append([]complex64(nil), noise...), voice...)
		opened := -1
		for s := 0; s+chunk <= len(iq); s += chunk {
			pm.process(iq[s : s+chunk])
			r := pm.noise.ratio()
			if s+chunk <= len(noise) {
				if r <= fmNoiseOpenRatio {
					t.Fatalf("chunk %d: opened on the noise ahead of the carrier (ratio %.3f)", chunk, r)
				}
				continue
			}
			if r <= fmNoiseOpenRatio {
				opened = s + chunk - len(noise)
				break
			}
		}
		if opened < 0 {
			t.Fatalf("chunk %d: never opened on the transmission", chunk)
		}
		got := float64(opened) / mmrRate
		// One chunk of slack: the reading is only looked at between chunks.
		if limit := fmNoiseLatency.Seconds() + float64(chunk)/mmrRate; got > limit {
			t.Errorf("chunk %d: opened %.1f ms after the carrier, want within %.1f ms", chunk, got*1e3, limit*1e3)
		}
	}
}

// fmVoice is an FM transmission at 48 kHz: speech-shaped noise (350 Hz to
// 2.8 kHz, with a syllable envelope) at peak deviation devHz.
func fmVoice(n int, devHz float64, seed int64) []complex64 {
	const rate = 48_000.0
	rng := rand.New(rand.NewSource(seed))
	mod := make([]float64, n)
	for i := range mod {
		mod[i] = rng.NormFloat64()
	}
	filter.NewLowPass(rate, 2800).Process(mod)
	filter.NewLowPass(rate, 2800).Process(mod)
	filter.NewHighPass(rate, 350).Process(mod)
	mag := make([]float64, n)
	for i := range mod {
		mod[i] *= 0.5 + 0.5*math.Sin(2*math.Pi*3*float64(i)/rate)
		mag[i] = math.Abs(mod[i])
	}
	sort.Float64s(mag)
	peak := mag[int(0.999*float64(n))]
	out := make([]complex64, n)
	var ph float64
	for i := range out {
		ph += 2 * math.Pi * devHz * math.Max(-1, math.Min(1, mod[i]/peak)) / rate
		out[i] = complex(float32(math.Cos(ph)), float32(math.Sin(ph)))
	}
	return out
}

// addNoiseAtCN adds white noise to a 48 kHz signal so its carrier-to-noise
// ratio in the ±8 kHz channel is cnDb.
func addNoiseAtCN(sig []complex64, cnDb float64, seed int64) []complex64 {
	var p float64
	for _, s := range sig {
		p += float64(real(s))*float64(real(s)) + float64(imag(s))*float64(imag(s))
	}
	p /= float64(len(sig))
	// Noise power in 16 kHz of the 48 kHz stream is a third of the total.
	sigma := math.Sqrt(p / math.Pow(10, cnDb/10) * 3 / 2)
	rng := rand.New(rand.NewSource(seed))
	out := make([]complex64, len(sig))
	for i, s := range sig {
		out[i] = s + complex(float32(rng.NormFloat64()*sigma), float32(rng.NormFloat64()*sigma))
	}
	return out
}

// How weak a signal still opens the squelch. The ratio follows the
// carrier-to-noise ratio in the channel, the same for a real handheld, NFM
// and wide FM speech: a signal 4 dB over the noise is open in every window,
// one at 2 dB in most, and noise is never open. 4 dB is under the FM
// threshold, so the gate takes nothing that could have been understood.
func TestFMNoiseSquelchFollowsCarrierToNoise(t *testing.T) {
	sources := []struct {
		name string
		iq   []complex64
	}{
		{"real NFM handheld", readCS16Fixture(t, radtelTone)},
		{"NFM speech, 2.5 kHz deviation", fmVoice(48_000*3, 2500, 3)},
		{"FM speech, 5 kHz deviation", fmVoice(48_000*3, 5000, 4)},
	}
	for _, src := range sources {
		var prev float64
		for _, tc := range []struct {
			cnDb         float64
			medLo, medHi float64
			openEvery    bool // every window at or under the open threshold
			openMostly   bool // the median window is
		}{
			{20, 0.001, 0.006, true, true},
			{10, 0.02, 0.045, true, true},
			{6, 0.07, 0.14, true, true},
			{4, 0.15, 0.27, true, true},
			{2, 0.28, 0.43, false, true},
			{0, 0.44, 0.60, false, false},
		} {
			r, _ := noiseRatios(t, addNoiseAtCN(src.iq, tc.cnDb, 9), 48_000, 164)
			_, med, hi := ratioRange(r)
			if med < tc.medLo || med > tc.medHi {
				t.Errorf("%s at %g dB: median ratio %.3f, want %.3f..%.3f", src.name, tc.cnDb, med, tc.medLo, tc.medHi)
			}
			if med <= prev {
				t.Errorf("%s: ratio did not rise as the signal weakened (%.3f at %g dB after %.3f)", src.name, med, tc.cnDb, prev)
			}
			prev = med
			if tc.openEvery && hi > fmNoiseOpenRatio {
				t.Errorf("%s at %g dB: a window read %.3f, want every window open (<= %.2f)", src.name, tc.cnDb, hi, fmNoiseOpenRatio)
			}
			if tc.openMostly && med > fmNoiseOpenRatio {
				t.Errorf("%s at %g dB: median %.3f, want open (<= %.2f)", src.name, tc.cnDb, med, fmNoiseOpenRatio)
			}
		}
	}
}

// The reference is calibrated per rate, so noise reads 1 at every rate a
// stream can arrive at, and at any chunk size; below fmNoiseMinRateHz there
// is no room for the measured band and no meter.
func TestFMNoiseSquelchAtEveryRate(t *testing.T) {
	rng := rand.New(rand.NewSource(5))
	for _, tc := range []struct {
		rate  float64
		chunk int
	}{
		{16_000, 512}, {25_000, 512}, {48_000, 164}, {48_000, 4096}, {60_000, 1000},
		{250_000, 4096}, {2_048_000, 8192}, {2_400_000, 16384}, {3_000_000, 8192}, {4_800_000, 8192},
	} {
		// A second of stream at the SDR rates, two at the slow ones: at
		// least 98 windows either way.
		seconds := 2.0
		if tc.rate > 100_000 {
			seconds = 1
		}
		r, _ := noiseRatios(t, whiteNoise(rng, int(seconds*tc.rate), 0.05), tc.rate, tc.chunk)
		lo, med, _ := ratioRange(r)
		if med < 0.93 || med > 1.08 || lo <= fmNoiseHoldRatio {
			t.Errorf("rate %v, chunk %d: ratio median %.3f, minimum %.3f; want about 1 and never under %.2f",
				tc.rate, tc.chunk, med, lo, fmNoiseHoldRatio)
		}
	}
	if pm := buildChannelPowerMeter(Channel{Mode: "fm"}, 8_000); pm == nil || pm.noise != nil {
		t.Errorf("an 8 kHz stream has no room for the noise band: want a level meter with no noise squelch")
	}
}

// A minute of noise, 6 000 windows: none reaches the hold threshold.
// (Two hours, 720 000 windows, gave a minimum of 0.66.)
func TestFMNoiseMeterNoiseNeverReadsQuiet(t *testing.T) {
	if testing.Short() {
		t.Skip("a minute of noise")
	}
	rng := rand.New(rand.NewSource(77))
	pm := buildChannelPowerMeter(Channel{Mode: "fm"}, 48_000)
	buf := make([]complex64, 480)
	lowest := math.Inf(1)
	for i := 0; i < 60*100; i++ {
		for j := range buf {
			buf[j] = complex(float32(rng.NormFloat64()*0.1), float32(rng.NormFloat64()*0.1))
		}
		pm.process(buf)
		lowest = math.Min(lowest, pm.noise.ratio())
	}
	t.Logf("lowest 20 ms window in a minute of noise: %.3f", lowest)
	if lowest <= fmNoiseHoldRatio {
		t.Errorf("a noise window read %.3f, want always above the hold threshold %.2f", lowest, fmNoiseHoldRatio)
	}
}

// The noise squelch belongs to analog FM channels and can be switched off;
// a reset forgets the previous visit, so a channel cannot open on the last
// one's reading.
func TestFMNoiseSquelchBuiltOnlyForAnalogFM(t *testing.T) {
	for _, tc := range []struct {
		ch   Channel
		want bool
	}{
		{Channel{Mode: "fm"}, true},
		{Channel{Mode: "nfm"}, true},
		{Channel{Mode: "fm", NoiseSquelchOff: true}, false},
		{Channel{Mode: ModeP25}, false},
		{Channel{Mode: ModeDMR}, false},
		{Channel{Mode: ModeYSF}, false},
	} {
		pm := buildChannelPowerMeter(tc.ch, 2_400_000)
		if pm == nil {
			t.Fatalf("mode %q: no level meter", tc.ch.Mode)
		}
		if got := pm.noise != nil; got != tc.want {
			t.Errorf("mode %q (off=%v): noise squelch built = %v, want %v", tc.ch.Mode, tc.ch.NoiseSquelchOff, got, tc.want)
		}
	}
	if pm := buildChannelPowerMeter(Channel{Mode: ModeAM}, 2_400_000); pm != nil {
		t.Errorf("an AM channel squelches on carrier-to-noise: want no level meter")
	}

	pm := buildChannelPowerMeter(Channel{Mode: "fm"}, 48_000)
	pm.process(readCS16Fixture(t, radtelNoTone))
	if r := pm.noise.ratio(); r > 0.01 {
		t.Fatalf("ratio %.3f on the real carrier, want quiet", r)
	}
	pm.reset()
	if r := pm.noise.ratio(); !math.IsInf(r, 1) {
		t.Errorf("ratio %.3f straight after a reset, want +Inf until a window is measured", r)
	}
	if _, ok := pm.noise.quietingDb(); ok {
		t.Errorf("quietingDb reported a reading straight after a reset")
	}
	if q, ok := pm.noise.shownQuietingDb(); !ok || q < 20 {
		t.Errorf("shownQuietingDb = %.1f, %v after a reset; want the last visit's reading (over 20 dB)", q, ok)
	}
}

// wideFM is an FM carrier at 2.4 MS/s modulated with noise up to 12 kHz
// (music-like) at peak deviation devHz, 40 dB over the span's noise.
func wideFM(seconds, devHz float64, seed int64) []complex64 {
	const rate = float64(mmrRate)
	n := int(rate * seconds)
	rng := rand.New(rand.NewSource(seed))
	mod := make([]float64, n)
	for i := range mod {
		mod[i] = rng.NormFloat64()
	}
	filter.NewLowPass(rate, 12_000).Process(mod)
	filter.NewLowPass(rate, 12_000).Process(mod)
	mag := make([]float64, n)
	for i, v := range mod {
		mag[i] = math.Abs(v)
	}
	sort.Float64s(mag)
	peak := mag[int(0.999*float64(n))]
	out := make([]complex64, n)
	var ph float64
	for i := range out {
		ph += 2 * math.Pi * devHz * math.Max(-1, math.Min(1, mod[i]/peak)) / rate
		out[i] = complex(
			float32(0.3*math.Cos(ph)+rng.NormFloat64()*0.003),
			float32(0.3*math.Sin(ph)+rng.NormFloat64()*0.003))
	}
	return out
}

// What the ±8 kHz channel makes of FM wider than itself. A broadcast station
// (±75 kHz) sweeps through the channel and reads as noise in every window:
// the squelch stays shut on it, steadily, however strong it is (the scan
// list has no wide-FM mode, and its voice chain never could demodulate
// one). A ±12 kHz signal, wider than any two-way radio's, still opens in
// every window, so nothing in between two-way FM and broadcast chatters.
func TestFMNoiseSquelchOnFMWiderThanTheChannel(t *testing.T) {
	r, lvl := noiseRatios(t, wideFM(0.4, 75_000, 3), mmrRate, 8192)
	lo, med, _ := ratioRange(r)
	t.Logf("broadcast FM, 75 kHz deviation: level %.1f dBFS, ratio median %.2f, minimum %.2f", lvl, med, lo)
	if lo <= fmNoiseHoldRatio {
		t.Errorf("a window read %.3f on broadcast FM, want always above the hold threshold %.2f (shut, without chatter)", lo, fmNoiseHoldRatio)
	}
	r, lvl = noiseRatios(t, wideFM(0.4, 12_000, 4), mmrRate, 8192)
	_, med, hi := ratioRange(r)
	t.Logf("FM, 12 kHz deviation: level %.1f dBFS, ratio median %.2f, maximum %.2f", lvl, med, hi)
	if hi > fmNoiseOpenRatio {
		t.Errorf("a window read %.3f on ±12 kHz FM, want every window open (<= %.2f)", hi, fmNoiseOpenRatio)
	}
}
