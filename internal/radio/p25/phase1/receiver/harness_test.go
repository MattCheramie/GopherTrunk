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
//   - TestHarnessCleanControlChannelLocks hard-asserts the clean C4FM
//     and CQPSK signals lock — a permanent regression guard.
//   - TestHarnessCQPSKChunkBoundary guards the Gardner timing-loop fix:
//     the recovered symbol count must not depend on the IQ chunk size.
//   - TestHarnessImpairedControlChannelCharacterization runs each
//     impairment and logs whether the lock survives — non-fatal, the
//     diagnostic deliverable that names the impairments that still
//     break decoding.
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

// demodModes is the set of demod paths the harness exercises.
var demodModes = []struct {
	name string
	mode DemodMode
}{
	{"c4fm", DemodC4FM},
	{"cqpsk", DemodCQPSK},
}

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

// TestHarnessCleanControlChannelLocks is the regression guard: an
// un-impaired synthetic signal, fed in RTL-realistic small chunks, must
// lock the control channel on both demod paths. The CQPSK path locking
// here is the proof point for the Gardner chunk-boundary fix — before
// it, the CQPSK demod emitted surplus dibits on small chunks and never
// locked.
func TestHarnessCleanControlChannelLocks(t *testing.T) {
	for _, m := range demodModes {
		t.Run(m.name, func(t *testing.T) {
			res := runHarness(m.mode, demod.Impairments{})
			if !res.locked {
				t.Fatalf("clean %s signal did not lock the control channel "+
					"(decodeErrors=%d, nidErrs min/max/n=%s)",
					m.name, res.decodeErrors, res.nidErrsSummary())
			}
			if res.nac != harnessNAC {
				t.Errorf("clean %s: locked NAC = %#x, want %#x", m.name, res.nac, harnessNAC)
			}
		})
	}
}

// TestHarnessCQPSKChunkBoundary guards the Gardner timing-loop fix for
// issue #275. The harness exercised the CQPSK path end-to-end for the
// first time and found the Gardner loop's recovered symbol count was
// chunk-size dependent: a single Process call recovered the transmitted
// dibit count, but the same clean signal fed in RTL-realistic small
// chunks inflated it by ~5% — surplus dibits that desynchronised the
// stream so the FSW correlator and control channel never locked. (Same
// bug class as #292, one stage earlier, in symbol timing recovery; the
// existing CQPSK unit tests missed it because they feed 4096-sample
// chunks where the drift is negligible.)
//
// The fix makes the Gardner stash pure look-back context. This test
// asserts the recovered dibit count is now independent of chunk size.
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

	if oneShot < len(canonical)-tolerance || oneShot > len(canonical)+tolerance {
		t.Errorf("one-shot CQPSK dibit count = %d, want within %d of %d", oneShot, tolerance, len(canonical))
	}
	if small < oneShot-tolerance || small > oneShot+tolerance {
		t.Errorf("small-chunk CQPSK dibit count = %d, want within %d of one-shot %d — the "+
			"Gardner timing loop is miscounting symbols across IQ-chunk boundaries (#275)",
			small, tolerance, oneShot)
	}
}

// TestHarnessC4FMToleratesCarrierOffset guards the coarse-AFC fix for
// issue #275. A residual RTL-SDR tuner offset appears as a DC bias on
// the FM-discriminator output; left in place it shifts the 4-level eye
// off the slicer's fixed thresholds, and at ≥500 Hz the C4FM Frame Sync
// Word stopped correlating entirely so the control channel never
// locked. The coarse-AFC stage tracks and subtracts that bias, so the
// channel locks across a realistic offset range.
func TestHarnessC4FMToleratesCarrierOffset(t *testing.T) {
	for _, offHz := range []float64{-1500, -800, -500, 500, 800, 1000, 1500} {
		t.Run(fmt.Sprintf("%+gHz", offHz), func(t *testing.T) {
			res := runHarness(DemodC4FM, demod.Impairments{FreqOffsetHz: offHz})
			if !res.locked {
				t.Errorf("C4FM did not lock at %+g Hz carrier offset "+
					"(decodeErrors=%d, nidErrs min/max/n=%s)",
					offHz, res.decodeErrors, res.nidErrsSummary())
			}
		})
	}
}

// TestHarnessImpairedControlChannelCharacterization runs each realistic
// RTL-SDR impairment through the real demod chain and logs whether the
// control channel still locks and how the NID decoder fared. It is
// intentionally non-fatal — the value is the logged characterisation,
// which names the impairment(s) that still break decoding and so points
// any further demod work at concrete, reproducible targets.
func TestHarnessImpairedControlChannelCharacterization(t *testing.T) {
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
		{"simulcast", demod.Impairments{
			Multipath: []complex64{1, 0, 0, 0, 0, 0, 0, 0, complex(0.7, 0.35)},
		}},
	}
	for _, m := range demodModes {
		for _, tc := range cases {
			t.Run(m.name+"/"+tc.name, func(t *testing.T) {
				res := runHarness(m.mode, tc.imp)
				t.Logf("#275 harness  %-5s %-18s  locked=%-5v  decodeErrors=%-3d  nidErrs(min/max/n)=%s",
					m.name, tc.name, res.locked, res.decodeErrors, res.nidErrsSummary())
			})
		}
	}
}

// TestHarnessCQPSKEqualizerRecoversSimulcast guards the CMA-equalizer
// fix for issue #275. P25 simulcast covers a channel with several
// synchronised transmitters; their differently-delayed copies sum to a
// multipath channel that closes the CQPSK constellation, so the Frame
// Sync Word stops correlating and the control channel never locks.
// Because LSM is a linear modulation the distortion is linear in the
// complex symbols, so the blind CMA equalizer reopens the constellation
// — the channel locks across a realistic simulcast-echo range.
func TestHarnessCQPSKEqualizerRecoversSimulcast(t *testing.T) {
	profiles := []struct {
		name string
		taps []complex64
	}{
		{"echo_0.56_at_0.5sym", []complex64{1, 0, 0, 0, 0, complex(0.5, 0.25)}},
		{"echo_0.78_at_0.8sym", []complex64{1, 0, 0, 0, 0, 0, 0, 0, complex(0.7, 0.35)}},
		{"echo_0.98_at_0.5sym", []complex64{1, 0, 0, 0, 0, complex(0.9, 0.4)}},
	}
	for _, p := range profiles {
		t.Run(p.name, func(t *testing.T) {
			res := runHarness(DemodCQPSK, demod.Impairments{Multipath: p.taps})
			if !res.locked {
				t.Errorf("CQPSK did not lock under simulcast multipath %s "+
					"(decodeErrors=%d, nidErrs min/max/n=%s)",
					p.name, res.decodeErrors, res.nidErrsSummary())
			}
		})
	}
}
