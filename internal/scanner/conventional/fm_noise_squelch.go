package conventional

import (
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

// FM noise squelch (issue #1329).
//
// squelch_dbfs is an absolute level. It screens a channel only when the
// operator has measured the receiver's noise floor at the gain in use and
// set the threshold above it. With the threshold below that floor, or under
// a tuner AGC (which raises whatever it receives, noise included, to its own
// target level), an empty channel reads "carrier present" for good: a
// PlutoSky scan list opened a call 100 ms after start-up and recorded 41
// minutes of noise. AM channels already squelch on carrier-to-noise for
// this reason (am_squelch.go).
//
// fmNoiseMeter is the FM answer, the noise-operated squelch of an FM
// receiver: the discriminator's output on noise alone is large and wideband
// whatever the input level, and a carrier quiets it. The meter
// FM-discriminates the channel's decimated, channel-filtered IQ (the stream
// channelPowerMeter already produces), high-passes the result above the
// voice and signalling band, and divides the power left by the power the
// same chain gives on noise alone (measured once per rate at construction,
// the dmrrx carrier gate's method). That ratio:
//
//   - is 1 on an empty channel at any gain and any sample rate. Measured
//     medians: 0.98–1.04 on real receiver noise (an RTL-SDR's at six empty
//     offsets of a 2.4 MS/s capture, and the air-band noise slice in
//     testdata) and 0.98–1.03 on white noise from -99 to -11 dBFS at
//     16, 25, 48 and 250 kHz, 2.048, 2.4, 3 and 4.8 MS/s;
//   - falls with the carrier-to-noise ratio in the ±8 kHz channel, whatever
//     the modulation: about 0.5 at 0 dB, 0.2 at 4 dB, 0.03 at 10 dB;
//   - is below 0.001 on the real-air FM handhelds in testdata, 0.02 on a
//     transmission with 57 % of its samples at the ADC rail, and 0.12 on a
//     linear (H-DQPSK) carrier, the least FM-like signal measured.
//
// The channel opens at fmNoiseOpenRatio and holds to fmNoiseHoldRatio, so a
// signal opens it from about 0 dB carrier-to-noise: well below the FM
// threshold, where nothing is intelligible anyway. The gate therefore takes
// nothing an operator could have heard. It is ANDed with squelch_dbfs, and
// the two cover each other's blind spot: digital silence, or a tuner with
// its gain so low that the ADC sees no noise, reads perfectly "quiet" and is
// stopped by the level; noise above the level is stopped here.
//
// What it costs: a channel opens one measurement window (fmNoiseLatency)
// after its carrier comes up, where the level alone opened on the first
// chunk.
const (
	// fmNoiseHighpassHz is where the measured band starts: above speech,
	// CTCSS/DCS and the 1200/1800 Hz data bursts, so modulation does not
	// count as noise. The band runs on to the channel edge and the
	// discriminator's out-of-band products beyond it.
	fmNoiseHighpassHz = 5_000
	// fmNoiseHighpassTaps and fmNoiseHighpassBeta shape the high-pass:
	// about 63 dB down below 3.5 kHz at 48 kHz.
	fmNoiseHighpassTaps = 63
	fmNoiseHighpassBeta = 6.0
	// fmNoiseBlock is the measurement block and fmNoiseBlocks how many the
	// ratio is taken over: a 20 ms sliding window, updated every 10 ms.
	// Over two hours of white noise (720 000 windows) the smallest reading
	// was 0.66; TestFMNoiseMeterNoiseNeverReadsQuiet runs a shorter stretch.
	fmNoiseBlock  = 10 * time.Millisecond
	fmNoiseBlocks = 2
	// fmNoiseOpenRatio opens the channel; fmNoiseHoldRatio is what a call
	// in progress must stay under. Noise reads 1.
	fmNoiseOpenRatio = 0.5
	fmNoiseHoldRatio = 0.6
	// fmNoiseLatency is how long after a carrier comes up the meter
	// reports it: the filters' fill plus one full window, rounded up.
	fmNoiseLatency = 25 * time.Millisecond
	// fmNoiseSettle is the one scan-window extension a channel that
	// cleared squelch_dbfs gets when the window ended before the meter's
	// first reading: the latency plus room for chunk delivery.
	fmNoiseSettle = 60 * time.Millisecond
	// fmNoiseMinRateHz is the lowest post-decimation rate with room for
	// the measured band; a slower stream gets no meter.
	fmNoiseMinRateHz = 12_000
	// fmNoiseQuietingCapDb bounds the quieting figure reported for a
	// carrier with (nearly) no noise at all.
	fmNoiseQuietingCapDb = 60.0
)

// fmNoiseMeter is one FM channel's noise squelch. Not safe for concurrent
// feed calls; ratio and quietingDb may be read from any goroutine.
type fmNoiseMeter struct {
	hp   *filter.RealFIR
	ref  float64 // high-passed discriminator power on noise alone
	last complex64
	disc []float32
	hf   []float32

	block int // samples per block
	warm  int // samples to discard after a reset (filter fill)
	skip  int // of those, how many are still to go
	acc   float64
	n     int
	sums  [fmNoiseBlocks]float64
	next  int
	have  int

	// bits is the latest ratio (math.Float64bits), +Inf until a full
	// window has been measured since the last reset.
	bits atomic.Uint64
	// shown is the last ratio measured, kept across resets for Snapshot:
	// a channel's reading stays up while the scanner is elsewhere and
	// through the first window of its next visit. NaN until the first.
	shown atomic.Uint64
}

// newFMNoiseMeter builds the meter for a channel stream at rate (the
// post-decimation rate), or nil when the rate leaves no room for it.
func newFMNoiseMeter(rate float64) *fmNoiseMeter {
	if rate < fmNoiseMinRateHz {
		return nil
	}
	m := newUncalibratedFMNoiseMeter(rate)
	m.ref = fmNoiseReference(rate)
	return m
}

func newUncalibratedFMNoiseMeter(rate float64) *fmNoiseMeter {
	lp := filter.LowpassKaiser(fmNoiseHighpassTaps, fmNoiseHighpassHz/rate, fmNoiseHighpassBeta)
	hp := make([]float32, len(lp))
	for i, t := range lp {
		hp[i] = -t
	}
	hp[len(hp)/2]++ // delta minus low-pass
	m := &fmNoiseMeter{
		hp:    filter.NewRealFIR(hp),
		ref:   1,
		last:  complex(1, 0),
		block: max(int(math.Round(rate*fmNoiseBlock.Seconds())), 1),
		// The channel filter ahead of the meter and the high-pass in it,
		// 63 taps each, plus the decimator's few output samples.
		warm: 2*fmNoiseHighpassTaps + 16,
	}
	m.skip = m.warm
	m.bits.Store(math.Float64bits(math.Inf(1)))
	m.shown.Store(math.Float64bits(math.NaN()))
	return m
}

// fmNoiseRefs caches fmNoiseReference by rate: every channel of a scan
// list shares its scanner's rate.
var fmNoiseRefs sync.Map

// fmNoiseReference is the meter's reading (ref = 1) on white noise through
// the channel chain at rate: what "no carrier" measures. Fixed seed, so the
// same rate always gives the same reference.
func fmNoiseReference(rate float64) float64 {
	if v, ok := fmNoiseRefs.Load(rate); ok {
		return v.(float64)
	}
	// The same channel filter channelPowerMeter builds for this rate. The
	// decimator ahead of it in a real stream is left out: it leaves the
	// noise white across the channel, and noise through it reads within a
	// few per cent of 1 against this reference at every rate tried
	// (TestFMNoiseSquelchAtEveryRate).
	pm := &channelPowerMeter{chanFilter: channelFilterFor(rate), level: PowerDbFS(nil)}
	m := newUncalibratedFMNoiseMeter(rate)
	rng := rand.New(rand.NewSource(1329))
	buf := make([]complex64, 4096)
	var acc float64
	var blocks int
	total := int(4 * rate) // 4 s: the estimate is good to about 0.5 %
	for done := 0; done < total; done += len(buf) {
		for i := range buf {
			buf[i] = complex(float32(rng.NormFloat64()), float32(rng.NormFloat64()))
		}
		x := buf
		if pm.chanFilter != nil {
			pm.chanBuf = pm.chanFilter.Process(pm.chanBuf[:0], buf)
			x = pm.chanBuf
		}
		m.feedBlocks(x, func(sum float64) {
			acc += sum
			blocks++
		})
	}
	ref := acc / float64(blocks*m.block)
	if !(ref > 0) {
		ref = 1
	}
	fmNoiseRefs.Store(rate, ref)
	return ref
}

// feed folds channel-filtered samples at the meter's rate into the ratio.
func (m *fmNoiseMeter) feed(iq []complex64) {
	m.feedBlocks(iq, m.closeBlock)
}

// feedBlocks discriminates and high-passes iq and calls done with each
// completed block's sum of squares.
func (m *fmNoiseMeter) feedBlocks(iq []complex64, done func(sum float64)) {
	if len(iq) == 0 {
		return
	}
	m.disc = m.disc[:0]
	for _, s := range iq {
		// arg(z[n] · conj(z[n-1])): radians per sample.
		ar := real(s)*real(m.last) + imag(s)*imag(m.last)
		ai := imag(s)*real(m.last) - real(s)*imag(m.last)
		m.disc = append(m.disc, float32(math.Atan2(float64(ai), float64(ar))))
		m.last = s
	}
	m.hf = m.hp.Process(m.hf, m.disc)
	for _, v := range m.hf {
		if m.skip > 0 {
			m.skip--
			continue
		}
		m.acc += float64(v) * float64(v)
		m.n++
		if m.n == m.block {
			done(m.acc)
			m.acc, m.n = 0, 0
		}
	}
}

func (m *fmNoiseMeter) closeBlock(sum float64) {
	m.sums[m.next] = sum
	m.next = (m.next + 1) % fmNoiseBlocks
	if m.have < fmNoiseBlocks {
		m.have++
		if m.have < fmNoiseBlocks {
			return
		}
	}
	var tot float64
	for _, s := range m.sums {
		tot += s
	}
	r := tot / (float64(fmNoiseBlocks*m.block) * m.ref)
	m.bits.Store(math.Float64bits(r))
	m.shown.Store(math.Float64bits(r))
}

// ratio is the high-passed discriminator power over what noise alone gives:
// 1 on an empty channel, towards 0 on a carrier. +Inf until a full window
// has been measured since the last reset, so a channel never opens on a
// reading the meter has not made yet.
func (m *fmNoiseMeter) ratio() float64 {
	return math.Float64frombits(m.bits.Load())
}

// quietingDb is the ratio as the quieting an FM receiver's squelch reads:
// 0 dB on noise, rising with the carrier. ok=false before the first window.
func (m *fmNoiseMeter) quietingDb() (db float64, ok bool) {
	return quietingOf(m.ratio())
}

// shownQuietingDb is quietingDb of the last window measured on any visit.
func (m *fmNoiseMeter) shownQuietingDb() (db float64, ok bool) {
	return quietingOf(math.Float64frombits(m.shown.Load()))
}

func quietingOf(r float64) (db float64, ok bool) {
	if math.IsInf(r, 1) || math.IsNaN(r) {
		return 0, false
	}
	if r <= 0 {
		return fmNoiseQuietingCapDb, true
	}
	return math.Min(-10*math.Log10(r), fmNoiseQuietingCapDb), true
}

// reset forgets the previous channel visit.
func (m *fmNoiseMeter) reset() {
	m.hp.Reset()
	m.last = complex(1, 0)
	m.skip = m.warm
	m.acc, m.n = 0, 0
	m.sums = [fmNoiseBlocks]float64{}
	m.next, m.have = 0, 0
	m.bits.Store(math.Float64bits(math.Inf(1)))
}

// noiseNoteChannels is how many channels noteNoiseAboveSquelch names before
// it stops listing them.
const noiseNoteChannels = 5

// noiseSquelchFor returns channel idx's FM noise squelch, or nil when the
// channel has none (not an FM channel, switched off, or no sample rate).
func (s *Scanner) noiseSquelchFor(idx int) *fmNoiseMeter {
	if pm := s.powerMeterFor(idx); pm != nil {
		return pm.noise
	}
	return nil
}

// noteNoiseAboveSquelch tells the operator, once per channel, that the
// channel sat above squelch_dbfs for a whole scan window with no carrier on
// it: the level squelch is below the receiver's noise floor at this gain
// (or the gain is automatic), so only the noise squelch is screening the
// channel. Scan goroutine only.
//
// The first noiseNoteChannels channels are named; after that one line says
// there are more, so a long scan list under an AGC does not log a line per
// channel.
func (s *Scanner) noteNoiseAboveSquelch(ch Channel, m *fmNoiseMeter, levelDb float64) {
	if s.noiseNoted == nil {
		s.noiseNoted = make(map[*fmNoiseMeter]bool)
	}
	if s.noiseNoted[m] {
		return
	}
	s.noiseNoted[m] = true
	if n := len(s.noiseNoted); n > noiseNoteChannels {
		if n == noiseNoteChannels+1 {
			s.log.Info("conv: more channels are above squelch_dbfs with no carrier on them; not naming the rest. quieting_db in GET /api/v1/scanner shows each channel's reading")
		}
		return
	}
	q, _ := m.quietingDb()
	s.log.Info("conv: channel is above squelch_dbfs with no carrier on it (noise, or a signal that is not FM); the noise squelch is keeping it shut. squelch_dbfs is below the noise floor at this gain, or the gain is automatic, so the level is not screening this channel",
		"freq_hz", ch.FrequencyHz, "label", ch.Label,
		"level_dbfs", roundTenth(levelDb), "squelch_dbfs", ch.SquelchDbFS, "quieting_db", roundTenth(q))
}

// logSquelchOpened records what the squelch measured as it opened: the line
// to read first when a channel opens that should not have, or does not open
// when it should (the level against its threshold, and on an analog FM
// channel how quiet the discriminator was).
func (s *Scanner) logSquelchOpened(idx int, ch Channel, level float64, nsq *fmNoiseMeter) {
	args := []any{"freq_hz", ch.FrequencyHz, "label", ch.Label, "mode", ch.Mode}
	if s.amMeterFor(idx) != nil {
		args = append(args, "level_cn_db", roundTenth(level), "squelch_cn_db", ch.SquelchCNDb)
	} else {
		args = append(args, "level_dbfs", roundTenth(level), "squelch_dbfs", ch.SquelchDbFS)
	}
	switch {
	case nsq != nil:
		if q, ok := nsq.quietingDb(); ok {
			args = append(args, "quieting_db", roundTenth(q))
		}
	case isAnalogFM(ch.Mode):
		args = append(args, "noise_squelch", "off")
	}
	s.log.Info("conv: squelch opened", args...)
}

func roundTenth(v float64) float64 { return math.Round(v*10) / 10 }
