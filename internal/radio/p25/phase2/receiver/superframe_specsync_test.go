package receiver_test

// Regression test for issue #813: P25 Phase 2 superframe sync never locked on
// real off-air traffic because OutboundSyncHex was a garbled 48-bit constant.
//
// The stimulus here is deliberately INDEPENDENT of GopherTrunk's own Phase 2
// modulator (demod.ModulatePiOver4DQPSK / EncodeSuperframe), which bakes in the
// same constants the decoder uses and so can never expose a wrong sync word.
// Instead we synthesize a spec-conformant P25 Phase 2 outbound sync directly
// from the authoritative value (0x575D57F7FF) and the standard π/4-DQPSK-family
// differential phase map (SDRtrunk Dibit.java: 00→+π/4, 01→+3π/4, 10→−π/4,
// 11→−3π/4), RRC pulse-shape it, and push it through the production receiver +
// SuperframeDecoder. It fails (0 locks) with the old constant and passes with
// the corrected one.

import (
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2/receiver"
)

// authoritative P25 Phase 2 outbound frame sync (OP25/SDRtrunk, TIA-102.BBAC).
const specSyncHex uint64 = 0x575D57F7FF

// standard π/4-DQPSK-family differential phase per (standard) dibit value.
func specSyncPhase(d uint8) float64 {
	switch d & 3 {
	case 0:
		return math.Pi / 4
	case 1:
		return 3 * math.Pi / 4
	case 2:
		return -math.Pi / 4
	default:
		return -3 * math.Pi / 4
	}
}

func hexToDibitsMSB(hex uint64, n int) []uint8 {
	out := make([]uint8, n)
	for i := 0; i < n; i++ {
		out[i] = uint8((hex >> uint(2*(n-1-i))) & 3)
	}
	return out
}

// specModulate differentially modulates dibits with the standard phase map and
// RRC-pulse-shapes to sps samples/symbol — an independent transmit model.
func specModulate(dibits []uint8, sps, span int, alpha float64) []complex64 {
	up := make([]complex64, len(dibits)*sps)
	theta := 0.0
	for i, d := range dibits {
		theta += specSyncPhase(d)
		up[i*sps] = complex(float32(math.Cos(theta)), float32(math.Sin(theta)))
	}
	return filter.NewFIR(filter.RootRaisedCosine(sps, span, alpha)).Process(nil, up)
}

func TestSuperframeLocksOnSpecSync(t *testing.T) {
	const (
		sps   = 8
		span  = 8
		alpha = 0.20
	)
	// Superframe grid: spec sync at the head of sub-frame 0, idle elsewhere.
	grid := make([]uint8, phase2.DibitsPerSuperframe)
	copy(grid[:phase2.SyncDibits], hexToDibitsMSB(specSyncHex, phase2.SyncDibits))
	var dibits []uint8
	for i := 0; i < 6; i++ {
		dibits = append(dibits, grid...)
	}
	iq := specModulate(dibits, sps, span, alpha)

	var got []uint8
	r := rx.New(rx.Options{
		SampleRateHz: sps * rx.SymbolRate,
		ClockMode:    rx.ClockGardner,
		DibitSink:    func(d []uint8, _ int) { got = append(got, d...) },
	})
	for off := 0; off < len(iq); off += 192 { // production-sized chunks
		end := off + 192
		if end > len(iq) {
			end = len(iq)
		}
		r.Process(iq[off:end])
	}

	locks := phase2.NewSuperframeDecoder().Process(got, 0)
	if len(locks) < 2 {
		t.Fatalf("spec-conformant P25 Phase 2 sync did not lock: got %d superframes, want >= 2 "+
			"(OutboundSyncHex=0x%X does not match real-air sync)", len(locks), phase2.OutboundSyncHex)
	}
}

// TestOutboundSyncIs40Bit guards the width invariant hexToDibits depends on: a
// value wider than SyncDibits*2 bits silently drops its top bits (the original
// issue #813 defect, a 48-bit constant used as 20 dibits).
func TestOutboundSyncIs40Bit(t *testing.T) {
	if phase2.OutboundSyncHex>>(2*phase2.SyncDibits) != 0 {
		t.Fatalf("OutboundSyncHex=0x%X exceeds %d bits; top bits are silently dropped by hexToDibits",
			phase2.OutboundSyncHex, 2*phase2.SyncDibits)
	}
}
