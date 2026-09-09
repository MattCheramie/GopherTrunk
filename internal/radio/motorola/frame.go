package motorola

// frame.go is the Motorola Type II / SmartZone OSW wire codec: the
// 84-bit control-channel frame (8-bit sync + 76 coded payload bits)
// and the interleave → convolutional-parity ECC → CRC-10 chain that
// reduces the payload to one 27-bit OSW.
//
// Every constant and transform here is ported from OP25's
// rx_smartnet.cc / rx_smartnet.h (Graham J. Norbury, itself derived
// from gr-smartnet and mottrunk.txt) — an implementation proven
// against live SmartNet/SmartZone systems — NOT from prose specs.
// The previous GopherTrunk framing (24-bit sync 0xA4D7AA + two
// BCH(64,16,11) codewords around a 32-bit OSW) matched no real
// reference and could never decode an on-air control channel
// (issue #1143); only its own synthetic fixtures decoded, the
// #764/#771 self-consistent-synthetic trap.
//
// Wire layout, MSB-first throughout:
//
//	[ 8-bit sync 10101100 ] [ 76-bit coded payload ]
//
// A frame is only trusted when the NEXT frame's sync arrives exactly
// 76 bits after the previous one ended (frames run back-to-back on
// the control channel), which together with the CRC-10 makes the
// short 8-bit sync safe. The 76 payload bits deinterleave (stride
// 19) into 38 (info, parity) pairs with parity[i] = info[i] ^
// info[i-1]; a doubled parity syndrome corrects single info-bit
// errors. The first 37 corrected bits are 27 data bits + 10 CRC
// bits (the 38th pair is spare). Data rides inverted on the wire:
// the address/command fields un-invert via the fixed XOR masks
// below and the CRC field is bitwise-complemented.
const (
	// SyncBits is the length of the frame sync preceding each OSW.
	SyncBits = 8
	// OutboundSyncHex is the 8-bit outbound sync word 10101100.
	OutboundSyncHex uint32 = 0xAC
	// PayloadBits is the coded payload length following sync.
	PayloadBits = 76
	// FrameBits is the full frame length (sync + payload).
	FrameBits = SyncBits + PayloadBits
	// oswDataBits is the OSW information length after ECC:
	// 16-bit address + 1 group bit + 10-bit command.
	oswDataBits = 27
	// oswCRCBits is the CRC field length after the data bits.
	oswCRCBits = 10
	// eccBits is how many corrected bits the ECC stage yields
	// (data + CRC; the 38th interleaved pair is unused spare).
	eccBits = oswDataBits + oswCRCBits

	// idXORMask / cmdXORMask un-invert the address and command
	// fields (the wire carries the data bits inverted; XOR with
	// the complemented masks recovers the true values in one
	// step). From rx_smartnet.h: ID_XOR 0x33C7, CMD_XOR 0x32A.
	idXORMask  uint16 = ^uint16(0x33C7)        // 0xCC38
	cmdXORMask uint16 = ^uint16(0x32A) & 0x3FF // 0x0D5

	// CRC-10 registers (rx_smartnet.cc crc_check).
	crcInit uint16 = 0x0393
	crcOp   uint16 = 0x036E
	crcPoly uint16 = 0x0225
)

// OutboundSyncBits returns the 8 bits of the outbound sync MSB-first.
func OutboundSyncBits() []byte {
	out := make([]byte, SyncBits)
	for i := 0; i < SyncBits; i++ {
		out[i] = byte((OutboundSyncHex >> uint(SyncBits-1-i)) & 1)
	}
	return out
}

// deinterleave76 undoes the transmit interleave. The wire order is
// payload[k + l*19] = seq[k*4 + l] for k in [0,19), l in [0,4); the
// receive side reads seq[k*4+l] = payload[k + l*19] (rx_smartnet.cc
// deinterleave).
func deinterleave76(dst, payload []byte) []byte {
	if cap(dst) < PayloadBits {
		dst = make([]byte, PayloadBits)
	} else {
		dst = dst[:PayloadBits]
	}
	for k := 0; k < PayloadBits/4; k++ {
		for l := 0; l < 4; l++ {
			dst[k*4+l] = payload[k+l*19] & 1
		}
	}
	return dst
}

// interleave76 is the transmit-side inverse of deinterleave76, used
// by the fixture encoder.
func interleave76(dst, seq []byte) []byte {
	if cap(dst) < PayloadBits {
		dst = make([]byte, PayloadBits)
	} else {
		dst = dst[:PayloadBits]
	}
	for k := 0; k < PayloadBits/4; k++ {
		for l := 0; l < 4; l++ {
			dst[k+l*19] = seq[k*4+l] & 1
		}
	}
	return dst
}

// eccDecode76 runs the convolutional-parity error correction over the
// deinterleaved 38 (info, parity) pairs and returns the first eccBits
// (37) corrected info bits plus how many it flipped. parity[i] =
// info[i] ^ info[i-1] on the wire; two consecutive flipped parity
// syndromes pinpoint a flipped info bit between them (rx_smartnet.cc
// error_correction).
func eccDecode76(dst, raw []byte) ([]byte, int) {
	if cap(dst) < eccBits {
		dst = make([]byte, eccBits)
	} else {
		dst = dst[:eccBits]
	}
	var syndrome [PayloadBits]byte
	// syndrome of the even (info) positions is zero by construction;
	// only the parity positions carry information.
	syndrome[1] = raw[1] ^ raw[0]
	for k := 2; k < PayloadBits; k += 2 {
		syndrome[k+1] = raw[k+1] ^ (raw[k]^raw[k-2])&1
	}
	flips := 0
	for k := 0; k < eccBits; k++ {
		if syndrome[2*k+1]&1 != 0 && syndrome[2*k+3]&1 != 0 {
			dst[k] = ^raw[2*k] & 1
			flips++
		} else {
			dst[k] = raw[2*k] & 1
		}
	}
	return dst, flips
}

// eccEncode expands info bits (eccBits+1 = 38, the last being the
// spare) into 76 interleaver-order (info, parity) pairs.
func eccEncode(dst, info []byte) []byte {
	if cap(dst) < PayloadBits {
		dst = make([]byte, PayloadBits)
	} else {
		dst = dst[:PayloadBits]
	}
	prev := byte(0)
	for k := 0; k < PayloadBits/2; k++ {
		b := info[k] & 1
		dst[2*k] = b
		dst[2*k+1] = b ^ prev
		prev = b
	}
	return dst
}

// crc10 computes the CRC-10 over the 27 data bits as they appear in
// the corrected (still-inverted) stream (rx_smartnet.cc crc_check).
func crc10(data []byte) uint16 {
	accum := crcInit
	op := crcOp
	for j := 0; j < oswDataBits; j++ {
		if op&1 != 0 {
			op = (op >> 1) ^ crcPoly
		} else {
			op >>= 1
		}
		if data[j]&1 != 0 {
			accum ^= op
		}
	}
	return accum
}

// DecodeOSWPayload runs the full 76-bit payload → OSW chain:
// deinterleave, ECC, CRC-10 check, field extraction with the wire
// inversion undone. Returns ok=false when the CRC rejects the frame.
func DecodeOSWPayload(payload []byte) (OSW, bool) {
	osw, _, ok := DecodeOSWPayloadDetail(payload)
	return osw, ok
}

// DecodeOSWPayloadDetail is DecodeOSWPayload with the ECC flip count
// exposed, for the siglab FEC explorer's clean/corrected tally.
func DecodeOSWPayloadDetail(payload []byte) (OSW, int, bool) {
	if len(payload) != PayloadBits {
		return OSW{}, 0, false
	}
	var deintBuf, eccBuf [PayloadBits]byte
	raw := deinterleave76(deintBuf[:0], payload)
	corrected, flips := eccDecode76(eccBuf[:0], raw)

	// CRC field rides inverted on the wire.
	var given uint16
	for j := 0; j < oswCRCBits; j++ {
		given = (given << 1) | uint16(^corrected[oswDataBits+j]&1)
	}
	if given != crc10(corrected) {
		return OSW{}, flips, false
	}

	var addr uint16
	for j := 0; j < 16; j++ {
		addr = (addr << 1) | uint16(corrected[j]&1)
	}
	var cmd uint16
	for j := 17; j < oswDataBits; j++ {
		cmd = (cmd << 1) | uint16(corrected[j]&1)
	}
	return OSW{
		Address: addr ^ idXORMask,
		Group:   corrected[16]&1 == 0,
		Command: cmd ^ cmdXORMask,
	}, flips, true
}

// EncodeOSWFrame renders one OSW as its FrameBits (84) on-air bits:
// sync + interleaved, parity-expanded, CRC'd, inverted payload. The
// inverse of DecodeOSWPayload, used by fixtures and tests. Note a
// decoder needs the NEXT frame's sync too — emit frames back-to-back
// (or append one trailing sync) so the last frame validates.
func EncodeOSWFrame(o OSW) []byte {
	var info [PayloadBits / 2]byte
	addr := o.Address ^ idXORMask
	for j := 0; j < 16; j++ {
		info[j] = byte(addr>>uint(15-j)) & 1
	}
	if !o.Group {
		info[16] = 1
	}
	cmd := (o.Command & 0x3FF) ^ cmdXORMask
	for j := 0; j < oswCRCBits; j++ {
		info[17+j] = byte(cmd>>uint(9-j)) & 1
	}
	crc := crc10(info[:oswDataBits])
	for j := 0; j < oswCRCBits; j++ {
		info[oswDataBits+j] = ^byte(crc>>uint(oswCRCBits-1-j)) & 1
	}
	// info[37] is the spare bit the decoder never reads; leave 0.

	var pairBuf, wireBuf [PayloadBits]byte
	pairs := eccEncode(pairBuf[:0], info[:])
	payload := interleave76(wireBuf[:0], pairs)

	out := make([]byte, 0, FrameBits)
	out = append(out, OutboundSyncBits()...)
	out = append(out, payload...)
	return out
}
