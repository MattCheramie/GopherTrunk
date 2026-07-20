package lora

// Frame is one decoded LoRa PHY frame.
type Frame struct {
	SF         int // spreading factor 7..12
	CR         int // payload coding rate 1..4 (4/5..4/8)
	BW         Bandwidth
	Payload    []byte  // de-whitened plaintext payload (excludes CRC)
	HasCRC     bool    // a payload CRC was present
	CRCOK      bool    // payload CRC verified (true when HasCRC is false)
	HeaderOK   bool    // explicit header checksum passed
	LDRO       bool    // payload decoded in reduced-rate (Low Data Rate Optimize) mode
	CFOHz      float64 // estimated carrier-frequency offset
	SNRdB      float64 // dechirp peak/runner-up ratio of the header, in dB
	RawSymbols []uint16
}

// headerSymbolCount is the number of symbols the header block occupies
// (one block at the fixed header coding rate).
func headerSymbolCount() int { return blockSymbols(HeaderCR) }

// encodeBlock turns exactly blockNibbles(ppm) data nibbles into
// blockSymbols(cr) symbol values via Hamming encode -> diagonal interleave
// -> Gray encode. Short nibble slices are zero-padded. ppm is the number of
// interleaver rows: SF for a full-rate block, SF-2 for a reduced-rate
// (LDRO / header) block. The returned symbol values span [0, 2^ppm); the
// caller scales them into the [0, 2^SF) chip domain for a reduced-rate
// block (see encodeFrameSymbols).
func encodeBlock(nibbles []uint8, ppm, cr int) []uint16 {
	codewords := make([]uint16, ppm)
	for i := 0; i < ppm; i++ {
		var nib uint8
		if i < len(nibbles) {
			nib = nibbles[i] & 0x0f
		}
		codewords[i] = HammingEncode4(nib, cr)
	}
	interleaved := Interleave(codewords, ppm, cr)
	syms := make([]uint16, len(interleaved))
	for i, s := range interleaved {
		syms[i] = grayEncode(s)
	}
	return syms
}

// decodeBlock inverts encodeBlock: blockSymbols(cr) symbol values (in the
// [0, 2^ppm) domain) -> blockNibbles(ppm) nibbles. ok is false only on a
// malformed length.
func decodeBlock(syms []uint16, ppm, cr int) (nibbles []uint8, ok bool) {
	if len(syms) < blockSymbols(cr) {
		return nil, false
	}
	gray := make([]uint16, blockSymbols(cr))
	for i := range gray {
		gray[i] = grayDecode(syms[i])
	}
	codewords := Deinterleave(gray, ppm, cr)
	nibbles = make([]uint8, ppm)
	for i, cw := range codewords {
		nib, _ := HammingDecode4(cw, cr)
		nibbles[i] = nib
	}
	return nibbles, true
}

// reducedRateShift is the number of least-significant chip bits a
// reduced-rate (LDRO) symbol leaves unused: the SF-2 meaningful bits are
// carried in the high chip positions, so the transmitted bin is always a
// multiple of four. Encoding shifts the [0, 2^(SF-2)) block value up into
// the [0, 2^SF) chip domain; decoding shifts the demodulated bin back down,
// which is robust to the +/-1 bin drift LDRO exists to tolerate.
const reducedRateShift = 2

// encodeFrameSymbols builds the full symbol sequence for a frame: the
// header block followed by the payload blocks. payload is the plaintext.
// When ldro is set the payload blocks are reduced-rate (SF-2 rows), and
// their block values are scaled into the [0, 2^SF) chip domain so the
// modulator can chirp them like any other symbol. The header block is
// always full-rate here (see header.go / ldro.go for the scope note).
func encodeFrameSymbols(payload []byte, sf, cr int, hasCRC, ldro bool) []uint16 {
	// Header block (fixed CR), carrying the header fields zero-padded to a
	// full block.
	hdr := encodeHeader(len(payload), cr, hasCRC)
	syms := encodeBlock(hdr, sf, HeaderCR)

	// Payload body: whitened payload + optional (un-whitened) CRC.
	body := make([]byte, len(payload))
	copy(body, payload)
	Whiten(body)
	if hasCRC {
		crc := PayloadCRC(payload)
		body = append(body, byte(crc>>8), byte(crc))
	}
	nibs := bytesToNibbles(body)

	ppm := payloadPPM(sf, ldro)
	per := blockNibbles(ppm)
	for off := 0; off < len(nibs); off += per {
		end := off + per
		if end > len(nibs) {
			end = len(nibs)
		}
		blk := encodeBlock(nibs[off:end], ppm, cr)
		if ldro {
			for i := range blk {
				blk[i] <<= reducedRateShift
			}
		}
		syms = append(syms, blk...)
	}
	return syms
}

// decodeFrameFromSymbols decodes a header block plus payload blocks from a
// flat symbol stream. It reads the header first to learn the payload
// length and coding rate, then decodes exactly that many payload symbols.
// ldro selects the reduced-rate payload geometry; it is fixed by the
// (SF, BW) pair, not carried in the header (see ldro.go).
func decodeFrameFromSymbols(syms []uint16, sf int, ldro bool) (Frame, bool) {
	hc := headerSymbolCount()
	if len(syms) < hc {
		return Frame{}, false
	}
	hdrNibs, ok := decodeBlock(syms[:hc], sf, HeaderCR)
	if !ok {
		return Frame{}, false
	}
	hdr := ParseHeader(hdrNibs)
	f := Frame{SF: sf, CR: hdr.CR, HasCRC: hdr.HasCRC, HeaderOK: hdr.Valid, LDRO: ldro}
	if !hdr.Valid {
		return f, false
	}

	nBlocks := payloadBlockCount(sf, hdr.PayloadLen, hdr.HasCRC, ldro)
	spb := blockSymbols(hdr.CR)
	need := hc + nBlocks*spb
	if len(syms) < need {
		return f, false
	}

	ppm := payloadPPM(sf, ldro)
	var nibs []uint8
	for b := 0; b < nBlocks; b++ {
		start := hc + b*spb
		blk := syms[start : start+spb]
		if ldro {
			// Undo the reduced-rate chip-domain scaling: shift each bin back
			// into the [0, 2^(SF-2)) block domain (integer /4, tolerant of a
			// +/-1 bin drift).
			shifted := make([]uint16, len(blk))
			for i, s := range blk {
				shifted[i] = s >> reducedRateShift
			}
			blk = shifted
		}
		dec, ok := decodeBlock(blk, ppm, hdr.CR)
		if !ok {
			return f, false
		}
		nibs = append(nibs, dec...)
	}
	body := nibblesToBytes(nibs)

	bodyLen := hdr.PayloadLen
	if hdr.HasCRC {
		bodyLen += 2
	}
	if len(body) < bodyLen {
		return f, false
	}
	body = body[:bodyLen]

	payload := make([]byte, hdr.PayloadLen)
	copy(payload, body[:hdr.PayloadLen])
	Whiten(payload) // de-whiten (self-inverse)
	f.Payload = payload

	f.CRCOK = true
	if hdr.HasCRC {
		rx := uint16(body[hdr.PayloadLen])<<8 | uint16(body[hdr.PayloadLen+1])
		f.CRCOK = rx == PayloadCRC(payload)
	}
	f.RawSymbols = syms[:need]
	return f, true
}
