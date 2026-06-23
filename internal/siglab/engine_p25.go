package siglab

import (
	"fmt"
	"log/slog"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	p25phase1 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	p25phase1rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1/receiver"
)

// p25DeviationHz is the P25 Phase 1 nominal peak deviation per TIA-102.BAAA-A
// — the same value the production ccdecoder pipeline calibrates the C4FM
// slicer against.
const p25DeviationHz = 1800.0

// buildP25DeepBundle constructs the P25 Phase 1 deep path: a directly-built
// control channel + receiver that exposes the per-symbol soft samples
// (SoftSink → the analyzer's eye), the receiver-state accessors, the
// per-frame CCStats, and the experimental bisect knobs (NID search span,
// decision-directed AFC, adaptive slicer) from issues #275/#402. It mirrors
// the production newP25Phase1Pipeline construction so a deep replay lock still
// implies an on-air lock, while surfacing the diagnostics the factory path
// cannot.
func buildP25DeepBundle(cfg Config, bus *events.Bus, logger *slog.Logger, receiverRate float64, symbolTap func([]uint8, bool, int), softTap func([]float32), constTap func([]complex64)) (*runBundle, error) {
	demodMode := p25phase1rx.DemodC4FM
	if cfg.DemodMode != "" {
		m, ok := p25phase1rx.ParseDemodMode(cfg.DemodMode)
		if !ok {
			return nil, fmt.Errorf("siglab: unknown demod mode %q (want c4fm or cqpsk)", cfg.DemodMode)
		}
		demodMode = m
	}
	// On a C4FM FM-discriminator stream only rotations 0 and 2 are physical
	// (issue #275); the CQPSK path keeps the all-rotation default.
	rotations := p25phase1.RotationsAll
	if demodMode == p25phase1rx.DemodC4FM {
		rotations = p25phase1.RotationsC4FM
	}
	nidSpan := cfg.NIDSearchSpan
	if nidSpan <= 0 {
		nidSpan = p25phase1.NIDSearchSpan
	}

	fswFallbackTol := 0
	if cfg.EnableSoftSync {
		fswFallbackTol = p25phase1.FSWFallbackToleranceDefault
	}
	cc := p25phase1.New(p25phase1.Options{
		Bus:                  bus,
		Log:                  logger,
		SystemName:           cfg.SystemName,
		FrequencyHz:          cfg.FrequencyHz,
		Rotations:            rotations,
		NIDSearchSpan:        nidSpan,
		FSWFallbackTolerance: fswFallbackTol,
	})
	rx := p25phase1rx.New(p25phase1rx.Options{
		SampleRateHz:              receiverRate,
		DeviationHz:               p25DeviationHz,
		DemodMode:                 demodMode,
		EnableDecisionDirectedAFC: cfg.EnableDDA,
		EnableAdaptiveC4FMSlicer:  cfg.EnableAdaptiveSlicer,
		SoftSink:                  softTap,
		// The CQPSK path fires SymbolSink with the post-carrier-recovery
		// complex constellation points; the C4FM path leaves it uncalled
		// (its quality view is the real soft eye via SoftSink). Feeding it to
		// the analyzer lets the demod-quality metrics use the constellation
		// EVM/SNR estimators on the linear path.
		SymbolSink: constTap,
		DibitSink: func(dibits []uint8, baseIdx int) {
			symbolTap(dibits, false, baseIdx)
			cc.Process(dibits, baseIdx)
		},
	})

	isCQPSK := demodMode == p25phase1rx.DemodCQPSK
	return &runBundle{
		proc: rx,
		stateAt: func(t float64) ReceiverState {
			return snapshotP25Receiver(rx, t, isCQPSK)
		},
		ccStats: func() *CCStatsBreakdown {
			s := cc.Stats()
			return &CCStatsBreakdown{
				NIDTrusted:        s.NIDTrusted,
				NIDMarginal:       s.NIDMarginal,
				NIDFailed:         s.NIDFailed,
				TSBKDecoded:       s.TSBKDecoded,
				TSBKTrellisFailed: s.TSBKTrellisFailed,
				TSBKCRCFailed:     s.TSBKCRCFailed,
				NetStatusSeen:     s.NetStatusSeen,
				RFSSStatusSeen:    s.RFSSStatusSeen,
				AdjacentSeen:      s.AdjacentSeen,
			}
		},
		topology: func() *TopologySnapshot { return cc.TopologySnapshot() },
	}, nil
}

// snapshotP25Receiver reads the P25 receiver's internal-loop state at stream
// time t. The C4FM and CQPSK paths expose disjoint accessor sets (the other
// path's accessors return zero), so we populate only the relevant fields.
func snapshotP25Receiver(rx *p25phase1rx.Receiver, t float64, isCQPSK bool) ReceiverState {
	if isCQPSK {
		return ReceiverState{
			TimeSec:      t,
			CQPSK:        true,
			CarrierHzEst: rx.CQPSKCarrierOffsetHz(),
			GardnerMu:    rx.GardnerMu(),
			GardnerSPS:   rx.GardnerSPS(),
			CQPSKAGCGain: rx.CQPSKAGCGain(),
			CMAError:     rx.CMAError(),
		}
	}
	return ReceiverState{
		TimeSec:          t,
		AFCHzEst:         rx.AFCOffsetHz(),
		AGCLevel:         rx.AGCLevel(),
		AGCTarget:        rx.AGCTarget(),
		MMMu:             rx.MMClockMu(),
		MMSPS:            rx.MMClockSPS(),
		DDAActive:        rx.DDAActive(),
		SlicerLevels:     rx.SlicerLevels(),
		SlicerThresholds: rx.SlicerThresholds(),
	}
}
