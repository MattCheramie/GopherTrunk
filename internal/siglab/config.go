package siglab

import (
	"log/slog"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/events"
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
	// EnableDCBlock inserts the pre-discriminator complex DC-removal high-pass
	// on the P25 C4FM path — strips the static zero-IF DC spur (e.g. the HackRF
	// centre spike) that otherwise collapses voice decode. Off by default; the
	// production voice composer enables it unconditionally, this exposes it for
	// diagnosing/decoding on-channel HackRF captures via replay.
	EnableDCBlock bool
	// EnableSoftSync opts the P25 control channel into the wider FSW-fallback
	// tolerance: sync words carrying 5–8 symbol errors are decoded if the
	// frame's TSBK CRC corroborates the NID, extending lock reach into
	// marginal-SNR captures. Gated by the existing NID/CRC validation, so it
	// cannot manufacture a false lock (issue #771; off in production).
	EnableSoftSync bool
	// CollectReceiverState captures a per-(stream-)second snapshot of the P25
	// receiver's AFC/AGC/clock/slicer state into the Result (the per-second
	// state log replay emitted to stderr, now structured + exportable).
	CollectReceiverState bool

	// ChunkSamples is the read-loop chunk size in IQ samples; 0 ⇒ default.
	ChunkSamples int

	// RecordDDCPath, when non-empty, writes the post-DDC narrowband IQ (the
	// exact channelized complex stream the receiver decodes, at the DDC output
	// rate — 144 kHz for TETRA, ~48 kHz for the C4FM family) to a two-channel
	// 16-bit baseband WAV at that path. This is the offline half of the "record
	// the digital down-converter output" tooling: point it at a fat 2.5/10 MS/s
	// cfile and it emits a small narrowband WAV that `replay -format wav`
	// decodes bit-for-bit the same way — a shareable, reproducible fixture that
	// is orders of magnitude smaller than the wideband capture. Empty ⇒ no
	// recording.
	RecordDDCPath string

	// CaptureIQ enables buffering a stride-decimated copy of the
	// post-DDC (channelized) IQ and the pre-slicer soft samples into
	// Result.IQTaps, so a web/visualization consumer can render the
	// constellation, spectrogram, PSD and eye diagram client-side. Off
	// by default since it is O(captured points) memory. The capture is
	// always decimated and hard-capped (see CaptureIQMaxPoints), so it
	// never ships full-rate IQ.
	CaptureIQ bool
	// CaptureIQMaxPoints caps the number of decimated IQ points (and,
	// separately, soft samples) retained when CaptureIQ is set. 0 ⇒
	// defaultCaptureIQMaxPoints. The read loop strides the channelized
	// stream so the retained density targets visualization fidelity
	// rather than faithful reconstruction.
	CaptureIQMaxPoints int

	// MaxSamples caps the number of input IQ samples processed (0 ⇒ whole
	// file). The signal identifier sets this to scan only a prefix of a
	// capture, bounding per-candidate cost regardless of capture length.
	MaxSamples int64

	// CollectPDUs retains a per-signaling-block dissection (Result.PDUs) of the
	// data PDUs the decoder parses — the data-over-RF inspector feed. Off by
	// default; P25 Phase 1 (TSBK) only for now. Capped at pduMaxRecords.
	CollectPDUs bool

	// Acceptance, when non-nil, is evaluated against the Result to produce a
	// pass/fail Verdict (the `test` harness sets it).
	Acceptance *Acceptance

	// Log receives the production pipeline's structured diagnostics. nil ⇒
	// a discard logger (the engine never wants a nil *slog.Logger reaching a
	// factory that does not nil-guard it).
	Log *slog.Logger

	// Bus, when non-nil, is the event bus the decode pipeline publishes grants
	// / decode errors to (and the internal collector subscribes to). Supplying
	// one lets a caller wire additional subscribers — e.g. a trunking engine +
	// voice composer + recorder that turn grants into recordings (replay
	// -record-voice). nil ⇒ the engine creates and closes its own bus (the
	// default; grants are only collected into the Result).
	Bus *events.Bus

	// OnChannelIQ, when non-nil, is called with each post-DDC channelized IQ
	// chunk (the exact complex stream the receiver decodes, at the DDC output
	// rate) as it is produced. The slice is reused after the call returns, so
	// an implementation that keeps the samples MUST copy them. This is the
	// live counterpart of RecordDDCPath: it feeds a same-carrier voice source
	// so replay can decode voice off the control carrier's own IQ. A slow
	// consumer paces the decode (the call is synchronous), which is what keeps
	// a fast file replay from overrunning the real-time voice chain. rateHz is
	// the DDC output (channelized) rate the samples are at.
	OnChannelIQ func(iq []complex64, rateHz float64)
}

const defaultChunkSamples = 8192

// defaultCaptureIQMaxPoints is the hard ceiling on retained decimated IQ
// points (and soft samples) when CaptureIQ is set but CaptureIQMaxPoints
// is not. ~200k float32 IQ pairs is ~1.6 MB on the wire — enough density
// for a spectrogram/PSD/constellation without unbounded memory growth.
const defaultCaptureIQMaxPoints = 200_000

// captureIQTargetRateHz is the post-decimation rate the IQ tap aims for.
// Higher than the live-constellation 2 ksps default (internal/dsp/diag)
// because offline FFTs want spectral fidelity, not just a lively canvas.
const captureIQTargetRateHz = 96_000

func (c Config) captureIQMaxPoints() int {
	if c.CaptureIQMaxPoints > 0 {
		return c.CaptureIQMaxPoints
	}
	return defaultCaptureIQMaxPoints
}

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
	return c.CollectIQDiag || c.CollectReceiverState || c.CollectPDUs ||
		c.NIDSearchSpan > 0 || c.EnableDDA || c.EnableAdaptiveSlicer || c.EnableDCBlock ||
		c.EnableSoftSync || c.DemodMode != ""
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
