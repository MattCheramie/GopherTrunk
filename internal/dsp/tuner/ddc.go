package tuner

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
)

// nco is a recursive complex-exponential generator. Each Step call
// returns the current phasor and advances by step = e^{-j 2π f / Fs}.
// Phasor magnitude drifts away from unity over time due to repeated
// float32 multiplies; renorm() folds it back to 1 every renormalize
// samples. The fix-up is cheap (one sqrt) and the drift is bounded
// well below the bank's overall numeric noise floor in between.
type nco struct {
	phasor       complex64
	step         complex64
	sinceRenorm  int
	renormEveryN int
}

func newNCO(offsetHz, sampleRateHz float64) *nco {
	n := &nco{phasor: complex(1, 0), renormEveryN: 4096}
	n.setOffset(offsetHz, sampleRateHz)
	return n
}

func (n *nco) setOffset(offsetHz, sampleRateHz float64) {
	theta := -2 * math.Pi * offsetHz / sampleRateHz
	n.step = complex(float32(math.Cos(theta)), float32(math.Sin(theta)))
}

func (n *nco) reset() {
	n.phasor = complex(1, 0)
	n.sinceRenorm = 0
}

func (n *nco) renorm() {
	r := real(n.phasor)
	i := imag(n.phasor)
	mag := float32(math.Sqrt(float64(r)*float64(r) + float64(i)*float64(i)))
	if mag > 0 {
		n.phasor = complex(r/mag, i/mag)
	} else {
		n.phasor = complex(1, 0)
	}
	n.sinceRenorm = 0
}

// DDCBank is a Bank built from one independent NCO mixer + rational
// polyphase resampler per tap. CPU cost is O(N_taps · N_samples).
//
// At high input rates a shared integer decimation stage runs once over the
// wideband stream (amortised across every tap) to bring it down to redRateHz
// before the per-tap NCO + resampler. This bounds the per-tap cost to the rate
// the single-stage resampler was tuned for (~2.5 MS/s) instead of letting a
// 208:1 single-stage decimation run per tap at the full input rate, which
// starved the live wideband pump goroutine and dropped samples at the hardware
// layer when the rate was raised above 2.5 MS/s (issue #764). Below
// ddcDecimateFloorHz·2 there is no shared stage and the path is byte-for-byte
// identical to the pre-#764 behaviour.
type DDCBank struct {
	inRateHz    float64
	redRateHz   float64 // rate after the shared decimation stage (== inRateHz when predec is nil)
	outRateHz   float64
	actualOutHz float64
	guardFrac   float64
	taps        []*ddcTap

	// predec is the shared decimate-by-D resampler applied to every chunk
	// before the per-tap mixers. A decimating dsp.Resampler (L=1, M=D) only
	// computes the kept output samples, so the shared cost is far below the
	// per-tap resamplers it offloads. Nil when inRateHz is at/below the floor.
	predec *dsp.Resampler
	decBuf []complex64 // reduced-rate scratch reused across Process calls

	// pool, when non-nil, runs the per-tap mix + resample + sink across worker
	// goroutines (EnableWorkers); nil keeps the serial loop.
	pool *tapPool
}

type ddcTap struct {
	offsetHz  float64
	actualOut float64 // this tap's emitted rate (may differ per tap; see AddTapAtRate)
	nco       *nco
	resampler *dsp.Resampler
	mixBuf    []complex64 // this tap's own mix scratch, so taps may run concurrently
	outBuf    []complex64
	sink      SinkFunc
}

// NewDDCBank constructs a DDCBank that takes wide-band IQ at inRateHz and
// produces narrow-band IQ at outRateHz per tap. The guardFrac (0..0.5)
// reserves a fraction of the IQ band at each edge as a guard against
// alias roll-off; AddTap rejects offsets outside the usable band. A typical
// value is 0.05 (5%).
func NewDDCBank(inRateHz, outRateHz, guardFrac float64) *DDCBank {
	return NewDDCBankForSpan(inRateHz, outRateHz, guardFrac, 0)
}

// NewDDCBankForSpan is NewDDCBank with an explicit minimum half-band the bank
// must keep usable: the shared decimator stops halving before the reduced
// rate would push the usable band (±redRate·(0.5-guardFrac)) below
// requiredHalfHz, so taps as far out as ±requiredHalfHz stay in band. Pass 0
// for the default behaviour (decimate down to ddcDecimateFloorHz, ±1.1 MHz at
// 2.5 MS/s). Use this when the tap set spans wider than that floor allows —
// e.g. a 70-channel DMR plan spread across ±1.9 MHz of a 10 MS/s capture,
// which the default floor would reject with ErrOffsetOutOfBand.
func NewDDCBankForSpan(inRateHz, outRateHz, guardFrac, requiredHalfHz float64) *DDCBank {
	return NewDDCBankForPlan(inRateHz, outRateHz, guardFrac, requiredHalfHz, 0)
}

// NewDDCBankForPlan is NewDDCBankForSpan for a caller that also knows the
// widest per-tap output rate its plan will register (maxTapRateHz, e.g. 144 kHz
// for a TETRA tap, 48 kHz for DMR / P25). With that known, the shared
// decimator may halve BELOW the 2.5 MS/s floor as long as every tap — out to
// ±requiredHalfHz, plus half its own output rate — stays inside the shared
// stage's flat passband (sharedDecimPassFrac of the reduced rate), so the
// per-tap resamplers run at the lowest rate the plan allows. That is the
// difference between keeping up and not on a small board (#1321): an Airspy
// Mini cannot stream below 3 MS/s, and two TETRA cells within ±200 kHz of its
// centre used to cost two full-rate 3 MS/s → 144 kHz resamplers (~250 taps per
// output each) on the single pump goroutine; decimating the shared stream to
// 375 kHz first makes each tap ~8× cheaper for one cheap shared stage. A
// plan wider than the floor allows keeps the floor: behaviour above it is
// unchanged. maxTapRateHz 0 is NewDDCBankForSpan.
func NewDDCBankForPlan(inRateHz, outRateHz, guardFrac, requiredHalfHz, maxTapRateHz float64) *DDCBank {
	minRedRateHz := 0.0
	if requiredHalfHz > 0 && guardFrac < 0.5 {
		minRedRateHz = requiredHalfHz / (0.5 - guardFrac)
	}
	floorHz := ddcDecimateFloorHz
	if maxTapRateHz > 0 {
		// The widest tap's channel must sit in the shared stage's passband:
		// |offset| + tapRate/2 ≤ sharedDecimPassFrac · redRate.
		if need := (requiredHalfHz + maxTapRateHz/2) / sharedDecimPassFrac; need < floorHz {
			floorHz = need
		}
	}
	redRateHz, decim := buildSharedDecimator(inRateHz, minRedRateHz, floorHz)
	l, m := rationalRatio(outRateHz, redRateHz)
	return &DDCBank{
		inRateHz:    inRateHz,
		redRateHz:   redRateHz,
		outRateHz:   outRateHz,
		actualOutHz: redRateHz * float64(l) / float64(m),
		guardFrac:   guardFrac,
		predec:      decim,
	}
}

// ddcDecimateFloorHz is the lowest reduced rate the shared decimator targets.
// Halving stops while the next halving would still leave the rate at/above
// this floor, so the reduced rate lands in [floor, 2·floor). 2.5 MS/s keeps
// the usable band ≥ ±1.1 MHz (after the 5% guard) and the per-tap resampler
// ratio in the same regime the single-stage path was tuned for.
const ddcDecimateFloorHz = 2_500_000.0

// sharedDecimPassFrac is the fraction of the reduced rate the shared
// decimator's passband is flat to. Its prototype is a Kaiser (β 7, ~72 dB)
// lowpass of ddcStopbandTaps·D taps at cutoff redRate/2, whose transition
// band is ≈0.37·redRate wide and centred on the cutoff — so content is
// passed cleanly (and its alias rejected) only out to ≈0.31·redRate.
// NewDDCBankForPlan lowers the floor only while every tap fits inside this.
const sharedDecimPassFrac = 0.30

// buildSharedDecimator picks the power-of-two decimation D that brings inRateHz
// down to the [floor, 2·floor) window and returns the reduced rate plus a
// shared decimate-by-D resampler (nil when inRateHz is already below 2·floor,
// leaving the legacy single-stage path untouched). The decimating resampler
// computes only the kept outputs, so the shared cost is bounded by the reduced
// rate, not the full input rate. tapsPerBranch mirrors the per-tap
// anti-aliasing convention (ddcStopbandTaps taps per unit of decimation).
//
// minRedRateHz raises the lower bound on the reduced rate: halving stops while
// the next halving would leave the rate at/above BOTH ddcDecimateFloorHz and
// minRedRateHz. A caller hosting taps spread wider than the default ±1.1 MHz
// passes the reduced rate its span demands so those taps stay in band; 0
// selects the legacy floor-only behaviour. floorHz is the lowest reduced rate
// to target (ddcDecimateFloorHz, or lower when the caller has proved its
// plan fits — NewDDCBankForPlan).
func buildSharedDecimator(inRateHz, minRedRateHz, floorHz float64) (redRateHz float64, predec *dsp.Resampler) {
	red := inRateHz
	d := 1
	for red/2 >= floorHz && red/2 >= minRedRateHz {
		red /= 2
		d *= 2
	}
	if d == 1 {
		return inRateHz, nil
	}
	tapsPerBranch := ddcStopbandTaps * d
	if tapsPerBranch < ddcMinTapsPerBranch {
		tapsPerBranch = ddcMinTapsPerBranch
	}
	return red, dsp.NewResampler(1, d, tapsPerBranch, ddcKaiserBeta)
}

// AddTap registers a new tap at the given offset, emitting the bank's default
// output rate (OutputRateHz).
func (b *DDCBank) AddTap(offsetHz float64, sink SinkFunc) error {
	_, err := b.AddTapAtRate(offsetHz, b.outRateHz, sink)
	return err
}

// AddTapAtRate registers a tap that emits its OWN output rate, independent of the
// bank default and of other taps. This lets one bank host, e.g., 48 kHz DMR/P25
// taps alongside a 144 kHz TETRA tap out of the same wideband capture: each tap
// owns an independent polyphase resampler from the shared reduced-rate stream, so
// mixed per-tap rates cost nothing extra. Returns the tap's actual emitted rate,
// which may differ from outRateHz by a fraction of a percent when the exact ratio
// exceeds the resampler caps (issue #550) — build that tap's symbol clock from the
// returned value. The offset gate is unchanged (the shared reduced band is common
// to all taps).
func (b *DDCBank) AddTapAtRate(offsetHz, outRateHz float64, sink SinkFunc) (actualHz float64, err error) {
	if !b.offsetInBand(offsetHz) {
		return 0, ErrOffsetOutOfBand
	}
	if outRateHz <= 0 {
		outRateHz = b.outRateHz
	}
	l, m := rationalRatio(outRateHz, b.redRateHz)
	tapsPerBranch := (ddcStopbandTaps*m + l - 1) / l
	if tapsPerBranch < ddcMinTapsPerBranch {
		tapsPerBranch = ddcMinTapsPerBranch
	}
	actual := b.redRateHz * float64(l) / float64(m)
	tap := &ddcTap{
		offsetHz:  offsetHz,
		actualOut: actual,
		nco:       newNCO(offsetHz, b.redRateHz),
		resampler: dsp.NewResampler(l, m, tapsPerBranch, ddcKaiserBeta),
		sink:      sink,
	}
	b.taps = append(b.taps, tap)
	return actual, nil
}

// offsetInBand tests the offset against the usable band AFTER the shared
// decimation cascade. A tap beyond the reduced Nyquist would alias once the
// cascade narrows the band, so it is rejected here rather than silently
// folded. The 2.5 MS/s reduced-rate floor keeps this band ≥ ±1.1 MHz, wider
// than anything the analog front end actually delivers at these offsets.
func (b *DDCBank) offsetInBand(offsetHz float64) bool {
	half := b.redRateHz * (0.5 - b.guardFrac)
	return offsetHz >= -half && offsetHz <= half
}

// EnableWorkers runs the per-tap chains (NCO mix, resample, sink) across up to
// workers goroutines once there are at least two taps; a no-op otherwise (the
// serial loop). Unlike the channelizer, whose per-tap work is a cheap
// fine-tune off a shared FFT, a DDC tap resamples the whole reduced-rate
// stream, so even two taps are worth splitting: on a four-core board the
// two-TETRA-cell pump (#1321) was one saturated thread beside idle cores. Each
// tap owns its mix / resampler / output buffers and reads the reduced stream
// read-only, so taps are independent; sinks run concurrently with each other,
// never with themselves, and Process barriers before returning. Call after
// the last AddTap and before the first Process; Close stops the workers.
func (b *DDCBank) EnableWorkers(workers int) {
	if workers <= 1 || len(b.taps) < 2 {
		return
	}
	b.pool = newTapPool(len(b.taps), workers)
}

// Close stops the worker pool, if any. Idempotent.
func (b *DDCBank) Close() {
	if b.pool != nil {
		b.pool.close()
	}
}

// processTap runs one tap's chain over the reduced-rate stream.
func (b *DDCBank) processTap(t *ddcTap, wide []complex64) {
	if cap(t.mixBuf) < len(wide) {
		t.mixBuf = make([]complex64, len(wide))
	} else {
		t.mixBuf = t.mixBuf[:len(wide)]
	}
	mixInto(t.mixBuf, wide, t.nco)
	t.outBuf = t.resampler.Process(t.outBuf, t.mixBuf)
	t.sink(t.outBuf)
}

// Process mixes each tap down to baseband and decimates to OutputRateHz,
// invoking each tap's sink with the resulting chunk.
func (b *DDCBank) Process(src []complex64) {
	if len(src) == 0 {
		for _, t := range b.taps {
			t.sink(nil)
		}
		return
	}
	// Run the shared decimator once; every tap mixes the reduced-rate stream.
	// When there is no shared stage (low input rate) this is src itself, so
	// the per-tap path is unchanged from the pre-#764 behaviour.
	wide := src
	if b.predec != nil {
		b.decBuf = b.predec.Process(b.decBuf, src)
		wide = b.decBuf
	}
	if b.pool != nil {
		b.pool.run(func(i int) { b.processTap(b.taps[i], wide) })
		return
	}
	for _, t := range b.taps {
		b.processTap(t, wide)
	}
}

// mixInto multiplies src by the NCO phasor and writes to dst (len(dst) ==
// len(src) is the caller's contract). The phasor is advanced and
// periodically renormalised to stay on the unit circle.
//
// The products are spelled out in float32: Go evaluates a complex64 multiply
// through float64 (widen, multiply, narrow), which on 32-bit and small ARM
// cores is the slow part of this loop (the fft.Radix2 lesson). The float32
// phasor recursion rounds differently from the complex64 one, so the mixed
// stream is not bit-identical to the pre-#1321 one; the NCO's periodic
// renormalisation bounds the drift exactly as before.
func mixInto(dst, src []complex64, n *nco) {
	pr, pi := real(n.phasor), imag(n.phasor)
	sr, si := real(n.step), imag(n.step)
	left := n.renormEveryN - n.sinceRenorm
	for i, x := range src {
		xr, xi := real(x), imag(x)
		dst[i] = complex(xr*pr-xi*pi, xr*pi+xi*pr)
		pr, pi = pr*sr-pi*si, pr*si+pi*sr
		if left--; left <= 0 {
			n.phasor = complex(pr, pi)
			n.renorm()
			pr, pi = real(n.phasor), imag(n.phasor)
			left = n.renormEveryN
		}
	}
	n.phasor = complex(pr, pi)
	n.sinceRenorm = n.renormEveryN - left
}

// InputRateHz returns the wide-band sample rate.
func (b *DDCBank) InputRateHz() float64 { return b.inRateHz }

// OutputRateHz returns the per-tap narrow-band sample rate the bank actually
// emits. This equals the construction target when the rational resampler can
// hit it exactly, but may differ by a fraction of a percent when the exact
// ratio exceeds the resampler caps (issue #550); consumers should build their
// symbol clocks from this value, not the nominal target.
func (b *DDCBank) OutputRateHz() float64 { return b.actualOutHz }

// ActualRateFor returns the rate a tap requesting outRateHz would actually emit,
// without adding a tap. Lets a caller size a receiver's symbol clock before it
// wires the tap's sink (AddTapAtRate needs that sink). outRateHz <= 0 selects the
// bank default. Uses the same rational-ratio computation as AddTapAtRate, so the
// two agree.
func (b *DDCBank) ActualRateFor(outRateHz float64) float64 {
	if outRateHz <= 0 {
		outRateHz = b.outRateHz
	}
	l, m := rationalRatio(outRateHz, b.redRateHz)
	return b.redRateHz * float64(l) / float64(m)
}

// Reset clears the shared decimator plus every tap's NCO and resampler state.
func (b *DDCBank) Reset() {
	if b.predec != nil {
		b.predec.Reset()
	}
	for _, t := range b.taps {
		t.nco.reset()
		t.resampler.Reset()
	}
}

// Constants mirror internal/scanner/ccdecoder/ddc.go so the wide-band
// taps yield the same anti-alias characteristics as the existing
// single-channel down-converter (>60 dB stopband, ~70 dB peak sidelobe).
const (
	ddcStopbandTaps     = 12
	ddcMinTapsPerBranch = 8
	ddcKaiserBeta       = 7.0
)

// rationalRatio reduces out/in to lowest L/M terms. When the exact reduction
// exceeds the resampler caps (L>64 branches or M>8192), it falls back to the
// closest representable ratio under those caps instead of a crude integer
// decimator. The old fallback (L=1, M=round(in/out)) silently changed the
// achieved rate by up to a few percent — e.g. for a polyphase channelizer bin
// rate of 390625 Hz (6.25 MS/s ÷16, a power of five with no factor of two) the
// exact 48000/390625 reduces to L=384/M=3125, trips L>64, and the old fallback
// produced 390625/8 = 48828 Hz: a 1.7% symbol-clock error that broke decode
// (issue #550). The bounded search instead returns L=7/M=57 (0.06% error), well
// within what the symbol-timing loops tolerate.
func rationalRatio(out, in float64) (l, m int) {
	outI := int(math.Round(out))
	inI := int(math.Round(in))
	if outI <= 0 || inI <= 0 {
		return 1, 1
	}
	g := gcdInt(outI, inI)
	l, m = outI/g, inI/g
	if l > 64 || m > 8192 {
		return bestRatioUnderCaps(float64(outI) / float64(inI))
	}
	return l, m
}

// bestRatioUnderCaps returns the L/M (L in 1..64, M in 1..8192) whose ratio is
// closest to r. O(64) and called only on the cap-exceeded fallback path, so it
// has no per-sample cost.
func bestRatioUnderCaps(r float64) (l, m int) {
	bestL, bestM := 1, 1
	bestErr := math.Inf(1)
	for li := 1; li <= 64; li++ {
		mi := int(math.Round(float64(li) / r))
		if mi < 1 {
			mi = 1
		} else if mi > 8192 {
			mi = 8192
		}
		if e := math.Abs(float64(li)/float64(mi) - r); e < bestErr {
			bestErr, bestL, bestM = e, li, mi
		}
	}
	return bestL, bestM
}

func gcdInt(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	if a < 0 {
		a = -a
	}
	return a
}
