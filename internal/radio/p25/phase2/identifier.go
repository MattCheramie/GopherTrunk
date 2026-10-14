package phase2

import (
	"errors"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/radio/p25"
)

// IdentifierUpdate carries a P25 Phase 2 MAC frequency-band definition —
// the channel-plan announcement a site repeats so a receiver can
// translate a voice grant's (Channel ID, Channel Number) pair into a
// downlink frequency. It is the Phase 2 MAC counterpart of the Phase 1
// IDENT_UP TSBK (phase1.IdentifierUpdate, opcode 0x3D).
//
// Field units follow TIA-102.AABF Table 14, identical to Phase 1:
//
//	BandwidthHz = BW   × 125  Hz
//	SpacingHz   = STEP × 125  Hz
//	TxOffsetHz  = OFF  × 250 kHz   (signed, sign-extended from 9 bits)
//	BaseHz      = FREQ × 5    Hz
//
// Layout note: the FDMA form (MAC opcode 0x7D) reuses the Phase 1
// opcode-0x3D bit packing (ParseIdentifierUpdate). A Phase 2 site
// announces its TDMA voice channels with the TDMA forms, 0x73 abbreviated
// and 0xF3 extended (ParseIdentifierUpdateTDMA), whose channel type sets
// SlotsPerCarrier — see BandPlan.Frequency (issue #1273).
type IdentifierUpdate struct {
	ChannelID   uint8  // 4-bit slot identifier (0..15)
	BandwidthHz uint32 // channel bandwidth
	SpacingHz   uint32 // channel-to-channel step
	TxOffsetHz  int64  // signed transmit offset
	BaseHz      uint64 // downlink base frequency for channel 0
	// AccessTDMA and SlotsPerCarrier come from the channel-type code of a
	// TDMA-form announcement (MAC opcode 0x73 / 0xF3). On a multi-slot
	// carrier a grant's channel number counts logical channels
	// (carrier × slots + slot), so BandPlan.Frequency divides by
	// SlotsPerCarrier. Zero / one means one call per channel number —
	// the FDMA 0x7D form (issue #1273).
	AccessTDMA      bool
	SlotsPerCarrier uint8
}

// ParseIdentifierUpdate decodes the 8-byte band-plan field. Bit layout,
// MSB first: [ ChanID(4) | BW(9) | OFF(9) | STEP(10) | FREQ(32) ].
// Returns ok=false if p is shorter than 8 bytes.
func ParseIdentifierUpdate(p []byte) (IdentifierUpdate, bool) {
	if len(p) < 8 {
		return IdentifierUpdate{}, false
	}
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
	}, true
}

// ParseIdentifierUpdateTDMA decodes the 8-byte field group of a TDMA-form
// identifier update: the payload of MAC opcode 0x73 (IDENTIFIER UPDATE TDMA
// ABBREVIATED) and bytes 1..8 of 0xF3 (EXTENDED). Bit layout, MSB first:
//
//	[ ChanID(4) | ChanType(4) | TxOffsetSign(1) | TxOffsetMag(13) | STEP(10) | FREQ(32) ]
//
// — the Phase 1 IDEN_UP_TDMA TSBK (0x33) layout. SDRTrunk
// FrequencyBandUpdateTDMAAbbreviated / Extended and OP25 tk_p25.py op 0xf3
// read the same fields at the same relative positions. The channel type
// gives access mode, bandwidth and slots per carrier from the table both
// P25 decoders share (p25.ParseTDMAChannelType). The tx offset is a
// sign-magnitude multiple of the spacing (sign set = positive, as the
// Phase 1 parser reads it); it never affects the downlink frequency.
// Returns ok=false if p is shorter than 8 bytes.
func ParseIdentifierUpdateTDMA(p []byte) (IdentifierUpdate, bool) {
	if len(p) < 8 {
		return IdentifierUpdate{}, false
	}
	chanID := p[0] >> 4
	ct := p25.ParseTDMAChannelType(p[0] & 0x0F)
	toff14 := uint16(p[1])<<6 | uint16(p[2])>>2
	step := uint16(p[2]&0x03)<<8 | uint16(p[3])
	freq5Hz := uint32(p[4])<<24 | uint32(p[5])<<16 | uint32(p[6])<<8 | uint32(p[7])

	spacingHz := uint32(step) * 125
	txOffsetHz := int64(toff14&0x1FFF) * int64(spacingHz)
	if toff14&0x2000 == 0 {
		txOffsetHz = -txOffsetHz
	}
	return IdentifierUpdate{
		ChannelID:       chanID,
		BandwidthHz:     ct.BandwidthHz,
		SpacingHz:       spacingHz,
		TxOffsetHz:      txOffsetHz,
		BaseHz:          uint64(freq5Hz) * 5,
		AccessTDMA:      ct.TDMA,
		SlotsPerCarrier: ct.SlotsPerCarrier,
	}, true
}

// AssembleIdentifierUpdate is the inverse of ParseIdentifierUpdate;
// it builds the 8-byte band-plan field for synthesized test streams.
// Out-of-range fields are silently truncated to their on-air widths.
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
var ErrUnknownChannelID = errors.New("p25/phase2: no IdentifierUpdate for channel ID")

// BandPlan accumulates IdentifierUpdate state — one slot per Channel
// ID — and resolves voice-grant (ChannelID, ChannelNumber) tuples to
// downlink frequencies. The zero value is ready to use. It is not safe
// for concurrent Apply / Frequency; the ControlChannel guards it with
// its own mutex.
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

// Frequency returns the downlink frequency in Hz for the given channel
// ID + channel number, or ErrUnknownChannelID if no IdentifierUpdate
// has been seen for that ID.
func (b *BandPlan) Frequency(channelID uint8, channelNumber uint16) (uint32, error) {
	if int(channelID) >= len(b.slots) || !b.known[channelID] {
		return 0, fmt.Errorf("%w: id=%d", ErrUnknownChannelID, channelID)
	}
	u := b.slots[channelID]
	// A TDMA channel number counts logical channels (carrier × slots +
	// slot): divide down to the carrier first, as SDRTrunk
	// FrequencyBandUpdateTDMA* getDownlinkFrequency and OP25
	// channel_id_to_frequency do (issue #1273).
	carrier := uint64(channelNumber)
	if u.SlotsPerCarrier > 1 {
		carrier /= uint64(u.SlotsPerCarrier)
	}
	hz := u.BaseHz + carrier*uint64(u.SpacingHz)
	if hz > 0xFFFFFFFF {
		return 0, fmt.Errorf("p25/phase2: resolved frequency %d Hz overflows uint32", hz)
	}
	return uint32(hz), nil
}

// Timeslot returns the 1-based TDMA timeslot a channel number selects on a
// multi-slot carrier (channel number mod slots-per-carrier, + 1), or 0 when
// the channel ID is unknown or one-slot — the trunking.Grant.Timeslot
// convention, where 0 means "no slot".
func (b *BandPlan) Timeslot(channelID uint8, channelNumber uint16) uint8 {
	if int(channelID) >= len(b.slots) || !b.known[channelID] {
		return 0
	}
	slots := b.slots[channelID].SlotsPerCarrier
	if slots < 2 {
		return 0
	}
	return uint8(channelNumber%uint16(slots)) + 1
}

// Known reports whether an IdentifierUpdate slot has been recorded.
func (b *BandPlan) Known(channelID uint8) bool {
	if int(channelID) >= len(b.known) {
		return false
	}
	return b.known[channelID]
}
