package receiver

import (
	"math"
	"math/cmplx"
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// Direct-mode (DMR Tier I / simplex) timing per ETSI TS 102 361-1 §4.2: a
// 60 ms TDMA frame of two 30 ms slots. A direct-mode MS transmits one 27.5 ms
// burst (132 dibits at 4800 baud) per frame and is OFF for the other 32.5 ms
// (156 dibit periods); a base station fills both slots continuously.
const (
	dmFrameDibits = 288
	dmOffDibits   = dmFrameDibits - dmr.BurstDibits
)

// dmVoiceLCHeaderBurst is one direct-mode Voice LC Header burst (a DATA burst,
// framed by the DM data sync).
func dmVoiceLCHeaderBurst(colorCode uint8, groupID, sourceID uint32) []uint8 {
	flcBytes := dmr.AssembleFLC(dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: groupID, SrcAddr: sourceID})
	var data [9]byte
	copy(data[:], flcBytes)
	cw := framing.EncodeRS12_9(data)
	for i := 0; i < 3; i++ {
		cw[9+i] ^= framing.RS129SeedVoiceLCHeader[i]
	}
	bits := make([]byte, 96)
	for i := 0; i < 96; i++ {
		bits[i] = (cw[i>>3] >> uint(7-(i&7))) & 1
	}
	payload := framing.BitsToDibits(framing.EncodeBPTC196_96(bits))
	slot := framing.BitsToDibits(dmr.AssembleSlotType(dmr.SlotType{ColorCode: colorCode, DataType: dmr.DTVoiceLCHeader}))
	burst := make([]uint8, 0, dmr.BurstDibits)
	burst = append(burst, payload[:dmr.HalfPayloadDibits]...)
	burst = append(burst, slot[:dmr.SlotTypeDibits]...)
	burst = append(burst, dmr.DMData1.Dibits[:]...)
	burst = append(burst, slot[dmr.SlotTypeDibits:]...)
	burst = append(burst, payload[dmr.HalfPayloadDibits:]...)
	return burst
}

// dmHeaderBursts is a cycle of direct-mode Voice LC Header bursts that differ
// in source ID, so the BPTC/RS parity scrambles the payload from burst to
// burst and the stream's symbol mean stays balanced the way real traffic
// (random AMBE voice payloads) is — one identical burst repeated for seconds
// carries a fixed symbol-mean bias no real transmission has, which the
// open-loop AFC would read as carrier drift (the #402 failure mode).
func dmHeaderBursts(n int) [][]uint8 {
	out := make([][]uint8, n)
	for i := range out {
		out[i] = dmVoiceLCHeaderBurst(0x1, 99, 0x123456+uint32(i)*0x1F3)
	}
	return out
}

// directModeIQ modulates `frames` bursts (cycling through bursts) on the
// direct-mode frame grid at 48 kHz / 10 sps C4FM. gapped renders the
// transmitter-OFF time as silence (plus the AWGN below) instead of modulated
// filler; offsetHz applies a constant carrier offset; noiseSigma is AWGN per
// axis against a unit-amplitude carrier (0.03 ≈ 27 dB SNR). The stream opens
// with 4 idle frames and closes with 2, like a real simplex channel around a
// PTT.
func directModeIQ(frames int, bursts [][]uint8, gapped bool, offsetHz, noiseSigma float64, seed int64) []complex64 {
	const sampleRate, sps = 48_000.0, 10
	rng := rand.New(rand.NewSource(seed))
	const leadFrames, tailFrames = 4, 2
	total := leadFrames + frames + tailFrames
	dibits := make([]uint8, 0, total*dmFrameDibits)
	for f := 0; f < total; f++ {
		if f >= leadFrames && f < leadFrames+frames {
			dibits = append(dibits, bursts[(f-leadFrames)%len(bursts)]...)
		} else {
			for i := 0; i < dmr.BurstDibits; i++ {
				dibits = append(dibits, uint8(rng.Intn(4)))
			}
		}
		for i := 0; i < dmOffDibits; i++ {
			dibits = append(dibits, uint8(rng.Intn(4)))
		}
	}
	const modSpan = 8
	iq := demod.ModulateC4FM(dibits, sps, modSpan, 0.20, sampleRate, 1944.0)
	if gapped {
		frameSamples := dmFrameDibits * sps
		burstSamples := dmr.BurstDibits * sps
		delay := modSpan * sps // RRC shaping-filter group delay
		guard := sps
		for f := 0; f < total; f++ {
			on := f >= leadFrames && f < leadFrames+frames
			base := f*frameSamples + delay
			for i := -delay; i < frameSamples-delay; i++ {
				j := base + i
				if j < 0 || j >= len(iq) {
					continue
				}
				if !(on && i >= -guard && i < burstSamples+guard) {
					iq[j] = 0
				}
			}
		}
	}
	for i := range iq {
		if offsetHz != 0 {
			iq[i] *= complex64(cmplx.Rect(1, 2*math.Pi*offsetHz*float64(i)/sampleRate))
		}
		if noiseSigma > 0 {
			iq[i] += complex(float32(rng.NormFloat64()*noiseSigma), float32(rng.NormFloat64()*noiseSigma))
		}
	}
	return iq
}

// countBurstSyncs runs iq through a calibrated receiver (gated or not) in
// RTL-realistic chunks and returns how many DM data-sync words the recovered
// dibit stream contains — in total and in its second half (the steady state,
// past any carrier acquisition) — plus the dibit stream itself.
func countBurstSyncs(iq []complex64, noGate bool) (total, steady int, dibits []uint8) {
	var got []uint8
	r := New(Options{
		SampleRateHz:  48_000,
		DeviationHz:   1944.0,
		ClockGain:     0.015, // the Tier I / II pipelines' gain
		NoCarrierGate: noGate,
		DibitSink:     func(dibits []uint8, baseIdx int) { got = append(got, dibits...) },
	})
	const chunk = 4096
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		r.Process(iq[i:end])
	}
	det := dmr.NewSyncDetector([]dmr.SyncPattern{dmr.DMData1}, 2)
	matches, _ := det.Process(nil, got, 0)
	for _, m := range matches {
		if m.Index >= len(got)/2 {
			steady++
		}
	}
	return len(matches), steady, got
}

// TestReceiverDecodesDirectModeBurstCadence is the issue #836 regression: a
// direct-mode (simplex) transmission — 27.5 ms bursts with 32.5 ms of receiver
// noise between them — must yield its burst syncs. Every earlier fixture laid
// bursts back-to-back like a base station, and the receiver's level / offset /
// timing trackers, fed the inter-burst noise, never produced a single sync on
// the real cadence (verified failing-first: NoCarrierGate reproduces it).
func TestReceiverDecodesDirectModeBurstCadence(t *testing.T) {
	const frames = 40
	bursts := dmHeaderBursts(8)
	for _, tc := range []struct {
		name     string
		offsetHz float64
		sigma    float64
	}{
		{"centred, 27 dB SNR", 0, 0.03},
		{"centred, 17 dB SNR", 0, 0.1},
		{"reporter's -1.2 kHz tuner offset, 27 dB SNR", -1200, 0.03},
		{"+2.5 kHz tuner offset, 27 dB SNR", 2500, 0.03},
	} {
		t.Run(tc.name, func(t *testing.T) {
			iq := directModeIQ(frames, bursts, true, tc.offsetHz, tc.sigma, 1)
			gated, gatedSteady, _ := countBurstSyncs(iq, false)
			ungated, _, _ := countBurstSyncs(iq, true)
			t.Logf("syncs: gated=%d (steady-state half %d/%d) ungated=%d of %d bursts", gated, gatedSteady, frames/2, ungated, frames)
			// A tuner offset costs the first transmission after a cold start its
			// coarse acquisition (two 512-present-symbol windows ≈ 8 bursts, plus
			// a burst or two of re-lock); every burst after that must sync, and
			// well over half of the transmission overall.
			if gatedSteady < frames/2-1 {
				t.Errorf("steady-state: %d of the last %d direct-mode bursts synced, want >= %d", gatedSteady, frames/2, frames/2-1)
			}
			if gated < frames*2/3 {
				t.Errorf("gated receiver found %d of %d direct-mode burst syncs, want >= %d", gated, frames, frames*2/3)
			}
		})
	}
}

// TestReceiverCarrierGateIsNoOpOnContinuousCarrier pins no-harm: on a
// continuous base-station-style stream every sample is present, so the gated
// receiver's dibit stream is the ungated one delayed by exactly the gate's
// lag (carrierGateSymbols symbols) and otherwise byte-identical — below the
// coarse acquirer's deadband, where nothing else in the chain re-syncs. At a
// large tuner offset the acquirer engages (its re-mix restarts the chain at
// a chunk that lands lag samples later on the gated path), so there the pin
// is that both decode every burst.
func TestReceiverCarrierGateIsNoOpOnContinuousCarrier(t *testing.T) {
	bursts := dmHeaderBursts(8)
	const lagDibits = int(carrierGateSymbols)
	for _, offset := range []float64{0, 300} {
		iq := directModeIQ(30, bursts, false, offset, 0.03, 2)
		_, _, gated := countBurstSyncs(iq, false)
		_, _, ungated := countBurstSyncs(iq, true)
		if len(gated) != len(ungated) {
			t.Fatalf("offset %.0f Hz: dibit count differs gated=%d ungated=%d", offset, len(gated), len(ungated))
		}
		// Skip the start-up transient: the delay line's leading zeros give
		// the timing loop a different first error than the ungated stream,
		// and with a carrier offset that first-order transient takes a few
		// hundred symbols to decay to the same steady state.
		const skip = 1200
		for i := skip; i+lagDibits < len(gated); i++ {
			if gated[i+lagDibits] != ungated[i] {
				t.Fatalf("offset %.0f Hz: dibit %d differs (gated=%d ungated=%d) — the gate must be a pure %d-dibit delay on a continuous carrier", offset, i, gated[i+lagDibits], ungated[i], lagDibits)
			}
		}
	}
	for _, offset := range []float64{-1200, 3000} {
		iq := directModeIQ(30, bursts, false, offset, 0.03, 2)
		gated, _, _ := countBurstSyncs(iq, false)
		ungated, _, _ := countBurstSyncs(iq, true)
		if gated < 27 || ungated < 27 {
			t.Errorf("offset %.0f Hz continuous carrier: gated=%d ungated=%d of 30 burst syncs, want both >= 27", offset, gated, ungated)
		}
	}
}

// TestCarrierGateSeparatesBurstFromNoise pins the detector's two populations
// on the discriminator statistics the receiver actually sees: a modulated
// burst reads present, band-limited noise and a dead (all-zero) input read
// absent.
func TestCarrierGateSeparatesBurstFromNoise(t *testing.T) {
	iq := directModeIQ(6, dmHeaderBursts(8), true, -1200, 0.03, 3)
	fm := demod.NewFM()
	disc := fm.Process(nil, iq)
	g := newCarrierGate(10)
	present := g.Process(nil, disc, iq)
	const frameSamples = dmFrameDibits * 10
	const burstSamples = dmr.BurstDibits * 10
	// Frame 5 (0-based) is the second transmitted burst: well past warm-up.
	// The flags are aligned with the DELAYED stream, i.e. lag samples after
	// the input grid.
	base := 5*frameSamples + 8*10 + g.Lag()
	on, off := 0, 0
	for i := 0; i < burstSamples; i++ {
		if present[base+i] {
			on++
		}
	}
	gapStart := base + burstSamples + 4*10
	for i := 0; i < dmOffDibits*10-8*10; i++ {
		if present[gapStart+i] {
			off++
		}
	}
	if frac := float64(on) / burstSamples; frac < 0.9 {
		t.Errorf("burst flagged present only %.0f%% of the time, want >= 90%%", 100*frac)
	}
	if frac := float64(off) / float64(dmOffDibits*10-80); frac > 0.1 {
		t.Errorf("gap noise flagged present %.0f%% of the time, want <= 10%%", 100*frac)
	}
}
