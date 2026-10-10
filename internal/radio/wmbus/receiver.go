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
	// ±50 ppm tuner error, and rejects neighbours further out.
	channelCutoffHz = 170_000
	// thresholdWindow is the slicing-threshold average: 32 chips. Both
	// preambles, the access code and T mode data are DC-balanced over it.
	thresholdWindow = 256
	// smoothLen is the post-discriminator boxcar: half a chip.
	smoothLen = 4
)

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
}

// ReceiverStats counts receiver activity.
type ReceiverStats struct {
	Samples int64
	Framer  FramerStats // both polarities summed
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

	// Run-length clock recovery on the sliced stream.
	n         int64   // processed-sample counter
	level     bool    // current sliced level
	lastCross float64 // time of the last confirmed transition
	pending   bool
	pendT     float64
	vPrev     float64
	spc       float64 // tracked samples per chip

	framers [2]*Framer // normal, inverted
	stats   ReceiverStats
	samples int64
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
	r.chf = filter.NewFIR(filter.LowpassKaiser(63, float64(channelCutoffHz)/procRateHz, 6))
	for i := range r.framers {
		inv := i == 1
		r.framers[i] = NewFramer(func(f Frame) {
			if o.OnFrame != nil {
				o.OnFrame(f, FrameInfo{
					LevelDBFS: 10 * math.Log10(r.power+1e-20),
					Sample:    int64(float64(r.n) * r.ratio),
					Inverted:  inv,
				})
			}
		})
	}
	return r, nil
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

// Process feeds IQ samples at the configured rate.
func (r *Receiver) Process(iq []complex64) {
	r.samples += int64(len(iq))
	if r.nco != nil {
		r.mixBuf = r.nco.Mix(r.mixBuf, iq)
		iq = r.mixBuf
	}
	if r.rs != nil {
		r.rsBuf = r.rs.Process(r.rsBuf[:0], iq)
		iq = r.rsBuf
	}
	r.chBuf = r.chf.Process(r.chBuf, iq)
	for _, x := range r.chBuf {
		r.sample(x)
	}
}

// Flush delivers the chips of the final run, so a frame that ends the
// input is not left waiting for a transition that never comes.
func (r *Receiver) Flush() {
	r.emitRun(float64(r.n))
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

// clock is the run-length clock recovery: it times each transition of the
// sliced stream (interpolated between samples, and only once the new
// level has held for two samples, which rejects one-sample glitches) and
// turns the run since the previous one into whole chips at the tracked
// chip period.
func (r *Receiver) clock(v float64) {
	t := float64(r.n)
	hi := v > 0
	switch {
	case !r.pending && hi != r.level:
		frac := 0.0
		if r.vPrev != v {
			frac = r.vPrev / (r.vPrev - v)
		}
		r.pending, r.pendT = true, t-1+frac
	case r.pending && hi != r.level:
		r.emitRun(r.pendT)
		r.level = hi
		r.lastCross = r.pendT
		r.pending = false
	case r.pending:
		r.pending = false // glitch
	}
	r.vPrev = v
}

func (r *Receiver) emitRun(at float64) {
	run := at - r.lastCross
	if run <= 0 {
		return
	}
	n := int(run/r.spc + 0.5)
	if n < 1 {
		n = 1
	}
	if n <= 4 {
		r.spc += 0.1 * (run/float64(n) - r.spc)
		r.spc = min(max(r.spc, nominalSPC*0.85), nominalSPC*1.15)
	} else if run > 200*nominalSPC {
		r.spc = nominalSPC // silence: start the next preamble from nominal
	}
	n = min(n, 4096)
	chip := uint8(0)
	if r.level {
		chip = 1
	}
	for range n {
		r.framers[0].Push(chip)
		r.framers[1].Push(chip ^ 1)
	}
}
