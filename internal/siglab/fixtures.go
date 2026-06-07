package siglab

import (
	"encoding/binary"
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr/tier3"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dpmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dstar"
	"github.com/MattCheramie/GopherTrunk/internal/radio/edacs"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/radio/ltr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/motorola"
	"github.com/MattCheramie/GopherTrunk/internal/radio/mpt1327"
	"github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
	p25phase1 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	p25phase2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	"github.com/MattCheramie/GopherTrunk/internal/radio/ysf"
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
	trunking.ProtocolP25Phase2: {
		build:       func() []uint8 { return buildP25Phase2MACPTTStream(8) },
		modulate:    func(d []uint8, _ float64) []complex64 { return demod.ModulatePiOver4DQPSK(d, 8, 8, 0.20, math.Pi/8) },
		sampleRate:  48_000,
		systemKnobs: map[string]string{"p25_phase2_trellis_mode": "on"},
		expected:    Acceptance{Lock: boolPtr(true)},
	},
	trunking.ProtocolDMRTier2: {
		build: func() []uint8 {
			return buildDMRConventionalVoiceLCHeaderDibits(80, 0x7, 0x123, 0x456789, dmr.BSData.Dibits[:])
		},
		modulate:   func(d []uint8, sr float64) []complex64 { return demod.ModulateC4FM(d, 10, 8, 0.20, sr, 1944.0) },
		sampleRate: 48_000,
		expected: Acceptance{
			Lock:       boolPtr(true),
			LockFields: map[string]any{"color_code": 0x7},
		},
	},
	trunking.ProtocolDMRTier1: {
		build: func() []uint8 {
			return buildDMRConventionalVoiceLCHeaderDibits(80, 0x7, 0x123, 0x456789, dmr.DMVoice1.Dibits[:])
		},
		modulate:   func(d []uint8, sr float64) []complex64 { return demod.ModulateC4FM(d, 10, 8, 0.20, sr, 1944.0) },
		sampleRate: 48_000,
		expected: Acceptance{
			Lock:       boolPtr(true),
			LockFields: map[string]any{"color_code": 0x7},
		},
	},
	trunking.ProtocolNXDN: {
		build:       func() []uint8 { return buildNXDNSpecEncodedDibits(20, 0xBEEF, 0x0042) },
		modulate:    func(d []uint8, sr float64) []complex64 { return demod.ModulateC4FM(d, 10, 8, 0.20, sr, 1800.0) },
		sampleRate:  48_000,
		systemKnobs: map[string]string{"nxdn_viterbi_mode": "spec"},
		expected: Acceptance{
			Lock:       boolPtr(true),
			LockFields: map[string]any{"site_id": "0xBEEF", "system_id": "0x0042"},
		},
	},
	trunking.ProtocolDPMR: {
		build:      func() []uint8 { return buildDPMRSiteBroadcastDibits(30, 0x123456) },
		modulate:   func(d []uint8, sr float64) []complex64 { return demod.ModulateC4FM(d, 20, 8, 0.20, sr, 900.0) },
		sampleRate: 48_000,
		expected: Acceptance{
			Lock:       boolPtr(true),
			LockFields: map[string]any{"system_id": "0x123456"},
		},
	},
	trunking.ProtocolYSF: {
		build:      func() []uint8 { return buildYSFFSWStream(30) },
		modulate:   func(d []uint8, sr float64) []complex64 { return demod.ModulateC4FM(d, 10, 8, 0.20, sr, 1800.0) },
		sampleRate: 48_000,
		expected:   Acceptance{Lock: boolPtr(true)},
	},
	trunking.ProtocolTETRA: {
		build:       func() []uint8 { return buildTETRASCHHDStream(100, 0x12345, 0x42) },
		modulate:    func(d []uint8, _ float64) []complex64 { return demod.ModulatePiOver4DQPSK(d, 4, 8, 0.35, math.Pi/4) },
		sampleRate:  72_000,
		systemKnobs: map[string]string{"tetra_colour_code": "0x12345", "tetra_channel": "sch/hd"},
		expected: Acceptance{
			Lock:       boolPtr(true),
			LockFields: map[string]any{"location_area": "0x42"},
		},
	},
	trunking.ProtocolEDACS: {
		build:       func() []uint8 { return buildEDACSSystemIDStream(30, 0x7B) },
		modulate:    func(d []uint8, sr float64) []complex64 { return demod.ModulateGFSK(d, 10, 4, 0.3, sr, 2400.0) },
		sampleRate:  96_000,
		systemKnobs: map[string]string{"edacs_bch_mode": "on"},
		expected: Acceptance{
			Lock:       boolPtr(true),
			LockFields: map[string]any{"system_id": "0x7B"},
		},
	},
	trunking.ProtocolMotorola: {
		build:       func() []uint8 { return buildMotorolaSystemIDStream(30, 0x4567) },
		modulate:    func(d []uint8, sr float64) []complex64 { return demod.ModulateGFSK(d, 27, 4, 0.5, sr, 1500.0) },
		sampleRate:  97_200,
		systemKnobs: map[string]string{"motorola_bch_mode": "on"},
		expected: Acceptance{
			Lock:       boolPtr(true),
			LockFields: map[string]any{"system_id": "0x4567"},
		},
	},
	trunking.ProtocolLTR: {
		build:       func() []uint8 { return buildLTRStatusStream(80, 7, 4) },
		modulate:    func(d []uint8, sr float64) []complex64 { return demod.ModulateSubAudibleNRZ(d, sr, 300.0, 0.05) },
		sampleRate:  48_000,
		systemKnobs: map[string]string{"ltr_manchester_mode": "off", "ltr_fcs_mode": "off"},
		expected: Acceptance{
			Lock:       boolPtr(true),
			LockFields: map[string]any{"area": 7, "repeater": 4},
		},
	},
	trunking.ProtocolMPT1327: {
		build:       func() []uint8 { return buildMPT1327AlohaStream(100, 0x5) },
		modulate:    func(d []uint8, sr float64) []complex64 { return demod.ModulateFFSK(d, sr, 1200.0, 1200.0, 1800.0) },
		sampleRate:  48_000,
		systemKnobs: map[string]string{"mpt1327_bch_mode": "on"},
		expected: Acceptance{
			Lock:       boolPtr(true),
			LockFields: map[string]any{"prefix": 0x5},
		},
	},
	trunking.ProtocolDStar: {
		build: func() []uint8 {
			return buildDStarHeaderStream(30, "CQCQCQ  ", "WB7XYZ  ", "KD0AAA B", "KD0AAA G")
		},
		modulate:   func(d []uint8, sr float64) []complex64 { return demod.ModulateGFSK(d, 10, 4, 0.5, sr, 1200.0) },
		sampleRate: 48_000,
		expected:   Acceptance{Lock: boolPtr(true)},
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

// buildP25Phase2MACPTTStream builds a P25 Phase 2 superframe stream of
// MAC-signaling subframes. Lifted from integration_cc_p25p2_test.go.
func buildP25Phase2MACPTTStream(repeats int) []uint8 {
	pdu := p25phase2.MACPDU{Opcode: p25phase2.OpGroupVoiceChannelUserAbbreviated, Payload: make([]byte, 17)}
	var subs [p25phase2.SubframesPerSuperframe][]uint8
	for i := range subs {
		subs[i] = p25phase2.EncodeMACSubframe(
			p25phase2.SlotTypeMACSignaling, uint8(i), pdu,
			p25phase2.TrellisOn, p25phase2.InterleaveOff)
	}
	superframe := p25phase2.EncodeSuperframe(subs)

	out := make([]uint8, 0, 400+repeats*len(superframe)+100)
	for i := 0; i < 400; i++ {
		out = append(out, uint8(i&3))
	}
	for r := 0; r < repeats; r++ {
		out = append(out, superframe...)
	}
	for i := 0; i < 100; i++ {
		out = append(out, uint8(i&3))
	}
	return out
}

// buildDMRTier2VoiceLCHeaderDibits builds a DMR Tier II conventional stream of
// repeated Voice LC Header bursts. Lifted from integration_cc_dmr_tier2_test.go.
// buildDMRConventionalVoiceLCHeaderDibits builds a DMR conventional /
// direct-mode Voice LC Header stream. The sync word distinguishes Tier II
// (base-station, dmr.BSData) from Tier I (direct-mode, dmr.DMVoice1); the rest
// of the wire format is identical. Lifted from integration_cc_dmr_tier2_test.go.
func buildDMRConventionalVoiceLCHeaderDibits(repeats int, colorCode uint8, groupID, sourceID uint32, sync []uint8) []uint8 {
	flc := dmr.FLC{
		FLCO:    dmr.FLCOGroupVoiceUser,
		DstAddr: groupID,
		SrcAddr: sourceID,
	}
	flcBytes := dmr.AssembleFLC(flc)
	var data [9]byte
	copy(data[:], flcBytes)
	cw := framing.EncodeRS12_9(data)
	for i := 0; i < 3; i++ {
		cw[9+i] ^= framing.RS129SeedVoiceLCHeader[i]
	}
	info := cw[:]
	bits := make([]byte, 96)
	for i := 0; i < 96; i++ {
		bits[i] = (info[i>>3] >> uint(7-(i&7))) & 1
	}
	channelBits := framing.EncodeBPTC196_96(bits)
	payloadDibits := framing.BitsToDibits(channelBits)

	slotBits := dmr.AssembleSlotType(dmr.SlotType{ColorCode: colorCode, DataType: dmr.DTVoiceLCHeader})
	slotDibits := framing.BitsToDibits(slotBits)

	burst := make([]uint8, 0, dmr.BurstDibits)
	burst = append(burst, payloadDibits[:dmr.HalfPayloadDibits]...)
	burst = append(burst, slotDibits[:dmr.SlotTypeDibits]...)
	burst = append(burst, sync...)
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

// buildNXDNSpecEncodedDibits builds an NXDN RCCH outbound control-channel
// stream (FSW + LICH + spec-encoded CAC SITE_INFO). Lifted from
// integration_cc_nxdn_test.go.
func buildNXDNSpecEncodedDibits(repeats int, siteID, systemID uint16) []uint8 {
	lichInfo := nxdn.AssembleLICH(nxdn.LICH{RFCh: nxdn.RFChControl})
	lichWire := nxdn.EncodeLICHWire(lichInfo)
	lichDibits := framing.BitsToDibits(lichWire)

	var payload [8]byte
	binary.BigEndian.PutUint16(payload[0:2], 0xAAAA)
	binary.BigEndian.PutUint16(payload[2:4], siteID)
	binary.BigEndian.PutUint16(payload[4:6], systemID)
	l3 := make([]byte, 9)
	l3[0] = byte(nxdn.RCCHSITEINFO)
	copy(l3[1:9], payload[:])

	info := make([]byte, nxdn.CACInfoBits)
	l3Bits := framing.UnpackBitsMSB(l3, 72)
	copy(info[8:8+72], l3Bits)

	channel := nxdn.EncodeCACChannel(info)
	cacDibits := framing.BitsToDibits(channel)

	frame := make([]uint8, 0, 8+len(lichDibits)+len(cacDibits))
	frame = append(frame, nxdn.FSWDibitsOutbound...)
	frame = append(frame, lichDibits...)
	frame = append(frame, cacDibits...)

	out := make([]uint8, 0, 300+repeats*(len(frame)+50)+100)
	for i := 0; i < 300; i++ {
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

// buildDPMRSiteBroadcastDibits builds a dPMR control-channel stream (FS3 +
// CSBK standing-service-status). Lifted from integration_cc_dpmr_test.go.
func buildDPMRSiteBroadcastDibits(repeats int, systemID uint32) []uint8 {
	csbk := dpmr.CSBK{Type: dpmr.MsgStandingServiceStatus, DestID: systemID, Extra: 0x42}
	csbkBits := dpmr.CSBKBits(csbk)
	csbkDibits := framing.BitsToDibits(csbkBits)

	frame := make([]uint8, 0, 24+len(csbkDibits))
	frame = append(frame, dpmr.FS3Dibits()...)
	frame = append(frame, csbkDibits...)

	out := make([]uint8, 0, 400+repeats*(len(frame)+16)+100)
	for i := 0; i < 400; i++ {
		out = append(out, uint8(i&3))
	}
	for r := 0; r < repeats; r++ {
		out = append(out, frame...)
		for i := 0; i < 16; i++ {
			out = append(out, uint8(i&3))
		}
	}
	for i := 0; i < 100; i++ {
		out = append(out, uint8(i&3))
	}
	return out
}

// buildYSFFSWStream builds a YSF stream of FSW-bearing frames. Lifted from
// integration_cc_ysf_test.go.
func buildYSFFSWStream(repeats int) []uint8 {
	frame := make([]uint8, ysf.FrameDibits)
	copy(frame[ysf.FSWOffset:], ysf.FSWPattern)

	out := make([]uint8, 0, 400+repeats*len(frame)+100)
	for i := 0; i < 400; i++ {
		out = append(out, uint8(i&3))
	}
	for r := 0; r < repeats; r++ {
		out = append(out, frame...)
	}
	for i := 0; i < 100; i++ {
		out = append(out, uint8(i&3))
	}
	return out
}

// buildTETRASCHHDStream builds a TETRA SCH/HD broadcast stream: a 400-dibit
// warmup so the matched filter + Gardner loop converge, then `repeats`
// back-to-back bursts of (11-dibit NTS1 + 108-dibit SCH/HD-coded MLE SYSINFO),
// then a 100-dibit flush. The MLE SYSINFO's LocationArea is varied per burst so
// the 108-dibit payload differs frame to frame — otherwise an identical payload
// makes any chance sync-alias inside the burst recur at the exact frame cadence
// and shadow the real training sequence. Bursts are continuous (no idle gap) so
// the matched filter stays in steady state, mirroring real TETRA's continuous
// downlink and keeping the short 11-dibit NTS1 free of inter-burst ISI.
func buildTETRASCHHDStream(repeats int, colourCode uint32, locationArea uint16) []uint8 {
	sync := tetra.NormalSyncDibits()
	out := make([]uint8, 0, 400+repeats*(len(sync)+108)+100)
	for i := 0; i < 400; i++ {
		out = append(out, uint8(i&3))
	}
	for r := 0; r < repeats; r++ {
		la := locationArea + uint16(r)
		payload := []byte{0x00, 0x00, 0x00, 0, 0}
		payload[3] = byte((la >> 6) & 0xFF)
		payload[4] = byte((la & 0x3F) << 2)
		pdu := tetra.PDU{
			Disc:    tetra.DiscMLE,
			Type:    uint8(tetra.MLESystemInfo),
			Payload: payload,
		}
		info := pduToType1BitsTETRA(pdu, 124)
		type5 := tetra.EncodeSCHHD(info, colourCode)
		out = append(out, sync...)
		out = append(out, tetra.TetraBitsToDibits(type5)...)
	}
	for i := 0; i < 100; i++ {
		out = append(out, uint8(i&3))
	}
	return out
}

// pduToType1BitsTETRA pads a PDU's header + bytes into exactly k1 bits,
// MSB-first per byte, zero-padded. Lifted from integration_cc_tetra_test.go.
func pduToType1BitsTETRA(pdu tetra.PDU, k1 int) []byte {
	bytes := tetra.AssemblePDU(pdu)
	if len(bytes)*8 > k1 {
		return nil
	}
	out := make([]byte, k1)
	for i, b := range bytes {
		for j := 0; j < 8; j++ {
			out[i*8+j] = (b >> uint(7-j)) & 1
		}
	}
	return out
}

// buildEDACSSystemIDStream builds an EDACS control-channel stream (outbound
// sync + BCH-encoded SystemID CCW). Lifted from integration_cc_edacs_test.go.
func buildEDACSSystemIDStream(repeats int, systemID uint16) []byte {
	info := uint32(edacs.CmdSystemID&0xF)<<24 |
		uint32(systemID&0xFFFF)<<4
	cw := framing.BCHEncodeEDACS(info)
	wire := make([]byte, 40)
	for i := 0; i < 40; i++ {
		if cw&(uint64(1)<<uint(39-i)) != 0 {
			wire[i] = 1
		}
	}

	frame := make([]byte, 0, 24+40)
	frame = append(frame, edacs.OutboundSyncBits()...)
	frame = append(frame, wire...)

	out := make([]byte, 0, 200+repeats*(len(frame)+16)+100)
	for i := 0; i < 200; i++ {
		out = append(out, byte(i&1))
	}
	for r := 0; r < repeats; r++ {
		out = append(out, frame...)
		for i := 0; i < 16; i++ {
			out = append(out, byte(i&1))
		}
	}
	for i := 0; i < 100; i++ {
		out = append(out, byte(i&1))
	}
	return out
}

// buildMotorolaSystemIDStream builds a Motorola Type II control-channel stream
// (outbound sync + two BCH(64,16) OSW halves). Lifted from
// integration_cc_motorola_test.go.
func buildMotorolaSystemIDStream(repeats int, systemID uint16) []byte {
	command := uint16(motorola.OpSystemIDExtended) << 4

	cw1 := framing.BCHEncode64_16(systemID)
	cw2 := framing.BCHEncode64_16(command)
	encoded := make([]byte, 128)
	for i := 0; i < 64; i++ {
		if cw1&(uint64(1)<<uint(63-i)) != 0 {
			encoded[i] = 1
		}
	}
	for i := 0; i < 64; i++ {
		if cw2&(uint64(1)<<uint(63-i)) != 0 {
			encoded[64+i] = 1
		}
	}

	frame := make([]byte, 0, 24+128)
	frame = append(frame, motorola.OutboundSyncBits()...)
	frame = append(frame, encoded...)

	out := make([]byte, 0, 200+repeats*(len(frame)+16)+100)
	for i := 0; i < 200; i++ {
		out = append(out, byte(i&1))
	}
	for r := 0; r < repeats; r++ {
		out = append(out, frame...)
		for i := 0; i < 16; i++ {
			out = append(out, byte(i&1))
		}
	}
	for i := 0; i < 100; i++ {
		out = append(out, byte(i&1))
	}
	return out
}

// buildLTRStatusStream builds an LTR sub-audible Status stream. Lifted from
// integration_cc_ltr_test.go.
func buildLTRStatusStream(repeats int, area, repeater uint8) []byte {
	status := ltr.Status{
		Sync:    true,
		Area:    area,
		Channel: 3,
		Home:    repeater,
		Free:    5,
	}
	frame := ltr.StatusBits(status)

	out := make([]byte, 0, 200+repeats*len(frame)+100)
	out = append(out, make([]byte, 200)...)
	for r := 0; r < repeats; r++ {
		out = append(out, frame...)
	}
	out = append(out, make([]byte, 100)...)
	return out
}

// buildMPT1327AlohaStream builds an MPT 1327 control-channel stream of
// BCH-encoded Aloha codewords. Lifted from integration_cc_mpt1327_test.go.
func buildMPT1327AlohaStream(repeats int, prefix uint8) []byte {
	aloha := mpt1327.Codeword{
		Type:     mpt1327.TypeAddress,
		Prefix:   prefix,
		Function: uint32(mpt1327.KindAloha) << 13,
	}
	wire48 := mpt1327.CodewordBits48(aloha)
	var info48 uint64
	for i := 0; i < 48; i++ {
		if wire48[i]&1 != 0 {
			info48 |= uint64(1) << uint(i)
		}
	}
	cw := framing.BCHEncodeMPT1327(info48)
	codeword := make([]byte, 64)
	for i := 0; i < 64; i++ {
		codeword[i] = byte((cw >> uint(i)) & 1)
	}

	out := make([]byte, 0, 200+repeats*(len(codeword)+16)+100)
	for i := 0; i < 200; i++ {
		out = append(out, byte(i&1))
	}
	for r := 0; r < repeats; r++ {
		out = append(out, codeword...)
		for i := 0; i < 16; i++ {
			out = append(out, byte(i&1))
		}
	}
	for i := 0; i < 100; i++ {
		out = append(out, byte(i&1))
	}
	return out
}

// buildDStarHeaderStream builds a D-STAR header stream (frame sync + 41-byte
// header with CRC, FEC-off). Lifted from integration_cc_dstar_test.go.
func buildDStarHeaderStream(repeats int, ur, my1, rpt2, rpt1 string) []byte {
	hdr := dstar.Header{
		Flag1: 0,
		Flag2: 0,
		Flag3: 0,
		RPT2:  rpt2,
		RPT1:  rpt1,
		UR:    ur,
		MY1:   my1,
		MY2:   "SUFX",
	}
	asm := dstar.AssembleHeader(hdr)
	hdr.CRC = dstar.ComputeCRC(asm[:39])
	asm = dstar.AssembleHeader(hdr)

	headerBits := make([]byte, 0, 328)
	for _, b := range asm {
		for i := 0; i < 8; i++ {
			headerBits = append(headerBits, (b>>uint(7-i))&1)
		}
	}

	frame := make([]byte, 0, 32+328)
	frame = append(frame, dstar.FrameSyncBitsSlice()...)
	frame = append(frame, headerBits...)

	out := make([]byte, 0, 256+repeats*(len(frame)+32)+100)
	for i := 0; i < 256; i++ {
		out = append(out, 1)
	}
	for r := 0; r < repeats; r++ {
		out = append(out, frame...)
		for i := 0; i < 32; i++ {
			out = append(out, 1)
		}
	}
	for i := 0; i < 100; i++ {
		out = append(out, 1)
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
