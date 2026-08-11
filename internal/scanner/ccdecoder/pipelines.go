package ccdecoder

import (
	"context"
	"log/slog"
	"math"
	"sort"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	dmrrx "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr/tier2"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr/tier3"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dpmr"
	dpmrrx "github.com/MattCheramie/GopherTrunk/internal/radio/dpmr/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dstar"
	dstarrx "github.com/MattCheramie/GopherTrunk/internal/radio/dstar/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/radio/edacs"
	edacsrx "github.com/MattCheramie/GopherTrunk/internal/radio/edacs/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/radio/ltr"
	ltrrx "github.com/MattCheramie/GopherTrunk/internal/radio/ltr/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/radio/motorola"
	motorolarx "github.com/MattCheramie/GopherTrunk/internal/radio/motorola/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/radio/mpt1327"
	mpt1327rx "github.com/MattCheramie/GopherTrunk/internal/radio/mpt1327/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
	nxdnrx "github.com/MattCheramie/GopherTrunk/internal/radio/nxdn/receiver"
	p25phase1 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	p25phase1rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1/receiver"
	p25phase2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	p25phase2rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	tetrarx "github.com/MattCheramie/GopherTrunk/internal/radio/tetra/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/radio/ysf"
	ysfrx "github.com/MattCheramie/GopherTrunk/internal/radio/ysf/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"github.com/MattCheramie/GopherTrunk/internal/version"
)

// ProtocolPipeline is the contract every per-protocol receiver
// pipeline satisfies. Process consumes one chunk of complex IQ;
// Reset clears symbol-domain state on stream re-sync; Close
// releases any held resources (it's idempotent and may return nil).
type ProtocolPipeline interface {
	Process(iq []complex64)
	Reset()
	Close() error
}

// PipelineOptions is the per-pipeline construction shape — the
// connector hands the bus + log down, plus the (system, frequency)
// the supervisor is currently attempting and the IQ sample rate
// the receiver needs to size its matched filter.
//
// System carries the full trunking.System the supervisor is hunting,
// so per-protocol factories can read protocol-specific config off it
// (TETRA colour code + expected channel, P25 WACN, etc.) without
// needing a new field on PipelineOptions per protocol. SystemName +
// FrequencyHz remain at the top level because they're consumed by
// every factory.
type PipelineOptions struct {
	Bus          *events.Bus
	Log          *slog.Logger
	SystemName   string
	FrequencyHz  uint32
	SampleRateHz float64
	System       trunking.System

	// SymbolTap, when non-nil, is invoked with every chunk of recovered
	// symbols a pipeline's receiver emits, just before they enter the
	// control-channel state machine. symbols holds dibits (values 0..3,
	// isBits=false) for the 4-level protocols and bits (values 0..1,
	// isBits=true) for the 2-level protocols; baseIdx is the absolute
	// symbol index the chunk starts at. It is a pure observation hook —
	// production never sets it. The offline siglab toolkit uses it to
	// count symbols and feed its protocol-agnostic signal-quality
	// analyzer for *every* protocol, without re-duplicating receiver
	// construction outside this factory. nil ⇒ zero overhead.
	SymbolTap func(symbols []uint8, isBits bool, baseIdx int)

	// FECObserver, when non-nil, receives a per-burst FEC correction
	// depth (channel bits the decode chain corrected) bound to this
	// system. The decoder supplies it only when metrics.detailed_fec is
	// enabled; today only newTETRAPipeline consumes it. nil ⇒ zero
	// overhead.
	FECObserver func(channel string, corrections int)

	// CarrierBiasHz, when non-nil, returns the frequency correction (Hz) the
	// decoder has already folded into the down-converter for the active
	// acquisition (the autotune-applied shift). newP25Phase1Pipeline adds it
	// to the receiver's residual AFCOffsetHz so the published
	// control_channel_carrier_offset_hz reports the TOTAL offset from the
	// configured frequency — matching the WARN in checkCarrierOffsetLocked —
	// instead of the residual-only value that drops toward 0 once autotune
	// re-centres the carrier (issue #815). Nil ⇒ residual only.
	CarrierBiasHz func() float64
}

// tapDibits / tapBits forward a recovered-symbol chunk to SymbolTap when
// one is wired, normalising the dibit ([]uint8) and bit ([]byte) sink
// shapes onto the single SymbolTap signature. byte and uint8 are the same
// underlying type, so the bit slice passes through without a copy.
func (o PipelineOptions) tapDibits(dibits []uint8, baseIdx int) {
	if o.SymbolTap != nil {
		o.SymbolTap(dibits, false, baseIdx)
	}
}

func (o PipelineOptions) tapBits(bits []byte, baseIdx int) {
	if o.SymbolTap != nil {
		o.SymbolTap(bits, true, baseIdx)
	}
}

// PipelineFactory constructs a fresh ProtocolPipeline for one tuned
// system. The factory returns an error when the protocol's
// per-receiver / per-state-machine wiring isn't complete enough to
// drive a live CC pipeline end-to-end yet — the connector skips the
// retune in that case and the system stays in `state=hunting`.
type PipelineFactory func(PipelineOptions) (ProtocolPipeline, error)

// factories maps a trunking.Protocol to its pipeline factory. Only
// protocols whose ControlChannel state machine already accepts a
// raw dibit / bit stream are wired here. Others land in follow-up
// PRs as the per-protocol Process(...) adapters ship.
//
// The Protocol enum currently lumps P25 Phase 1 and Phase 2
// together; this factory targets Phase 1 (the more common
// deployment + the protocol with a complete IQ → dibits → CC →
// bus chain shipping today). A future PR splits Phase 1 / Phase 2
// once the daemon's config grows a per-system phase selector.
//
// DMR / NXDN / dPMR / EDACS / MPT 1327 / LTR / Motorola Type II /
// TETRA all have IQ → symbol receivers shipping but their
// ControlChannel state machines still consume pre-parsed PDUs.
// Adding `Process(stream, baseIdx)` adapters that buffer +
// detect sync + frame + dispatch into the existing parsers is a
// follow-up.
// SetTestFactory replaces the registered pipeline factory for a
// single protocol and returns a restore function the caller is
// expected to defer. INTENDED FOR INTEGRATION TESTS ONLY — the
// in-package unit tests substitute factories by mutating the
// unexported map directly. Out-of-package integration tests
// (e.g. cmd/gophertrunk's end-to-end "lights up live trunked
// reception" check) need an exported hook so they can pump
// known-good dibit streams through the daemon's real ccdecoder
// without owning a working C4FM modulator.
//
// Production code MUST NOT call this — the factory map is
// initialised once at package load and the daemon assumes it
// stays stable for the rest of the process lifetime.
func SetTestFactory(protocol trunking.Protocol, f PipelineFactory) (restore func()) {
	saved, hadSaved := factories[protocol]
	factories[protocol] = f
	return func() {
		if hadSaved {
			factories[protocol] = saved
		} else {
			delete(factories, protocol)
		}
	}
}

// NewPipeline constructs the registered pipeline for protocol p with the
// given options. ok is false (and the pipeline nil) when no factory is
// registered for p — callers should treat that as "this protocol cannot
// be driven offline yet" rather than an error. err propagates a factory's
// own construction failure (e.g. incomplete per-system wiring).
//
// This is the out-of-package entry point the offline siglab toolkit uses
// to drive any protocol through the same production pipelines the daemon
// runs; the daemon itself keeps using the internal factory map directly.
func NewPipeline(p trunking.Protocol, opts PipelineOptions) (pipe ProtocolPipeline, ok bool, err error) {
	f, ok := factories[p]
	if !ok {
		return nil, false, nil
	}
	pipe, err = f(opts)
	return pipe, true, err
}

// HasFactory reports whether a pipeline factory is registered for p.
func HasFactory(p trunking.Protocol) bool {
	_, ok := factories[p]
	return ok
}

// RegisteredProtocols returns the protocols with a registered pipeline
// factory, in a stable (ascending Protocol-value) order so callers — the
// siglab TUI's protocol picker, a gen→test sweep — render a deterministic
// list.
func RegisteredProtocols() []trunking.Protocol {
	out := make([]trunking.Protocol, 0, len(factories))
	for p := range factories {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

var factories = map[trunking.Protocol]PipelineFactory{
	trunking.ProtocolP25:       newP25Phase1Pipeline,
	trunking.ProtocolP25Phase2: newP25Phase2Pipeline,
	trunking.ProtocolDMR:       newDMRTier3Pipeline,
	trunking.ProtocolDPMR:      newDPMRPipeline,
	trunking.ProtocolNXDN:      newNXDNPipeline,
	trunking.ProtocolEDACS:     newEDACSPipeline,
	trunking.ProtocolMotorola:  newMotorolaPipeline,
	trunking.ProtocolLTR:       newLTRPipeline,
	trunking.ProtocolMPT1327:   newMPT1327Pipeline,
	trunking.ProtocolTETRA:     newTETRAPipeline,
	trunking.ProtocolYSF:       newYSFPipeline,
	trunking.ProtocolDStar:     newDStarPipeline,
	trunking.ProtocolDMRTier2:  newDMRTier2Pipeline,
	trunking.ProtocolDMRTier1:  newDMRTier1Pipeline,
}

// newP25Phase1Pipeline wires the existing
// internal/radio/p25/phase1/receiver into
// phase1.ControlChannel.Process. The receiver's DibitSink
// forwards dibits + baseIdx straight into the state machine,
// which publishes events.KindCCLocked + events.KindGrant on the
// bus when the supervisor's tuned frequency carries valid P25
// traffic.
func newP25Phase1Pipeline(opts PipelineOptions) (ProtocolPipeline, error) {
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	demodMode, ok := p25phase1rx.ParseDemodMode(opts.System.P25Phase1DemodMode)
	if !ok {
		log.Warn("ccdecoder: unrecognised p25_phase1_demod_mode; falling back to c4fm",
			"system", opts.SystemName, "value", opts.System.P25Phase1DemodMode)
	}
	// On a C4FM FM-discriminator stream only rotations 0 (identity)
	// and 2 (discriminator polarity flip) are physical; rotations 1
	// and 3 are non-physical on that path, so restricting the FSW +
	// NID rotation search to {0, 2} stops the BCH decoder from
	// miscorrecting misaligned dibits into a parity-valid pseudo-NID
	// at a wrong rotation (issue #275). The CQPSK path has a genuine
	// four-fold phase ambiguity and keeps the all-rotation default.
	rotations := p25phase1.RotationsAll
	if demodMode == p25phase1rx.DemodC4FM {
		rotations = p25phase1.RotationsC4FM
	}
	// Seed the BandPlan with operator-supplied IDEN_UP entries (issue
	// #345). Sites that route grants through a channel ID they never
	// broadcast an IDEN_UP TSBK for would otherwise have those grants
	// silently dropped; this gives the operator a startup floor that
	// over-the-air IDEN_UPs naturally override later via the same
	// BandPlan.Apply path.
	var bandPlan *p25phase1.BandPlan
	if len(opts.System.P25BandPlan) > 0 {
		bandPlan = &p25phase1.BandPlan{}
		for _, e := range opts.System.P25BandPlan {
			bandPlan.Apply(p25phase1.IdentifierUpdate{
				ChannelID:   e.ChannelID,
				BaseHz:      e.BaseHz,
				SpacingHz:   e.SpacingHz,
				TxOffsetHz:  e.TxOffsetHz,
				BandwidthHz: e.BandwidthHz,
			})
			log.Info("ccdecoder: p25/phase1 band-plan seeded",
				"system", opts.SystemName,
				"id", e.ChannelID, "base_hz", e.BaseHz,
				"spacing_hz", e.SpacingHz, "tx_offset_hz", e.TxOffsetHz)
		}
	}
	// Parse the same Phase 2 FEC YAML knobs newP25Phase2Pipeline reads
	// (below), so a hybrid Phase 1 CC + Phase 2 TC system (issue #376,
	// MMR) can stamp its TDMA grants with the FEC config the voice
	// composer's Phase 2 chain needs. Unrecognised values produce the
	// same warn-then-fallback behaviour as the Phase 2 pipeline; an
	// empty per-system value retains the same default (Trellis=on,
	// everything else=off). Operators with Protocol="p25" override per
	// system via the existing p25_phase2_*_mode YAML keys.
	p2Trellis, p2TrellisOK := p25phase2.ParseTrellisMode(opts.System.P25Phase2TrellisMode)
	if !p2TrellisOK {
		log.Warn("ccdecoder: unrecognised p25_phase2_trellis_mode; falling back to on",
			"system", opts.SystemName, "value", opts.System.P25Phase2TrellisMode)
	}
	p2RS, p2RSOK := p25phase2.ParseRSMode(opts.System.P25Phase2RSMode)
	if !p2RSOK {
		log.Warn("ccdecoder: unrecognised p25_phase2_rs_mode; falling back to off",
			"system", opts.SystemName, "value", opts.System.P25Phase2RSMode)
	}
	p2Interleave, p2IlOK := p25phase2.ParseInterleaveMode(opts.System.P25Phase2InterleaveMode)
	if !p2IlOK {
		log.Warn("ccdecoder: unrecognised p25_phase2_interleave_mode; falling back to off",
			"system", opts.SystemName, "value", opts.System.P25Phase2InterleaveMode)
	}
	p2Scrambler, p2ScrOK := p25phase2.ParseScramblerMode(opts.System.P25Phase2ScramblerMode)
	if !p2ScrOK {
		log.Warn("ccdecoder: unrecognised p25_phase2_scrambler_mode; falling back to on",
			"system", opts.SystemName, "value", opts.System.P25Phase2ScramblerMode)
	}
	p2SoftDecision, p2SoftOK := p25phase2rx.ParseSoftDecision(opts.System.P25Phase2SoftDecision)
	if !p2SoftOK {
		log.Warn("ccdecoder: unrecognised p25_phase2_soft_decision; falling back to off",
			"system", opts.SystemName, "value", opts.System.P25Phase2SoftDecision)
	}
	p2Equalizer, p2EqOK := p25phase2rx.ParseEqualizer(opts.System.P25Phase2Equalizer)
	if !p2EqOK {
		log.Warn("ccdecoder: unrecognised p25_phase2_equalizer; falling back to off",
			"system", opts.SystemName, "value", opts.System.P25Phase2Equalizer)
	}
	// rx is forward-declared so the control channel's CarrierOffsetHz provider
	// can close over it; the closure is only called later, at site-update
	// publish time, well after rx is assigned below (issue #815).
	var rx *p25phase1rx.Receiver
	cc := p25phase1.New(p25phase1.Options{
		Bus:                   opts.Bus,
		Log:                   opts.Log,
		SystemName:            opts.SystemName,
		FrequencyHz:           opts.FrequencyHz,
		BandPlan:              bandPlan,
		Rotations:             rotations,
		P25Phase1DemodMode:    opts.System.P25Phase1DemodMode,
		P25Phase2Trellis:      uint8(p2Trellis),
		P25Phase2RS:           uint8(p2RS),
		P25Phase2Interleave:   uint8(p2Interleave),
		P25Phase2Scrambler:    uint8(p2Scrambler),
		P25Phase2SoftDecision: p2SoftDecision,
		P25Phase2Equalizer:    p2Equalizer,
		// Report the TOTAL carrier offset from the configured frequency:
		// the receiver's residual AFC plus any correction the decoder folded
		// into the DDC (CarrierBiasHz). Residual-only would drop toward 0
		// once autotune re-centres, hiding an offset the WARN still flags
		// (issue #815). With autotune off CarrierBiasHz returns 0.
		CarrierOffsetHz: func() float64 {
			off := rx.AFCOffsetHz()
			if opts.CarrierBiasHz != nil {
				off += opts.CarrierBiasHz()
			}
			return off
		},
	})
	rx = p25phase1rx.New(p25phase1rx.Options{
		SampleRateHz: opts.SampleRateHz,
		// P25 Phase 1 nominal peak deviation per TIA-102.BAAA-A
		// — calibrates the slicer thresholds against the
		// FM-discriminator output level (see
		// p25phase1rx.Options.DeviationHz). Hardcoded since the
		// air-interface deviation is spec-fixed; if a future
		// site uses non-standard deviation the connector can
		// expose this as a per-system YAML key. Only consulted on
		// the C4FM path; the CQPSK path is amplitude-invariant
		// after the matched filter so DeviationHz isn't used.
		DeviationHz: 1800.0,
		DemodMode:   demodMode,
		// EnableDecisionDirectedAFC intentionally left false: the
		// daemon runs CoarseAFC-alone (the pre-DDA behaviour). The DDA
		// can stably false-lock with no FSW/CC-lock feedback to catch
		// it and was a net regression on the issue #402 capture; keep
		// it off here until the eye-skew root cause is pinned.
		DibitSink: func(dibits []uint8, baseIdx int) {
			opts.tapDibits(dibits, baseIdx)
			cc.Process(dibits, baseIdx)
		},
	})
	// build is the link-time version stamp. Decoder log excerpts are
	// what issue reporters paste, and a git-describe string there is
	// the only reliable way to tell which fixes a build contains —
	// issue #275 saw a retest invalidated by a silently stale build.
	//
	// rotations / nid_search_span / nid_accept_errs / nid_marginal_max
	// pin the same retest-verification surface to the decode parameters
	// themselves: a stale build that still uses RotationsAll on a C4FM
	// site or the old ±2 search will show its signature in this line
	// without anyone having to read source.
	log.Info("ccdecoder: p25/phase1 pipeline configured",
		"system", opts.SystemName, "freq_hz", opts.FrequencyHz,
		"demod", demodModeLabel(demodMode),
		"rotations", rotations,
		"nid_search_span", p25phase1.NIDSearchSpan,
		"nid_accept_errs", p25phase1.NIDAcceptErrs,
		"nid_marginal_max", p25phase1.NIDMarginalMaxErrs,
		"build", version.String())
	return &p25Phase1Pipeline{rx: rx, cc: cc}, nil
}

func demodModeLabel(m p25phase1rx.DemodMode) string {
	switch m {
	case p25phase1rx.DemodCQPSK:
		return "cqpsk"
	default:
		return "c4fm"
	}
}

type p25Phase1Pipeline struct {
	rx *p25phase1rx.Receiver
	cc *p25phase1.ControlChannel
}

func (p *p25Phase1Pipeline) Process(iq []complex64) { p.rx.Process(iq) }
func (p *p25Phase1Pipeline) Reset()                 { p.rx.Reset() }
func (p *p25Phase1Pipeline) Close() error           { return nil }

// AFCOffsetHz reports the receiver's current estimate of the true carrier
// offset in Hz, so the decoder's autotune tracker can fold it into the
// per-dongle running average. 0 on the CQPSK path (no AFC stage). Satisfies
// the unexported afcReporter capability the decoder type-asserts for.
func (p *p25Phase1Pipeline) AFCOffsetHz() float64 { return p.rx.AFCOffsetHz() }

// appliesAutotune marks P25 Phase 1 as a protocol whose measured control-channel
// carrier error is consumed downstream (the P25 Phase 1 voice composer reads the
// per-dongle autotune Manager to pre-rotate voice IQ). Only marked pipelines are
// sampled + logged by sampleAutotuneLocked, so a protocol without an autotune
// consumer (e.g. TETRA) no longer floods the debug log with a correction it never
// applies. See autotuneApplier in decoder.go.
func (p *p25Phase1Pipeline) appliesAutotune() {}

// TSBKCounts reports the control channel's cumulative decoded and failed
// (Viterbi + CRC) TSBK block counts. Satisfies the unexported ccHealthReporter
// capability the decoder type-asserts for, so it can watch the live TSBK error
// rate and nudge the operator toward dc_avoid when a zero-IF lock decodes
// poorly (issue #402).
func (p *p25Phase1Pipeline) TSBKCounts() (decoded, failed int64) {
	s := p.cc.Stats()
	return s.TSBKDecoded, s.TSBKTrellisFailed + s.TSBKCRCFailed
}

// TopologySnapshot surfaces the P25 Phase 1 system topology (identity, primary/
// secondary control channels, neighbours, band plan) the control channel
// accumulated from its status broadcasts. Satisfies trunking.TopologyProvider so
// the decoder can log the network-configuration report at lock.
func (p *p25Phase1Pipeline) TopologySnapshot() *trunking.TopologySnapshot {
	return p.cc.TopologySnapshot()
}

// newP25Phase2Pipeline wires internal/radio/p25/phase2/receiver into
// p25phase2.ControlChannel.Process. The receiver's DibitSink forwards
// H-DQPSK dibits into the state machine (20-dibit outbound sync
// detect → 146-channel-dibit trellis decode → MAC PDU parse →
// Ingest), which publishes cc.locked on the first non-idle MAC PDU
// and grants on GroupVoiceChannelGrant variants.
//
// Trellis FEC is on by default: the factory always runs
// p25phase2.ParseTrellisMode over the per-system config string,
// which maps an empty string to TrellisOn. Operators feeding
// pre-stripped MAC-PDU fixtures opt out per-system with
// `p25_phase2_trellis_mode: off`.
//
// The connector wires the receiver with `ClockMode: ClockGardner`
// — Gardner timing recovery on complex IQ replaces the receiver's
// default naive decimation, which matters for noisier live SDR
// captures where the symbol clock isn't aligned with the sample
// clock. The legacy ClockNaive path stays callable for in-package
// tests that synthesize sample-aligned IQ fixtures.
func newP25Phase2Pipeline(opts PipelineOptions) (ProtocolPipeline, error) {
	cc := p25phase2.New(p25phase2.Options{
		Bus:         opts.Bus,
		Log:         opts.Log,
		SystemName:  opts.SystemName,
		FrequencyHz: opts.FrequencyHz,
	})
	trellisMode, ok := p25phase2.ParseTrellisMode(opts.System.P25Phase2TrellisMode)
	if !ok {
		opts.Log.Warn("ccdecoder: unrecognised p25_phase2_trellis_mode; falling back to on",
			"system", opts.SystemName, "value", opts.System.P25Phase2TrellisMode)
	}
	cc.SetTrellisMode(trellisMode)
	rsMode, rsOK := p25phase2.ParseRSMode(opts.System.P25Phase2RSMode)
	if !rsOK {
		opts.Log.Warn("ccdecoder: unrecognised p25_phase2_rs_mode; falling back to off",
			"system", opts.SystemName, "value", opts.System.P25Phase2RSMode)
	}
	cc.SetRSMode(rsMode)
	interleaveMode, ilOK := p25phase2.ParseInterleaveMode(opts.System.P25Phase2InterleaveMode)
	if !ilOK {
		opts.Log.Warn("ccdecoder: unrecognised p25_phase2_interleave_mode; falling back to off",
			"system", opts.SystemName, "value", opts.System.P25Phase2InterleaveMode)
	}
	cc.SetInterleaveMode(interleaveMode)
	scramblerMode, scrOK := p25phase2.ParseScramblerMode(opts.System.P25Phase2ScramblerMode)
	if !scrOK {
		opts.Log.Warn("ccdecoder: unrecognised p25_phase2_scrambler_mode; falling back to on",
			"system", opts.SystemName, "value", opts.System.P25Phase2ScramblerMode)
	}
	if scramblerMode == p25phase2.ScramblerProbe && !rsMode.Enabled() {
		opts.Log.Warn("ccdecoder: p25_phase2_scrambler_mode=probe requires p25_phase2_rs_mode=on or =correct; descrambler will degrade to offset 0",
			"system", opts.SystemName)
	}
	cc.SetScramblerMode(scramblerMode)
	// Derive the PN44 seed from (WACN, SystemID, low-12 bits of Site
	// as the spec's Color Code = NAC) per TIA-102.BBAC-1 §7.2.5
	// equation (5). System operators who haven't configured these
	// values end up with a zero-input seed that maps to (2^44 - 1)
	// per spec — the descrambler runs but with an unlikely-to-help
	// sequence. Future PRs derive the seed from the Network Status
	// Broadcast MAC message at runtime instead of static config.
	cc.SetScramblerSeed(framing.PN44SeedFromIdentity(
		opts.System.WACN, opts.System.SystemID, uint16(opts.System.Site),
	))
	softDecision, softOK := p25phase2rx.ParseSoftDecision(opts.System.P25Phase2SoftDecision)
	if !softOK {
		opts.Log.Warn("ccdecoder: unrecognised p25_phase2_soft_decision; falling back to off",
			"system", opts.SystemName, "value", opts.System.P25Phase2SoftDecision)
	}
	cc.SetSoftDecision(softDecision)
	equalizer, eqOK := p25phase2rx.ParseEqualizer(opts.System.P25Phase2Equalizer)
	if !eqOK {
		opts.Log.Warn("ccdecoder: unrecognised p25_phase2_equalizer; falling back to off",
			"system", opts.SystemName, "value", opts.System.P25Phase2Equalizer)
	}
	cc.SetEqualizer(equalizer)
	clockMode, clockOK := p25phase2rx.ParseClockMode(opts.System.P25Phase2ClockMode)
	if !clockOK {
		opts.Log.Warn("ccdecoder: unrecognised p25_phase2_clock_mode; falling back to gardner",
			"system", opts.SystemName, "value", opts.System.P25Phase2ClockMode)
	}
	// The receiver's dibit stream drives a SuperframeDecoder: it locks
	// the 360 ms TDMA superframe, slices the 12 sub-frames, decodes
	// each ISCH SlotType, and IngestSuperframe routes the MAC-bearing
	// sub-frames into the control-channel state machine. cc.Process
	// (the flat sync-window adapter) stays available for callers that
	// feed pre-stripped fixtures, but the live pipeline is structured.
	sfDec := p25phase2.NewSuperframeDecoder()
	rx := p25phase2rx.New(p25phase2rx.Options{
		SampleRateHz: opts.SampleRateHz,
		DibitSink: func(dibits []uint8, baseIdx int) {
			opts.tapDibits(dibits, baseIdx)
			for _, sf := range sfDec.Process(dibits, baseIdx) {
				cc.IngestSuperframe(sf)
			}
		},
		ClockMode: clockMode,
		// Tuned smaller than the 0.03 default — H-DQPSK at
		// 6000 sym/s has the same slip behaviour as TETRA's
		// π/4-DQPSK at the default gain (see PR #154). 0.005
		// tracks both clean synthesized IQ and noisier on-air
		// captures within the loop's lock-acquisition margin.
		// Only applied when ClockMode == ClockGardner.
		GardnerGain: 0.005,
	})
	return &p25Phase2Pipeline{rx: rx, cc: cc, sfDec: sfDec}, nil
}

type p25Phase2Pipeline struct {
	rx    *p25phase2rx.Receiver
	cc    *p25phase2.ControlChannel
	sfDec *p25phase2.SuperframeDecoder
}

func (p *p25Phase2Pipeline) Process(iq []complex64) { p.rx.Process(iq) }
func (p *p25Phase2Pipeline) Reset() {
	p.rx.Reset()
	p.sfDec.Reset()
}
func (p *p25Phase2Pipeline) Close() error { return nil }

// newTETRAPipeline wires internal/radio/tetra/receiver into
// tetra.ControlChannel.Process. The receiver's DibitSink forwards
// π/4-DQPSK dibits into the state machine.
//
// Channel coding is on by default: the factory always runs
// tetra.ParseChannelCoding over the per-system config string, which
// maps an empty string to ChannelCodingOn, then slices per the
// configured TETRAChannel (default ChannelSCHHD) and runs the full
// ETSI EN 300 392-2 §8.3.1 type-5 → type-1 decode chain (descramble +
// deinterleave + depuncture + Viterbi + CRC-16 verify + tail strip)
// per burst. The TETRAColourCode seeds the descrambler — zero is
// only valid for BSCH; non-BSCH channels need the per-cell colour
// code or descrambling produces garbage. Operators feeding pre-
// stripped DSD-FME / OP25 fixtures opt out per-system with
// `tetra_channel_coding: off`.
//
// The connector wires the receiver with `ClockMode: ClockGardner`
// — Gardner timing recovery on complex IQ replaces the receiver's
// default naive decimation. Same pattern as the P25 Phase 2
// pipeline; the legacy ClockNaive path stays callable for
// in-package tests that synthesize sample-aligned IQ fixtures.
func newTETRAPipeline(opts PipelineOptions) (ProtocolPipeline, error) {
	cc := tetra.New(tetra.Options{
		Bus:         opts.Bus,
		Log:         opts.Log,
		SystemName:  opts.SystemName,
		FrequencyHz: opts.FrequencyHz,
		// Opt-in FEC correction-depth histogram (metrics.detailed_fec);
		// nil unless the daemon wired the observer.
		FECObserver: opts.FECObserver,
	})
	codingMode, ok := tetra.ParseChannelCoding(opts.System.TETRAChannelCoding)
	if !ok {
		opts.Log.Warn("ccdecoder: unrecognised tetra_channel_coding; falling back to on",
			"system", opts.SystemName, "value", opts.System.TETRAChannelCoding)
	}
	cc.SetChannelCoding(codingMode)
	if codingMode == tetra.ChannelCodingOn {
		ch, chOK := tetra.ParseChannelType(opts.System.TETRAChannel)
		if !chOK {
			opts.Log.Warn("ccdecoder: unrecognised tetra_channel; falling back to SCH/HD",
				"system", opts.SystemName, "value", opts.System.TETRAChannel)
		}
		cc.SetExpectedChannel(ch)
		cc.SetColourCode(opts.System.TETRAColourCode)
		if opts.System.TETRAColourCode == 0 && ch != tetra.ChannelBSCH {
			// Not fatal: the decoder auto-acquires the colour code from a BSCH
			// synchronisation burst (scrambled with colour 0) and then descrambles
			// the SCH/HD stream (issue #553). Setting tetra_colour_code only avoids
			// the cold-start wait for the first SB burst. Debug, not Warn — this is
			// the normal state during blind identify (#648).
			opts.Log.Debug("ccdecoder: tetra zero colour code; will auto-acquire from the BSCH synchronisation burst",
				"system", opts.SystemName, "channel", opts.System.TETRAChannel)
		}
	}
	tetraClockMode, tetraClockOK := tetrarx.ParseClockMode(opts.System.TETRAClockMode)
	if !tetraClockOK {
		opts.Log.Warn("ccdecoder: unrecognised tetra_clock_mode; falling back to gardner",
			"system", opts.SystemName, "value", opts.System.TETRAClockMode)
	}
	rx := tetrarx.New(tetrarx.Options{
		SampleRateHz: opts.SampleRateHz,
		DibitSink: func(dibits []uint8, baseIdx int) {
			opts.tapDibits(dibits, baseIdx)
			cc.Process(dibits, baseIdx)
		},
		// Soft differentials for soft-decision channel decoding, stashed
		// just before the matching DibitSink → Process call (issue #553).
		SoftSink: func(diffs []complex64, baseIdx int) {
			cc.StashSoft(diffs, baseIdx)
		},
		ClockMode: tetraClockMode,
		// Tuned smaller than the 0.03 default — at TETRA's 18000
		// sym/s the standard gain over-corrects on clean signals
		// and slips. 0.005 tracks both clean synthesized IQ (the
		// integration-cc test) and noisier on-air captures within
		// the loop's lock-acquisition margin. Same pattern as the
		// DMR Tier III ClockGain tweak in PR #150. Only applied
		// when ClockMode == ClockGardner.
		GardnerGain: 0.005,
		// The live DDC has no AFC; a control channel that is not
		// perfectly centred (a coarse tuning offset, or a tuner a few
		// hundred Hz off) leaves a constant per-symbol phase offset
		// that biases every dibit and stops the training sequence
		// correlating (issue #553). The AFC acquires ~0 on a centred
		// signal, so it is a near-noop there.
		EnableAFC: true,
		// The channelised stream is far wider than a 25 kHz TETRA
		// channel, so adjacent carriers leak through to the matched
		// filter. The channel-select filter rejects them — measured to
		// cut the on-air symbol error rate by ~10x (issue #553).
		EnableChannelFilter: true,
		// Blind SnapshotCMA equalizer between symbol timing and the
		// differential decoder, inverting the multipath / ISI / band-edge
		// group delay that smears the π/4-DQPSK constellation on real
		// captures. Already enabled on the voice composer and wideband-T2
		// CC paths (widebandt2/tetra.go: "mirror the ccdecoder pipeline's
		// on-air settings") — this closes the gap where the primary live
		// CC path alone lacked it. On the reporter's ~10 dB re-acquisition
		// capture it lifts CRC-clean BSCH yield from ~12% to ~100%, which
		// is the difference between riding through a marginal dip and
		// dropping lock → re-hunt (the ~210 CC transitions/hour symptom).
		// Differential-decoder-safe (frozen snapshot) and a near-noop on
		// already-clean signals. EnableDCBlock stays OFF — reserved for the
		// voice receivers; it must not disturb steady-state CC decode.
		EnableEqualizer: true,
	})
	// Emission cadence of the decode-status line, operator-configurable via
	// tetra_status_interval_secs; falls back to the 5 s default when unset or
	// non-positive.
	statusInterval := tetraStatusInterval
	if s := opts.System.TETRAStatusIntervalSecs; s > 0 {
		statusInterval = time.Duration(s * float64(time.Second))
	}
	return &tetraPipeline{
		rx:             rx,
		cc:             cc,
		log:            opts.Log,
		system:         opts.SystemName,
		rateHz:         opts.SampleRateHz,
		now:            time.Now,
		statusInterval: statusInterval,
		// Gate the periodic decode-status line on debug once, up front, so a
		// production (info-level) decode does no per-chunk clock reads. The
		// control channel gates its own counter accumulation the same way.
		debug: opts.Log != nil && opts.Log.Enabled(context.Background(), slog.LevelDebug),
	}, nil
}

// tetraStatusInterval is the DEFAULT throttle for the aggregate "tetra: decode
// status" debug line so a long capture emits a compact health summary a few
// times a minute rather than once per IQ chunk. Mirrors the ~1 s AFC-status
// throttle in decoder.go. Operators can override the cadence per system via
// tetra_status_interval_secs (see newTETRAPipeline / tetraPipeline.statusInterval).
const tetraStatusInterval = 5 * time.Second

// tetraLockStaleTimeout is how long a locked TETRA control channel may decode
// nothing before the watchdog (ControlChannel.CheckStale) declares it lost and
// publishes cc.lost, so the cchunt supervisor leaves StateLocked and re-hunts.
// Generous (~5 missed multiframes at the ~1 s BSCH cadence) so brief fades or a
// momentarily busy carrier never trip it, while a genuinely dead carrier is
// surfaced within a few seconds. tetraStaleCheckInterval throttles how often
// Process runs the (cheap) check.
const (
	tetraLockStaleTimeout   = 5 * time.Second
	tetraStaleCheckInterval = 1 * time.Second
)

// tetraResyncTimeout is how much CONTROL-CHANNEL SIGNAL the receiver may process
// without a single CRC-clean decode before the pipeline forces a fast DSP
// re-acquire (reset the symbol-timing and AFC loops to centre, then rebuild the
// CC's dibit-sync scratch). A noise burst wanders the Gardner timing phase
// off-lock and, at the production GardnerGain, it re-converges only very slowly —
// the field symptom was a control channel taking tens of seconds to re-lock after
// brief wideband noise while a from-cold receiver acquires in ~one AFC block.
// Resetting to centre on a genuine decode drought reacquires in that same time.
//
// It is expressed as a duration but measured against PROCESSED-SIGNAL time, not
// wall clock: checkResync counts the post-DDC IQ samples actually fed to the
// receiver since the last decode and compares them against a sample budget
// (tetraResyncTimeout × the channel rate). This is the fix for the multislot
// "losing dsp sync" garble. Reset-to-centre is destructive — it discards the
// receiver's *converged* Gardner timing and AFC — so it must fire only on genuine
// loss, never on a transient scheduling stall. Under concurrent same-carrier voice
// load the CPU-heavy shared voice demux momentarily starves this CC decode
// goroutine, and its input IQ is meanwhile dropped upstream (forwardIQ). A
// wall-clock window then aged out purely because the goroutine was descheduled —
// not because timing drifted — firing a destructive reset every window and
// churning through resets that discard a still-good lock (the reporter's field
// captures show ~46 such resyncs). A signal-time budget is immune: a descheduled
// goroutine processes no samples, so the budget never advances and no reset fires;
// a healthy lock decodes a sync burst (~1/s) well within the budget and never
// trips it; only a genuine off-lock that processes a full window of real signal
// with no decode reacquires. 1.5 s of signal sits above the ~1 s BSCH cadence yet
// well below the 5 s tetraLockStaleTimeout backstop, so a genuinely dead carrier
// still re-hunts. Because starvation can no longer trigger a reset, the earlier
// exponential back-off (whose only job was to damp the starvation-driven reset
// storm) is no longer needed and has been removed.
const tetraResyncTimeout = 1500 * time.Millisecond

type tetraPipeline struct {
	rx             *tetrarx.Receiver
	cc             *tetra.ControlChannel
	log            *slog.Logger
	system         string
	debug          bool
	rateHz         float64          // post-DDC channel rate; converts the resync window into a sample budget
	now            func() time.Time // injectable for tests; set to time.Now at construction
	statusInterval time.Duration    // decode-status emission cadence (tetra_status_interval_secs; default 5 s)
	lastLog        time.Time        // wall clock of the previous status line (zero ⇒ not primed)
	lastStale      time.Time        // wall clock of the previous lock-staleness check
	// Signal-time DSP-resync trigger: the receiver must PROCESS a full
	// tetraResyncTimeout-worth of post-DDC samples with no CRC-clean CC decode
	// before a destructive reset-to-centre is allowed. Counting processed signal
	// (not wall clock) makes the reset immune to CPU starvation, which ages a wall
	// clock without advancing the sample count. See tetraResyncTimeout.
	samplesSinceDecode int64
	lastSeenActivity   int64 // cc heartbeat nano last observed; a change ⇒ a decode landed since
}

func (p *tetraPipeline) Process(iq []complex64) {
	p.rx.Process(iq)
	p.checkResync(len(iq))
	p.checkLockStale()
	p.maybeLogStatus()
}

// AFCOffsetHz reports the receiver's measured carrier offset (Hz), so the Decoder
// can surface control_channel_carrier_offset for TETRA. Satisfies afcReporter.
func (p *tetraPipeline) AFCOffsetHz() float64 { return p.rx.CarrierOffsetHz() }

// BSCHCounts exposes the control channel's always-on cumulative BSCH (ok, fail)
// decode counts, from which the Decoder derives a TETRA decode-quality bucket —
// the analogue of the P25 pipeline's TSBKCounts. Satisfies bschHealthReporter.
func (p *tetraPipeline) BSCHCounts() (ok, fail int64) { return p.cc.BSCHCounts() }

// checkResync forces a fast DSP re-acquire once the control channel has PROCESSED
// a full tetraResyncTimeout-worth of post-DDC signal (n counts the samples just
// fed to the receiver) without a single CRC-clean decode. It resets the receiver's
// timing/AFC loops to centre and drops the CC's dibit-sync scratch (kept in
// lock-step because rx.Reset restarts the dibit index at 0), so a channel knocked
// off-lock by a noise burst reacquires in ~one AFC block instead of waiting for
// the slow steady-state Gardner loop to drift back.
//
// The trigger is PROCESSED-SIGNAL time, not wall clock (see tetraResyncTimeout):
// a CC decode since the last call (heartbeat advanced) clears the sample budget;
// otherwise the budget accumulates and exactly one reset fires per window-worth of
// real signal, resetting the budget again — an inherent throttle and reacquire
// window. Counting processed signal is what makes the reset immune to CPU
// starvation: a descheduled goroutine (whose IQ is meanwhile dropped upstream)
// feeds no samples, so the budget never advances and a still-good lock is never
// discarded. Unlike maybeLogStatus this runs in production (not debug-gated): the
// reacquire must happen regardless of log level; only the diagnostic line is
// level-filtered. Called after rx.Process so a decode from the current chunk is
// credited before the budget grows.
func (p *tetraPipeline) checkResync(n int) {
	act := p.cc.LastActivityNano()
	if act == 0 {
		// No decode has ever landed — nothing to reacquire toward. Mirrors
		// NeedsResync/CheckStale, both no-ops before the first decode.
		return
	}
	if act != p.lastSeenActivity {
		// A decode landed since the last check: the drought (if any) is over, so
		// reset the budget and keep the lock.
		p.lastSeenActivity = act
		p.samplesSinceDecode = 0
		return
	}
	if p.rateHz <= 0 {
		return // defensive; tetrarx.New already requires SampleRateHz > 0
	}
	p.samplesSinceDecode += int64(n)
	if p.samplesSinceDecode < int64(tetraResyncTimeout.Seconds()*p.rateHz) {
		return
	}
	// A full window of real signal processed with no decode: a genuine off-lock.
	// Reset the budget first so the next reset needs another full window (throttle
	// + reacquire window), then reacquire the symbol timing from centre.
	p.samplesSinceDecode = 0
	p.rx.Reset()
	p.cc.ResyncReset()
	if p.log != nil {
		p.log.Debug("tetra: dsp resync (signal-time decode drought; reacquiring symbol timing from centre)",
			"system", p.system)
	}
}

// checkLockStale runs the control-channel lock watchdog on a light throttle.
// Unlike maybeLogStatus it is NOT debug-gated — the watchdog must run in
// production so a silent carrier surfaces cc.lost and the supervisor re-hunts.
func (p *tetraPipeline) checkLockStale() {
	now := p.now()
	if now.Sub(p.lastStale) < tetraStaleCheckInterval {
		return
	}
	p.lastStale = now
	p.cc.CheckStale(now, tetraLockStaleTimeout)
}

// tetraEffectiveBaud derives the effective symbol rate (baud) and its
// percentage deviation from the nominal 18000 sym/s over a window of `dibits`
// symbols spanning `elapsed`. This is the same figure siglab reports as
// EffectiveBaud = symbols / duration. Returns zeros for a non-positive window.
func tetraEffectiveBaud(dibits int64, elapsed time.Duration) (baud, devPct float64) {
	secs := elapsed.Seconds()
	if secs <= 0 {
		return 0, 0
	}
	baud = float64(dibits) / secs
	devPct = (baud - tetrarx.SymbolRate) / tetrarx.SymbolRate * 100
	return baud, devPct
}

// maybeLogStatus emits a throttled decode-health summary at debug level:
// effective baud + deviation (symbols over wall time, the same figure siglab
// reports as EffectiveBaud), the AFC's residual carrier offset, lock state and
// the per-window decode counts. No-op unless the logger is at debug level.
func (p *tetraPipeline) maybeLogStatus() {
	if !p.debug || p.log == nil {
		return
	}
	now := p.now()
	if p.lastLog.IsZero() {
		p.lastLog = now // prime the window; first line lands one interval later
		return
	}
	elapsed := now.Sub(p.lastLog)
	if elapsed < p.statusInterval {
		return
	}
	p.lastLog = now

	st := p.cc.DrainStats()
	baud, dev := tetraEffectiveBaud(st.Dibits, elapsed)
	p.log.Debug("tetra: decode status",
		"system", p.system,
		"locked", p.cc.Locked(),
		"carrier_off_hz", math.Round(p.rx.CarrierOffsetHz()*10)/10,
		"baud", math.Round(baud),
		"baud_dev_pct", math.Round(dev*100)/100,
		"sb_bursts", st.SBBursts,
		"bsch_ok", st.BSCHOK,
		"bsch_fail", st.BSCHFail,
		"sysinfo", st.SysInfo,
		"sch_pdus", st.SCHPDUs,
		"sch_pdus_fail", st.SCHPDUsFail,
		"grants", st.Grants,
		"colour_code", p.cc.Topology().ColourCode&0x3F,
	)
}

func (p *tetraPipeline) Reset() {
	p.rx.Reset()
	// rx.Reset restarts the receiver's dibit index at 0; keep the CC's
	// absolute-indexed sync scratch in step so a post-reset stream can
	// reacquire (see ControlChannel.ResyncReset).
	p.cc.ResyncReset()
	p.lastLog = time.Time{}
	p.lastStale = time.Time{}
	p.samplesSinceDecode = 0
	p.lastSeenActivity = 0
}
func (p *tetraPipeline) Close() error { return nil }

// TopologySnapshot surfaces the TETRA single-cell identity (MCC/MNC/LA + colour
// code) the control channel learned. No adjacent cells for TETRA.
func (p *tetraPipeline) TopologySnapshot() *trunking.TopologySnapshot {
	t := p.cc.Topology()
	snap := &trunking.TopologySnapshot{
		MCC:          t.MCC,
		MNC:          t.MNC,
		LocationArea: t.LocationArea,
		// The ETSI 6-bit colour code is the low 6 bits of the 30-bit extended
		// colour code (MCC<<20 | MNC<<6 | CC); masking 0xFF would drag in two
		// stray MNC bits and corrupt the surfaced value.
		ColorCode: uint8(t.ColourCode & 0x3F),
	}
	// Surface the cell's own control carrier with its offset-corrected downlink
	// frequency (§21.4.4.1) — the true carrier, not the tuned frequency — so the
	// network-config report shows where the CC actually is.
	if t.DownlinkHz != 0 {
		snap.PrimaryCC = &trunking.TopoChannelRef{
			ChannelNumber: t.MainCarrier,
			FrequencyHz:   t.DownlinkHz,
			// The uplink is the offset-corrected duplex pair (§21.4.4.1), derived
			// directly from the SYSINFO duplex spacing rather than a band plan.
			UplinkHz: t.UplinkHz,
		}
	}
	return snap
}

// newYSFPipeline wires the existing internal/radio/ysf/receiver
// into ysf.ControlChannel.Process. YSF is the System Fusion
// (Yaesu) C4FM amateur trunked variant — same 4800-baud
// modulation as P25 P1 / NXDN / DMR / dPMR, with α = 0.20 RRC and
// the standard 1800 Hz peak deviation.
func newYSFPipeline(opts PipelineOptions) (ProtocolPipeline, error) {
	cc := ysf.New(ysf.Options{
		Bus:         opts.Bus,
		Log:         opts.Log,
		SystemName:  opts.SystemName,
		FrequencyHz: opts.FrequencyHz,
	})
	rx := ysfrx.New(ysfrx.Options{
		SampleRateHz: opts.SampleRateHz,
		// YSF spec peak deviation, same calibration knob the
		// P25 P1 / NXDN / DMR / dPMR receivers picked up so live
		// captures slice correctly out of the box.
		DeviationHz: 1800.0,
		DibitSink: func(dibits []uint8, baseIdx int) {
			opts.tapDibits(dibits, baseIdx)
			cc.Process(dibits, baseIdx)
		},
	})
	return &ysfPipeline{rx: rx, cc: cc}, nil
}

type ysfPipeline struct {
	rx *ysfrx.Receiver
	cc *ysf.ControlChannel
}

func (p *ysfPipeline) Process(iq []complex64) { p.rx.Process(iq) }
func (p *ysfPipeline) Reset()                 { p.rx.Reset() }
func (p *ysfPipeline) Close() error           { return nil }

// newDPMRPipeline wires internal/radio/dpmr/receiver into
// dpmr.ControlChannel.Process. The receiver's DibitSink forwards
// dibits + baseIdx straight into the state machine's Process
// method (sync detect → 80-bit CSBK slice → CSBKFromBits →
// Ingest), which publishes events.KindCCLocked +
// events.KindGrant on the bus.
func newDPMRPipeline(opts PipelineOptions) (ProtocolPipeline, error) {
	cc := dpmr.New(dpmr.Options{
		Bus:         opts.Bus,
		Log:         opts.Log,
		SystemName:  opts.SystemName,
		FrequencyHz: opts.FrequencyHz,
	})
	rx := dpmrrx.New(dpmrrx.Options{
		SampleRateHz: opts.SampleRateHz,
		// dPMR Mode 3 peak deviation — half of P25 / DMR / YSF,
		// matching the 6.25 kHz channel spacing. Calibrates
		// slicer thresholds against the FM-discriminator output
		// level so live captures slice correctly.
		DeviationHz: 900.0,
		DibitSink: func(dibits []uint8, baseIdx int) {
			opts.tapDibits(dibits, baseIdx)
			cc.Process(dibits, baseIdx)
		},
	})
	return &dpmrPipeline{rx: rx, cc: cc}, nil
}

type dpmrPipeline struct {
	rx *dpmrrx.Receiver
	cc *dpmr.ControlChannel
}

func (p *dpmrPipeline) Process(iq []complex64) { p.rx.Process(iq) }
func (p *dpmrPipeline) Reset()                 { p.rx.Reset() }
func (p *dpmrPipeline) Close() error           { return nil }

// newDMRTier3Pipeline wires internal/radio/dmr/receiver into
// dmr/tier3.ControlChannel.Process. The receiver's DibitSink
// forwards dibits into the adapter (multi-pattern sync detect
// across all 9 ETSI sync words → 132-dibit burst slice →
// slot-type Hamming(20,8) decode → IngestBurst → BPTC(196,96) →
// CSBK CRC → cc.locked / grant publication).
func newDMRTier3Pipeline(opts PipelineOptions) (ProtocolPipeline, error) {
	cc := tier3.New(tier3.Options{
		Bus:         opts.Bus,
		Log:         opts.Log,
		SystemName:  opts.SystemName,
		FrequencyHz: opts.FrequencyHz,
		// LCN→downlink resolver from the system's dmr_band_plan; nil
		// when unconfigured, in which case T3 voice grants drop with
		// decode.error stage=no-bandplan (the daemon warns at load).
		Resolver:         tier3.ResolverFromPlan(opts.System.DMRBandPlan),
		InterleavedVoice: opts.System.DMRInterleavedVoice,
	})
	rx := dmrrx.New(dmrrx.Options{
		SampleRateHz: opts.SampleRateHz,
		// DMR spec peak deviation per ETSI TS 102 361-1 §6.3.
		// Calibrates the slicer thresholds against the
		// FM-discriminator output level so live captures slice
		// correctly out of the box.
		DeviationHz: 1944.0,
		// ClockGain tuned smaller than the 0.05 default — at
		// 1944 Hz deviation the per-sample phase excursion is
		// ~8% larger than P25 P1's, and the standard MM gain
		// slips on the harder symbol transitions during burst
		// payloads. 0.025 tracks cleanly on synthesized IQ
		// and stays well within the loop's noise margin for
		// live captures.
		ClockGain: 0.025,
		DibitSink: func(dibits []uint8, baseIdx int) {
			opts.tapDibits(dibits, baseIdx)
			cc.Process(dibits, baseIdx)
		},
	})
	return &dmrPipeline{rx: rx, cc: cc}, nil
}

type dmrPipeline struct {
	rx *dmrrx.Receiver
	cc *tier3.ControlChannel
}

// dmrPipeline holds dmr import alive for the package-level
// import-grouping rule; the underlying Receiver type is in dmrrx.
var _ = dmr.BurstDibits

func (p *dmrPipeline) Process(iq []complex64) { p.rx.Process(iq) }
func (p *dmrPipeline) Reset()                 { p.rx.Reset() }
func (p *dmrPipeline) Close() error           { return nil }

// TopologySnapshot surfaces the DMR Tier III system topology (identity +
// adjacent sites) the control channel accumulated, mapped to the neutral
// trunking shape so the signal-lab engine attaches it to the decode Result.
func (p *dmrPipeline) TopologySnapshot() *trunking.TopologySnapshot {
	t := p.cc.Topology()
	snap := &trunking.TopologySnapshot{
		SystemID:  uint32(t.SystemID),
		RFSS:      t.RFSS,
		Site:      t.Site,
		ColorCode: t.ColorCode,
	}
	for _, n := range t.Neighbors {
		ref := trunking.TopoNeighborRef{
			Site:          uint8(n.SiteID),
			ChannelNumber: n.LCN,
		}
		if hz, ok := p.cc.NeighborFrequency(n.LCN); ok {
			ref.FrequencyHz = hz
		}
		snap.Neighbors = append(snap.Neighbors, ref)
	}
	return snap
}

// newDMRTier2Pipeline wires internal/radio/dmr/receiver into
// dmr/tier2.ConventionalChannel.Process. DMR Tier II is conventional
// (per-repeater) rather than trunked — there's no dedicated control
// channel. The pipeline still slots into the trunked-decoder model
// because the ConventionalChannel state machine emits a `protocol =
// "dmr-tier2"` grant on every Voice LC Header burst (deduped per
// call, cleared on Terminator-with-LC) plus cc.locked on the first
// valid slot-type decode, so the engine + recorder + composer don't
// need to know the protocol is conventional.
//
// Receiver chain is identical to Tier III: C4FM dibits via the
// shared dmr/receiver. Differences live in the per-protocol state
// machine — Tier II doesn't read CSBK, it reads Voice LC Headers
// (BPTC(196,96) + RS(12,9,4) parity check) for call setup.
func newDMRTier2Pipeline(opts PipelineOptions) (ProtocolPipeline, error) {
	cc := tier2.New(tier2.Options{
		Bus:              opts.Bus,
		Log:              opts.Log,
		SystemName:       opts.SystemName,
		FrequencyHz:      opts.FrequencyHz,
		ColorCodeFilter:  opts.System.DMRColorCode,
		InterleavedVoice: opts.System.DMRInterleavedVoice,
	})
	rx := dmrrx.New(dmrrx.Options{
		SampleRateHz: opts.SampleRateHz,
		DeviationHz:  1944.0,
		// ClockGain lowered to 0.015 vs Tier III's 0.025 because Tier
		// II Voice LC Header bursts have a higher per-symbol
		// transition magnitude than Tier III's CSBK Aloha bursts
		// (1.27 vs 0.90, see TestDMRTier2VsTier3SymbolDensity in
		// cmd/gophertrunk/dmr_tier2_diagnostic_test.go). The RS(12, 9)
		// seed 0x96 0x96 0x96 and the BPTC(196, 96) parity rows
		// distribute high-Hamming-weight bits throughout the
		// channel-bit output; the resulting rapid-transition dibit
		// stream slips the loop at 0.025. A more conservative gain
		// converges slower but stays locked under the harder
		// symbol distribution. Live captures benefit equally — the
		// 0.015 value still sits well within the loop's noise
		// margin per the MM stability bound.
		ClockGain: 0.015,
		DibitSink: func(dibits []uint8, baseIdx int) {
			opts.tapDibits(dibits, baseIdx)
			cc.Process(dibits, baseIdx)
		},
	})
	return &dmrTier2Pipeline{rx: rx, cc: cc}, nil
}

type dmrTier2Pipeline struct {
	rx *dmrrx.Receiver
	cc *tier2.ConventionalChannel
}

func (p *dmrTier2Pipeline) Process(iq []complex64) { p.rx.Process(iq) }
func (p *dmrTier2Pipeline) Reset()                 { p.rx.Reset() }
func (p *dmrTier2Pipeline) Close() error           { return nil }

// newDMRTier1Pipeline wires the shared DMR receiver into the
// tier2.ConventionalChannel state machine in *direct-mode* configuration:
// the same wire format as Tier II (C4FM dibits → burst → slot-type
// Hamming → Voice LC Header BPTC(196,96) + RS(12,9,4) → grant), but the
// burst-sync detector is restricted to the four ETSI direct-mode sync
// words (DM-Voice/Data, TS1/TS2) and grants/decode-errors are tagged
// "dmr-tier1". DMR Tier I is license-free simplex (PMR446); it has no
// repeater or control channel, so a Voice LC Header marks each
// transmission start exactly as in Tier II conventional.
func newDMRTier1Pipeline(opts PipelineOptions) (ProtocolPipeline, error) {
	cc := tier2.New(tier2.Options{
		Bus:             opts.Bus,
		Log:             opts.Log,
		SystemName:      opts.SystemName,
		FrequencyHz:     opts.FrequencyHz,
		ProtocolTag:     "dmr-tier1",
		ColorCodeFilter: opts.System.DMRColorCode,
		SyncPatterns: []dmr.SyncPattern{
			dmr.DMVoice1, dmr.DMVoice2, dmr.DMData1, dmr.DMData2,
		},
	})
	rx := dmrrx.New(dmrrx.Options{
		SampleRateHz: opts.SampleRateHz,
		DeviationHz:  1944.0,
		ClockGain:    0.015, // same as Tier II (identical burst symbol statistics)
		DibitSink: func(dibits []uint8, baseIdx int) {
			opts.tapDibits(dibits, baseIdx)
			cc.Process(dibits, baseIdx)
		},
	})
	return &dmrTier1Pipeline{rx: rx, cc: cc}, nil
}

type dmrTier1Pipeline struct {
	rx *dmrrx.Receiver
	cc *tier2.ConventionalChannel
}

func (p *dmrTier1Pipeline) Process(iq []complex64) { p.rx.Process(iq) }
func (p *dmrTier1Pipeline) Reset()                 { p.rx.Reset() }
func (p *dmrTier1Pipeline) Close() error           { return nil }

// newNXDNPipeline wires internal/radio/nxdn/receiver into
// nxdn.ControlChannel.Process. The receiver's DibitSink forwards
// dibits into the state machine, which detects the outbound 8-dibit
// FSW, parses the LICH from the next 16 wire bits, and pulls the
// first 44 dibits of the Info field as raw CAC bits. The CAC FEC
// layer (K=5 ½-rate Viterbi + interleaver + puncture) is a
// follow-up; until it ships the adapter will sync on FSW + LICH
// but typically fail the CAC CRC on real on-air signals.
func newNXDNPipeline(opts PipelineOptions) (ProtocolPipeline, error) {
	cc := nxdn.NewControlChannel(opts.Bus, opts.Log, opts.FrequencyHz, nxdn.Rate9600)
	viterbiMode, ok := nxdn.ParseViterbiMode(opts.System.NXDNViterbiMode)
	if !ok {
		opts.Log.Warn("ccdecoder: unrecognised nxdn_viterbi_mode; falling back to spec",
			"system", opts.SystemName, "value", opts.System.NXDNViterbiMode)
	}
	cc.SetViterbiMode(viterbiMode)
	// Wire the system name + band plan so VCALL_ASSGN voice grants
	// resolve a traffic-channel number to a downlink frequency and the
	// engine follows the call. With no band plan configured, grants are
	// dropped (logged) — the control channel still locks and decodes.
	cc.SetSystemName(opts.SystemName)
	cc.SetBandPlan(nxdn.ResolverFromPlan(opts.System.NXDNBandPlan))
	// NXDN spec peak deviation per the Common Air Interface (same
	// value P25 Phase 1 uses). Calibrates the slicer thresholds
	// against the FM-discriminator output level so live captures
	// slice correctly out of the box. Per-system override via
	// nxdn_deviation_hz for transmitters that deviate from spec —
	// see samples/nxdn/README.md.
	deviationHz := 1800.0
	if opts.System.NXDNDeviationHz > 0 {
		deviationHz = opts.System.NXDNDeviationHz
	}
	rx := nxdnrx.New(nxdnrx.Options{
		SampleRateHz: opts.SampleRateHz,
		DeviationHz:  deviationHz,
		DibitSink: func(dibits []uint8, baseIdx int) {
			opts.tapDibits(dibits, baseIdx)
			cc.Process(dibits, baseIdx)
		},
	})
	return &nxdnPipeline{rx: rx, cc: cc, deviationHz: deviationHz}, nil
}

type nxdnPipeline struct {
	rx          *nxdnrx.Receiver
	cc          *nxdn.ControlChannel
	deviationHz float64
}

func (p *nxdnPipeline) Process(iq []complex64) { p.rx.Process(iq) }
func (p *nxdnPipeline) Reset()                 { p.rx.Reset() }
func (p *nxdnPipeline) Close() error           { return nil }

// TopologySnapshot surfaces the NXDN single-site identity (System/Site/Location)
// the control channel accumulated. No adjacent sites for NXDN.
func (p *nxdnPipeline) TopologySnapshot() *trunking.TopologySnapshot {
	t := p.cc.Topology()
	return &trunking.TopologySnapshot{
		SystemID:     uint32(t.SystemID),
		Site:         uint8(t.SiteID),
		LocationArea: t.LocationID,
	}
}

// newEDACSPipeline wires internal/radio/edacs/receiver into
// edacs.ControlChannel.Process. The receiver's BitSink forwards
// bits + baseIdx into the state machine (24-bit sync detect →
// 40-bit CCW slice → CCWFromBits → Ingest). The per-CCW BCH(40,
// 28, 2) FEC layer flips on via edacs_bch_mode: on in the
// system's YAML; BCH is the only on-wire FEC layer on the
// Standard EDACS CCW per the lwvmobile/edacs-fm reference.
func newEDACSPipeline(opts PipelineOptions) (ProtocolPipeline, error) {
	cc := edacs.New(edacs.Options{
		Bus:         opts.Bus,
		Log:         opts.Log,
		SystemName:  opts.SystemName,
		FrequencyHz: opts.FrequencyHz,
	})
	bchMode, ok := edacs.ParseBCHMode(opts.System.EDACSBCHMode)
	if !ok {
		opts.Log.Warn("ccdecoder: unrecognised edacs_bch_mode; falling back to on",
			"system", opts.SystemName, "value", opts.System.EDACSBCHMode)
	}
	cc.SetBCHMode(bchMode)
	rx := edacsrx.New(edacsrx.Options{
		SampleRateHz: opts.SampleRateHz,
		BitSink: func(bits []byte, baseIdx int) {
			opts.tapBits(bits, baseIdx)
			cc.Process(bits, baseIdx)
		},
	})
	return &edacsPipeline{rx: rx, cc: cc}, nil
}

type edacsPipeline struct {
	rx *edacsrx.Receiver
	cc *edacs.ControlChannel
}

func (p *edacsPipeline) Process(iq []complex64) { p.rx.Process(iq) }
func (p *edacsPipeline) Reset()                 { p.rx.Reset() }
func (p *edacsPipeline) Close() error           { return nil }

// TopologySnapshot surfaces the EDACS system identity + adjacent sites the
// control channel accumulated. EDACS has no RFSS; only SystemID + neighbors.
func (p *edacsPipeline) TopologySnapshot() *trunking.TopologySnapshot {
	t := p.cc.Topology()
	snap := &trunking.TopologySnapshot{SystemID: uint32(t.SystemID)}
	for _, n := range t.Neighbors {
		ref := trunking.TopoNeighborRef{
			Site:          uint8(n.SiteID),
			ChannelNumber: uint16(n.LCN),
		}
		if hz, ok := p.cc.NeighborFrequency(n.LCN); ok {
			ref.FrequencyHz = hz
		}
		snap.Neighbors = append(snap.Neighbors, ref)
	}
	return snap
}

// newMotorolaPipeline wires internal/radio/motorola/receiver into
// motorola.ControlChannel.Process. The receiver's BitSink forwards
// bits + baseIdx into the state machine (24-bit sync detect →
// 32-bit OSW slice → OSWFromBits → Ingest).
//
// The BCH(64, 16, 11) FEC layer is gated on per-system config:
// trunking.System.MotorolaBCHMode (the `motorola_bch_mode` YAML
// key) flips SetBCHMode on the ControlChannel before any sample
// flows. Empty string preserves the legacy 32-bit raw-OSW path so
// existing synthesized-fixture tests stay green; live Motorola
// Type II captures typically need `motorola_bch_mode: on` to pass
// the FEC layer. Unknown values warn-log and fall back to off
// rather than failing the retune.
func newMotorolaPipeline(opts PipelineOptions) (ProtocolPipeline, error) {
	cc := motorola.New(motorola.Options{
		Bus:         opts.Bus,
		Log:         opts.Log,
		SystemName:  opts.SystemName,
		FrequencyHz: opts.FrequencyHz,
	})
	bchMode, ok := motorola.ParseBCHMode(opts.System.MotorolaBCHMode)
	if !ok {
		opts.Log.Warn("ccdecoder: unrecognised motorola_bch_mode; falling back to on",
			"system", opts.SystemName, "value", opts.System.MotorolaBCHMode)
	}
	cc.SetBCHMode(bchMode)
	rx := motorolarx.New(motorolarx.Options{
		SampleRateHz: opts.SampleRateHz,
		BitSink: func(bits []byte, baseIdx int) {
			opts.tapBits(bits, baseIdx)
			cc.Process(bits, baseIdx)
		},
	})
	return &motorolaPipeline{rx: rx, cc: cc}, nil
}

type motorolaPipeline struct {
	rx *motorolarx.Receiver
	cc *motorola.ControlChannel
}

func (p *motorolaPipeline) Process(iq []complex64) { p.rx.Process(iq) }
func (p *motorolaPipeline) Reset()                 { p.rx.Reset() }
func (p *motorolaPipeline) Close() error           { return nil }

// TopologySnapshot surfaces the Motorola system identity + adjacent sites the
// control channel accumulated. Motorola has no RFSS; only SystemID + neighbors.
func (p *motorolaPipeline) TopologySnapshot() *trunking.TopologySnapshot {
	t := p.cc.Topology()
	snap := &trunking.TopologySnapshot{SystemID: uint32(t.SystemID)}
	for _, n := range t.Neighbors {
		ref := trunking.TopoNeighborRef{
			Site:          uint8(n.SiteID),
			ChannelNumber: n.LCN,
		}
		if hz, ok := p.cc.NeighborFrequency(n.LCN); ok {
			ref.FrequencyHz = hz
		}
		snap.Neighbors = append(snap.Neighbors, ref)
	}
	return snap
}

// newLTRPipeline wires internal/radio/ltr/receiver into
// ltr.ControlChannel.Process. The receiver's BitSink forwards
// sub-audible bits into the state machine, which slides a 41-bit
// window across the stream, commits to the first Sync=1 alignment
// it finds, and dispatches each Status word into the existing
// Ingest path.
//
// FCS verification + Manchester decoding are gated on per-system
// config: trunking.System.LTRFCSMode and LTRManchesterMode (the
// `ltr_fcs_mode` + `ltr_manchester_mode` YAML keys) flip the
// corresponding modes on the ControlChannel before any sample
// flows. Empty strings preserve the legacy raw-NRZ + no-CRC path
// so existing synthesized-fixture tests stay green; live captures
// of sub-audible LTR signaling typically need
// `ltr_manchester_mode: soft` + `ltr_fcs_mode: on`. Unknown values
// warn-log and fall back to the off / NRZ default rather than
// failing the retune.
func newLTRPipeline(opts PipelineOptions) (ProtocolPipeline, error) {
	cc := ltr.New(ltr.Options{
		Bus:         opts.Bus,
		Log:         opts.Log,
		SystemName:  opts.SystemName,
		FrequencyHz: opts.FrequencyHz,
	})
	fcsMode, fcsOK := ltr.ParseFCSMode(opts.System.LTRFCSMode)
	if !fcsOK {
		opts.Log.Warn("ccdecoder: unrecognised ltr_fcs_mode; falling back to on",
			"system", opts.SystemName, "value", opts.System.LTRFCSMode)
	}
	cc.SetFCSMode(fcsMode)
	manchesterMode, manchesterOK := ltr.ParseManchesterMode(opts.System.LTRManchesterMode)
	if !manchesterOK {
		opts.Log.Warn("ccdecoder: unrecognised ltr_manchester_mode; falling back to soft",
			"system", opts.SystemName, "value", opts.System.LTRManchesterMode)
	}
	cc.SetManchesterMode(manchesterMode)
	rx := ltrrx.New(ltrrx.Options{
		SampleRateHz: opts.SampleRateHz,
		BitSink: func(bits []byte, baseIdx int) {
			opts.tapBits(bits, baseIdx)
			cc.Process(bits, baseIdx)
		},
	})
	return &ltrPipeline{rx: rx, cc: cc}, nil
}

type ltrPipeline struct {
	rx *ltrrx.Receiver
	cc *ltr.ControlChannel
}

func (p *ltrPipeline) Process(iq []complex64) { p.rx.Process(iq) }
func (p *ltrPipeline) Reset()                 { p.rx.Reset() }
func (p *ltrPipeline) Close() error           { return nil }

// newMPT1327Pipeline wires internal/radio/mpt1327/receiver into
// mpt1327.ControlChannel.Process. The receiver's BitSink forwards
// FFSK bits into the state machine, which slides a 38-bit window
// over the stream + commits to the first window that parses as a
// recognised Address codeword + follows the alignment with an
// auto-unlock on extended runs of unrecognised codewords. The
// 64-bit on-air codeword's BCH(63,38) FEC + de-interleaving are
// follow-ups; without them the adapter works on noise-free test
// fixtures but typically fails on captured MPT 1327 traffic.
// mpt1327ProdMinConfirm is the production confirmation threshold for an MPT 1327
// lock: how many recognised Address codewords must arrive before the control
// channel publishes cc.locked. 2 removes single-codeword false locks at
// negligible latency on a continuously-broadcasting real control channel.
const mpt1327ProdMinConfirm = 2

func newMPT1327Pipeline(opts PipelineOptions) (ProtocolPipeline, error) {
	cc := mpt1327.New(mpt1327.Options{
		Bus:         opts.Bus,
		Log:         opts.Log,
		SystemName:  opts.SystemName,
		FrequencyHz: opts.FrequencyHz,
	})
	bchMode, ok := mpt1327.ParseBCHMode(opts.System.MPT1327BCHMode)
	if !ok {
		opts.Log.Warn("ccdecoder: unrecognised mpt1327_bch_mode; falling back to on",
			"system", opts.SystemName, "value", opts.System.MPT1327BCHMode)
	}
	cc.SetBCHMode(bchMode)
	cwscTol, ok := mpt1327.ParseCWSCTolerance(opts.System.MPT1327CWSCTolerance)
	if !ok {
		opts.Log.Warn("ccdecoder: unrecognised mpt1327_cwsc_tolerance; falling back to default",
			"system", opts.SystemName, "value", opts.System.MPT1327CWSCTolerance)
	}
	cc.SetCWSCTolerance(cwscTol)
	// Require a couple of recognised codewords before an MPT 1327 lock: a real
	// control channel streams them continuously, so this is near-instant on a
	// genuine CC but stops a single cross-protocol false parse (e.g. an
	// off-channel P25/DMR carrier handed to the identifier) from locking MPT.
	cc.SetMinConfirm(mpt1327ProdMinConfirm)
	rx := mpt1327rx.New(mpt1327rx.Options{
		SampleRateHz: opts.SampleRateHz,
		BitSink: func(bits []byte, baseIdx int) {
			opts.tapBits(bits, baseIdx)
			cc.Process(bits, baseIdx)
		},
	})
	return &mpt1327Pipeline{rx: rx, cc: cc}, nil
}

type mpt1327Pipeline struct {
	rx *mpt1327rx.Receiver
	cc *mpt1327.ControlChannel
}

func (p *mpt1327Pipeline) Process(iq []complex64) { p.rx.Process(iq) }
func (p *mpt1327Pipeline) Reset()                 { p.rx.Reset() }
func (p *mpt1327Pipeline) Close() error           { return nil }

// newDStarPipeline wires internal/radio/dstar/receiver into
// dstar.ControlChannel.Process. D-STAR is the JARL DV-mode amateur
// digital voice + data protocol — GMSK at 4800 bps with BT = 0.5,
// same 2-level shape as EDACS but at half the symbol rate.
//
// D-STAR isn't trunked in the cellular sense: each repeater is its
// own conventional channel and there's no separate control channel
// granting traffic onto a different frequency. The pipeline still
// fits the trunked connector model because the ControlChannel state
// machine treats each PCH header as a synthetic grant on the same
// tuned frequency, so the engine + recorder + composer don't need to
// know D-STAR is conventional.
//
// The convolutional rate-1/2 inner FEC + scrambler + interleaver
// the on-air PCH carries are documented follow-ups; this pipeline
// works on synthesized fixtures and pre-FEC-stripped inputs.
func newDStarPipeline(opts PipelineOptions) (ProtocolPipeline, error) {
	cc := dstar.New(dstar.Options{
		Bus:         opts.Bus,
		Log:         opts.Log,
		SystemName:  opts.SystemName,
		FrequencyHz: opts.FrequencyHz,
	})
	fecMode, fecOK := dstar.ParseFECMode(opts.System.DStarFECMode)
	if !fecOK {
		opts.Log.Warn("ccdecoder: unrecognised dstar_fec_mode; falling back to off",
			"system", opts.SystemName, "value", opts.System.DStarFECMode)
	}
	cc.SetFECMode(fecMode)
	rx := dstarrx.New(dstarrx.Options{
		SampleRateHz: opts.SampleRateHz,
		BitSink: func(bits []byte, baseIdx int) {
			opts.tapBits(bits, baseIdx)
			cc.Process(bits, baseIdx)
		},
	})
	return &dstarPipeline{rx: rx, cc: cc}, nil
}

type dstarPipeline struct {
	rx *dstarrx.Receiver
	cc *dstar.ControlChannel
}

func (p *dstarPipeline) Process(iq []complex64) { p.rx.Process(iq) }
func (p *dstarPipeline) Reset()                 { p.rx.Reset() }
func (p *dstarPipeline) Close() error           { return nil }
