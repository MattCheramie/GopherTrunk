package phase1

import (
	"errors"
	"fmt"
)

// IdentifierUpdate carries a P25 IDENT_UP TSBK announcement that
// defines the band plan for one channel-ID slot. A site repeats these
// periodically alongside its status broadcasts; the receiver
// accumulates them and uses each slot to translate a (Channel ID,
// Channel Number) pair from a voice grant into a downlink frequency.
//
// Three on-air variants encode the same frequency fields with
// different byte-0-low-nibble semantics; all three parse into this
// struct:
//
//	0x3D  IDEN_UP        — standard 700/800/900 MHz form (ParseIdentifierUpdate)
//	0x34  IDEN_UP_VUHF   — VHF/UHF form         (ParseIdentifierUpdateVUHF)
//	0x33  IDEN_UP_TDMA   — Phase 2 TDMA form    (ParseIdentifierUpdateTDMA)
//
// Field units after parsing (independent of variant):
//
//	BandwidthHz   — channel bandwidth in Hz (informational only;
//	                BandPlan.Frequency does not consult it)
//	SpacingHz     — channel-to-channel step in Hz
//	TxOffsetHz    — signed transmit offset (uplink = downlink + offset)
//	BaseHz        — downlink base frequency for channel 0 in Hz
//
// The TDMA variant additionally carries a slot count + access mode in
// its channel-type code (TDMA-1 / TDMA-2 / TDMA-4). The frequency
// resolver doesn't care; downstream Phase 2 voice decoding would, but
// that wiring lives outside this package. See
// internal/radio/p25/phase2/identifier.go for the Phase 2 variant
// shipped today (which is also a 0x3D-style packing).
type IdentifierUpdate struct {
	ChannelID   uint8  // 4-bit slot identifier (0..15)
	BandwidthHz uint32 // channel bandwidth
	SpacingHz   uint32 // channel-to-channel step
	TxOffsetHz  int64  // signed transmit offset (uplink = downlink + offset)
	BaseHz      uint64 // downlink base frequency for channel 0
	// AccessTDMA is true when this slot was advertised via opcode 0x33
	// (OpIdentifierUpdateTDMA) — i.e. the granted voice channels are
	// P25 Phase 2 H-DQPSK TDMA carriers. The Phase 1 control channel
	// uses this to route grants on TDMA channels into the Phase 2
	// voice composer chain (Protocol="p25-phase2") rather than the
	// Phase 1 FDMA chain. Issue #376: without this, MMR-class systems
	// (Phase 1 CC + Phase 2 TC) silently lose their Phase 2 MAC PDU
	// dispatch (talker alias, in-call src/enc backfill).
	AccessTDMA bool
}

// ParseIdentifierUpdate decodes the 8-byte payload of opcode 0x3D
// (OpIdentifierUpdate). The bit layout is, MSB first:
//
//	[ ChanID(4) | BW(9) | OFF(9) | STEP(10) | FREQ(32) ]
func ParseIdentifierUpdate(p [8]byte) IdentifierUpdate {
	chanID := p[0] >> 4
	bw := uint16(p[0]&0x0F)<<5 | uint16(p[1])>>3
	offRaw := uint16(p[1]&0x07)<<6 | uint16(p[2])>>2
	step := uint16(p[2]&0x03)<<8 | uint16(p[3])
	freq5Hz := uint32(p[4])<<24 | uint32(p[5])<<16 | uint32(p[6])<<8 | uint32(p[7])

	// Sign-extend OFF (9 bits, two's complement).
	off := int16(offRaw)
	if off&0x100 != 0 {
		off -= 0x200
	}
	return IdentifierUpdate{
		ChannelID:   chanID,
		BandwidthHz: uint32(bw) * 125,
		SpacingHz:   uint32(step) * 125,
		TxOffsetHz:  int64(off) * 250_000,
		BaseHz:      uint64(freq5Hz) * 5,
	}
}

// ParseIdentifierUpdateVUHF decodes the 8-byte payload of opcode 0x34
// (OpIdentifierUpdateVUHF), the VHF/UHF variant. The bit layout, MSB
// first, per TIA-102.AABF Table 14a:
//
//	[ ChanID(4) | BW(4) | TxOffsetSign(1) | TxOffsetMag(13) | STEP(10) | FREQ(32) ]
//
// Differences from the 0x3D variant:
//
//   - BW is a 4-bit lookup code (Table 16) rather than a 9-bit ×125 Hz
//     field: 0x4 → 6.25 kHz, 0x5 → 12.5 kHz, others → 0 (unknown). The
//     value is informational only — BandPlan.Frequency does not
//     consult it.
//
//   - TxOffset is a 1-bit sign + 13-bit magnitude pair, and the
//     magnitude unit is the channel step (SpacingHz) rather than a
//     fixed 250 kHz. The same convention as 0x3D applies to the
//     stored TxOffsetHz: positive = uplink above downlink ("mob Tx+"),
//     negative = uplink below downlink ("mob Tx-").
//
// STEP, FREQ, and the resulting SpacingHz / BaseHz values use the same
// scaling as 0x3D (×125 Hz / ×5 Hz).
//
// Cross-checked against OP25 (`op25/gr-op25_repeater/apps/trunking.py`
// `iden_up vhf uhf`) and SDRTrunk
// (`FrequencyBandUpdateVUHF.java`).
func ParseIdentifierUpdateVUHF(p [8]byte) IdentifierUpdate {
	chanID := p[0] >> 4
	bwCode := p[0] & 0x0F
	toff14 := uint16(p[1])<<6 | uint16(p[2])>>2
	toffSign := (toff14 >> 13) & 0x1
	toffMag := toff14 & 0x1FFF
	step := uint16(p[2]&0x03)<<8 | uint16(p[3])
	freq5Hz := uint32(p[4])<<24 | uint32(p[5])<<16 | uint32(p[6])<<8 | uint32(p[7])

	spacingHz := uint32(step) * 125
	offsetMagHz := int64(toffMag) * int64(spacingHz)
	txOffsetHz := offsetMagHz
	if toffSign == 0 {
		txOffsetHz = -offsetMagHz
	}

	return IdentifierUpdate{
		ChannelID:   chanID,
		BandwidthHz: vuhfBandwidthHz(bwCode),
		SpacingHz:   spacingHz,
		TxOffsetHz:  txOffsetHz,
		BaseHz:      uint64(freq5Hz) * 5,
	}
}

// AssembleIdentifierUpdateVUHF is the inverse of ParseIdentifierUpdateVUHF;
// useful for synthetic test streams. Out-of-range fields are silently
// truncated to their on-air widths. TxOffsetHz that isn't an integer
// multiple of SpacingHz is truncated toward zero; a zero SpacingHz
// produces a zero on-air offset regardless of TxOffsetHz.
func AssembleIdentifierUpdateVUHF(u IdentifierUpdate) [8]byte {
	bwCode := vuhfBandwidthCode(u.BandwidthHz) & 0x0F
	step := uint16(u.SpacingHz/125) & 0x3FF

	var toffSign uint8
	var toffMag uint16
	if u.TxOffsetHz >= 0 {
		toffSign = 1
		if u.SpacingHz > 0 {
			toffMag = uint16(uint64(u.TxOffsetHz)/uint64(u.SpacingHz)) & 0x1FFF
		}
	} else {
		toffSign = 0
		if u.SpacingHz > 0 {
			toffMag = uint16(uint64(-u.TxOffsetHz)/uint64(u.SpacingHz)) & 0x1FFF
		}
	}
	toff14 := uint16(toffSign)<<13 | (toffMag & 0x1FFF)

	freq5 := uint32(u.BaseHz / 5)

	var p [8]byte
	p[0] = (u.ChannelID&0x0F)<<4 | bwCode
	p[1] = byte(toff14 >> 6)
	p[2] = byte((toff14&0x3F)<<2) | byte(step>>8)
	p[3] = byte(step)
	p[4] = byte(freq5 >> 24)
	p[5] = byte(freq5 >> 16)
	p[6] = byte(freq5 >> 8)
	p[7] = byte(freq5)
	return p
}

// ParseIdentifierUpdateTDMA decodes the 8-byte payload of opcode 0x33
// (OpIdentifierUpdateTDMA), the Phase-2 TDMA variant. The bit layout,
// MSB first, per TIA-102.AABF Table 14:
//
//	[ ChanID(4) | ChanType(4) | TxOffsetSign(1) | TxOffsetMag(13) | STEP(10) | FREQ(32) ]
//
// The packing is identical to VUHF (opcode 0x34) for every field
// EXCEPT byte 0's lower nibble: where VUHF carries a 4-bit bandwidth
// code, TDMA carries a 4-bit Channel Type code that simultaneously
// encodes the slot count (TDMA-1 / TDMA-2 / TDMA-4) and the access
// mode (FDMA / TDMA) plus a nominal bandwidth. The bandwidth half of
// that code is mapped into BandwidthHz here for log readability; the
// slot count is intentionally NOT carried on IdentifierUpdate because
// BandPlan.Frequency uses only Base + Spacing + ChannelNumber, and
// downstream Phase 2 voice decoding consumes slot count from the
// grant payload, not from the band plan.
//
// TxOffset semantics match VUHF: 1-bit sign + 13-bit magnitude, the
// magnitude scaled by SpacingHz (not 250 kHz like the 0x3D form).
// STEP, FREQ scaling is the same ×125 Hz / ×5 Hz as the other two
// variants.
//
// Cross-checked against OP25 (`op25/gr-op25_repeater/apps/trunking.py`
// `iden_up_tdma`) and SDRTrunk
// (`FrequencyBandUpdateTDMA.java`).
func ParseIdentifierUpdateTDMA(p [8]byte) IdentifierUpdate {
	chanID := p[0] >> 4
	chanType := p[0] & 0x0F
	toff14 := uint16(p[1])<<6 | uint16(p[2])>>2
	toffSign := (toff14 >> 13) & 0x1
	toffMag := toff14 & 0x1FFF
	step := uint16(p[2]&0x03)<<8 | uint16(p[3])
	freq5Hz := uint32(p[4])<<24 | uint32(p[5])<<16 | uint32(p[6])<<8 | uint32(p[7])

	spacingHz := uint32(step) * 125
	offsetMagHz := int64(toffMag) * int64(spacingHz)
	txOffsetHz := offsetMagHz
	if toffSign == 0 {
		txOffsetHz = -offsetMagHz
	}

	return IdentifierUpdate{
		ChannelID:   chanID,
		BandwidthHz: tdmaChannelTypeBandwidthHz(chanType),
		SpacingHz:   spacingHz,
		TxOffsetHz:  txOffsetHz,
		BaseHz:      uint64(freq5Hz) * 5,
		AccessTDMA:  true,
	}
}

// AssembleIdentifierUpdateTDMA is the inverse of
// ParseIdentifierUpdateTDMA; useful for synthetic test streams.
// Out-of-range fields are silently truncated to their on-air widths.
// TxOffsetHz that isn't an integer multiple of SpacingHz is truncated
// toward zero; a zero SpacingHz produces a zero on-air offset
// regardless of TxOffsetHz. ChannelType is reverse-mapped from
// BandwidthHz via tdmaChannelTypeBandwidthCode; the slot-count half
// of the channel-type encoding isn't carried on IdentifierUpdate so
// only the bandwidth-derived form round-trips.
func AssembleIdentifierUpdateTDMA(u IdentifierUpdate) [8]byte {
	chanType := tdmaChannelTypeBandwidthCode(u.BandwidthHz) & 0x0F
	step := uint16(u.SpacingHz/125) & 0x3FF

	var toffSign uint8
	var toffMag uint16
	if u.TxOffsetHz >= 0 {
		toffSign = 1
		if u.SpacingHz > 0 {
			toffMag = uint16(uint64(u.TxOffsetHz)/uint64(u.SpacingHz)) & 0x1FFF
		}
	} else {
		toffSign = 0
		if u.SpacingHz > 0 {
			toffMag = uint16(uint64(-u.TxOffsetHz)/uint64(u.SpacingHz)) & 0x1FFF
		}
	}
	toff14 := uint16(toffSign)<<13 | (toffMag & 0x1FFF)

	freq5 := uint32(u.BaseHz / 5)

	var p [8]byte
	p[0] = (u.ChannelID&0x0F)<<4 | chanType
	p[1] = byte(toff14 >> 6)
	p[2] = byte((toff14&0x3F)<<2) | byte(step>>8)
	p[3] = byte(step)
	p[4] = byte(freq5 >> 24)
	p[5] = byte(freq5 >> 16)
	p[6] = byte(freq5 >> 8)
	p[7] = byte(freq5)
	return p
}

// tdmaChannelTypeBandwidthHz maps the 4-bit Channel Type code in
// opcode 0x33 to the nominal channel bandwidth in Hz. The Channel
// Type field also encodes slot count (TDMA-1 / TDMA-2 / TDMA-4) and
// access mode; that information isn't carried on IdentifierUpdate.
// Unknown codes return 0; BandPlan.Frequency does not consult this
// field, so an unknown code does not prevent grant resolution.
//
// The mapping covers the codes a Phase-2 site typically broadcasts;
// per TIA-102.AABF most TDMA codes target 12.5 kHz outbound on a
// 6.25 kHz inbound, and the BandwidthHz reported is the inbound /
// per-slot value most operators expect to see in logs.
func tdmaChannelTypeBandwidthHz(code uint8) uint32 {
	switch code & 0x0F {
	case 0x1, 0x3:
		return 6250
	case 0x2:
		return 12500
	default:
		return 0
	}
}

// tdmaChannelTypeBandwidthCode is the inverse of
// tdmaChannelTypeBandwidthHz used by AssembleIdentifierUpdateTDMA.
// Unrecognised bandwidths emit code 0.
func tdmaChannelTypeBandwidthCode(hz uint32) uint8 {
	switch hz {
	case 6250:
		return 0x1
	case 12500:
		return 0x2
	default:
		return 0
	}
}

// vuhfBandwidthHz maps the 4-bit BW code in opcode 0x34 to channel
// bandwidth in Hz per TIA-102.AABF Table 16. Unknown codes return 0;
// BandPlan.Frequency does not consult this field.
func vuhfBandwidthHz(code uint8) uint32 {
	switch code & 0x0F {
	case 0x4:
		return 6250
	case 0x5:
		return 12500
	default:
		return 0
	}
}

// vuhfBandwidthCode is the inverse of vuhfBandwidthHz used by
// AssembleIdentifierUpdateVUHF. Unrecognised bandwidths emit code 0.
func vuhfBandwidthCode(hz uint32) uint8 {
	switch hz {
	case 6250:
		return 0x4
	case 12500:
		return 0x5
	default:
		return 0
	}
}

// AssembleIdentifierUpdate is the inverse of ParseIdentifierUpdate;
// useful for synthetic test streams. Out-of-range fields are silently
// truncated to their on-air widths, matching the ParseTSBK contract.
func AssembleIdentifierUpdate(u IdentifierUpdate) [8]byte {
	bw := uint16(u.BandwidthHz/125) & 0x1FF
	step := uint16(u.SpacingHz/125) & 0x3FF
	off := uint16(u.TxOffsetHz/250_000) & 0x1FF
	freq5 := uint32(u.BaseHz / 5)

	var p [8]byte
	p[0] = (u.ChannelID&0x0F)<<4 | byte(bw>>5)
	p[1] = byte(bw<<3) | byte(off>>6)
	p[2] = byte(off<<2) | byte(step>>8)
	p[3] = byte(step)
	p[4] = byte(freq5 >> 24)
	p[5] = byte(freq5 >> 16)
	p[6] = byte(freq5 >> 8)
	p[7] = byte(freq5)
	return p
}

// ErrUnknownChannelID is returned by BandPlan.Frequency when no
// IdentifierUpdate has been observed for the requested channel ID.
// The control channel maps this to a `decode.error` with
// stage="no-bandplan" so the gap is visible in metrics.
var ErrUnknownChannelID = errors.New("p25/phase1: no IdentifierUpdate for channel ID")

// BandPlan accumulates IdentifierUpdate state — one slot per Channel
// ID — and resolves voice-grant (ChannelID, ChannelNumber) tuples to
// downlink frequencies. Zero-value BandPlan{} is ready to use; not
// safe for concurrent Apply / Frequency without external locking, but
// the control channel reads/writes from a single goroutine so that's
// the natural usage shape.
type BandPlan struct {
	slots [16]IdentifierUpdate
	known [16]bool
}

// Apply records (or replaces) the band-plan slot for u.ChannelID.
func (b *BandPlan) Apply(u IdentifierUpdate) {
	if int(u.ChannelID) >= len(b.slots) {
		return
	}
	b.slots[u.ChannelID] = u
	b.known[u.ChannelID] = true
}

// Frequency returns the downlink frequency in Hz for the given
// channel ID + channel number, or ErrUnknownChannelID if no
// IdentifierUpdate has been seen for that ID.
func (b *BandPlan) Frequency(channelID uint8, channelNumber uint16) (uint32, error) {
	if int(channelID) >= len(b.slots) || !b.known[channelID] {
		return 0, fmt.Errorf("%w: id=%d", ErrUnknownChannelID, channelID)
	}
	u := b.slots[channelID]
	hz := u.BaseHz + uint64(channelNumber)*uint64(u.SpacingHz)
	// P25 sits well below 4.29 GHz; guard anyway so a malformed
	// IdentifierUpdate can't silently wrap.
	if hz > 0xFFFFFFFF {
		return 0, fmt.Errorf("p25/phase1: resolved frequency %d Hz overflows uint32", hz)
	}
	return uint32(hz), nil
}

// Known reports whether an IdentifierUpdate slot has been recorded.
// Intended for tests + diagnostics.
func (b *BandPlan) Known(channelID uint8) bool {
	if int(channelID) >= len(b.known) {
		return false
	}
	return b.known[channelID]
}

// IsTDMA reports whether the channel ID was advertised via opcode 0x33
// (OpIdentifierUpdateTDMA). Returns false for unknown channel IDs and
// for FDMA channels. The Phase 1 control channel queries this at grant
// publish time so voice grants targeting Phase 2 TDMA carriers get
// routed to the composer's Phase 2 MAC dispatch path.
func (b *BandPlan) IsTDMA(channelID uint8) bool {
	if int(channelID) >= len(b.known) || !b.known[channelID] {
		return false
	}
	return b.slots[channelID].AccessTDMA
}

// Snapshot returns the IdentifierUpdate entries for every known channel ID,
// in ascending channel-ID order. Used by the signal-lab / hunt layers to
// surface the band plan a control channel advertised so a discovered system
// can be documented and exported.
func (b *BandPlan) Snapshot() []IdentifierUpdate {
	var out []IdentifierUpdate
	for id := 0; id < len(b.slots); id++ {
		if b.known[id] {
			out = append(out, b.slots[id])
		}
	}
	return out
}
