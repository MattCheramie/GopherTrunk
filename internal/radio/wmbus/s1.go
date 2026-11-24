package wmbus

import (
	"math"
	"math/bits"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

// Mode S1 (EN 13757-4 stationary mode, meter to other), issue #1256.
//
// S1 meters transmit on 868.3 MHz, 650 kHz below the T1 / C1 channel: 2-FSK
// at ±50 kHz, 32.768 kchip/s, Manchester coded (a 1 is the chip pair 01, a 0
// is 10, so 16.384 kbit/s), a long preamble of alternating chips, the 24-chip
// access code 0x547696 (which breaks the Manchester rule on purpose, so no
// data can imitate it), then a format A frame, bytes MSB first, with the same
// block CRCs as T1. Facts as rtl-wmbus (xaelsouth, BSD-2) implements them:
// ACCESS_CODE_S1, MANCHESTER_IEEE_802_3 ("01" = one, "10" = zero) and its
// format A length table in s1_packet_decoder.h.
//
// The receiver here is its own chain, run beside the T / C one on the same
// IQ (ReceiverOptions.S1): its own offset mixer, resampler, channel filter,
// burst gate and chip clock. Each Manchester bit is decided from its two
// chips' soft values (second minus first), so there are no "invalid pair"
// aborts, and the magnitude ranks the bits for the same CRC repair C mode
// frames get.

const (
	// S1ChannelHz is the S1 meter-to-other channel centre.
	S1ChannelHz = 868_300_000
	// S1ChipRateHz is the S mode chip rate.
	S1ChipRateHz = 32_768
	// S1OffsetFromChannelHz is where the S1 channel sits relative to the
	// T1 / C1 channel (ChannelHz).
	S1OffsetFromChannelHz = S1ChannelHz - ChannelHz

	s1AccessCode  = 0x547696
	s1AccessMask  = 0xFFFFFF
	s1SyncErrors  = 1
	s1SPC         = float64(procRateHz) / S1ChipRateHz // 12.2 samples per chip
	s1Box         = 12                                 // matched filter: one chip
	s1ThrWindow   = 512                                // carrier estimate: 42 chips
	s1CutoffHz    = 150_000                            // ±50 kHz FSK + tuner error
	s1LoopGain    = 0.15
	s1LoopRate    = 0.004
	s1MaxRateDiff = 0.05
)

// s1Receiver demodulates S1 frames.
type s1Receiver struct {
	nco   *dsp.NCO
	rs    *dsp.Resampler
	chf   *filter.FIR
	ratio float64

	mixBuf, rsBuf, chBuf []complex64
	prev                 complex64

	ring   [s1ThrWindow]float64
	ringN  int
	rIdx   int
	rSum   float64
	thr    float64
	frozen bool

	box    [s1Box]float64
	boxIdx int
	boxSum float64

	power float64
	n     int64

	level     bool
	lastCross float64
	vPrev     float64
	spc       float64
	tSample   float64
	hist      [4]float64

	framers  [2]*s1Framer
	gate     *burstGate
	skipFrac float64
	bursts   int64
	demodded int64
}

func newS1Receiver(o ReceiverOptions, deliver func(Frame, FrameInfo)) *s1Receiver {
	r := &s1Receiver{spc: s1SPC, lastCross: math.Inf(-1)}
	r.nco = dsp.NewNCO(o.S1OffsetHz, o.SampleRateHz)
	in := int(math.Round(o.SampleRateHz))
	if in != procRateHz {
		g := gcd(procRateHz, in)
		l, m := procRateHz/g, in/g
		taps := (6*max(l, m) + l - 1) / l
		r.rs = dsp.NewResampler(l, m, max(taps, 8), 6)
	}
	r.ratio = o.SampleRateHz / procRateHz
	r.chf = filter.NewFIR(filter.LowpassKaiser(31, float64(s1CutoffHz)/procRateHz, 6))
	if !o.NoBurstGate {
		r.gate = newBurstGate(o.SampleRateHz)
	}
	for i := range r.framers {
		inv := i == 1
		r.framers[i] = &s1Framer{deliver: func(f Frame) {
			off := r.thr * procRateHz / (2 * math.Pi)
			if inv {
				off = -off
			}
			deliver(f, FrameInfo{
				LevelDBFS:       10 * math.Log10(r.power+1e-20),
				Sample:          int64(float64(r.n) * r.ratio),
				Inverted:        inv,
				CarrierOffsetHz: off,
			})
		}}
	}
	return r
}

func (r *s1Receiver) process(iq []complex64) {
	r.mixBuf = r.nco.Mix(r.mixBuf, iq)
	iq = r.mixBuf
	if r.gate != nil {
		r.gate.feed(r, iq)
		return
	}
	for len(iq) > 0 {
		n := min(len(iq), processBlock)
		r.demod(iq[:n])
		iq = iq[n:]
	}
}

func (r *s1Receiver) flush() {
	if r.gate != nil {
		r.gate.flush(r)
	}
	// Push the samples still inside the matched filter's span.
	for i := 0; i < s1Box; i++ {
		r.sample(0)
	}
}

func (r *s1Receiver) countBurst() { r.bursts++ }

func (r *s1Receiver) busy() bool { return r.framers[0].busy() || r.framers[1].busy() }

func (r *s1Receiver) skip(inputSamples int64) {
	r.skipFrac += float64(inputSamples) / r.ratio
	whole := math.Floor(r.skipFrac)
	r.skipFrac -= whole
	r.n += int64(whole)
	r.ring = [s1ThrWindow]float64{}
	r.ringN, r.rIdx, r.rSum = 0, 0, 0
	r.box = [s1Box]float64{}
	r.boxIdx, r.boxSum = 0, 0
	r.prev = 0
	r.level, r.vPrev = false, 0
	r.hist = [len(r.hist)]float64{}
	r.lastCross = math.Inf(-1)
	r.tSample = float64(r.n) + r.spc/2
}

func (r *s1Receiver) demod(iq []complex64) {
	r.demodded += int64(len(iq))
	if r.rs != nil {
		r.rsBuf = r.rs.Process(r.rsBuf[:0], iq)
		iq = r.rsBuf
	}
	r.chBuf = r.chf.Process(r.chBuf, iq)
	for _, x := range r.chBuf {
		r.sample(x)
	}
}

func (r *s1Receiver) sample(x complex64) {
	r.n++
	p := float64(real(x))*float64(real(x)) + float64(imag(x))*float64(imag(x))
	r.power += (p - r.power) / 512
	d := x * complex(real(r.prev), -imag(r.prev))
	r.prev = x
	f := math.Atan2(float64(imag(d)), float64(real(d)))

	// The carrier: the discriminator's mean over the window. Manchester
	// data is DC-balanced, so it holds through a frame too, but it is
	// frozen there anyway so the frame's own carrier is what it reports.
	r.rSum += f - r.ring[r.rIdx]
	r.ring[r.rIdx] = f
	r.rIdx = (r.rIdx + 1) % s1ThrWindow
	// The matched filter: the discriminator integrated over one chip.
	r.boxSum += f - r.box[r.boxIdx]
	r.box[r.boxIdx] = f
	r.boxIdx = (r.boxIdx + 1) % s1Box
	if r.ringN < s1ThrWindow {
		r.ringN++
		return
	}
	busy := r.busy()
	if !busy || !r.frozen {
		r.thr = r.rSum / s1ThrWindow
	}
	r.frozen = busy
	r.clock(r.boxSum/s1Box - r.thr)
}

// clock is the T / C receiver's chip clock (see Receiver.clock) at the S
// mode chip rate. The matched filter's output peaks at the end of each chip
// and crosses zero half a chip after a chip edge, so a crossing again falls
// half a chip before the decision instant.
func (r *s1Receiver) clock(v float64) {
	t := float64(r.n)
	if hi := v > 0; hi != r.level {
		frac := 0.0
		if r.vPrev != v {
			frac = r.vPrev / (r.vPrev - v)
		}
		tc := t - 1 + frac
		run := tc - r.lastCross
		switch {
		case run > 200*s1SPC:
			r.spc = s1SPC
			r.tSample = tc + r.spc/2
		case run < r.spc/2:
			// A noise spike through the threshold: measures nothing.
			r.level = hi
			r.push(v)
			return
		default:
			e := tc - (r.tSample - r.spc/2)
			e -= r.spc * math.Round(e/r.spc)
			r.tSample += s1LoopGain * e
			r.spc += s1LoopRate * e
			if n := math.Round(run / r.spc); n >= 1 && n <= 2 && !r.frozen {
				r.spc += 0.1 * (run/n - r.spc)
			}
			r.spc = min(max(r.spc, s1SPC*(1-s1MaxRateDiff)), s1SPC*(1+s1MaxRateDiff))
		}
		r.level = hi
		r.lastCross = tc
	}
	r.push(v)
}

func (r *s1Receiver) push(v float64) {
	copy(r.hist[:], r.hist[1:])
	r.hist[len(r.hist)-1] = v
	r.vPrev = v
	t := float64(r.n)
	for r.tSample <= t {
		x := interpAt(r.hist[:], r.tSample-(t-float64(len(r.hist)-1)))
		r.framers[0].push(x)
		r.framers[1].push(-x)
		r.tSample += r.spc
	}
}

func (r *s1Receiver) addStats(s *ReceiverStats) {
	s.Bursts += r.bursts
	s.Demodulated += r.demodded
	for _, f := range r.framers {
		fs := f.stats
		s.Framer.Syncs += fs.Syncs
		s.Framer.Frames += fs.Frames
		s.Framer.CRCOK += fs.CRCOK
		s.Framer.BadLength += fs.BadLength
		s.Framer.Repaired += fs.Repaired
		s.S1Framer.Syncs += fs.Syncs
		s.S1Framer.Frames += fs.Frames
		s.S1Framer.CRCOK += fs.CRCOK
		s.S1Framer.BadLength += fs.BadLength
		s.S1Framer.Repaired += fs.Repaired
	}
}

// s1Framer turns soft S mode chips into frames: the access code on the hard
// chips, then Manchester bits from chip pairs.
type s1Framer struct {
	deliver func(Frame)
	stats   FramerStats

	inFrame bool
	shift   uint32
	first   float64
	odd     bool
	cur     byte
	nbits   int
	need    int
	buf     []byte
	rel     []float32
}

func (f *s1Framer) busy() bool { return f.inFrame }

func (f *s1Framer) reset() {
	f.inFrame, f.shift, f.odd, f.nbits, f.need = false, 0, false, 0, 0
	f.buf, f.rel = f.buf[:0], f.rel[:0]
}

func (f *s1Framer) push(v float64) {
	if !f.inFrame {
		f.shift <<= 1
		if v > 0 {
			f.shift |= 1
		}
		if bits.OnesCount32((f.shift^s1AccessCode)&s1AccessMask) <= s1SyncErrors {
			f.stats.Syncs++
			f.reset()
			f.inFrame = true
		}
		return
	}
	if !f.odd {
		f.first, f.odd = v, true
		return
	}
	f.odd = false
	d := v - f.first // 01 = one: the second chip above the first
	f.cur <<= 1
	if d > 0 {
		f.cur |= 1
	}
	f.rel = append(f.rel, float32(math.Abs(d)))
	if f.nbits++; f.nbits < 8 {
		return
	}
	f.nbits = 0
	f.buf = append(f.buf, f.cur)
	if len(f.buf) == 1 {
		if f.need = frameLength(f.cur, FormatA); f.need == 0 {
			f.stats.BadLength++
			f.reset()
		}
		return
	}
	if len(f.buf) < f.need {
		return
	}
	raw := append([]byte(nil), f.buf...)
	rel := append([]float32(nil), f.rel...)
	f.reset()
	data, blocks, failed, err := checkBlocks(raw, FormatA)
	if err != nil {
		f.stats.BadLength++
		return
	}
	corrected := 0
	if len(failed) > 0 {
		if corrected = repairBlocks(raw, rel, FormatA, failed); corrected > 0 {
			data, blocks, failed, _ = checkBlocks(raw, FormatA)
			if len(failed) == 0 {
				f.stats.Repaired++
			}
		}
	}
	f.stats.Frames++
	ok := len(failed) == 0
	if ok {
		f.stats.CRCOK++
	}
	f.deliver(Frame{Mode: ModeS1, Format: FormatA, Raw: raw, Data: data, CRCOK: ok,
		Blocks: blocks, FailedBlocks: failed, CorrectedBits: corrected})
}

// encodeS1Chips is the S1 transmitter's chip stream, for tests and the
// synthesiser: preamble (alternating chips), the access code, the frame's
// bytes Manchester coded MSB first, and a short postamble.
func encodeS1Chips(raw []byte, preambleChips int) []uint8 {
	var out []uint8
	for i := range preambleChips {
		out = append(out, uint8(i&1))
	}
	for i := s1AccessChips - 1; i >= 0; i-- {
		out = append(out, uint8(uint32(s1AccessCode)>>i&1))
	}
	for _, b := range raw {
		for i := 7; i >= 0; i-- {
			if b>>i&1 == 1 {
				out = append(out, 0, 1)
			} else {
				out = append(out, 1, 0)
			}
		}
	}
	return append(out, 0, 1, 0, 1)
}

const s1AccessChips = 24
