package receiver

import (
	"io"
	"log/slog"
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
)

// omegaFromHz converts a carrier offset in Hz to rad/symbol at the TETRA
// symbol rate, matching carrierAFC.OffsetHz's inverse.
func omegaFromHz(hz float64) float64 { return hz * 2 * math.Pi / SymbolRate }

// TestAFCTrackClusteredRejectsReprime pins the escape hatch at the unit
// level: an EMA latched in the wrong ±f_sym/4 alias bucket (the 19 Aug field
// state: −3630 Hz reported while the true carrier sat near 0) re-primes after
// omegaReprimeStreak consecutive rejected estimates that agree with each
// other, instead of rejecting the correct estimate forever.
func TestAFCTrackClusteredRejectsReprime(t *testing.T) {
	a := newCarrierAFC(8)
	a.omegaEMA = omegaFromHz(-3630)
	a.omegaPrimed = true

	// True carrier ~0 Hz: every raw estimate lands near 0, > omegaAliasReject
	// away from the latched EMA. Small jitter keeps the cluster realistic.
	raws := []float64{omegaFromHz(40), omegaFromHz(-60), omegaFromHz(25)}
	for i, r := range raws {
		a.track(r)
		if i < len(raws)-1 && math.Abs(a.omegaEMA-omegaFromHz(-3630)) > 1e-9 {
			t.Fatalf("EMA moved before the streak completed (after %d rejects): %f", i+1, a.omegaEMA)
		}
	}
	if gotHz := a.OffsetHz(SymbolRate); math.Abs(gotHz) > 100 {
		t.Fatalf("EMA did not re-prime from the reject cluster: reporting %.0f Hz, want ~0", gotHz)
	}
}

// TestAFCTrackSporadicOutlierHolds pins the anti-flap side: sporadic or
// scattered alias-sized outliers — the ISI condition the EMA reject exists
// for — must NOT re-prime the mean.
func TestAFCTrackSporadicOutlierHolds(t *testing.T) {
	alias := omegaFromHz(4500)

	// Sporadic: outliers interleaved with accepted estimates. Each accepted
	// block zeroes the streak, so the mean must hold at ~0.
	a := newCarrierAFC(8)
	a.track(0) // prime at 0
	for _, r := range []float64{0, alias, 0, -alias, 0, alias, 0} {
		a.track(r)
	}
	if got := a.OffsetHz(SymbolRate); math.Abs(got) > 50 {
		t.Fatalf("sporadic outliers moved the EMA: %.0f Hz, want ~0", got)
	}

	// Scattered: consecutive rejects that do NOT agree with each other
	// (alternating alias buckets) restart the cluster each time and must
	// never reach omegaReprimeStreak.
	b := newCarrierAFC(8)
	b.track(0)
	for i := 0; i < 6; i++ {
		if i%2 == 0 {
			b.track(alias)
		} else {
			b.track(-alias)
		}
	}
	if got := b.OffsetHz(SymbolRate); math.Abs(got) > 50 {
		t.Fatalf("scattered outliers re-primed the EMA: %.0f Hz, want ~0", got)
	}
}

// TestAFCEscapesWrongAliasSeed drives the full block pipeline: a clean,
// perfectly-centred π/4-DQPSK stream through a carrierAFC whose EMA has been
// poisoned into the wrong alias bucket (what a one-block weak-signal seed
// after Reset() produced in the field). Before the escape hatch the latch is
// permanent — every correct per-block estimate differs by ~3630 Hz >
// omegaAliasReject and is discarded; this test fails on that code.
func TestAFCEscapesWrongAliasSeed(t *testing.T) {
	const (
		sps   = 8
		span  = 8
		alpha = 0.35
	)
	dibits := make([]uint8, 0, 12000)
	lcg := uint32(424242)
	for i := 0; i < 12000; i++ {
		lcg = lcg*1664525 + 1013904223
		dibits = append(dibits, uint8((lcg>>16)&3))
	}
	iq := demod.ModulatePiOver4DQPSK(dibits, sps, span, alpha, math.Pi/4)

	a := newCarrierAFC(sps)
	a.omegaEMA = omegaFromHz(-3630)
	a.omegaPrimed = true

	dst := make([]complex64, 0, len(iq))
	const chunk = 4096
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		// The pre-matched (wide) stream is the same clean block here — at zero
		// true CFO the coarse stage reads ~0 from it, exactly the estimates the
		// wrong latch keeps rejecting.
		dst = a.Process(dst, iq[i:end], iq[i:end])
	}

	if got := a.OffsetHz(SymbolRate); math.Abs(got) > 300 {
		t.Fatalf("AFC still latched at %.0f Hz on a centred stream — the wrong alias seed was never escaped", got)
	}
}

// TestReceiverRecoversFromAliasLatchedAFC is the receiver-level regression for
// the 19 Aug field failure: the production 144 kHz Gardner+AFC+channel-filter
// path, its AFC poisoned into the wrong alias bucket mid-stream (as a
// decode-drought resync's one-block re-prime did on air), must recover and
// lock on a clean centred control channel instead of staying "locked but
// deaf" forever.
func TestReceiverRecoversFromAliasLatchedAFC(t *testing.T) {
	const (
		controlFreqHz        = 412_062_500
		colourCode           = uint32(0x12345)
		locationArea  uint16 = 0x42
		sps                  = 8
		span                 = 8
		alpha                = 0.35
		gardnerGain          = 0.005
		sampleRateHz         = 144_000.0
	)

	dibits := buildTETRASCHHDDibits(120, colourCode, locationArea)
	iq := demod.ModulatePiOver4DQPSK(dibits, sps, span, alpha, math.Pi/4)

	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := tetra.New(tetra.Options{
		Bus:         bus,
		Log:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		SystemName:  "TETRASite",
		FrequencyHz: controlFreqHz,
	})
	cc.SetChannelCoding(tetra.ChannelCodingOn)
	cc.SetExpectedChannel(tetra.ChannelSCHHD)
	cc.SetColourCode(colourCode)

	rx := New(Options{
		SampleRateHz:        sampleRateHz,
		ClockMode:           ClockGardner,
		GardnerGain:         gardnerGain,
		EnableAFC:           true,
		EnableChannelFilter: true,
		DibitSink:           func(d []uint8, b int) { cc.Process(d, b) },
		SoftSink:            func(diffs []complex64, b int) { cc.StashSoft(diffs, b) },
	})

	// Poison the AFC the way the field failure did: latched in the wrong
	// ±4500 Hz alias bucket while the true carrier is centred.
	rx.afc.omegaEMA = omegaFromHz(-3630)
	rx.afc.omegaPrimed = true

	const chunk = 4096
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		rx.Process(iq[i:end])
	}

	locked := false
drain:
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				locked = true
			}
		default:
			break drain
		}
	}
	if !locked {
		t.Fatalf("no cc.locked on a centred stream with an alias-latched AFC "+
			"(afc reporting %.0f Hz): the escape hatch did not fire", rx.afc.OffsetHz(SymbolRate))
	}
}
