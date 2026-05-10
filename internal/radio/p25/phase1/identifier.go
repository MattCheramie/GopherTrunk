package phase1

import (
	"errors"
	"fmt"
)

// IdentifierUpdate carries a P25 IDENT_UP TSBK (opcode 0x3D), the
// "Identifier Update" announcement that defines the band plan for one
// channel-ID slot. A site repeats these periodically alongside its
// status broadcasts; the receiver accumulates them and uses each slot
// to translate a (Channel ID, Channel Number) pair from a voice grant
// into a downlink frequency.
//
// Field units come straight from TIA-102.AABF Table 14:
//
//	BandwidthHz   = BW    × 125  Hz
//	SpacingHz     = STEP  × 125  Hz   (channel step)
//	TxOffsetHz    = OFF   × 250 kHz   (signed; sign-extended from 9-bit)
//	BaseHz        = FREQ  × 5    Hz
//
// Only the standard 700/800/900 MHz form is parsed here (opcode 0x3D).
// VHF/UHF (0x34) and Phase-2 TDMA (0x33) variants encode the same
// fields with different bit packings and are out of scope for this
// pass; lighting them up is incremental.
type IdentifierUpdate struct {
	ChannelID   uint8  // 4-bit slot identifier (0..15)
	BandwidthHz uint32 // channel bandwidth
	SpacingHz   uint32 // channel-to-channel step
	TxOffsetHz  int64  // signed transmit offset (uplink = downlink + offset)
	BaseHz      uint64 // downlink base frequency for channel 0
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
