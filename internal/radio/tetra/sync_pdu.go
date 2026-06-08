package tetra

// SyncPDU is the MAC SYNC broadcast carried by the BSCH (block 1 of the
// synchronisation downlink burst), per ETSI EN 300 392-2 §21.4.4.2. The
// BSCH is scrambled with colour code 0, so a cold receiver — one with no
// pre-configured colour code — can always decode it. It carries the
// fields needed to form the cell's scrambling code, so once the BSCH is
// decoded every other logical channel (BNCH SYSINFO, SCH/HD, SCH/F)
// becomes decodable without operator configuration.
//
// Field layout in the 60 decoded type-1 bits, MSB-first (matching the
// osmo-tetra reference parser):
//
//	bits  4..9   colour code        (6)
//	bits 10..11  timeslot number    (2)
//	bits 12..16  frame number       (5)
//	bits 17..22  multiframe number  (6)
//	bits 31..40  mobile country code (10)
//	bits 41..54  mobile network code (14)
type SyncPDU struct {
	ColourCode uint8  // 6-bit colour code
	TN         uint8  // timeslot number (1..4 encoded 0..3)
	FN         uint8  // frame number
	MN         uint8  // multiframe number
	MCC        uint16 // mobile country code
	MNC        uint16 // mobile network code
}

// bitsToUint reads n bits (MSB-first) starting at off from a slice of
// 0/1 bytes. Returns 0 if the range exceeds the slice.
func bitsToUint(bits []byte, off, n int) uint32 {
	if off < 0 || off+n > len(bits) {
		return 0
	}
	var v uint32
	for i := 0; i < n; i++ {
		v = (v << 1) | uint32(bits[off+i]&1)
	}
	return v
}

// ParseSyncPDU parses the 60 type-1 bits recovered from a BSCH (each
// element 0 or 1, MSB-first) into a SyncPDU. Returns false if the input
// is too short.
func ParseSyncPDU(bits []byte) (SyncPDU, bool) {
	if len(bits) < 55 {
		return SyncPDU{}, false
	}
	return SyncPDU{
		ColourCode: uint8(bitsToUint(bits, 4, 6)),
		TN:         uint8(bitsToUint(bits, 10, 2)),
		FN:         uint8(bitsToUint(bits, 12, 5)),
		MN:         uint8(bitsToUint(bits, 17, 6)),
		MCC:        uint16(bitsToUint(bits, 31, 10)),
		MNC:        uint16(bitsToUint(bits, 41, 14)),
	}, true
}

// ExtendedColourCode forms the 30-bit extended colour code that seeds
// the TETRA scrambler for every channel other than the BSCH, per ETSI
// EN 300 392-2 §8.2.5.2: MCC (10 bits) | MNC (14 bits) | colour code
// (6 bits). This is the value passed to SetColourCode / the Decode*
// helpers (framing.NewScramblerTetra) so BNCH/SCH descramble.
func (s SyncPDU) ExtendedColourCode() uint32 {
	return (uint32(s.MCC&0x3FF) << 20) | (uint32(s.MNC&0x3FFF) << 6) | uint32(s.ColourCode&0x3F)
}

// ExtendedColourCode is the package-level form of SyncPDU.ExtendedColourCode,
// for callers that already hold MCC/MNC/CC separately (e.g. config that
// lets an operator give the three components instead of the packed value).
func ExtendedColourCode(mcc, mnc uint16, colour uint8) uint32 {
	return (uint32(mcc&0x3FF) << 20) | (uint32(mnc&0x3FFF) << 6) | uint32(colour&0x3F)
}
