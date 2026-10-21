package wmbus

import (
	"errors"
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

// ChannelHz is the T1 / C1 meter-to-other channel centre.
const ChannelHz = 868_950_000

// ChipRateHz is the nominal T and C mode chip rate. T mode meters may be
// several percent off it; the clock recovery tracks ±15 %.
const ChipRateHz = 100_000

// procRateHz is the rate the receiver demodulates at: 8 samples per chip.
const procRateHz = 800_000

const (
	nominalSPC = float64(procRateHz) / ChipRateHz // samples per chip
	// channelCutoffHz keeps the ±50 kHz (T1) / ±45 kHz (C1) FSK plus a
	// tuner error of about ±120 kHz (±140 ppm), and rejects neighbours
	// further out. It is the filter until the carrier has been found.
	channelCutoffHz = 170_000
	// lockedCutoffHz is the filter once the carrier is found (see afc):
	// the FSK plus room for one meter's carrier sitting off another's.
	lockedCutoffHz = 100_000
	// afcGain smooths the carrier estimate across frames: each CRC-valid
	// frame pulls it this far toward the carrier that frame was heard on.
	afcGain = 0.25
	// afcMaxHz bounds the correction: the wide filter cannot hear a
	// carrier further out than this anyway.
	afcMaxHz = 150_000
	// afcHoldSeconds is how long the narrow filter is kept without a
	// CRC-valid frame before the receiver widens it again, in case the
	// tuner has drifted away from the estimate.
	afcHoldSeconds = 120
	// thresholdWindow is the slicing-threshold average: 32 chips. Both
	// preambles, the access code and T mode data are DC-balanced over it.
	thresholdWindow = 256
	// smoothLen is the post-discriminator boxcar: half a chip.
	smoothLen = 4
	// sampleLoopGain and sampleLoopRateGain are the mid-chip clock's
	// phase and period gains, per transition.
	sampleLoopGain     = 0.15
	sampleLoopRateGain = 0.004
	// decideSpan is how many samples either side of a chip's centre (in
	// total, plus the centre) its decision averages: with the half-chip
	// boxcar that makes about a chip-long matched filter.
	decideSpan = 4
)

// interpAt linearly interpolates h at fractional index x, clamped to h.
func interpAt(h []float64, x float64) float64 {
	if x <= 0 {
		return h[0]
	}
	if x >= float64(len(h)-1) {
		return h[len(h)-1]
	}
	i := int(x)
	a := x - float64(i)
	return h[i] + a*(h[i+1]-h[i])
}

// ReceiverOptions configures a Receiver.
type ReceiverOptions struct {
	// SampleRateHz is the input IQ rate. Required.
	SampleRateHz float64
	// OffsetHz is where the wM-Bus channel sits relative to the input's
	// centre frequency; 0 when tuned to ChannelHz.
	OffsetHz float64
	// OnFrame receives every framed packet, including CRC failures.
	OnFrame func(Frame, FrameInfo)
}

// FrameInfo is receiver-side metadata for a frame.
type FrameInfo struct {
	// LevelDBFS is the channel power around the end of the frame.
	LevelDBFS float64
	// Sample is the input sample index at which the frame ended.
	Sample int64
	// Inverted reports that the frame was framed on the inverted chip
	// stream (a spectrum-inverted front end).
	Inverted bool
	// CarrierOffsetHz is where the frame's carrier sat relative to the
	// channel (ChannelHz, or the configured OffsetHz from the input's
	// centre), measured on its preamble. Every meter heard through one
	// tuner shares that tuner's frequency error, so on a run it mostly
	// measures the tuner.
	CarrierOffsetHz float64
}

// ReceiverStats counts receiver activity.
type ReceiverStats struct {
	Samples int64
	Framer  FramerStats // both polarities summed
	// CarrierOffsetHz is the receiver's carrier estimate relative to the
	// channel, learned from CRC-valid frames; CarrierLocked reports that
	// it has one (and is running the narrow channel filter).
	CarrierOffsetHz float64
	CarrierLocked   bool
}

// Receiver turns IQ into wM-Bus frames.
type Receiver struct {
	opts  ReceiverOptions
	nco   *dsp.NCO
	rs    *dsp.Resampler
	chf   *filter.FIR
	ratio float64 // input samples per processed sample

	mixBuf, rsBuf, chBuf []complex64

	prev   complex64
	smooth [smoothLen]float64
	smIdx  int
	smSum  float64

	ring   [thresholdWindow]float64
	ringN  int
	rIdx   int
	rSum   float64
	frozen bool
	thr    float64

	power float64

	// Chip clock recovery on the sliced stream.
	n         int64   // processed-sample counter
	level     bool    // current sliced level
	lastCross float64 // time of the last transition
	vPrev     float64
	spc       float64                 // tracked samples per chip
	tSample   float64                 // the next chip centre
	hist      [decideSpan + 2]float64 // recent sliced values, newest last

	framers [2]*Framer // normal, inverted
	stats   ReceiverStats
	samples int64

	// Carrier tracking (afc): the learned offset from opts.OffsetHz, and
	// the processed-sample index of the last frame that set it.
	afcHz     float64
	appliedHz float64 // afcHz as it was when the block being sliced was mixed
	afcLocked bool
	narrow    bool
	afcAt     int64
}

// NewReceiver builds a Receiver.
func NewReceiver(o ReceiverOptions) (*Receiver, error) {
	if o.SampleRateHz < procRateHz/2 {
		return nil, errors.New("wmbus: sample rate must be at least 400 kS/s")
	}
	r := &Receiver{opts: o, spc: nominalSPC}
	if o.OffsetHz != 0 {
		r.nco = dsp.NewNCO(o.OffsetHz, o.SampleRateHz)
	}
	in := int(math.Round(o.SampleRateHz))
	if in != procRateHz {
		g := gcd(procRateHz, in)
		l, m := procRateHz/g, in/g
		taps := (12*max(l, m) + l - 1) / l
		r.rs = dsp.NewResampler(l, m, max(taps, 8), 6)
	}
	r.ratio = o.SampleRateHz / procRateHz
	r.chf = channelFilter(channelCutoffHz)
	for i := range r.framers {
		inv := i == 1
		r.framers[i] = NewFramer(func(f Frame) {
			off := r.carrierHz(inv)
			if f.CRCOK && !inv {
				r.afc(off)
			}
			if o.OnFrame != nil {
				o.OnFrame(f, FrameInfo{
					LevelDBFS:       10 * math.Log10(r.power+1e-20),
					Sample:          int64(float64(r.n) * r.ratio),
					Inverted:        inv,
					CarrierOffsetHz: off,
				})
			}
		})
	}
	return r, nil
}

// channelFilter is the channel filter at the processing rate.
func channelFilter(cutoffHz float64) *filter.FIR {
	return filter.NewFIR(filter.LowpassKaiser(63, cutoffHz/procRateHz, 6))
}

// carrierHz is where the frame being delivered sat relative to the
// channel: the receiver's current correction plus what is left of the
// offset after it, which the slicing threshold measured on the preamble
// (the 0101… preamble is DC-balanced, and the threshold is held from the
// access code to the end of the frame). The inverted framer sees the
// discriminator negated.
func (r *Receiver) carrierHz(inverted bool) float64 {
	residual := r.thr * procRateHz / (2 * math.Pi)
	if inverted {
		residual = -residual
	}
	return r.appliedHz + residual
}

// afc re-centres the receiver on the carrier a CRC-valid frame was heard
// on, and narrows the channel filter around it.
//
// A real tuner can sit far off: the S5 + RTL-SDR in issue #1256 heard its
// meters about 50 kHz (58 ppm) low, so the lower FSK tone sat near -95 kHz
// and the wide filter needed for that let in noise from +170 kHz too. On
// that reporter's 156 saved telegrams, fed through one receiver in order,
// this took the CRC-valid count from 97 to 128.
// The offset is the tuner's, shared by every meter it hears, so it is
// learned across frames: the first CRC-valid frame sets it, later ones
// refine it. Only CRC-valid frames count, so noise cannot steer it, and
// the filter widens again after afcHoldSeconds without one, should the
// tuner drift outside it.
func (r *Receiver) afc(carrierHz float64) {
	carrierHz = min(max(carrierHz, -afcMaxHz), afcMaxHz)
	if r.afcLocked {
		r.afcHz += afcGain * (carrierHz - r.afcHz)
	} else {
		r.afcHz = carrierHz
	}
	r.afcLocked = true
	r.afcAt = r.n
	off := r.opts.OffsetHz + r.afcHz
	if r.nco == nil {
		r.nco = dsp.NewNCO(off, r.opts.SampleRateHz)
	} else {
		r.nco.SetOffset(off, r.opts.SampleRateHz)
	}
	// The slicing threshold follows the new offset within its window
	// (32 chips), well before the next frame's access code.
	if !r.narrow {
		r.chf = channelFilter(lockedCutoffHz)
		r.narrow = true
	}
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// Stats returns the receiver counters.
func (r *Receiver) Stats() ReceiverStats {
	s := r.stats
	s.Samples = r.samples
	s.CarrierOffsetHz, s.CarrierLocked = r.afcHz, r.afcLocked
	for _, f := range r.framers {
		fs := f.Stats()
		s.Framer.Syncs += fs.Syncs
		s.Framer.Frames += fs.Frames
		s.Framer.CRCOK += fs.CRCOK
		s.Framer.CodingError += fs.CodingError
		s.Framer.BadLength += fs.BadLength
	}
	return s
}

// processBlock bounds how many input samples are mixed with one carrier
// correction, so a correction learned from a frame applies within a few
// milliseconds whatever chunk size the caller feeds.
const processBlock = 8192

// Process feeds IQ samples at the configured rate.
func (r *Receiver) Process(iq []complex64) {
	for len(iq) > 0 {
		n := min(len(iq), processBlock)
		r.processBlock(iq[:n])
		iq = iq[n:]
	}
}

func (r *Receiver) processBlock(iq []complex64) {
	r.samples += int64(len(iq))
	// The correction this block is mixed with: a frame decoded from it
	// measures its residual offset against this, even if a frame earlier
	// in the block has already moved the NCO for the next one.
	r.appliedHz = r.afcHz
	if r.nco != nil {
		r.mixBuf = r.nco.Mix(r.mixBuf, iq)
		iq = r.mixBuf
	}
	if r.rs != nil {
		r.rsBuf = r.rs.Process(r.rsBuf[:0], iq)
		iq = r.rsBuf
	}
	if r.narrow && r.n-r.afcAt > afcHoldSeconds*procRateHz {
		// No CRC-valid frame for a while: widen the filter again around
		// the estimate, in case the tuner drifted outside the narrow one.
		r.chf = channelFilter(channelCutoffHz)
		r.narrow = false
	}
	r.chBuf = r.chf.Process(r.chBuf, iq)
	for _, x := range r.chBuf {
		r.sample(x)
	}
}

// Flush slices the samples still waiting in the threshold window and
// decides the chips after them, so a frame that ends the input is
// delivered. Call it once, at the end of the input.
func (r *Receiver) Flush() {
	if r.ringN < thresholdWindow {
		return
	}
	for i := 1; i < thresholdWindow/2; i++ {
		r.n++
		r.clock(r.ring[(r.rIdx+thresholdWindow/2+i)%thresholdWindow] - r.thr)
	}
	r.decide(float64(r.n) + decideSpan/2)
}

func (r *Receiver) sample(x complex64) {
	r.n++
	p := float64(real(x))*float64(real(x)) + float64(imag(x))*float64(imag(x))
	r.power += (p - r.power) / 512

	// FM discriminator, then a half-chip boxcar.
	d := x * complex(real(r.prev), -imag(r.prev))
	r.prev = x
	f := math.Atan2(float64(imag(d)), float64(real(d)))
	r.smSum += f - r.smooth[r.smIdx]
	r.smooth[r.smIdx] = f
	r.smIdx = (r.smIdx + 1) % smoothLen
	s := r.smSum / smoothLen

	// Slice the sample from the middle of the threshold window against
	// the window's mean. The mean absorbs a carrier offset of any size
	// within the channel filter; it is frozen while a frame is being
	// received, because C mode NRZ data is not DC-balanced.
	r.rSum += s - r.ring[r.rIdx]
	r.ring[r.rIdx] = s
	r.rIdx = (r.rIdx + 1) % thresholdWindow
	if r.ringN < thresholdWindow {
		r.ringN++
		return
	}
	busy := r.framers[0].Busy() || r.framers[1].Busy()
	if !busy || !r.frozen {
		r.thr = r.rSum / thresholdWindow
	}
	r.frozen = busy
	v := r.ring[(r.rIdx+thresholdWindow/2)%thresholdWindow] - r.thr
	r.clock(v)
}

// clock is the chip clock: a timing loop that decides each chip from the
// sliced stream around the chip's centre. Every transition pulls the next
// centre toward half a chip after it, and the chip period toward the
// measured spacing.
//
// It replaced counting whole chips from the run between transitions, where
// a noise glitch inside a long run added two transitions and so extra
// chips: one glitch slipped the bit stream and failed every CRC block
// after it (issue #1256, strong real-air frames failing their CRC; on
// simulated 192-byte C mode frames at 13 dB SNR every failure was such a
// slip). Decided at the centre, a glitch costs at most one chip.
func (r *Receiver) clock(v float64) {
	t := float64(r.n)
	if hi := v > 0; hi != r.level {
		frac := 0.0
		if r.vPrev != v {
			frac = r.vPrev / (r.vPrev - v)
		}
		tc := t - 1 + frac
		if tc-r.lastCross > 200*nominalSPC {
			// Silence before this burst: restart the loop on it.
			r.spc = nominalSPC
			r.tSample = tc + r.spc/2
		} else {
			// The transition should fall half a chip before a centre;
			// e is how late it came, wrapped to ±half a chip.
			e := tc - (r.tSample - r.spc/2)
			e -= r.spc * math.Round(e/r.spc)
			r.tSample += sampleLoopGain * e
			r.spc += sampleLoopRateGain * e
			// Between frames the period also follows short runs
			// directly: the loop alone cannot pull in a T mode meter
			// several percent off nominal within a preamble. Inside a
			// frame that estimate's noise would cost more than it
			// tracks, so only the loop runs there.
			run := tc - r.lastCross
			if n := math.Round(run / r.spc); n >= 1 && n <= 4 && !r.frozen {
				r.spc += 0.1 * (run/n - r.spc)
			}
			r.spc = min(max(r.spc, nominalSPC*0.85), nominalSPC*1.15)
		}
		r.level = hi
		r.lastCross = tc
	}
	copy(r.hist[:], r.hist[1:])
	r.hist[len(r.hist)-1] = v
	r.vPrev = v
	r.decide(t)
}

// decide pushes every chip whose centre lies at least decideSpan/2 samples
// before limit. A chip's value is the sliced stream averaged over
// decideSpan samples around its centre, which with the half-chip boxcar
// ahead of it spans about one chip.
func (r *Receiver) decide(limit float64) {
	t := float64(r.n)
	for r.tSample+decideSpan/2 <= limit {
		x := 0.0
		c := r.tSample - (t - float64(len(r.hist)-1)) // the centre, as an index into hist
		for k := -decideSpan / 2; k <= decideSpan/2; k++ {
			x += interpAt(r.hist[:], c+float64(k))
		}
		chip := uint8(0)
		if x > 0 {
			chip = 1
		}
		r.framers[0].Push(chip)
		r.framers[1].Push(chip ^ 1)
		r.tSample += r.spc
	}
}
