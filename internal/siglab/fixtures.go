package siglab

import (
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr/tier3"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	p25phase1 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// fixture describes how to synthesize a known-good locking capture for one
// protocol: a symbol-stream builder, the modulator that shapes it to IQ, the
// capture sample rate, the decoder knobs needed to decode it, and the
// acceptance criteria a correct decode must satisfy. The `gen` subcommand
// uses these to emit a capture + metadata sidecar that the `test` harness
// then validates — a closed synthesize→decode→grade loop.
//
// The symbol-stream builders are lifted from the per-protocol
// integration_cc_*_test.go fixtures so synthesis exercises exactly the
// streams the production decoders are tested against. Protocols not yet in
// the registry are reported by Fixtures(); porting their builders is a
// mechanical follow-up (each lives in its integration_cc_<proto>_test.go).
type fixture struct {
	build       func() []uint8
	modulate    func(symbols []uint8, sampleRateHz float64) []complex64
	sampleRate  float64
	systemKnobs map[string]string
	expected    Acceptance
}

func boolPtr(b bool) *bool { return &b }

// fixtures is the synthesis registry keyed by protocol.
var fixtures = map[trunking.Protocol]fixture{
	trunking.ProtocolP25: {
		build:      func() []uint8 { return buildP25LockedDibits(0x293, 40) },
		modulate:   func(d []uint8, sr float64) []complex64 { return demod.ModulateP25C4FM(d, sr, 1800.0) },
		sampleRate: 48_000,
		expected: Acceptance{
			Lock:             boolPtr(true),
			LockFields:       map[string]any{"nac": "0x293"},
			BaudTolerancePct: 5,
		},
	},
	trunking.ProtocolDMR: {
		build:      func() []uint8 { return buildDMRTier3CSBKDibits(80, 0xA, 0x1234) },
		modulate:   func(d []uint8, sr float64) []complex64 { return demod.ModulateC4FM(d, 10, 8, 0.20, sr, 1944.0) },
		sampleRate: 48_000,
		expected: Acceptance{
			Lock:             boolPtr(true),
			LockFields:       map[string]any{"color_code": 0xA},
			BaudTolerancePct: 5,
		},
	},
}

// Fixtures returns the protocols with a registered synthesis fixture, in
// stable order, for the `gen` usage message and protocol picker.
func Fixtures() []trunking.Protocol {
	out := make([]trunking.Protocol, 0, len(fixtures))
	for p := range fixtures {
		out = append(out, p)
	}
	// Reuse the ccdecoder ordering convention (ascending Protocol value).
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// HasFixture reports whether a synthesis fixture is registered for p.
func HasFixture(p trunking.Protocol) bool {
	_, ok := fixtures[p]
	return ok
}

// --- Lifted builders (from cmd/gophertrunk/integration_cc_*_test.go) ---

// buildP25LockedDibits builds a P25 Phase 1 control-channel dibit stream:
// a warmup ramp, then repeated FSW + NID(TSDU) + RFSS-status-broadcast TSBK
// frames with on-air status symbols injected. Lifted verbatim from
// integration_cc_test.go's buildP25LockedIQDibits.
func buildP25LockedDibits(nac uint16, repeats int) []uint8 {
	frame := make([]uint8, 0, 24+32+98)
	frame = append(frame, p25phase1.FrameSyncWord[:]...)
	nidBits := p25phase1.EncodeNIDBits(nac, p25phase1.DUIDTrunkingSignaling)
	for i := 0; i < 32; i++ {
		frame = append(frame, (nidBits[2*i]<<1)|nidBits[2*i+1])
	}
	tsbk := p25phase1.AssembleTSBK(p25phase1.TSBK{LB: true, Opcode: p25phase1.OpRFSSStatusBroadcast})
	frame = append(frame, p25phase1.EncodeTSBKChannel(tsbk)...)
	frame = p25phase1.InjectControlStatusSymbols(frame)

	out := make([]uint8, 0, 200+repeats*(len(frame)+50)+100)
	for i := 0; i < 200; i++ {
		out = append(out, uint8(i&3))
	}
	for r := 0; r < repeats; r++ {
		out = append(out, frame...)
		for i := 0; i < 50; i++ {
			out = append(out, uint8(i&3))
		}
	}
	for i := 0; i < 100; i++ {
		out = append(out, uint8(i&3))
	}
	return out
}

// buildDMRTier3CSBKDibits builds a DMR Tier III control-channel dibit stream
// of repeated CSBK Aloha bursts (BPTC(196,96) payload + slot-type Hamming),
// with a long warmup for the lower-gain clock loop. Lifted verbatim from
// integration_cc_dmr_test.go.
func buildDMRTier3CSBKDibits(repeats int, colorCode uint8, systemID uint16) []uint8 {
	csbk := tier3.CSBK{Opcode: tier3.OpAloha, LB: true}
	csbk.Payload[2] = byte(systemID >> 8)
	csbk.Payload[3] = byte(systemID & 0xFF)
	csbkBytes := tier3.AssembleCSBK(csbk)
	infoBits := framing.UnpackBitsMSB(csbkBytes, 96)
	channelBits := framing.EncodeBPTC196_96(infoBits)
	payloadDibits := framing.BitsToDibits(channelBits)

	slotBits := dmr.AssembleSlotType(dmr.SlotType{ColorCode: colorCode, DataType: dmr.DTCSBK})
	slotDibits := framing.BitsToDibits(slotBits)

	burst := make([]uint8, 0, dmr.BurstDibits)
	burst = append(burst, payloadDibits[:dmr.HalfPayloadDibits]...)
	burst = append(burst, slotDibits[:dmr.SlotTypeDibits]...)
	burst = append(burst, dmr.BSData.Dibits[:]...)
	burst = append(burst, slotDibits[dmr.SlotTypeDibits:]...)
	burst = append(burst, payloadDibits[dmr.HalfPayloadDibits:]...)

	out := make([]uint8, 0, 800+repeats*(len(burst)+32)+100)
	for i := 0; i < 800; i++ {
		out = append(out, uint8(i&3))
	}
	for r := 0; r < repeats; r++ {
		out = append(out, burst...)
		for i := 0; i < 32; i++ {
			out = append(out, uint8(i&3))
		}
	}
	for i := 0; i < 100; i++ {
		out = append(out, uint8(i&3))
	}
	return out
}
