package wmbus

import "math"

// The burst gate keeps the demodulator off the IQ between bursts.
//
// A meter is on the air for a few milliseconds every few minutes. On the
// issue #1256 walk of 16 Nov the 770 telegrams the receiver framed add up to
// about 5 s of air in 57 minutes of IQ: 0.2 %. The receiver demodulated all
// of it: resampler, channel filter, discriminator, tone correlators and chip
// clock on every sample, about 0.06 of a desktop core, and on the reporter's
// phone enough that whenever the phone slowed the process down (screen off)
// the decoder got through only 26-51 % of real time and the SDR dropped the
// rest: 108 s, 159 s, 231 s and 139 s stretches of unheard air on that walk.
//
// The gate watches the channel's power with a detector that costs two
// additions per input sample, and runs the demodulator only on the IQ around
// a rise above the noise floor. What it has to guarantee is that everything
// the demodulator could decode still reaches it:
//
//   - The detector is a boxcar over gateDecim input samples (down to about
//     the processing rate), DC removed, then the mean power of a block of
//     gateBlockLen of those: 0.16 ms. The boxcar passes the channel with
//     under 1.5 dB of droop at the FSK tones of a tuner 50 kHz off.
//   - The gate opens on gateOpenBlocks consecutive blocks whose power
//     exceeds the tracked floor by gateOpenDB. On the 16 Nov walk the
//     weakest telegram that decodes raises the detector 4.6 dB over the
//     floor, and the weakest one that was framed at all (CRC failed)
//     3.2 dB; the gate opens at 1.5 dB. A telegram lasts dozens of blocks,
//     so asking for two costs it nothing, and it keeps noise from opening
//     the gate: one block in a few hundred crosses 1.5 dB by chance, which
//     alone had the demodulator on 7 % of pure noise.
//   - The demodulator needs the IQ before the burst (the threshold window,
//     the filters and the preamble), so the last gateLookback blocks are
//     kept and run first when the gate opens: 1.3 ms, where the threshold
//     window and the filters together need 0.4 ms.
//   - The gate stays open for gateHold blocks after the last block over the
//     threshold, and for as long as either framer is inside a frame, so a
//     fade inside a telegram does not close it.
const (
	// gateBlockLen is the detector samples per block.
	gateBlockLen = 64
	// gateOpenDB is how far over the floor a block's power must be, and
	// gateOpenBlocks for how many blocks in a row.
	gateOpenDB     = 1.5
	gateOpenBlocks = 2
	// gateLookback is how many blocks before the opening one are
	// demodulated first; gateHold how many after the last one over the
	// threshold.
	gateLookback = 8
	gateHold     = 4
	// gateWarmBlocks is how long the gate stays open at the start of a
	// stream, while the floor and the DC estimate settle: 20 ms.
	gateWarmBlocks = 128
	// gateFloorBlocks and gateDCBlocks are the floor's and the DC
	// estimate's averaging lengths, in blocks.
	gateFloorBlocks = 64
	gateDCBlocks    = 32
	// gateRise is how fast the floor climbs while blocks stay over the
	// threshold: it doubles in 1.8 s. A lasting rise in the noise (a new
	// place, a gain change) therefore holds the gate open for a few
	// seconds and then stops; a transmission elsewhere in the tuner's span
	// that lasts a second holds it open throughout, as the receiver ran
	// before the gate, and does not hide a weak telegram behind a raised
	// floor; a 40 ms telegram moves the floor 0.01 dB. Downwards the floor
	// follows the blocks under the threshold, in gateFloorBlocks.
	gateRise = 1.0 / 16384
)

var gateOpenRatio = math.Pow(10, gateOpenDB/10)

type burstGate struct {
	decim int // input samples per detector sample
	block int // input samples per block

	floor      float64
	dcR, dcI   float32 // DC of the detector samples
	seeded     bool
	warm, hold int
	hotRun     int // consecutive blocks over the threshold

	// carry is the partial block left by the previous Process call.
	carry []complex64
	// pend holds the last gateLookback blocks that were not demodulated,
	// oldest at pendHead; skipped counts the input samples that fell out
	// of it since the demodulator last ran.
	pend     []complex64
	pendHead int
	pendN    int
	skipped  int64
}

// gatedDemod is the demodulator a burst gate runs: the T/C Receiver, or the
// S1 receiver.
type gatedDemod interface {
	demod(iq []complex64)
	busy() bool
	skip(inputSamples int64)
	countBurst()
}

func newBurstGate(sampleRateHz float64) *burstGate {
	d := max(1, int(math.Round(sampleRateHz/procRateHz)))
	g := &burstGate{decim: d, block: d * gateBlockLen, warm: gateWarmBlocks}
	g.pend = make([]complex64, gateLookback*g.block)
	g.carry = make([]complex64, 0, g.block)
	return g
}

// feed cuts the input into blocks and runs each through the gate.
func (g *burstGate) feed(r gatedDemod, iq []complex64) {
	if len(g.carry) > 0 {
		need := g.block - len(g.carry)
		if len(iq) < need {
			g.carry = append(g.carry, iq...)
			return
		}
		g.carry = append(g.carry, iq[:need]...)
		iq = iq[need:]
		g.blockIn(r, g.carry)
		g.carry = g.carry[:0]
	}
	for len(iq) >= g.block {
		g.blockIn(r, iq[:g.block])
		iq = iq[g.block:]
	}
	g.carry = append(g.carry, iq...)
}

func (g *burstGate) blockIn(r gatedDemod, b []complex64) {
	if g.measure(b) {
		g.hotRun++
	} else {
		g.hotRun = 0
	}
	burst := g.hotRun >= gateOpenBlocks
	if burst {
		if g.hold == 0 && g.warm == 0 {
			r.countBurst()
		}
		g.hold = gateHold
	}
	if !(g.hold > 0 || g.warm > 0 || r.busy()) {
		g.keep(b)
		return
	}
	g.runPending(r)
	r.demod(b)
	if !burst && g.hold > 0 {
		g.hold--
	}
	if g.warm > 0 {
		g.warm--
	}
}

// measure updates the floor with a block and reports whether the block is
// over the threshold.
func (g *burstGate) measure(b []complex64) bool {
	d := g.decim
	var acc, sumR, sumI float32
	for i := 0; i+d <= len(b); i += d {
		var ar, ai float32
		for _, x := range b[i : i+d] {
			ar += real(x)
			ai += imag(x)
		}
		sumR += ar
		sumI += ai
		ar -= g.dcR
		ai -= g.dcI
		acc += ar*ar + ai*ai
	}
	n := float32(len(b) / d)
	p := float64(acc / n)
	if !g.seeded {
		// The first block: its own mean is the DC estimate, and its power
		// about it the floor. The gate is held open while both settle.
		g.seeded = true
		g.dcR, g.dcI = sumR/n, sumI/n
		g.floor = p
		return false
	}
	g.dcR += (sumR/n - g.dcR) / gateDCBlocks
	g.dcI += (sumI/n - g.dcI) / gateDCBlocks
	hot := p > gateOpenRatio*g.floor
	if hot {
		g.floor *= 1 + gateRise
	} else {
		g.floor += (p - g.floor) / gateFloorBlocks
	}
	return hot
}

// keep stores a block the demodulator did not run on, in case the gate
// opens within the lookback.
func (g *burstGate) keep(b []complex64) {
	slot := (g.pendHead + g.pendN) % gateLookback
	if g.pendN == gateLookback {
		// The oldest block falls out of the lookback: it is skipped.
		g.pendHead = (g.pendHead + 1) % gateLookback
		g.skipped += int64(g.block)
	} else {
		g.pendN++
	}
	copy(g.pend[slot*g.block:(slot+1)*g.block], b)
}

// runPending demodulates the kept blocks, after telling the receiver how
// much input was skipped before them.
func (g *burstGate) runPending(r gatedDemod) {
	if g.skipped > 0 {
		r.skip(g.skipped)
		g.skipped = 0
	}
	for ; g.pendN > 0; g.pendN-- {
		r.demod(g.pend[g.pendHead*g.block : (g.pendHead+1)*g.block])
		g.pendHead = (g.pendHead + 1) % gateLookback
	}
}

// flush demodulates a partial last block if the gate is open, so a frame
// that ends the input is delivered.
func (g *burstGate) flush(r gatedDemod) {
	if len(g.carry) > 0 && (g.hold > 0 || g.warm > 0 || r.busy()) {
		g.runPending(r)
		r.demod(g.carry)
	}
	g.carry = g.carry[:0]
}

// busy reports that a framer is inside a frame.
func (r *Receiver) countBurst() { r.stats.Bursts++ }

func (r *Receiver) busy() bool {
	return r.framers[0].busy() || r.framers[1].busy()
}

// skip accounts for input the demodulator did not see: the sample counter
// moves on, so frame positions and the carrier hold still count stream
// time, and the sliding windows and the chip clock start again, as they
// would after silence.
func (r *Receiver) skip(inputSamples int64) {
	r.skipFrac += float64(inputSamples) / r.ratio
	whole := math.Floor(r.skipFrac)
	r.skipFrac -= whole
	r.n += int64(whole)
	// The threshold window refills before anything is clocked again. Its
	// old contents go too: rSum is a running sum that subtracts the value
	// each new one replaces.
	r.ring = [thresholdWindow]float64{}
	r.ringN, r.rIdx, r.rSum = 0, 0, 0
	r.prev = 0
	r.toneAge = 0
	r.level = false
	r.vPrev = 0
	r.hist = [len(r.hist)]float64{}
	// The chip clock restarts on the next transition, and no chip is
	// decided for the gap.
	r.lastCross = math.Inf(-1)
	r.tSample = float64(r.n) + r.spc/2
}
