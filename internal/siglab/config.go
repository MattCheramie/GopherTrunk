package siglab

import (
	"log/slog"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// Config drives a single offline run of the engine over one capture (or a
// synthesized stream). The zero value is not runnable — Protocol, a sample
// source, and SampleRateHz must be set.
type Config struct {
	// Protocol selects the production pipeline to drive. Every protocol the
	// ccdecoder factory map registers is supported.
	Protocol trunking.Protocol
	// System carries per-protocol knobs (demod mode, colour code, band
	// plan, FEC modes, …) exactly as the daemon reads them off a configured
	// trunking.System. SystemName/FrequencyHz are surfaced separately
	// because every factory consumes them.
	System      trunking.System
	SystemName  string
	FrequencyHz uint32

	// SampleRateHz is the capture's IQ sample rate (input rate, before any
	// down-conversion). Required.
	SampleRateHz float64
	// Format is the on-disk sample encoding.
	Format SampleFormat

	// TuneHz frequency-shifts the capture so a channel at +TuneHz lands at
	// 0 Hz before demod. AutoTune estimates it from a prefix of the file
	// (and overrides TuneHz when set).
	TuneHz   float64
	AutoTune bool

	// Conjugate negates Q before channelization (spectrum-inverted /
	// I-Q-swapped front-end, issue #264). IQCorrect applies blind
	// I/Q-imbalance correction to the raw IQ before decimation (issue #402).
	Conjugate bool
	IQCorrect bool

	// CollectIQDiag enables the protocol-agnostic signal-quality analyzer
	// (symbol histogram, IQ imbalance, soft-sample distribution where
	// available). Off by default since it buffers per-symbol observations.
	CollectIQDiag bool

	// --- P25 Phase 1 deep-diagnostic knobs ---
	// These drive the engine's P25 "deep" construction path (a directly-built
	// receiver + control channel that exposes soft samples, receiver state,
	// and the experimental bisect knobs from issues #275/#402). They are
	// ignored for every other protocol. The deep path engages when the
	// protocol is P25 Phase 1 and any of CollectIQDiag / CollectReceiverState
	// / a non-default knob below is set.

	// DemodMode is the P25 Phase 1 symbol-recovery path: "c4fm" (default) or
	// "cqpsk". Empty ⇒ c4fm.
	DemodMode string
	// NIDSearchSpan overrides the NID-alignment search radius in dibits
	// (issue #275 bisect knob). 0 ⇒ the production default.
	NIDSearchSpan int
	// EnableDDA opts the C4FM path into the experimental decision-directed
	// AFC (issue #402; off in production).
	EnableDDA bool
	// EnableAdaptiveSlicer opts the C4FM path into the adaptive slicer
	// (issue #402; off in production).
	EnableAdaptiveSlicer bool
	// CollectReceiverState captures a per-(stream-)second snapshot of the P25
	// receiver's AFC/AGC/clock/slicer state into the Result (the per-second
	// state log replay emitted to stderr, now structured + exportable).
	CollectReceiverState bool

	// ChunkSamples is the read-loop chunk size in IQ samples; 0 ⇒ default.
	ChunkSamples int

	// Acceptance, when non-nil, is evaluated against the Result to produce a
	// pass/fail Verdict (the `test` harness sets it).
	Acceptance *Acceptance

	// Log receives the production pipeline's structured diagnostics. nil ⇒
	// a discard logger (the engine never wants a nil *slog.Logger reaching a
	// factory that does not nil-guard it).
	Log *slog.Logger
}

const defaultChunkSamples = 8192

func (c Config) logger() *slog.Logger {
	if c.Log != nil {
		return c.Log
	}
	return slog.New(slog.DiscardHandler)
}

func (c Config) chunkSamples() int {
	if c.ChunkSamples > 0 {
		return c.ChunkSamples
	}
	return defaultChunkSamples
}

// wantP25Deep reports whether the engine should drive the P25 Phase 1 deep
// path (direct receiver + control-channel build with soft/state capture and
// the experimental bisect knobs) rather than the generic factory path. It
// engages for P25 Phase 1 whenever deep analysis or any non-default deep knob
// is requested.
func (c Config) wantP25Deep() bool {
	if c.Protocol != trunking.ProtocolP25 {
		return false
	}
	return c.CollectIQDiag || c.CollectReceiverState ||
		c.NIDSearchSpan > 0 || c.EnableDDA || c.EnableAdaptiveSlicer ||
		c.DemodMode != ""
}

// ParseProtocolCLI maps a command-line protocol name to a trunking.Protocol.
// It first honours the legacy replay aliases ("p25p1" → P25 Phase 1,
// "dmr-tier3" → DMR Tier III) so existing replay invocations keep working,
// then falls back to the canonical trunking.ParseProtocol (which covers all
// config spellings: p25, p25-phase2, dmr, dmr-tier2, nxdn, dpmr, edacs,
// motorola, ltr, mpt1327, tetra, ysf, dstar).
func ParseProtocolCLI(s string) (trunking.Protocol, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "p25p1", "p25-phase1", "p25_phase1":
		return trunking.ProtocolP25, nil
	case "dmr-tier3", "dmr_tier3", "dmr-t3", "dmrtier3", "tier3":
		return trunking.ProtocolDMR, nil
	}
	return trunking.ParseProtocol(s)
}

// expectedSymbolRate returns the nominal recovered-symbol rate (dibits/s for
// the 4-level protocols, bits/s for the 2-level ones) the receiver emits for
// a protocol, used for the effective-baud self-diagnostic that flags a
// mislabeled capture (a capture whose true sample rate differs from the one
// the operator passed reports a symbol rate far from this value). Returns 0
// when no nominal rate is pinned, in which case the deviation check is
// skipped rather than reported against a guess.
func expectedSymbolRate(p trunking.Protocol) float64 {
	switch p {
	case trunking.ProtocolP25, trunking.ProtocolDMR, trunking.ProtocolDMRTier2,
		trunking.ProtocolNXDN, trunking.ProtocolYSF:
		return 4800
	case trunking.ProtocolP25Phase2:
		return 6000
	case trunking.ProtocolDPMR:
		return 2400
	case trunking.ProtocolTETRA:
		return 18000
	case trunking.ProtocolEDACS:
		return 9600
	case trunking.ProtocolMotorola:
		return 3600
	case trunking.ProtocolMPT1327:
		return 1200
	case trunking.ProtocolDStar:
		return 4800
	case trunking.ProtocolLTR:
		return 300
	default:
		return 0
	}
}
