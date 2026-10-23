package conventional

import (
	"context"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

const amTestRate = 2_400_000

// amIQ is an AM signal (800 Hz tone, depth m, carrier offset offHz) at the
// given carrier-to-noise ratio in one meter bin (~188 Hz), over complex
// white noise of per-axis standard deviation noiseSD, at the scanner's
// 2.4 MS/s. cnBinDb < -100 means noise only.
func amIQ(r *rand.Rand, n int, noiseSD, cnBinDb, m, offHz float64) []complex64 {
	noiseBin := 2 * noiseSD * noiseSD * (amCNRefRateHz / amCNFFTSize) / amTestRate
	amp := 0.0
	if cnBinDb > -100 {
		amp = math.Sqrt(noiseBin * math.Pow(10, cnBinDb/10))
	}
	x := make([]complex64, n)
	for i := range x {
		t := float64(i) / amTestRate
		env := amp * (1 + m*math.Cos(2*math.Pi*800*t))
		ph := 2 * math.Pi * offHz * t
		x[i] = complex(
			float32(env*math.Cos(ph)+r.NormFloat64()*noiseSD),
			float32(env*math.Sin(ph)+r.NormFloat64()*noiseSD))
	}
	return x
}

func chunked(x []complex64, n int) [][]complex64 {
	var out [][]complex64
	for i := 0; i+n <= len(x); i += n {
		out = append(out, x[i:i+n])
	}
	return out
}

func meterRange(x []complex64) (lo, hi float64) {
	a := newAMCNMeter(amTestRate)
	lo, hi = math.Inf(1), math.Inf(-1)
	for _, c := range chunked(x, 16384) {
		v := a.process(c)
		if math.IsInf(v, -1) {
			continue
		}
		lo, hi = math.Min(lo, v), math.Max(hi, v)
	}
	return lo, hi
}

// Noise alone — at any level — never reaches even the default CLOSE level
// (open − hysteresis), so an idle AM channel cannot hold a dwell open.
func TestAMCNMeterNoiseStaysClosed(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for _, sd := range []float64{1e-4, 0.1} {
		_, hi := meterRange(amIQ(r, 10*amTestRate, sd, -200, 0, 0))
		if hi >= DefaultAMSquelchCNDb-defaultSquelchHysteresisDb {
			t.Errorf("noise sd %g: meter peaked at %.1f dB over 10 s, want < %.1f", sd, hi, DefaultAMSquelchCNDb-defaultSquelchHysteresisDb)
		}
	}
}

// The meter reads the carrier's C/N, not its level: the same C/N at noise
// floors 60 dB apart reads the same, within the measurement spread.
func TestAMCNMeterIsLevelIndependent(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	for _, cn := range []float64{20, 35} {
		loA, hiA := meterRange(amIQ(r, amTestRate, 1e-5, cn, 0.8, 1200))
		loB, hiB := meterRange(amIQ(r, amTestRate, 1e-2, cn, 0.8, 1200))
		for _, v := range []float64{loA, hiA, loB, hiB} {
			if math.Abs(v-cn) > 3 {
				t.Errorf("C/N %v dB: readings %.1f..%.1f (floor 1e-5) and %.1f..%.1f (floor 1e-2), want within 3 dB", cn, loA, hiA, loB, hiB)
				break
			}
		}
	}
}

func newAMScanner(t *testing.T, iq IQSource, tuner *fakeTuner, eng *fakeEngine) *Scanner {
	t.Helper()
	s, err := New(Options{
		Tuner: tuner, IQ: iq, Engine: eng, Recorder: fakeRecorder{},
		DeviceSerial: "CONV-AM",
		Channels: []Channel{{
			Label: "Tower", FrequencyHz: 118_700_000, Mode: ModeAM,
			SquelchDbFS: -50, Hangtime: 200 * time.Millisecond,
		}},
		SampleRateHz: amTestRate,
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// scanOpens runs one scan window over the chunks and reports whether the
// squelch opened.
func scanOpens(t *testing.T, s *Scanner, chunks [][]complex64) bool {
	t.Helper()
	stream := make(chan []complex64, len(chunks))
	for _, c := range chunks {
		stream <- c
	}
	close(stream)
	return s.scanWindow(context.Background(), 0, s.channels[0], stream, nil)
}

// TestAMSquelchIgnoresAbsoluteLevel is the #1219 squelch pin: an AM
// channel opens on a weak carrier at -85 dBFS (35 dB under squelch_dbfs)
// and stays shut on receiver noise at -17 dBFS (33 dB over it). The
// IQ-power squelch the FM channels use gets both backwards.
func TestAMSquelchIgnoresAbsoluteLevel(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	weak := amIQ(r, amTestRate/4, 1e-4, 25, 0.8, 900)
	loud := amIQ(r, amTestRate/4, 0.1, -200, 0, 0)
	if p := PowerDbFS(weak); p > -70 {
		t.Fatalf("fixture: weak signal at %.1f dBFS, want well under -50", p)
	}
	if p := PowerDbFS(loud); p < -30 {
		t.Fatalf("fixture: noise at %.1f dBFS, want well over -50", p)
	}

	s := newAMScanner(t, &fakeIQ{}, &fakeTuner{}, &fakeEngine{})
	if !scanOpens(t, s, chunked(weak, 16384)) {
		t.Error("AM channel did not open on a 25 dB C/N carrier at -85 dBFS")
	}
	s.amMeterFor(0).reset()
	if scanOpens(t, s, chunked(loud, 16384)) {
		t.Error("AM channel opened on receiver noise at -17 dBFS")
	}
}

// An AM channel's call is granted as "am-conv" (the composer's AM chain),
// and ends on hangtime when the carrier drops.
func TestAMChannelGrantsAMConvAndEndsOnCarrierDrop(t *testing.T) {
	r := rand.New(rand.NewSource(4))
	tuner := &fakeTuner{}
	eng := &fakeEngine{}
	sig := chunked(amIQ(r, amTestRate/2, 1e-4, 30, 0.8, 0), 16384)
	idle := chunked(amIQ(r, amTestRate, 1e-4, -200, 0, 0), 16384)
	iq := &fakeIQ{chunks: map[uint32][][]complex64{118_700_000: append(sig, idle...)}, tuner: tuner}
	s := newAMScanner(t, iq, tuner, eng)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go func() { _ = s.Run(ctx) }()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && eng.normalEndCount() == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	eng.mu.Lock()
	defer eng.mu.Unlock()
	if len(eng.starts) == 0 {
		t.Fatal("no call started on an AM carrier")
	}
	if got := eng.starts[0].Protocol; got != "am-conv" {
		t.Errorf("grant protocol %q, want am-conv", got)
	}
	normal := 0
	for _, r := range eng.endReasons {
		if r == trunking.EndReasonNormal {
			normal++
		}
	}
	if normal == 0 {
		t.Errorf("call did not end on hangtime after the carrier dropped (end reasons %v)", eng.endReasons)
	}
}

func TestAMModeValidationAndDefaults(t *testing.T) {
	if _, err := New(Options{
		Tuner: &fakeTuner{}, IQ: &fakeIQ{}, Engine: &fakeEngine{}, Recorder: fakeRecorder{},
		DeviceSerial: "X", Channels: []Channel{{FrequencyHz: 1, Mode: "usb"}},
	}); err == nil {
		t.Error("mode usb accepted")
	}
	s := newAMScanner(t, &fakeIQ{}, &fakeTuner{}, &fakeEngine{})
	if got := s.channels[0].SquelchCNDb; got != DefaultAMSquelchCNDb {
		t.Errorf("SquelchCNDb default %v, want %v", got, DefaultAMSquelchCNDb)
	}
	if s.opts.MinDwellPerChannel < amMinDwell {
		t.Errorf("min dwell %v, want >= %v for the meter's first average", s.opts.MinDwellPerChannel, amMinDwell)
	}
	if s.amMeterFor(0) == nil {
		t.Error("no AM meter on an AM channel")
	}
}
