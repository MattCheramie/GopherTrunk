package phase2

import "github.com/MattCheramie/GopherTrunk/internal/radio/framing"

// P25 Phase 2 ACCH framing: lifting a MAC PDU out of one 180-dibit TDMA
// burst. This is the layer TIA-102.BBAB/BBAC define between the demodulated
// dibit stream and mac.go's PDU parse, and it is the layer whose absence kept
// GopherTrunk from decoding any Phase 2 signalling (issue #915).
//
// The spec is paywalled. Every constant here was instead read independently
// out of the two working implementations on which the whole chain was
// cross-checked — SDRtrunk's Java (Timeslot / FacchTimeslot / SacchTimeslot)
// and OP25's C++ (p25p2_tdma.cc::handle_acch_frame) — which agree exactly, and
// then validated against real air: on a real P25 Phase 2 traffic channel this
// code recovers FACCH-S PDUs byte-identical to SDRtrunk's decode of the same
// capture, and produces no PDU SDRtrunk does not also produce.
//
// Burst geometry, on which everything else hangs:
//
//	dibit  0..19   40-bit ISCH region — carries the frame sync in the six
//	               S-ISCH slots, an I-ISCH codeword in the other six
//	dibit 20..179  320-bit payload
//
// The DUID that names the burst type is 8 bits scattered across four payload
// dibits and is read BEFORE descrambling (both references do). The payload is
// then XORed with the PN44 sequence, whose origin sits mid-ISCH at slot 0 bit
// 20 — so slot k's payload begins at sequence bit k*360+20.
//
// Note what is NOT here: no trellis code and no deinterleaver. The ACCH path
// runs dibits → bits → 6-bit symbols → RS(63,35,29) → CRC-12, with nothing in
// between.

// Burst geometry.
const (
	// BurstDibits is the on-wire width of one TDMA burst (360 bits).
	BurstDibits = 180
	// ISCHRegionDibits is the width of the leading ISCH / frame-sync
	// region. The payload starts immediately after it.
	ISCHRegionDibits = 20
	// PayloadDibits is the width of the burst payload.
	PayloadDibits = BurstDibits - ISCHRegionDibits
	// ScrambleOriginBit is where the PN44 sequence's bit 0 lands within
	// slot 0: mid-ISCH, 20 bits before the payload. Both references
	// place it here — SDRtrunk takes its per-slot segment at sequence
	// offset 20+k*360, OP25 descrambles from burst dibit 10.
	ScrambleOriginBit = 20
	// SuperframeScrambleBits is the length of one PN44 superframe
	// sequence: 12 slots x 360 bits. It restarts every superframe.
	SuperframeScrambleBits = SubframesPerSuperframe * 2 * BurstDibits
)

// BurstType names what a TDMA burst carries, as decoded from its DUID.
// Values are the DUID code's own, so they can be compared directly against
// either reference implementation.
type BurstType uint8

const (
	BurstVoice4           BurstType = 0  // 4V — four voice frames
	BurstSACCHScrambled   BurstType = 3  // scrambled slow ACCH
	BurstLCCHScrambled    BurstType = 4  // scrambled link control channel
	BurstVoice2           BurstType = 6  // 2V — two voice frames + ESS
	BurstFACCHScrambled   BurstType = 9  // scrambled fast ACCH
	BurstSACCHUnscrambled BurstType = 12 // unscrambled slow ACCH
	BurstLCCHUnscrambled  BurstType = 13 // unscrambled LCCH / TDMA CC
	BurstFACCHUnscrambled BurstType = 15 // unscrambled fast ACCH
	// BurstInvalid is returned when the received DUID codeword is more
	// than one bit from every valid codeword.
	BurstInvalid BurstType = 0xFF
)

// IsACCH reports whether the burst carries a signalling PDU this file can
// decode. Voice bursts and the LCCH (used only for inbound IECI, which
// neither reference decodes) are excluded.
func (b BurstType) IsACCH() bool {
	switch b {
	case BurstSACCHScrambled, BurstFACCHScrambled, BurstSACCHUnscrambled, BurstFACCHUnscrambled:
		return true
	}
	return false
}

// IsFast reports whether the burst is a FACCH-S rather than a SACCH-S. The
// two differ in their dibit windows, message length and RS block placement.
func (b BurstType) IsFast() bool {
	return b == BurstFACCHScrambled || b == BurstFACCHUnscrambled
}

// IsScrambled reports whether the payload is PN44-scrambled.
func (b BurstType) IsScrambled() bool {
	return b == BurstSACCHScrambled || b == BurstFACCHScrambled || b == BurstLCCHScrambled
}

// duidDibitPositions are the four burst dibits carrying the scattered DUID,
// equivalently payload bits 0-1, 74-75, 244-245 and 318-319.
var duidDibitPositions = [4]int{20, 57, 142, 179}

// duidParityRows is the parity half of the systematic (8, 4) DUID code, one
// row per bit of the burst-type value, LSB first: a value contributes row i
// when its bit i is set. So type 1 encodes as 0x17, type 2 as 0x2E, type 4 as
// 0x4B and type 8 as 0x8D, and the rest follow by linearity.
//
// The row order is the whole content of this constant, and it is easy to get
// backwards — indexing these MSB-first bit-reverses every value. Four burst
// types are their own bit reversal (0 and 6, the voice bursts; 9 and 15, the
// two FACCH forms), so a reversed table decodes all of those correctly and
// silently swaps the rest: scrambled SACCH (3) reads as unscrambled SACCH (12)
// and is never descrambled. That was live here until 2026-09-01 and showed up
// only as an absence — sdrtrunk found 377 SACCH-S bursts in a capture set
// where this decoder found none, while agreeing on every FACCH-S.
var duidParityRows = [4]byte{0x7, 0xE, 0xB, 0xD}

// duidDecodeTable maps a received 8-bit DUID codeword to its burst type, or
// to BurstInvalid. The code is a systematic (8, 4) linear code with minimum
// distance 3 — the value in the high nibble, four parity bits in the low one —
// so the table corrects any single bit error and is generated here from
// duidParityRows rather than transcribed from a reference implementation.
var duidDecodeTable = func() [256]BurstType {
	var tbl [256]BurstType
	for i := range tbl {
		tbl[i] = BurstInvalid
	}
	for v := 0; v < 16; v++ {
		var parity byte
		for bit := 0; bit < 4; bit++ {
			if v&(1<<bit) != 0 {
				parity ^= duidParityRows[bit]
			}
		}
		cw := byte(v)<<4 | parity
		tbl[cw] = BurstType(v)
		for bit := 0; bit < 8; bit++ {
			tbl[cw^(1<<bit)] = BurstType(v)
		}
	}
	return tbl
}()

// BurstTypeOf decodes the DUID of a BurstDibits-long burst. It reads the raw,
// still-scrambled dibits: the DUID sits outside the scrambled field.
func BurstTypeOf(burst []uint8) BurstType {
	if len(burst) < BurstDibits {
		return BurstInvalid
	}
	var cw byte
	for i, pos := range duidDibitPositions {
		cw |= (burst[pos] & 3) << (6 - 2*i)
	}
	return duidDecodeTable[cw]
}

// ScrambleSequence returns one superframe of the PN44 sequence for the given
// seed, ready to hand to DecodeACCHBurst. Callers derive the seed with
// framing.PN44SeedFromIdentity from the (WACN, System ID, NAC) the Network
// Status Broadcast publishes. The sequence restarts every superframe, so one
// slice serves for the life of a channel.
func ScrambleSequence(seed uint64) []byte {
	seq := make([]byte, SuperframeScrambleBits)
	s := framing.NewPN44Scrambler(seed)
	for i := range seq {
		seq[i] = s.Next()
	}
	return seq
}

// acchWindows returns the payload-relative dibit windows that carry the coded
// ACCH bits — the payload minus the four DUID dibits and the 21-dibit gap
// across the middle of the burst. FACCH-S fills 135 dibits (270 coded bits),
// SACCH-S 156 (312).
func acchWindows(fast bool) [][2]int {
	if fast {
		return [][2]int{{1, 37}, {38, 69}, {90, 122}, {123, 159}}
	}
	return [][2]int{{1, 37}, {38, 122}, {123, 159}}
}

// acchLayout gives the shortened, punctured RS(63,35,29) placement for each
// ACCH kind: the block index the first received symbol occupies, the length of
// the recovered message in bits (information bits plus the 12 CRC bits), and
// the parity positions that were never transmitted.
//
// The two ends of the block are not the same thing, and the distinction is
// worth its correction budget. The information symbols before rsStart are
// *shortened* — the encoder treated them as zero and so does the decoder, for
// free. The parity symbols after the last received one are *punctured* — real
// values that simply did not travel, so they are erasures, unknown at known
// positions. 2·errors + erasures ≤ 28 then leaves a FACCH-S burst 9 correctable
// symbol errors and a SACCH-S burst 11.
func acchLayout(fast bool) (rsStart, msgBits int, erasures []int) {
	if fast {
		// 45 symbols received into block[9..53]: 26 information
		// (144 bits + CRC-12), 19 parity; block[54..62] punctured.
		return 9, 156, []int{54, 55, 56, 57, 58, 59, 60, 61, 62}
	}
	// 52 symbols into block[5..56]: 30 information (168 bits + CRC-12),
	// 22 parity; block[57..62] punctured.
	return 5, 180, []int{57, 58, 59, 60, 61, 62}
}

// ACCHResult is one decoded ACCH burst.
type ACCHResult struct {
	// Burst is the burst type the DUID named.
	Burst BurstType
	// Message is the ACCH message: information bits followed by their
	// CRC-12, 156 bits for FACCH-S and 180 for SACCH-S — byte-for-byte what
	// SDRtrunk hands to its own MAC parser.
	Message []byte
	// RSValid reports that the outer RS(63,35,29) closed on this burst,
	// with RSErrors symbol errors repaired. When it is false the message
	// passed only its CRC-12, which happens when the damage sits in the
	// parity symbols the message itself does not depend on — the
	// information is still good, but one fewer check stands behind it.
	//
	// Callers that write a decode into anything durable should require it.
	// CRC-12 alone admits 1 word in 4096, and a MAC PDU that parses as a
	// GROUP VOICE CHANNEL USER carries a source RID with it (issue #915).
	RSValid  bool
	RSErrors int
}

// DecodeACCHBurst recovers the MAC PDU message from one TDMA burst.
//
// burst must be BurstDibits long and in the canonical dibit convention. slot
// is the burst's position 0..11 within the superframe, which selects the
// scramble phase; seq is one superframe of PN44 from ScrambleSequence, and may
// be nil for an unscrambled burst.
//
// ok is false when the burst is not an ACCH burst, or when neither the outer
// RS nor the CRC-12 can vouch for what came out of it.
func DecodeACCHBurst(burst []uint8, slot int, seq []byte) (ACCHResult, bool) {
	res := ACCHResult{Burst: BurstTypeOf(burst)}
	if !res.Burst.IsACCH() {
		return res, false
	}
	payload := make([]uint8, PayloadDibits)
	copy(payload, burst[ISCHRegionDibits:BurstDibits])
	if res.Burst.IsScrambled() {
		if len(seq) < SuperframeScrambleBits {
			return res, false
		}
		bits := framing.DibitsToBits(payload)
		off := slot*2*BurstDibits + ScrambleOriginBit
		for i := range bits {
			bits[i] ^= seq[(off+i)%SuperframeScrambleBits]
		}
		payload = framing.BitsToDibits(bits)
	}

	block := acchSymbolBlock(payload, res.Burst.IsFast())
	rsStart, msgBits, erasures := acchLayout(res.Burst.IsFast())

	// Correct first, then check. A burst whose RS closes is trusted on two
	// independent codes; one that does not may still be intact in the
	// information symbols, with the damage confined to parity — so fall
	// through to the CRC on the uncorrected block rather than dropping it.
	if corrected, nErr, err := framing.DecodeRS63_35(block, erasures); err == nil {
		candidate := acchMessageFrom(corrected, rsStart, msgBits)
		if framing.CRC12P25P2OK(candidate) {
			res.Message, res.RSValid, res.RSErrors = candidate, true, nErr
			return res, true
		}
	}
	candidate := acchMessageFrom(block, rsStart, msgBits)
	if !framing.CRC12P25P2OK(candidate) {
		return res, false
	}
	res.Message = candidate
	return res, true
}

// acchSymbolBlock extracts the coded bits from the payload's dibit windows and
// packs them into the 63-symbol RS block, 6 bits per symbol, leaving the
// shortened and punctured positions zero.
func acchSymbolBlock(payload []uint8, fast bool) []byte {
	var coded []byte
	for _, w := range acchWindows(fast) {
		for i := w[0]; i < w[1]; i++ {
			coded = append(coded, payload[i]>>1&1, payload[i]&1)
		}
	}
	rsStart, _, _ := acchLayout(fast)
	block := make([]byte, 63)
	for i, sym := 0, rsStart; i+5 < len(coded) && sym < len(block); i, sym = i+6, sym+1 {
		var v byte
		for k := 0; k < 6; k++ {
			v = v<<1 | coded[i+k]
		}
		block[sym] = v
	}
	return block
}

// acchMessageFrom reads the message bits back out of the block's information
// symbols.
func acchMessageFrom(block []byte, rsStart, msgBits int) []byte {
	msg := make([]byte, msgBits)
	for i, sym := 0, rsStart; i < msgBits; i, sym = i+6, sym+1 {
		for k := 0; k < 6; k++ {
			msg[i+k] = block[sym] >> (5 - k) & 1
		}
	}
	return msg
}

// EncodeACCHBurst builds a BurstDibits-long TDMA burst carrying one MAC
// message: the exact inverse of DecodeACCHBurst, and the generator that lets a
// synthesized fixture exercise the real framing rather than a model of it.
//
// structures are laid out after the header byte and the remainder is left
// zero, which is NULL INFORMATION padding — the same shape real idle traffic
// has. For a whole-PDU type (PTT, END PTT, and the two reserved types) there
// is no inner opcode at all: the header byte is the opcode, so only the first
// structure's Payload is written and its Opcode is ignored. slot selects
// the scramble phase; seq is one superframe of PN44 and may
// be nil for an unscrambled burst type. The ISCH region is left zero for the
// caller to fill (EncodeSuperframe stamps the frame sync into it).
//
// Structures that overflow the payload are truncated, since a MAC message has
// no continuation: 18 bytes for FACCH-S and 21 for SACCH-S, one of which is
// the header.
func EncodeACCHBurst(bt BurstType, pduType MACPDUType, voiceOffset uint8, structures []MACPDU, slot int, seq []byte) []uint8 {
	rsStart, msgBits, erasures := acchLayout(bt.IsFast())
	info := make([]byte, (msgBits-12)/8)
	info[0] = byte(pduType)<<5 | (voiceOffset&0x7)<<2
	if !pduType.carriesStructureSequence() {
		// A whole-PDU type has no inner opcode: the header byte is the
		// opcode, and the body follows it. The first structure's Payload is
		// that body; its Opcode is ignored, because the header supplies it.
		if len(structures) > 0 {
			copy(info[1:], structures[0].Payload)
		}
	} else {
		ptr := 1
		for _, s := range structures {
			b := AssembleMACPDU(s)
			if ptr+len(b) > len(info) {
				break
			}
			copy(info[ptr:], b)
			ptr += len(b)
		}
	}

	msg := framing.UnpackBitsMSB(info, len(info)*8)
	crc := framing.CRC12P25P2(msg)
	for i := 0; i < 12; i++ {
		msg = append(msg, byte(crc>>(11-i)&1))
	}

	// Message bits → information symbols → a full RS codeword, of which only
	// the un-punctured span is transmitted.
	var block [63]byte
	for i, sym := 0, rsStart; i < len(msg); i, sym = i+6, sym+1 {
		var v byte
		for k := 0; k < 6; k++ {
			v = v<<1 | msg[i+k]
		}
		block[sym] = v
	}
	var infoSyms [35]byte
	copy(infoSyms[:], block[:35])
	cw := framing.EncodeRS63_35(infoSyms)
	lastSent := 62
	for _, e := range erasures {
		if e <= lastSent {
			lastSent = e - 1
		}
	}

	// Symbols → coded bits → the payload's dibit windows.
	coded := make([]byte, 0, (lastSent-rsStart+1)*6)
	for sym := rsStart; sym <= lastSent; sym++ {
		for k := 0; k < 6; k++ {
			coded = append(coded, cw[sym]>>(5-k)&1)
		}
	}
	payload := make([]uint8, PayloadDibits)
	ci := 0
	for _, w := range acchWindows(bt.IsFast()) {
		for i := w[0]; i < w[1] && ci+1 < len(coded); i++ {
			payload[i] = coded[ci]<<1 | coded[ci+1]
			ci += 2
		}
	}

	if bt.IsScrambled() && len(seq) >= SuperframeScrambleBits {
		bits := framing.DibitsToBits(payload)
		off := slot*2*BurstDibits + ScrambleOriginBit
		for i := range bits {
			bits[i] ^= seq[(off+i)%SuperframeScrambleBits]
		}
		payload = framing.BitsToDibits(bits)
	}

	burst := make([]uint8, BurstDibits)
	copy(burst[ISCHRegionDibits:], payload)
	// The DUID rides outside the scrambled field — both reference decoders
	// read it before descrambling — so it is stamped last, over the top.
	var parity byte
	for bit := 0; bit < 4; bit++ {
		if byte(bt)&(1<<bit) != 0 {
			parity ^= duidParityRows[bit]
		}
	}
	cwDUID := byte(bt)<<4 | parity
	for i, pos := range duidDibitPositions {
		burst[pos] = cwDUID >> (6 - 2*i) & 3
	}
	return burst
}

// String names the burst type for logs and census buckets.
func (b BurstType) String() string {
	switch b {
	case BurstVoice4:
		return "4V"
	case BurstVoice2:
		return "2V"
	case BurstSACCHScrambled:
		return "SACCH_S"
	case BurstSACCHUnscrambled:
		return "SACCH_U"
	case BurstFACCHScrambled:
		return "FACCH_S"
	case BurstFACCHUnscrambled:
		return "FACCH_U"
	case BurstLCCHScrambled:
		return "LCCH_S"
	case BurstLCCHUnscrambled:
		return "LCCH_U"
	case BurstInvalid:
		return "INVALID"
	}
	return "RESERVED"
}

// acchSlotProbe is the cheap form of DecodeACCHBurst used by the slot-phase
// vote: it extracts the message and checks only the CRC-12, skipping the outer
// RS. The vote needs to tell a right phase from a wrong one, not to recover
// the PDU — and the RS decode is by far the most expensive step in the chain,
// which the vote would otherwise pay for on six candidates before the real
// decode runs once.
//
// A wrong phase clears the CRC once in 4096, so a burst may occasionally vote
// for the wrong candidate; the vote aggregates over the superframe, where the
// right phase clears every burst.
func acchSlotProbe(burst []uint8, slot int, seq []byte) bool {
	bt := BurstTypeOf(burst)
	if !bt.IsACCH() {
		return false
	}
	payload := make([]uint8, PayloadDibits)
	copy(payload, burst[ISCHRegionDibits:BurstDibits])
	if bt.IsScrambled() {
		if len(seq) < SuperframeScrambleBits {
			return false
		}
		bits := framing.DibitsToBits(payload)
		off := slot*2*BurstDibits + ScrambleOriginBit
		for i := range bits {
			bits[i] ^= seq[(off+i)%SuperframeScrambleBits]
		}
		payload = framing.BitsToDibits(bits)
	}
	rsStart, msgBits, _ := acchLayout(bt.IsFast())
	return framing.CRC12P25P2OK(acchMessageFrom(acchSymbolBlock(payload, bt.IsFast()), rsStart, msgBits))
}
