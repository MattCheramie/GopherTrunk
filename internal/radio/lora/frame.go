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
	FECFixed   int     // Hamming codewords with a corrected single-bit error
	FECErrors  int     // Hamming codewords with a detected, uncorrected error
	RawSymbols []uint16
}

// encodeBlock turns up to rows nibbles into cwLen = 4+cr chip values:
// Hamming encode -> diagonal interleave -> Gray decode / +1 (with the x4
// scaling of a reduced-rate block). rows is SF for a full-rate block and
// SF-2 for a reduced-rate one (the first block, and every LDRO block).
// A short nibble slice is padded with zero codewords.
func encodeBlock(nibbles []uint8, sf, rows, cr int, reduced bool) []uint16 {
	codewords := make([]uint16, len(nibbles))
	for i, nib := range nibbles {
		codewords[i] = HammingEncode4(nib&0x0f, cr)
	}
	syms := Interleave(codewords, rows, 4+cr)
	for i, s := range syms {
		syms[i] = symbolToChip(s, sf, reduced)
	}
	return syms
}

// blockDecode is decodeBlock's result.
type blockDecode struct {
	nibbles      []uint8
	fixed, fails int
}

// decodeBlock inverts encodeBlock: 4+cr chip values -> rows nibbles.
func decodeBlock(chips []uint16, sf, rows, cr int, reduced bool) blockDecode {
	syms := make([]uint16, 4+cr)
	for i := range syms {
		syms[i] = chipToSymbol(chips[i], sf, reduced)
	}
	var out blockDecode
	for _, cw := range Deinterleave(syms, rows, 4+cr) {
		nib, c := HammingDecode4(cw, cr)
		switch c {
		case 1:
			out.fixed++
		case -1:
			out.fails++
		}
		out.nibbles = append(out.nibbles, nib)
	}
	return out
}

// encodeFrameSymbols builds the chip values for a frame's data symbols
// (everything after the SFD). payload is the plaintext. The nibble stream
// is header, whitened payload, then the un-whitened CRC; its first SF-2
// nibbles form the reduced-rate 4/8 first block and the rest are cut into
// blocks of payloadPPM(sf, ldro) nibbles at the payload coding rate.
func encodeFrameSymbols(payload []byte, sf, cr int, hasCRC, ldro bool) []uint16 {
	body := make([]byte, len(payload))
	copy(body, payload)
	Whiten(body)
	nibs := encodeHeader(len(payload), cr, hasCRC)
	nibs = append(nibs, bytesToNibbles(body)...)
	if hasCRC {
		nibs = append(nibs, crcNibbles(PayloadCRC(payload))...)
	}

	first := firstBlockNibbles(sf)
	if first > len(nibs) {
		first = len(nibs)
	}
	syms := encodeBlock(nibs[:first], sf, sf-2, HeaderCR, true)
	per := payloadPPM(sf, ldro)
	for off := first; off < len(nibs); off += per {
		end := off + per
		if end > len(nibs) {
			end = len(nibs)
		}
		syms = append(syms, encodeBlock(nibs[off:end], sf, per, cr, ldro)...)
	}
	return syms
}

// decodeHeaderBlock decodes the first HeaderSymbols chips: the header and
// the SF-7 payload nibbles that share its block.
func decodeHeaderBlock(chips []uint16, sf int) (Header, blockDecode, bool) {
	if len(chips) < HeaderSymbols {
		return Header{}, blockDecode{}, false
	}
	blk := decodeBlock(chips[:HeaderSymbols], sf, sf-2, HeaderCR, true)
	return ParseHeader(blk.nibbles), blk, true
}

// decodeFrameFromSymbols decodes a frame from its demodulated data-symbol
// chips. It reads the header block first to learn the payload length and
// coding rate, then decodes exactly that many payload symbols. ldro
// selects the reduced-rate payload geometry; it is fixed by the (SF, BW)
// pair, not carried in the header (see ldro.go).
func decodeFrameFromSymbols(chips []uint16, sf int, ldro bool) (Frame, bool) {
	hdr, first, ok := decodeHeaderBlock(chips, sf)
	if !ok {
		return Frame{}, false
	}
	f := Frame{SF: sf, CR: hdr.CR, HasCRC: hdr.HasCRC, HeaderOK: hdr.Valid, LDRO: ldro,
		FECFixed: first.fixed, FECErrors: first.fails}
	if !hdr.Valid {
		return f, false
	}
	need := FrameSymbolCount(sf, hdr.CR, hdr.PayloadLen, hdr.HasCRC, ldro)
	if len(chips) < need {
		return f, false
	}

	nibs := append([]uint8(nil), first.nibbles[HeaderNibbles:]...)
	per := payloadPPM(sf, ldro)
	spb := 4 + hdr.CR
	for start := HeaderSymbols; start < need; start += spb {
		blk := decodeBlock(chips[start:start+spb], sf, per, hdr.CR, ldro)
		f.FECFixed += blk.fixed
		f.FECErrors += blk.fails
		nibs = append(nibs, blk.nibbles...)
	}

	bodyNibs := 2 * hdr.PayloadLen
	if hdr.HasCRC {
		bodyNibs += 4
	}
	if len(nibs) < bodyNibs {
		return f, false
	}
	payload := nibblesToBytes(nibs[:2*hdr.PayloadLen])
	Whiten(payload) // de-whiten (self-inverse)
	f.Payload = payload

	f.CRCOK = true
	if hdr.HasCRC {
		c := nibs[2*hdr.PayloadLen:]
		rx := uint16(c[0]) | uint16(c[1])<<4 | uint16(c[2])<<8 | uint16(c[3])<<12
		f.CRCOK = rx == PayloadCRC(payload)
	}
	f.RawSymbols = chips[:need]
	return f, true
}
