package ccdecoder

import (
	"log/slog"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	dmrrx "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr/tier3"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dpmr"
	dpmrrx "github.com/MattCheramie/GopherTrunk/internal/radio/dpmr/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/radio/edacs"
	edacsrx "github.com/MattCheramie/GopherTrunk/internal/radio/edacs/receiver"
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
}

// newP25Phase1Pipeline wires the existing
// internal/radio/p25/phase1/receiver into
// phase1.ControlChannel.Process. The receiver's DibitSink
// forwards dibits + baseIdx straight into the state machine,
// which publishes events.KindCCLocked + events.KindGrant on the
// bus when the supervisor's tuned frequency carries valid P25
// traffic.
func newP25Phase1Pipeline(opts PipelineOptions) (ProtocolPipeline, error) {
	cc := p25phase1.New(p25phase1.Options{
		Bus:         opts.Bus,
		Log:         opts.Log,
		SystemName:  opts.SystemName,
		FrequencyHz: opts.FrequencyHz,
	})
	rx := p25phase1rx.New(p25phase1rx.Options{
		SampleRateHz: opts.SampleRateHz,
		DibitSink: func(dibits []uint8, baseIdx int) {
			cc.Process(dibits, baseIdx)
		},
	})
	return &p25Phase1Pipeline{rx: rx, cc: cc}, nil
}

type p25Phase1Pipeline struct {
	rx *p25phase1rx.Receiver
	cc *p25phase1.ControlChannel
}

func (p *p25Phase1Pipeline) Process(iq []complex64) { p.rx.Process(iq) }
func (p *p25Phase1Pipeline) Reset()                  { p.rx.Reset() }
func (p *p25Phase1Pipeline) Close() error            { return nil }

// newP25Phase2Pipeline wires internal/radio/p25/phase2/receiver into
// p25phase2.ControlChannel.Process. The receiver's DibitSink forwards
// H-DQPSK dibits into the state machine (20-dibit outbound sync
// detect → 72-dibit MAC PDU slice → ParseMACPDU → Ingest), which
// publishes cc.locked on the first non-idle MAC PDU and grants on
// GroupVoiceChannelGrant variants. Trellis FEC + slot-type
// extraction across the full 180-dibit subframe are follow-ups.
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
	if v := opts.System.P25Phase2TrellisMode; v != "" {
		mode, ok := p25phase2.ParseTrellisMode(v)
		if !ok {
			opts.Log.Warn("ccdecoder: unrecognised p25_phase2_trellis_mode; falling back to off",
				"system", opts.SystemName, "value", v)
		}
		cc.SetTrellisMode(mode)
	}
	rx := p25phase2rx.New(p25phase2rx.Options{
		SampleRateHz: opts.SampleRateHz,
		DibitSink: func(dibits []uint8, baseIdx int) {
			cc.Process(dibits, baseIdx)
		},
		ClockMode: p25phase2rx.ClockGardner,
	})
	return &p25Phase2Pipeline{rx: rx, cc: cc}, nil
}

type p25Phase2Pipeline struct {
	rx *p25phase2rx.Receiver
	cc *p25phase2.ControlChannel
}

func (p *p25Phase2Pipeline) Process(iq []complex64) { p.rx.Process(iq) }
func (p *p25Phase2Pipeline) Reset()                  { p.rx.Reset() }
func (p *p25Phase2Pipeline) Close() error            { return nil }

// newTETRAPipeline wires internal/radio/tetra/receiver into
// tetra.ControlChannel.Process. The receiver's DibitSink forwards
// π/4-DQPSK dibits into the state machine.
//
// When the supplied trunking.System carries a non-zero TETRAColourCode,
// the factory flips the CC into ChannelCodingOn — slicing per the
// configured TETRAChannel (default ChannelSCHHD) and running the full
// ETSI EN 300 392-2 §8.3.1 type-5 → type-1 decode chain (descramble +
// deinterleave + depuncture + Viterbi + CRC-16 verify + tail strip)
// per burst. Leaving TETRAColourCode at zero keeps the legacy
// ChannelCodingOff raw-dibit path (38-dibit normal training-sequence
// detect → 48-dibit PDU slice → ParsePDU → Ingest), which still
// works on synthesized fixtures but won't lock on live FEC-encoded
// captures.
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
	})
	if opts.System.TETRAColourCode != 0 {
		ch, ok := tetra.ParseChannelType(opts.System.TETRAChannel)
		if !ok {
			opts.Log.Warn("ccdecoder: unrecognised tetra_channel; falling back to SCH/HD",
				"system", opts.SystemName, "value", opts.System.TETRAChannel)
		}
		cc.SetChannelCoding(tetra.ChannelCodingOn)
		cc.SetExpectedChannel(ch)
		cc.SetColourCode(opts.System.TETRAColourCode)
	}
	rx := tetrarx.New(tetrarx.Options{
		SampleRateHz: opts.SampleRateHz,
		DibitSink: func(dibits []uint8, baseIdx int) {
			cc.Process(dibits, baseIdx)
		},
		ClockMode: tetrarx.ClockGardner,
	})
	return &tetraPipeline{rx: rx, cc: cc}, nil
}

type tetraPipeline struct {
	rx *tetrarx.Receiver
	cc *tetra.ControlChannel
}

func (p *tetraPipeline) Process(iq []complex64) { p.rx.Process(iq) }
func (p *tetraPipeline) Reset()                  { p.rx.Reset() }
func (p *tetraPipeline) Close() error            { return nil }

// newYSFPipeline wires the existing internal/radio/ysf/receiver
// into ysf.ControlChannel.Process. YSF lacks a published
// trunking.Protocol enum value today (the config layer expects
// "p25" / "dmr" / "nxdn"); the factory is exposed for direct use
// by the daemon + tests rather than via the factory map. Once the
// Protocol enum gains a YSF entry the factory map gains a row.
func newYSFPipeline(opts PipelineOptions) (ProtocolPipeline, error) {
	cc := ysf.New(ysf.Options{
		Bus:         opts.Bus,
		Log:         opts.Log,
		SystemName:  opts.SystemName,
		FrequencyHz: opts.FrequencyHz,
	})
	rx := ysfrx.New(ysfrx.Options{
		SampleRateHz: opts.SampleRateHz,
		DibitSink: func(dibits []uint8, baseIdx int) {
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
func (p *ysfPipeline) Reset()                  { p.rx.Reset() }
func (p *ysfPipeline) Close() error            { return nil }

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
		DibitSink: func(dibits []uint8, baseIdx int) {
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
func (p *dpmrPipeline) Reset()                  { p.rx.Reset() }
func (p *dpmrPipeline) Close() error            { return nil }

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
	})
	rx := dmrrx.New(dmrrx.Options{
		SampleRateHz: opts.SampleRateHz,
		DibitSink: func(dibits []uint8, baseIdx int) {
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
func (p *dmrPipeline) Reset()                  { p.rx.Reset() }
func (p *dmrPipeline) Close() error            { return nil }

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
	if v := opts.System.NXDNViterbiMode; v != "" {
		mode, ok := nxdn.ParseViterbiMode(v)
		if !ok {
			opts.Log.Warn("ccdecoder: unrecognised nxdn_viterbi_mode; falling back to off",
				"system", opts.SystemName, "value", v)
		}
		cc.SetViterbiMode(mode)
	}
	rx := nxdnrx.New(nxdnrx.Options{
		SampleRateHz: opts.SampleRateHz,
		DibitSink: func(dibits []uint8, baseIdx int) {
			cc.Process(dibits, baseIdx)
		},
	})
	return &nxdnPipeline{rx: rx, cc: cc}, nil
}

type nxdnPipeline struct {
	rx *nxdnrx.Receiver
	cc *nxdn.ControlChannel
}

func (p *nxdnPipeline) Process(iq []complex64) { p.rx.Process(iq) }
func (p *nxdnPipeline) Reset()                  { p.rx.Reset() }
func (p *nxdnPipeline) Close() error            { return nil }

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
	if v := opts.System.EDACSBCHMode; v != "" {
		mode, ok := edacs.ParseBCHMode(v)
		if !ok {
			opts.Log.Warn("ccdecoder: unrecognised edacs_bch_mode; falling back to off",
				"system", opts.SystemName, "value", v)
		}
		cc.SetBCHMode(mode)
	}
	rx := edacsrx.New(edacsrx.Options{
		SampleRateHz: opts.SampleRateHz,
		BitSink: func(bits []byte, baseIdx int) {
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
func (p *edacsPipeline) Reset()                  { p.rx.Reset() }
func (p *edacsPipeline) Close() error            { return nil }

// newMotorolaPipeline wires internal/radio/motorola/receiver into
// motorola.ControlChannel.Process. The receiver's BitSink forwards
// bits + baseIdx into the state machine (24-bit sync detect →
// 32-bit OSW slice → OSWFromBits → Ingest). The BCH(64,16,11)
// FEC over the OSW is a follow-up; until it lands the adapter
// sync-locks but typically fails OSW parsing on noisy signals.
func newMotorolaPipeline(opts PipelineOptions) (ProtocolPipeline, error) {
	cc := motorola.New(motorola.Options{
		Bus:         opts.Bus,
		Log:         opts.Log,
		SystemName:  opts.SystemName,
		FrequencyHz: opts.FrequencyHz,
	})
	rx := motorolarx.New(motorolarx.Options{
		SampleRateHz: opts.SampleRateHz,
		BitSink: func(bits []byte, baseIdx int) {
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
func (p *motorolaPipeline) Reset()                  { p.rx.Reset() }
func (p *motorolaPipeline) Close() error            { return nil }

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
	if v := opts.System.LTRFCSMode; v != "" {
		mode, ok := ltr.ParseFCSMode(v)
		if !ok {
			opts.Log.Warn("ccdecoder: unrecognised ltr_fcs_mode; falling back to off",
				"system", opts.SystemName, "value", v)
		}
		cc.SetFCSMode(mode)
	}
	if v := opts.System.LTRManchesterMode; v != "" {
		mode, ok := ltr.ParseManchesterMode(v)
		if !ok {
			opts.Log.Warn("ccdecoder: unrecognised ltr_manchester_mode; falling back to off",
				"system", opts.SystemName, "value", v)
		}
		cc.SetManchesterMode(mode)
	}
	rx := ltrrx.New(ltrrx.Options{
		SampleRateHz: opts.SampleRateHz,
		BitSink: func(bits []byte, baseIdx int) {
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
func (p *ltrPipeline) Reset()                  { p.rx.Reset() }
func (p *ltrPipeline) Close() error            { return nil }

// newMPT1327Pipeline wires internal/radio/mpt1327/receiver into
// mpt1327.ControlChannel.Process. The receiver's BitSink forwards
// FFSK bits into the state machine, which slides a 38-bit window
// over the stream + commits to the first window that parses as a
// recognised Address codeword + follows the alignment with an
// auto-unlock on extended runs of unrecognised codewords. The
// 64-bit on-air codeword's BCH(63,38) FEC + de-interleaving are
// follow-ups; without them the adapter works on noise-free test
// fixtures but typically fails on captured MPT 1327 traffic.
func newMPT1327Pipeline(opts PipelineOptions) (ProtocolPipeline, error) {
	cc := mpt1327.New(mpt1327.Options{
		Bus:         opts.Bus,
		Log:         opts.Log,
		SystemName:  opts.SystemName,
		FrequencyHz: opts.FrequencyHz,
	})
	if v := opts.System.MPT1327BCHMode; v != "" {
		mode, ok := mpt1327.ParseBCHMode(v)
		if !ok {
			opts.Log.Warn("ccdecoder: unrecognised mpt1327_bch_mode; falling back to off",
				"system", opts.SystemName, "value", v)
		}
		cc.SetBCHMode(mode)
	}
	rx := mpt1327rx.New(mpt1327rx.Options{
		SampleRateHz: opts.SampleRateHz,
		BitSink: func(bits []byte, baseIdx int) {
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
func (p *mpt1327Pipeline) Reset()                  { p.rx.Reset() }
func (p *mpt1327Pipeline) Close() error            { return nil }
