package phase1

import "time"

// Motorola P25 Phase 1 voice-channel talker-alias reassembly and
// proprietary-cipher decode.
//
// Real Motorola P25 systems do NOT emit the TIA-102.AABF "standard"
// HEADER+BLOCK1+BLOCK2 form that an earlier working model assumed
// (issue #376). They emit a Motorola-specific variant carried on
// the voice channel as Link Control words:
//
//	LCO 0x15 — Motorola talker-alias header
//	LCO 0x17 — Motorola talker-alias data block (variable count)
//
// LCO 0x16 from the TIA-102.AABF spec is not used by the Motorola
// form; if a voice chain ever observes one, this assembler ignores
// it.
//
// Carriage note (#376): SDRTrunk ground truth on Victorian MMR shows
// these 0x15/0x17 LCs ride primarily in the TERMINATOR link control
// (TDULC, DUID 0xF), not LDU1. p25p1_voice.go currently feeds this
// buffer only from LDU1 and returns early on the terminator, so the
// TDULC-borne aliases are missed until a TDULC LC extractor is added
// — fixture-backed follow-up. This buffer's reassembly is carriage-
// independent, so the future TDULC path can reuse it unchanged.
//
// The 9-octet LC content layouts (HEADER + DATA BLOCK), the message
// framing (WACN + System + RadioID + encoded alias + CRC), and the
// per-byte obfuscation cipher are documented publicly in the
// reverse-engineering work of the SDRTrunk project. GopherTrunk's
// implementation is a clean-room Go port of the algorithm — the
// expression here is original, the algorithm is treated as a fact
// about Motorola's wire protocol.
//
// HEADER content layout (octet 0 is the LCFormat byte, already
// known to be LCOTalkerAliasHeader before AddHeader is called):
//
//	octet 1     : MFID
//	octets 2-3  : talkgroup ID (big-endian 16-bit)
//	octet 4     : block count — number of LCO 0x17 data blocks that
//	              follow before the alias is complete
//	octet 5     : format identifier (1 = Unicode, observed)
//	octet 6     : 0x00 (reserved / unknown)
//	octet 7..8  : 4-bit sequence number + 12-bit checksum, packed
//	              high-nibble-first into octet 7
//
// DATA BLOCK content layout:
//
//	octet 1     : MFID
//	octet 2     : block number (1-based; 1..block_count)
//	octet 3 hi  : 4-bit sequence number (must match the header's)
//	octet 3 lo +
//	  octets 4-8: 44-bit fragment payload
//
// Per-block fragment contribution: 44 bits. The full reassembled
// message is N × 44 bits where N = block_count from the header.
// That message is then framed:
//
//	bits 0..19   : WACN
//	bits 20..31  : System ID
//	bits 32..55  : Radio (subscriber) ID
//	bits 56..end-16 : encoded alias bytes
//	last 16 bits : CRC-16/GSM (currently informational; not verified).
//	               Validating it would reject the junk alias a
//	               partial/garbled block sequence can otherwise surface
//	               (#376) — a fixture-backed hardening, pending a real
//	               frame to confirm the CRC parameters.
//
// Encoded alias bytes are byte-aligned starting at bit 56. Each
// encoded byte passes through the cipher (see decodeAliasBytes) to
// produce one decoded byte; pairs of decoded bytes form one UTF-16
// big-endian character. ASCII aliases come through with a 0x00 high
// byte that the existing cleanAlias filter strips.

// Motorola alias assembler bounds.
const (
	// motorolaAliasStaleAfter is how long an incomplete alias is
	// kept before the next AddDataBlock evicts it. Voice-channel
	// LCs ride at ~9 LDU1s per second; a 10s gap covers a brief
	// decoder stall without dragging stale fragments forward.
	motorolaAliasStaleAfter = 10 * time.Second

	// motorolaMaxBlocks caps the block count an alias header may
	// declare. Real aliases are 1..16 blocks; the cap stops a
	// corrupted header byte from preallocating gigabytes.
	motorolaMaxBlocks = 32

	// motorolaFragmentBits is the bit-width each data block
	// contributes to the reassembled message.
	motorolaFragmentBits = 44

	// motorolaSUIDBits = WACN(20) + System(12) + RadioID(24).
	motorolaSUIDBits = 56

	// motorolaCRCBits is the trailing CRC-16/GSM field.
	motorolaCRCBits = 16
)

// MotorolaTalkerAliasBuf reassembles the Motorola voice-channel
// talker alias for one active call. Construct one per voice chain;
// each chain owns exactly one buffer and is single-goroutine.
//
// Call AddFragment with every LCO 0x15 / 0x17 LC the chain observes;
// AddFragment returns the decoded alias once the header's declared
// block_count has arrived with matching sequence numbers. The same
// alias is not returned twice (per-emission fingerprint dedupe).
type MotorolaTalkerAliasBuf struct {
	now func() time.Time

	haveHeader  bool
	blockCount  uint8
	sequence    uint8
	talkgroupID uint16

	// blocks[i] holds the 44-bit fragment of data block i+1, or nil
	// if not yet received. Indexed 0..blockCount-1.
	blocks [][]byte

	lastUpdate  time.Time
	emittedHash uint64
}

// NewMotorolaTalkerAliasBuf returns a ready buffer. now is
// injectable for tests; nil defaults to time.Now.
func NewMotorolaTalkerAliasBuf(now func() time.Time) *MotorolaTalkerAliasBuf {
	if now == nil {
		now = time.Now
	}
	return &MotorolaTalkerAliasBuf{now: now}
}

// AddFragment feeds one talker-alias LC's raw 9-octet content into
// the buffer. lcf must be LCOTalkerAliasHeader (0x15) or
// LCOTalkerAliasBlock2 (0x17, what Motorola uses as a data block).
// Other LCOs are ignored.
//
// Returns (alias, true) when the header + every data block has
// arrived with consistent sequence numbers; ("", false) otherwise.
// Repeated emission of the same alias inside one sequence is
// suppressed — a new header (different sequence number) resets the
// dedupe fingerprint.
func (b *MotorolaTalkerAliasBuf) AddFragment(lcf uint8, content [lcContentOctets]byte) (string, bool) {
	now := b.now()
	if !b.lastUpdate.IsZero() && now.Sub(b.lastUpdate) > motorolaAliasStaleAfter {
		b.resetExceptEmitted()
	}
	switch lcf {
	case LCOTalkerAliasHeader:
		count := content[4]
		seq := content[7] >> 4
		if count == 0 || count > motorolaMaxBlocks {
			return "", false
		}
		// A header with a different sequence number is a new alias
		// transmission — drop any in-flight data blocks from the
		// previous one so they don't blend.
		if !b.haveHeader || seq != b.sequence || count != b.blockCount {
			b.blocks = make([][]byte, count)
		}
		b.haveHeader = true
		b.blockCount = count
		b.sequence = seq
		b.talkgroupID = uint16(content[2])<<8 | uint16(content[3])
	case LCOTalkerAliasBlock2:
		// LCO 0x17 in Motorola = data block. block_number is 1-based;
		// sequence in octet 3 high nibble must match the header.
		blockNum := content[2]
		seq := content[3] >> 4
		if blockNum == 0 || !b.haveHeader || seq != b.sequence ||
			blockNum > b.blockCount {
			return "", false
		}
		// Fragment = low nibble of octet 3 (4 bits) + octets 4..8
		// (40 bits) = 44 bits total. Pack into 6 bytes left-aligned
		// for later bit-stream concatenation.
		frag := make([]byte, 6)
		frag[0] = content[3] & 0x0F   // 4 bits of payload
		copy(frag[1:6], content[4:9]) // 40 bits of payload
		b.blocks[blockNum-1] = frag
	default:
		return "", false
	}
	b.lastUpdate = now

	if !b.haveHeader {
		return "", false
	}
	for _, blk := range b.blocks {
		if blk == nil {
			return "", false
		}
	}

	bits := assembleMotorolaBitStream(b.blocks)
	alias := decodeMotorolaAlias(bits)
	if alias == "" {
		return "", false
	}
	h := aliasFingerprint(alias)
	if h == b.emittedHash {
		return "", false
	}
	b.emittedHash = h
	return alias, true
}

// Reset clears the buffer including the de-duplication fingerprint
// — the voice composer calls this when a new call starts on the
// same device so the previous call's alias doesn't leak forward.
func (b *MotorolaTalkerAliasBuf) Reset() {
	b.resetExceptEmitted()
	b.emittedHash = 0
}

func (b *MotorolaTalkerAliasBuf) resetExceptEmitted() {
	b.haveHeader = false
	b.blockCount = 0
	b.sequence = 0
	b.talkgroupID = 0
	b.blocks = nil
	b.lastUpdate = time.Time{}
}

// assembleMotorolaBitStream concatenates the 44-bit fragments from
// each data block (in block-number order) into a contiguous bit
// stream packed into bytes, MSB-first. Each input fragment is
// already stored left-aligned in 6 bytes (4 bits in [0], 40 bits in
// [1:6]).
func assembleMotorolaBitStream(blocks [][]byte) []byte {
	// Each block contributes motorolaFragmentBits bits = 44.
	totalBits := len(blocks) * motorolaFragmentBits
	out := make([]byte, (totalBits+7)/8)
	bitPos := 0
	for _, blk := range blocks {
		// Bit 0..3 from blk[0] low nibble, then 40 bits from blk[1..6].
		for i := 0; i < 4; i++ {
			if (blk[0]>>(3-i))&1 == 1 {
				out[bitPos/8] |= 1 << (7 - bitPos%8)
			}
			bitPos++
		}
		for j := 1; j < 6; j++ {
			for i := 0; i < 8; i++ {
				if (blk[j]>>(7-i))&1 == 1 {
					out[bitPos/8] |= 1 << (7 - bitPos%8)
				}
				bitPos++
			}
		}
	}
	return out
}

// aliasFingerprint is a cheap FNV-1a hash used only to suppress
// repeat emissions of the same alias inside one sequence.
func aliasFingerprint(s string) uint64 {
	const (
		offset uint64 = 14695981039346656037
		prime  uint64 = 1099511628211
	)
	h := offset
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= prime
	}
	return h
}

// decodeMotorolaAlias extracts the encoded alias bytes from the
// reassembled bit stream, runs them through the proprietary
// per-byte cipher, then filters the decoded UTF-16 BE characters
// to printable ASCII via cleanAlias. Empty result on too-short
// input or all-non-printable decoded bytes.
func decodeMotorolaAlias(bits []byte) string {
	totalBits := len(bits) * 8
	// Encoded alias starts at bit motorolaSUIDBits (after WACN +
	// System + Radio ID) and runs until motorolaCRCBits before the
	// end. Must be byte-aligned and at least 2 bytes (one decoded
	// character).
	if totalBits < motorolaSUIDBits+motorolaCRCBits+16 {
		return ""
	}
	aliasBits := totalBits - motorolaSUIDBits - motorolaCRCBits
	if aliasBits%8 != 0 {
		// Trim to the nearest whole byte; a real header always
		// produces a byte-aligned alias section, so a non-aligned
		// bit count signals truncation.
		aliasBits -= aliasBits % 8
	}
	encStart := motorolaSUIDBits / 8 // SUID is exactly 7 bytes
	encByteLen := aliasBits / 8
	if encStart+encByteLen > len(bits) {
		return ""
	}
	encoded := bits[encStart : encStart+encByteLen]
	decoded := decodeAliasBytes(encoded)
	return cleanAlias(decoded)
}
