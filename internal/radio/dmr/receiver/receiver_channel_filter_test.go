package receiver

import (
	"math"
	"math/cmplx"
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
)

// neighbourC4FM is a second, unrelated DMR-like C4FM carrier (random dibits)
// of the same amplitude as the wanted signal, shifted to offsetHz — the
// co-passband emitter the wideband tap's ±~22 kHz DDC output admits.
func neighbourC4FM(n int, offsetHz float64, seed int64) []complex64 {
	const sampleRate, sps = 48_000.0, 10
	rng := rand.New(rand.NewSource(seed))
	dibits := make([]uint8, n/sps+64)
	for i := range dibits {
		dibits[i] = uint8(rng.Intn(4))
	}
	iq := demod.ModulateC4FM(dibits, sps, 8, 0.20, sampleRate, 1944.0)
	if len(iq) > n {
		iq = iq[:n]
	}
	for i := range iq {
		iq[i] *= complex64(cmplx.Rect(1, 2*math.Pi*offsetHz*float64(i)/sampleRate))
	}
	return iq
}

// countSyncsFiltered is countBurstSyncs with the channel-select filter
// switchable.
func countSyncsFiltered(iq []complex64, chanFilter bool) (total, steady int) {
	var got []uint8
	r := New(Options{
		SampleRateHz:        48_000,
		DeviationHz:         1944.0,
		ClockGain:           0.015,
		EnableChannelFilter: chanFilter,
		DibitSink:           func(dibits []uint8, baseIdx int) { got = append(got, dibits...) },
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
	return len(matches), steady
}

// TestReceiverChannelFilterRejectsCoPassbandNeighbour is the 15/16 Sep IPSC
// "deaf tap" regression. The wideband channelizer / DDC hands the receiver a
// 48 kHz stream whose ±~22 kHz passband is two DMR channels wide either side,
// and the FM discriminator cannot separate co-passband carriers of comparable
// power: on the operator's 442.3875 MHz tap an emitter at −20.1 kHz sat at the
// tap's own level through every idle gap, and the tap decoded nothing while it
// was up (`sync_hits=0` for minutes at the channel's normal −49..−51 dBFS).
//
// Failing-first: a continuous wanted C4FM stream plus an equal-power C4FM
// neighbour at −20 kHz yields (almost) no burst syncs through the receiver
// without the channel filter, and the full count with it.
func TestReceiverChannelFilterRejectsCoPassbandNeighbour(t *testing.T) {
	const frames = 40
	bursts := dmHeaderBursts(8)
	clean := directModeIQ(frames, bursts, false, 0, 0.03, 1)
	nb := neighbourC4FM(len(clean), -20_000, 7)
	mixed := make([]complex64, len(clean))
	for i := range clean {
		mixed[i] = clean[i] + nb[i]
	}

	cleanTotal, _ := countSyncsFiltered(clean, false)
	if cleanTotal < frames*8/10 {
		t.Fatalf("clean fixture: %d syncs of %d, fixture broken", cleanTotal, frames)
	}
	oldTotal, _ := countSyncsFiltered(mixed, false)
	newTotal, _ := countSyncsFiltered(mixed, true)
	t.Logf("syncs: clean=%d neighbour(unfiltered)=%d neighbour(filtered)=%d of %d", cleanTotal, oldTotal, newTotal, frames)
	if oldTotal > cleanTotal/2 {
		t.Fatalf("unfiltered receiver decoded %d/%d syncs with an equal-power −20 kHz neighbour; the fixture does not reproduce the deaf tap", oldTotal, cleanTotal)
	}
	if newTotal < cleanTotal*9/10 {
		t.Fatalf("channel-filtered receiver decoded %d syncs with the neighbour, want ≥ 90%% of the clean %d", newTotal, cleanTotal)
	}
}

// TestReceiverChannelFilterNoHarmOnCleanSignal: the filter passes the wanted
// signal — a clean stream (with and without a small tuner offset) decodes the
// same burst syncs with the filter as without.
func TestReceiverChannelFilterNoHarmOnCleanSignal(t *testing.T) {
	const frames = 40
	bursts := dmHeaderBursts(8)
	for _, offset := range []float64{0, 400, -1200} {
		iq := directModeIQ(frames, bursts, false, offset, 0.03, 3)
		off, offSteady := countSyncsFiltered(iq, false)
		on, onSteady := countSyncsFiltered(iq, true)
		t.Logf("offset %+.0f Hz: syncs unfiltered=%d (steady %d) filtered=%d (steady %d)", offset, off, offSteady, on, onSteady)
		if onSteady < offSteady-1 {
			t.Errorf("offset %+.0f Hz: channel filter lost steady-state syncs: %d → %d", offset, offSteady, onSteady)
		}
	}
}

// TestChannelFilterKeepsCarrierGateOnDirectMode: the channel filter narrows
// the receiver noise, and the carrier gate's thresholds must follow it —
// with the wideband constants a filtered inter-burst gap read as a quiet
// carrier (variance ~0.5 rad² < the 1.0 open threshold), the gate never
// closed, and the direct-mode fixture that pins #836 decoded ZERO syncs
// through the filtered receiver (verified failing-first). The filtered
// receiver must decode the gapped direct-mode fixture like the unfiltered one.
func TestChannelFilterKeepsCarrierGateOnDirectMode(t *testing.T) {
	const frames = 40
	bursts := dmHeaderBursts(8)
	for _, offset := range []float64{0, -1200} {
		iq := directModeIQ(frames, bursts, true, offset, 0.03, 1)
		off, offSteady := countSyncsFiltered(iq, false)
		on, onSteady := countSyncsFiltered(iq, true)
		t.Logf("gapped, offset %+.0f Hz: syncs unfiltered=%d (steady %d) filtered=%d (steady %d)", offset, off, offSteady, on, onSteady)
		if onSteady < offSteady-1 {
			t.Errorf("offset %+.0f Hz: filtered receiver lost direct-mode syncs: %d → %d (carrier gate not calibrated to the filtered noise)", offset, offSteady, onSteady)
		}
	}
	v := filteredNoiseDiscVariance(filter.LowpassKaiser(181, ChannelCutoffHz/48_000, channelFilterBeta))
	t.Logf("filtered noise discriminator variance at 48 kHz: %.3f rad² (wideband %.3f)", v, carrierGateNoiseVarianceWideband)
	if v < 0.2 || v > 1.0 {
		t.Errorf("filtered noise variance %.3f outside the measured 0.2..1.0 band", v)
	}
}
