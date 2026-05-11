package ccdecoder

import (
	"log/slog"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dpmr"
	dpmrrx "github.com/MattCheramie/GopherTrunk/internal/radio/dpmr/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/radio/edacs"
	edacsrx "github.com/MattCheramie/GopherTrunk/internal/radio/edacs/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/radio/motorola"
	motorolarx "github.com/MattCheramie/GopherTrunk/internal/radio/motorola/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
	nxdnrx "github.com/MattCheramie/GopherTrunk/internal/radio/nxdn/receiver"
	p25phase1 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	p25phase1rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1/receiver"
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
type PipelineOptions struct {
	Bus          *events.Bus
	Log          *slog.Logger
	SystemName   string
	FrequencyHz  uint32
	SampleRateHz float64
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
	trunking.ProtocolP25:      newP25Phase1Pipeline,
	trunking.ProtocolDPMR:     newDPMRPipeline,
	trunking.ProtocolNXDN:     newNXDNPipeline,
	trunking.ProtocolEDACS:    newEDACSPipeline,
	trunking.ProtocolMotorola: newMotorolaPipeline,
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
// 40-bit CCW slice → CCWFromBits → Ingest). The interleaved
// Reed-Solomon-derived FEC over the CCW is a follow-up; until
// it lands the adapter sync-locks but typically fails CCW
// parsing on noisy on-air signals.
func newEDACSPipeline(opts PipelineOptions) (ProtocolPipeline, error) {
	cc := edacs.New(edacs.Options{
		Bus:         opts.Bus,
		Log:         opts.Log,
		SystemName:  opts.SystemName,
		FrequencyHz: opts.FrequencyHz,
	})
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
