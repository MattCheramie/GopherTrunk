package composer

import (
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

// oldFrontEnd reproduces the pre-change front end —
// `decimateComplex(lpf.Process(nil, iq), decim)` with the same 81-tap
// Kaiser LPF the chains built — so the decimating FIR can be pinned
// byte-for-byte against the code it replaced. filterAtUnity mirrors the
// two decim==1 behaviours: the digital chains' raw `samples := iq`
// (false) and the FM chain's filtered `decimateComplex(lpf.Process,1)`
// (true).
func oldFrontEnd(iq []complex64, inRateHz, targetHz, bwHz float64, filterAtUnity bool) []complex64 {
	decim := int(inRateHz) / int(targetHz)
	if decim < 1 {
		decim = 1
	}
	if decim == 1 && !filterAtUnity {
		return iq // old `samples := iq` no-op
	}
	cutoff := bwHz / inRateHz
	if cutoff > 0.45 {
		cutoff = 0.45
	}
	lpf := filter.NewFIR(filter.LowpassKaiser(81, cutoff, 8.6))
	return decimateComplex(lpf.Process(nil, iq), decim)
}

func tone(n int) []complex64 {
	out := make([]complex64, n)
	for i := range out {
		// A deterministic, slowly-rotating tone with a little structure so
		// the filter actually does something. No math/rand (banned in this
		// harness) and no Date dependency.
		p := float64(i) * 0.013
		q := float64(i) * 0.0007
		out[i] = complex(float32(0.7*math.Cos(p)+0.2*math.Cos(q)), float32(0.7*math.Sin(p)-0.2*math.Sin(q)))
	}
	return out
}

// TestDecimatingFIRMatchesOldFrontEnd is the core regression guard: the
// decimating FIR must produce bit-for-bit the same samples as the old
// filter-everything-then-stride front end, at the field-reported 2.4 MS/s
// (decim=50) and a couple of other ratios, both as one call and fed in
// streaming chunks (history + decimation phase must carry across chunks).
// This is what lets the change claim "decode unchanged, only faster".
func TestDecimatingFIRMatchesOldFrontEnd(t *testing.T) {
	const (
		bw     = 12_500.0
		target = 48_000.0
	)
	// false = digital-chain decim==1 semantics (raw passthrough); true =
	// FM-chain semantics (band-limit even at unity). Both must stay
	// byte-for-byte equal to the old front end at every rate.
	for _, filterAtUnity := range []bool{false, true} {
		for _, rate := range []float64{96_000, 240_000, 2_400_000} {
			iq := tone(20_000)
			want := oldFrontEnd(iq, rate, target, bw, filterAtUnity)

			// Single call.
			fe := newDecimatingFIR(rate, target, bw, filterAtUnity)
			got := fe.Process(nil, iq)
			if fe.OutRateHz() != rate/float64(int(rate)/int(target)) {
				t.Errorf("rate=%.0f OutRateHz=%.4f", rate, fe.OutRateHz())
			}
			if len(got) != len(want) {
				t.Fatalf("rate=%.0f unity=%v: len got=%d want=%d", rate, filterAtUnity, len(got), len(want))
			}
			for i := range got {
				if got[i] != want[i] {
					t.Fatalf("rate=%.0f unity=%v single: sample %d got=%v want=%v", rate, filterAtUnity, i, got[i], want[i])
				}
			}

			// Streaming chunks of an awkward size that is not a multiple of
			// decim, to prove the decimation phase carries across boundaries.
			fe2 := newDecimatingFIR(rate, target, bw, filterAtUnity)
			var streamed []complex64
			var dst []complex64
			for i := 0; i < len(iq); i += 777 {
				e := i + 777
				if e > len(iq) {
					e = len(iq)
				}
				dst = fe2.Process(dst, iq[i:e])
				streamed = append(streamed, dst...)
			}
			if len(streamed) != len(want) {
				t.Fatalf("rate=%.0f unity=%v: streamed len=%d want=%d", rate, filterAtUnity, len(streamed), len(want))
			}
			for i := range streamed {
				if streamed[i] != want[i] {
					t.Fatalf("rate=%.0f unity=%v chunked: sample %d got=%v want=%v", rate, filterAtUnity, i, streamed[i], want[i])
				}
			}
		}
	}
}

// TestDecimatingFIRFilterAtUnity pins the FM-chain decim==1 behaviour:
// with filterAtUnity the front end still band-limits (it does not return
// raw), matching the old FM `decimateComplex(lpf.Process(iq), 1)`.
func TestDecimatingFIRFilterAtUnity(t *testing.T) {
	fe := newDecimatingFIR(48_000, 48_000, 12_500, true)
	if fe.OutRateHz() != 48_000 {
		t.Errorf("unity OutRateHz=%.3f, want 48000", fe.OutRateHz())
	}
	in := tone(2048)
	got := fe.Process(nil, in)
	want := oldFrontEnd(in, 48_000, 48_000, 12_500, true)
	if len(got) != len(in) {
		t.Fatalf("unity len got=%d, want %d (no decimation)", len(got), len(in))
	}
	if &got[0] == &in[0] {
		t.Fatal("filterAtUnity must filter, not return the input slice unchanged")
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("unity sample %d got=%v want=%v", i, got[i], want[i])
		}
	}
}

// TestDecimatingFIRPassThrough pins the decim==1 parity path: a source
// already at the intermediate rate is returned unchanged (no filtering),
// byte-exact with the old `samples := iq`. The existing 48 kHz chain
// tests rely on this.
func TestDecimatingFIRPassThrough(t *testing.T) {
	fe := newDecimatingFIR(48_000, 48_000, 12_500, false)
	if fe.OutRateHz() != 48_000 {
		t.Errorf("pass-through OutRateHz=%.3f, want 48000", fe.OutRateHz())
	}
	in := tone(64)
	out := fe.Process(nil, in)
	if len(out) != len(in) || &out[0] != &in[0] {
		t.Errorf("pass-through must return the input slice unchanged (len %d want %d, aliased=%v)",
			len(out), len(in), len(out) > 0 && &out[0] == &in[0])
	}
}
