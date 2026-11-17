package receiver

import (
	"fmt"
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/radio/pager/flex"
)

// SampleRateHz is the rate the discriminator output is resampled to: 16
// samples per symbol through the 1600 symbols/s opening of every frame and 8
// per symbol in a 3200 symbols/s frame body.
const SampleRateHz = 25600

const (
	huntSPS   = SampleRateHz / flex.SyncSymbolRate // 16
	huntBox   = huntSPS / 2                        // boxcar length while hunting
	syncBits  = 64
	ringSize  = 4096 // > the 64 sync symbols the level calibration looks back over
	ringMask  = ringSize - 1
	timingGfs = 0.08 // timing loop gain, fraction of the measured error
	// sliceFrac places the inner/outer decision between the 4-FSK levels at
	// ±1/3 and ±1 of the outer deviation.
	sliceFrac = 2.0 / 3.0
)

// Demod turns FM discriminator output into FLEX pages: it finds each frame's
// sync, measures the signal's centre and deviation from it, then slices the
// frame's symbols — at the rate and level count the sync named — into a
// flex.Decoder.
//
// Every frame is acquired on its own. While hunting, the sync is searched at
// all 16 sample phases of the 1600 symbols/s grid at once and the middle of
// the run of phases that match fixes the symbol timing, so there is no timing
// loop to pull in (a loop hunting at 1600 symbols/s through the 3200
// symbols/s body of a frame it missed would be dragged off). Inside the frame
// a slow loop on the level transitions follows the sample-clock error.
type Demod struct {
	dec *flex.Decoder

	ring [ringSize]float32
	n    int64 // samples consumed; ring[(n-1)&ringMask] is the newest

	// Hunt.
	box      float32 // sum of the newest huntBox samples
	dc       float32 // slow mean: the discriminator's carrier-offset bias
	regs     [huntSPS]uint64
	inRun    bool
	runFirst int64
	runLast  int64
	runMode  flex.Mode
	runInv   bool

	// Frame.
	tracking bool
	symStart float64 // time of the current symbol's leading edge, in samples
	sps      float64
	zero     float32
	env      float32
	prevSym  byte
	havePrev bool

	// OnFrame, when set, is called as each frame's sync is acquired.
	OnFrame func(mode flex.Mode, inverted bool)
	// OnSymbol, when set, sees every symbol sliced after a frame's sync, as
	// the decoder is about to (frequency sense not yet rectified).
	OnSymbol func(sym byte)
}

// NewDemod returns a Demod hunting for a frame.
func NewDemod() *Demod { return &Demod{dec: flex.NewDecoder()} }

// Decoder exposes the frame decoder (for its counters).
func (d *Demod) Decoder() *flex.Decoder { return d.dec }

// at returns the sample at absolute index i (which must be within the ring).
func (d *Demod) at(i int64) float32 { return d.ring[i&ringMask] }

// mean averages the samples at integer times in [from, to).
func (d *Demod) mean(from, to float64) float32 {
	lo := int64(from + 0.999999)
	hi := int64(to + 0.999999)
	if hi <= lo {
		hi = lo + 1
	}
	var s float32
	for i := lo; i < hi; i++ {
		s += d.at(i)
	}
	return s / float32(hi-lo)
}

// Process consumes discriminator samples at SampleRateHz and returns the
// pages completed within them.
func (d *Demod) Process(samples []float32) []flex.Message {
	var out []flex.Message
	for _, s := range samples {
		idx := d.n
		d.ring[idx&ringMask] = s
		d.n++
		if idx&ringMask == 0 {
			// Re-sum exactly now and then: a running float sum drifts.
			d.box = 0
			for k := int64(0); k < huntBox; k++ {
				d.box += d.at(idx - k)
			}
		} else {
			d.box += s - d.at(idx-huntBox)
		}
		if d.tracking {
			out = append(out, d.track()...)
		} else {
			d.hunt(idx, s)
		}
	}
	return out
}

// hunt advances the sync search by the sample at index idx.
func (d *Demod) hunt(idx int64, s float32) {
	d.dc += (s - d.dc) * (1.0 / 2048)
	p := idx & (huntSPS - 1)
	d.regs[p] <<= 1
	if d.box/huntBox-d.dc < 0 {
		d.regs[p] |= 1 // sync sense: 1 is the low deviation
	}
	mode, ok := flex.MatchSync(d.regs[p])
	inv := false
	if !ok {
		mode, ok = flex.MatchSync(^d.regs[p])
		inv = true
	}
	if ok && !d.inRun {
		d.inRun, d.runFirst, d.runMode, d.runInv = true, idx, mode, inv
	}
	if ok {
		d.runLast = idx
	}
	if d.inRun && (!ok || idx-d.runFirst >= huntSPS-1) {
		d.inRun = false
		d.acquire(float64(d.runFirst+d.runLast) / 2)
	}
}

// acquire starts a frame. centre is the sample index at which the hunting
// boxcar sat squarely inside the last sync symbol: the boxcar spans the
// huntBox samples ending there, so that symbol's midpoint is (huntBox-1)/2
// earlier and the next symbol starts half a symbol after the midpoint.
func (d *Demod) acquire(centre float64) {
	mid := centre - float64(huntBox-1)/2
	// Levels, from the middle half of each of the 64 sync symbols — all sent
	// at the full deviation, in a near-balanced pattern.
	var pos, neg float32
	var nPos, nNeg int
	for k := 0; k < syncBits; k++ {
		m := mid - float64(k*huntSPS)
		v := d.mean(m-huntSPS/4, m+huntSPS/4)
		if v > d.dc {
			pos += v
			nPos++
		} else {
			neg += v
			nNeg++
		}
	}
	for i := range d.regs {
		d.regs[i] = 0
	}
	if nPos == 0 || nNeg == 0 {
		return
	}
	pos /= float32(nPos)
	neg /= float32(nNeg)
	d.zero = (pos + neg) / 2
	d.env = (pos - neg) / 2
	if d.env <= 0 {
		return
	}
	d.sps = huntSPS
	d.symStart = mid + huntSPS/2
	d.havePrev = false
	d.tracking = true
	d.dec.Begin(d.runMode, d.runInv)
	if d.OnFrame != nil {
		d.OnFrame(d.runMode, d.runInv)
	}
}

// track emits the current symbol once its middle half has arrived.
func (d *Demod) track() []flex.Message {
	if float64(d.n-1) < d.symStart+0.75*d.sps {
		return nil
	}
	v := d.mean(d.symStart+0.25*d.sps, d.symStart+0.75*d.sps) - d.zero
	var sym byte
	switch {
	case v > d.env*sliceFrac:
		sym = 3
	case v > 0:
		sym = 2
	case v > -d.env*sliceFrac:
		sym = 1
	}
	// Timing: stepping from one level to another, the signal passes the
	// midpoint of the two at the symbol edge. Every transition is used, not
	// just the ones through the centre: a frame's idle tail can go thousands
	// of symbols without one of those, and a 60 ppm clock error left to
	// free-run that long slides the slicer a third of a symbol off.
	if d.havePrev && sym != d.prevSym {
		mid := (d.level(sym) + d.level(d.prevSym)) / 2
		if x, ok := d.crossing(d.symStart-0.5*d.sps, d.symStart+0.5*d.sps, mid); ok {
			d.symStart += timingGfs * (x - d.symStart)
		}
	}
	d.prevSym, d.havePrev = sym, true

	if d.OnSymbol != nil {
		d.OnSymbol(sym)
	}
	msgs := d.dec.PushSymbol(sym)
	d.symStart += d.sps
	d.sps = float64(SampleRateHz) / float64(d.dec.SymbolRate())
	if d.dec.Hunting() {
		// Frame over (or its information word failed): hunt for the next,
		// keeping the centre this frame measured.
		d.tracking = false
		d.dc = d.zero
	}
	return msgs
}

// level is the nominal value of a symbol, relative to the centre: the outer
// levels at the measured deviation, the inner ones at a third of it.
func (d *Demod) level(sym byte) float32 {
	return d.env * (float32(sym)*2 - 3) / 3
}

// crossing finds where the (3-sample smoothed) signal first passes the given
// level between times from and to, by linear interpolation.
func (d *Demod) crossing(from, to float64, level float32) (float64, bool) {
	lo := int64(from)
	hi := int64(to)
	if hi > d.n-2 {
		hi = d.n - 2
	}
	sm := func(i int64) float32 { return (d.at(i-1)+d.at(i)+d.at(i+1))/3 - d.zero - level }
	prev := sm(lo)
	for i := lo + 1; i <= hi; i++ {
		cur := sm(i)
		if (prev < 0) != (cur < 0) {
			den := cur - prev
			if den == 0 {
				return float64(i), true
			}
			return float64(i-1) + float64(-prev/den), true
		}
		prev = cur
	}
	return 0, false
}

// IQDemod runs the whole front end over IQ: FM discriminator, resampling to
// SampleRateHz, then Demod. It is what the live Receiver and the offline
// survey both use.
type IQDemod struct {
	*Demod
	fm       *demod.FM
	rsmp     *dsp.RealResampler
	demodBuf []float32
	rsmpBuf  []float32
}

// NewIQDemod builds the front end for IQ at inputRateHz.
func NewIQDemod(inputRateHz uint32) (*IQDemod, error) {
	rsmp, err := newResampler(inputRateHz)
	if err != nil {
		return nil, err
	}
	return &IQDemod{Demod: NewDemod(), fm: demod.NewFM(), rsmp: rsmp}, nil
}

// ProcessIQ consumes one IQ chunk and returns the pages completed within it.
func (q *IQDemod) ProcessIQ(chunk []complex64) []flex.Message {
	q.demodBuf = q.fm.Process(q.demodBuf, chunk)
	q.rsmpBuf = q.rsmp.Process(q.rsmpBuf, q.demodBuf)
	return q.Demod.Process(q.rsmpBuf)
}

// AudioDemod runs Demod over discriminator audio (a receiver's unfiltered FM
// output) at any sample rate — the form FLEX recordings usually come in.
type AudioDemod struct {
	*Demod
	rsmp *dsp.RealResampler
	buf  []float32
}

// NewAudioDemod builds a demodulator for discriminator audio at rateHz.
func NewAudioDemod(rateHz uint32) (*AudioDemod, error) {
	rsmp, err := newResampler(rateHz)
	if err != nil {
		return nil, err
	}
	return &AudioDemod{Demod: NewDemod(), rsmp: rsmp}, nil
}

// ProcessAudio consumes discriminator samples and returns the pages completed
// within them.
func (a *AudioDemod) ProcessAudio(samples []float32) []flex.Message {
	a.buf = a.rsmp.Process(a.buf, samples)
	return a.Demod.Process(a.buf)
}

func newResampler(inputRateHz uint32) (*dsp.RealResampler, error) {
	if inputRateHz == 0 {
		return nil, fmt.Errorf("flex/receiver: input rate is required")
	}
	g := gcd(SampleRateHz, inputRateHz)
	L := int(SampleRateHz / g)
	M := int(inputRateHz / g)
	if L > maxInterp {
		// An awkward rate: take the closest L/M with a small L. The symbol
		// timing loop absorbs the residual like any sample-clock error, so
		// only a ratio that is far off is refused.
		want := float64(SampleRateHz) / float64(inputRateHz)
		best := 1.0
		for l := 1; l <= maxInterp; l++ {
			m := int(float64(l)/want + 0.5)
			if m < 1 {
				continue
			}
			if e := math.Abs(float64(l)/float64(m)/want - 1); e < best {
				best, L, M = e, l, m
			}
		}
		if best > maxRateErr {
			return nil, fmt.Errorf("flex/receiver: no resampler from %d Hz to %d Hz within %.0f ppm",
				inputRateHz, SampleRateHz, maxRateErr*1e6)
		}
	}
	return dsp.NewRealResampler(L, M, 16, 7.0), nil
}

const (
	maxInterp  = 1024  // largest interpolation factor (16 taps per branch)
	maxRateErr = 50e-6 // tolerated error of an approximated resampling ratio
)

// gcd computes the greatest common divisor via Euclid's algorithm.
func gcd(a, b uint32) uint32 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
