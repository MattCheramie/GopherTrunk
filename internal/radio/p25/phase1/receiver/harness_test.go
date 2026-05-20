package receiver

// This file is the reproduction harness for issue #275 ("Unable to
// lock / decode Control Channel on P25 system").
//
// The end-to-end integration test (cmd/gophertrunk/integration_cc_test.go)
// synthesises a mathematically ideal IQ stream — no carrier offset, no
// noise, no DC spike, no IQ imbalance — and feeds it in large chunks, so
// the whole demod chain passes while real RTL-SDR captures fail. That
// gap is why #275 went through six guess-fix-retest PRs.
//
// The harness here drives the *real* receiver + control-channel chain
// (receiver.New → DibitSink → phase1.ControlChannel.Process) against
// clean and deliberately-impaired IQ, fed in RTL-realistic small chunks:
//
//   - TestHarnessC4FMCleanLocks hard-asserts the clean C4FM signal
//     locks — a permanent regression guard.
//   - TestHarnessCQPSKChunkBoundary isolates the CQPSK-side root cause:
//     the Gardner timing loop's symbol count is chunk-size dependent.
//   - TestHarnessImpairedC4FMCharacterization runs each impairment and
//     logs whether the lock survives — non-fatal, the diagnostic
//     deliverable that names the impairments that break decoding.
//
// Run the full harness with:
//
//	go test -v -run Harness ./internal/radio/p25/phase1/receiver/

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
)

const (
	harnessNAC          = 0x293
	harnessControlFreq  = 420_087_500
	harnessSampleRateHz = 48_000.0
	harnessSPS          = 10 // 48 kHz / 4800 baud
	harnessSpan         = 8
	harnessAlpha        = 0.2
	harnessDeviationHz  = 1800.0
	harnessFrameRepeats = 40
	// harnessChunk is the IQ chunk size fed to Receiver.Process. ~19
	// symbols' worth — close to a real 16 KiB RTL-SDR USB transfer —
	// so the cross-chunk frame assembly added in #292 stays exercised.
	harnessChunk = 192
)

// buildHarnessDibits assembles a canonical P25 Phase 1 control-channel
// dibit stream: a 200-dibit warmup so the symbol-clock loop converges,
// then `repeats` × (FSW + NID + TSBK + 50 idle dibits), then a
// 100-dibit trailer. Mirrors buildP25LockedIQDibits in the integration
// test so the two harnesses stay comparable.
func buildHarnessDibits(nac uint16, repeats int) []uint8 {
	frame := make([]uint8, 0, 24+32+98)
	frame = append(frame, phase1.FrameSyncWord[:]...)
	nidBits := phase1.EncodeNIDBits(nac, phase1.DUIDTrunkingSignaling)
	for i := 0; i < 32; i++ {
		frame = append(frame, (nidBits[2*i]<<1)|nidBits[2*i+1])
	}
	tsbk := phase1.AssembleTSBK(phase1.TSBK{LB: true, Opcode: phase1.OpRFSSStatusBroadcast})
	frame = append(frame, phase1.EncodeTSBKChannel(tsbk)...)

	out := make([]uint8, 0, 200+repeats*(len(frame)+50)+100)
	for i := 0; i < 200; i++ {
		out = append(out, uint8(i&3))
	}
	for r := 0; r < repeats; r++ {
		out = append(out, frame...)
		for i := 0; i < 50; i++ {
			out = append(out, uint8(i&3))
		}
	}
	for i := 0; i < 100; i++ {
		out = append(out, uint8(i&3))
	}
	return out
}

// modulateHarness synthesises an IQ stream carrying the canonical dibit
// sequence for the given demod path. The CQPSK path applies
// lsmDibitRemap (an involution swapping dibits 2↔3) after the DQPSK
// quadrant decode, so the modulator must be fed the remapped dibits for
// the receiver to recover the canonical stream.
func modulateHarness(canonical []uint8, mode DemodMode) []complex64 {
	if mode == DemodCQPSK {
		modIn := make([]uint8, len(canonical))
		for i, d := range canonical {
			modIn[i] = lsmDibitRemap[d&3]
		}
		return demod.ModulatePiOver4DQPSK(modIn, harnessSPS, harnessSpan, harnessAlpha, math.Pi/4)
	}
	return demod.ModulateC4FM(canonical, harnessSPS, harnessSpan, harnessAlpha,
		harnessSampleRateHz, harnessDeviationHz)
}

// harnessResult is the outcome of one runHarness pass.
type harnessResult struct {
	locked       bool
	nac          uint16
	decodeErrors int
	nidErrs      []int64 // every "errs" value the NID decoder logged
}

// nidErrsSummary renders the captured NID error counts as min/max/count.
func (r harnessResult) nidErrsSummary() string {
	if len(r.nidErrs) == 0 {
		return "none"
	}
	lo, hi := r.nidErrs[0], r.nidErrs[0]
	for _, e := range r.nidErrs {
		if e < lo {
			lo = e
		}
		if e > hi {
			hi = e
		}
	}
	return fmt.Sprintf("%d/%d/%d", lo, hi, len(r.nidErrs))
}

// nidLogCapture is a slog.Handler that records the "errs" attribute of
// every NID-decode log line, so the harness can report how badly the
// dibits feeding the BCH decoder were corrupted.
type nidLogCapture struct {
	mu   sync.Mutex
	errs []int64
}

func (h *nidLogCapture) Enabled(context.Context, slog.Level) bool { return true }

func (h *nidLogCapture) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "errs" {
			h.errs = append(h.errs, a.Value.Int64())
		}
		return true
	})
	return nil
}

func (h *nidLogCapture) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *nidLogCapture) WithGroup(string) slog.Handler      { return h }

// runHarness modulates the canonical control-channel stream for one
// demod path, applies imp, and pumps the resulting IQ through the real
// receiver + control-channel chain in RTL-realistic small chunks. It
// reports whether the control channel locked and how the NID decoder
// fared.
func runHarness(mode DemodMode, imp demod.Impairments) harnessResult {
	canonical := buildHarnessDibits(harnessNAC, harnessFrameRepeats)
	iq := demod.ApplyImpairments(modulateHarness(canonical, mode), harnessSampleRateHz, imp)

	bus := events.NewBus(4096)
	sub := bus.Subscribe()
	defer sub.Close()

	logCap := &nidLogCapture{}
	cc := phase1.New(phase1.Options{
		Bus:         bus,
		Log:         slog.New(logCap),
		SystemName:  "Harness",
		FrequencyHz: harnessControlFreq,
	})

	r := New(Options{
		SampleRateHz: harnessSampleRateHz,
		DeviationHz:  harnessDeviationHz,
		DemodMode:    mode,
		DibitSink:    func(dibits []uint8, baseIdx int) { cc.Process(dibits, baseIdx) },
	})

	for i := 0; i < len(iq); i += harnessChunk {
		end := i + harnessChunk
		if end > len(iq) {
			end = len(iq)
		}
		r.Process(iq[i:end])
	}

	var res harnessResult
	for draining := true; draining; {
		select {
		case ev := <-sub.C:
			switch ev.Kind {
			case events.KindCCLocked:
				if ls, ok := ev.Payload.(phase1.LockState); ok {
					res.locked = true
					res.nac = ls.NAC
				}
			case events.KindDecodeError:
				res.decodeErrors++
			}
		default:
			draining = false
		}
	}
	logCap.mu.Lock()
	res.nidErrs = logCap.errs
	logCap.mu.Unlock()
	return res
}

// TestHarnessC4FMCleanLocks is the regression guard: an un-impaired
// synthetic C4FM signal, fed in RTL-realistic small chunks, must lock
// the control channel.
func TestHarnessC4FMCleanLocks(t *testing.T) {
	res := runHarness(DemodC4FM, demod.Impairments{})
	if !res.locked {
		t.Fatalf("clean C4FM signal did not lock the control channel "+
			"(decodeErrors=%d, nidErrs min/max/n=%s)", res.decodeErrors, res.nidErrsSummary())
	}
	if res.nac != harnessNAC {
		t.Errorf("clean C4FM: locked NAC = %#x, want %#x", res.nac, harnessNAC)
	}
}

// TestHarnessCQPSKChunkBoundary isolates the CQPSK-side root cause for
// issue #275. The harness exercised the CQPSK path end-to-end for the
// first time and found that the Gardner timing loop emits a different
// number of symbols depending on the IQ chunk size: a single Process
// call recovers the transmitted dibit count almost exactly, but feeding
// the same clean signal in RTL-realistic small chunks inflates the
// count by ~5% — spurious dibits that desynchronise the stream so the
// FSW correlator and control channel never lock.
//
// This is the same bug class as #292 (the frame assembler was
// chunk-size sensitive) but one stage earlier, in symbol timing
// recovery. The existing CQPSK unit tests miss it because they feed
// 4096-sample chunks, large enough to keep the drift negligible.
//
// The test asserts the demod core is sound (one-shot count is correct)
// and logs the small-chunk inflation as the reproduced defect. When the
// Gardner fix lands, the final block should be promoted to assert that
// the small-chunk count matches the one-shot count.
func TestHarnessCQPSKChunkBoundary(t *testing.T) {
	canonical := buildHarnessDibits(harnessNAC, harnessFrameRepeats)
	iq := modulateHarness(canonical, DemodCQPSK)

	dibitCount := func(chunk int) int {
		var n int
		r := New(Options{
			SampleRateHz: harnessSampleRateHz,
			DemodMode:    DemodCQPSK,
			DibitSink:    func(d []uint8, _ int) { n += len(d) },
		})
		for i := 0; i < len(iq); i += chunk {
			end := i + chunk
			if end > len(iq) {
				end = len(iq)
			}
			r.Process(iq[i:end])
		}
		return n
	}

	oneShot := dibitCount(len(iq))
	small := dibitCount(harnessChunk)
	tolerance := len(canonical) / 100 // 1%

	t.Logf("#275 CQPSK chunk-boundary: transmitted≈%d dibits  one-shot=%d  small-chunk(%d)=%d (%+d)",
		len(canonical), oneShot, harnessChunk, small, small-oneShot)

	// The demod core is correct: one Process call recovers very close
	// to the transmitted dibit count.
	if oneShot < len(canonical)-tolerance || oneShot > len(canonical)+tolerance {
		t.Errorf("one-shot CQPSK dibit count = %d, want within %d of %d — CQPSK demod core broken",
			oneShot, tolerance, len(canonical))
	}

	// The defect: small (RTL-realistic) chunks inflate the symbol
	// count. Logged rather than failed — the fix is the #275 follow-up.
	if small > oneShot+tolerance {
		t.Logf("REPRODUCED #275 (CQPSK): small chunks emit %d surplus dibits (%.1f%%) — the "+
			"Gardner timing loop miscounts symbols across IQ-chunk boundaries; the control "+
			"channel cannot lock until this is fixed",
			small-oneShot, 100*float64(small-oneShot)/float64(oneShot))
	} else {
		t.Logf("CQPSK small-chunk dibit count is within tolerance of one-shot — #275 CQPSK " +
			"chunk-boundary defect appears fixed; promote this to a hard assertion")
	}
}

// TestHarnessImpairedC4FMCharacterization reproduces issue #275 on the
// C4FM path: it runs each realistic RTL-SDR impairment through the real
// demod chain and logs whether the control channel still locks and how
// the NID decoder fared. It is intentionally non-fatal — the value is
// the logged characterisation, which names the impairment(s) that break
// decoding and so points the follow-up demod fix at concrete,
// reproducible targets.
//
// (The CQPSK path is not characterised here: its chunk-boundary defect,
// covered by TestHarnessCQPSKChunkBoundary, breaks it before any
// impairment is even applied.)
func TestHarnessImpairedC4FMCharacterization(t *testing.T) {
	cases := []struct {
		name string
		imp  demod.Impairments
	}{
		{"freq_offset_500hz", demod.Impairments{FreqOffsetHz: 500}},
		{"freq_offset_1500hz", demod.Impairments{FreqOffsetHz: 1500}},
		{"dc_spike", demod.Impairments{DCOffset: complex(0.15, 0.10)}},
		{"iq_imbalance", demod.Impairments{IQGainImbalance: 1.15, IQPhaseSkewRad: 0.12}},
		{"awgn_20db", demod.Impairments{SNRdB: 20, Seed: 1}},
		{"awgn_10db", demod.Impairments{SNRdB: 10, Seed: 1}},
		{"combined", demod.Impairments{
			FreqOffsetHz: 600, DCOffset: complex(0.08, 0.05),
			IQGainImbalance: 1.08, SNRdB: 18, Seed: 1,
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := runHarness(DemodC4FM, tc.imp)
			t.Logf("#275 harness  c4fm  %-18s  locked=%-5v  decodeErrors=%-3d  nidErrs(min/max/n)=%s",
				tc.name, res.locked, res.decodeErrors, res.nidErrsSummary())
		})
	}
}
