package tetra

import "github.com/MattCheramie/GopherTrunk/internal/radio/framing"

// TCH/S full-rate speech traffic-channel coding, ETSI EN 300 395-2 V1.3.3 §5.5
// (normal, no frame-stealing). One transmission slot carries TWO 137-bit speech
// frames coded together into 432 type-5 bits. This file decodes the 432
// (descrambled) bits back to the two speech frames and encodes them (for the
// round-trip test). The 137-bit frames feed the ACELP speech decoder.
//
// Chain (encode): 2×137 speech → reorder to type-2 order (class0 102 ++ class1
// 112 ++ class2 60) → CRC(8) over class 2 → convolutional RCPC (rate 8/12 over
// class 1, 8/18 over class 2+CRC+tail, one continuous K=5 mother stream) →
// 432 type-3 → 24×18 matrix interleave → type-4. Scrambling (type-4→type-5) is
// handled by the traffic extractor; here we work in descrambled (type-4) bits.

const (
	tchSpeechFrameBits = 137
	tchType2SpeechN    = 274                                                      // 2 × 137
	tchType3Bits       = 432                                                      // = tchClass0Bits + 168 + 162
	tchClass1Coded     = 168                                                      // (2×56)  × 3/2   (rate 8/12)
	tchClass2Coded     = 162                                                      // (60+8+4) × 18/8 (rate 8/18)
	tchConvIn          = tchClass1Bits + tchClass2Bits + tchCRCBits + tchTailBits // 184
)

// crcTCHClass2 computes the 8 CRC bits protecting the 60 class-2 bits
// (§5.5.1): 7 parity bits from G(X)=1+X³+X⁷, then an 8th overall even-parity
// bit over the class-2 bits plus the 7 parity bits. class2 is 60 bit-per-byte.
func crcTCHClass2(class2 []byte) []byte {
	var reg [7]byte // reg[k] holds the x^k coefficient
	for _, b := range class2 {
		fb := (b & 1) ^ reg[6]
		reg = [7]byte{fb, reg[0], reg[1], reg[2] ^ fb, reg[3], reg[4], reg[5]}
	}
	out := make([]byte, tchCRCBits)
	var parity byte
	for k := 0; k < 7; k++ {
		out[k] = reg[k]
		parity ^= reg[k]
	}
	for _, b := range class2 {
		parity ^= b & 1
	}
	out[7] = parity
	return out
}

// speechToType2 reorders the two 137-bit speech frames into the 274 type-2
// speech bits (class0 ++ class1 ++ class2) per Table 5.
func speechToType2(frameA, frameB []byte) []byte {
	out := make([]byte, tchType2SpeechN)
	for i, sb := range tchType2SpeechOrder {
		if sb.Frame == 0 {
			out[i] = frameA[sb.Bit-1] & 1
		} else {
			out[i] = frameB[sb.Bit-1] & 1
		}
	}
	return out
}

// type2ToSpeech is the inverse of speechToType2.
func type2ToSpeech(type2 []byte) (frameA, frameB []byte) {
	frameA = make([]byte, tchSpeechFrameBits)
	frameB = make([]byte, tchSpeechFrameBits)
	for i, sb := range tchType2SpeechOrder {
		if sb.Frame == 0 {
			frameA[sb.Bit-1] = type2[i] & 1
		} else {
			frameB[sb.Bit-1] = type2[i] & 1
		}
	}
	return frameA, frameB
}

// tchInterleave maps 432 type-3 bits to 432 type-4 bits: C4(i·24+j)=C3(j·18+i)
// — reading a 24×18 matrix column by column (§5.5.3).
func tchInterleave(type3 []byte) []byte {
	out := make([]byte, tchType3Bits)
	for i := 0; i < 18; i++ {
		for j := 0; j < 24; j++ {
			out[i*24+j] = type3[j*18+i]
		}
	}
	return out
}

func tchDeinterleave(type4 []byte) []byte {
	out := make([]byte, tchType3Bits)
	for i := 0; i < 18; i++ {
		for j := 0; j < 24; j++ {
			out[j*18+i] = type4[i*24+j]
		}
	}
	return out
}

// EncodeTCHS codes two 137-bit speech frames (bit-per-byte) into a 54-byte
// packed type-4 (descrambled type-5) traffic frame. Mirrors DecodeTCHS for the
// round-trip test; not used in the live receive path.
func EncodeTCHS(frameA, frameB []byte) []byte {
	type2 := speechToType2(frameA, frameB)
	class0 := type2[:tchClass0Bits]
	class1 := type2[tchClass0Bits : tchClass0Bits+tchClass1Bits]
	class2 := type2[tchClass0Bits+tchClass1Bits:]

	conv := make([]byte, 0, tchConvIn)
	conv = append(conv, class1...)
	conv = append(conv, class2...)
	conv = append(conv, crcTCHClass2(class2)...)
	conv = append(conv, make([]byte, tchTailBits)...) // flush tail
	mother := framing.EncodeRCPCTetraMother(conv)     // 3 × 184 = 552

	split := 3 * tchClass1Bits // 336: mother bits from the class-1 input region
	c1 := framing.PunctureRCPCTetra(mother[:split], framing.RCPCTetraPeriod23, framing.RCPCTetraPuncture23, tchClass1Coded)
	c2 := framing.PunctureRCPCTetra(mother[split:], framing.RCPCTetraPeriod818, framing.RCPCTetraPuncture818, tchClass2Coded)

	type3 := make([]byte, 0, tchType3Bits)
	type3 = append(type3, class0...)
	type3 = append(type3, c1...)
	type3 = append(type3, c2...)
	return framing.PackBitsMSB(tchInterleave(type3))
}

// TCHSpeechFrames decodes one descrambled 54-byte type-5 traffic frame and, if
// its class-2 CRC verifies, returns the two 137-bit speech frames packed
// MSB-first (18 bytes each) ready for the ACELP vocoder. A frame that fails the
// CRC — a non-TCH/S burst (signalling on another timeslot) or a badly corrupted
// slot — returns nil. On a single-call carrier this CRC gate is what isolates
// the granted call's speech from the other TDMA timeslots' bursts: a burst that
// is not TCH/S speech descrambles to essentially random class-2 bits and passes
// the 8-bit CRC only ~1/256 of the time.
func TCHSpeechFrames(traffic []byte) [][]byte {
	frameA, frameB, crcOK, _, ok := DecodeTCHS(traffic)
	if !ok || !crcOK {
		return nil
	}
	return [][]byte{framing.PackBitsMSB(frameA), framing.PackBitsMSB(frameB)}
}

// DecodeTCHS decodes a 54-byte packed, descrambled type-5 (= type-4) traffic
// frame into two 137-bit speech frames (bit-per-byte). crcOK reports whether
// the class-2 CRC verified. errs is the Viterbi path metric (corrected bit
// count; 0 = clean). Returns ok=false if the frame is the wrong size.
func DecodeTCHS(frame []byte) (frameA, frameB []byte, crcOK bool, errs int, ok bool) {
	if len(frame)*8 < tchType3Bits {
		return nil, nil, false, 0, false
	}
	type4 := framing.UnpackBitsMSB(frame, tchType3Bits)
	type3 := tchDeinterleave(type4)

	class0 := type3[:tchClass0Bits]
	c1 := type3[tchClass0Bits : tchClass0Bits+tchClass1Coded]
	c2 := type3[tchClass0Bits+tchClass1Coded:]

	m1 := framing.DepunctureRCPCTetra(c1, framing.RCPCTetraPeriod23, framing.RCPCTetraPuncture23, 3*tchClass1Bits)
	m2 := framing.DepunctureRCPCTetra(c2, framing.RCPCTetraPeriod818, framing.RCPCTetraPuncture818, 3*(tchClass2Bits+tchCRCBits+tchTailBits))
	mother := append(append([]byte{}, m1...), m2...)

	conv, metric := framing.DecodeRCPCTetraMother(mother, tchConvIn)
	class1 := conv[:tchClass1Bits]
	class2 := conv[tchClass1Bits : tchClass1Bits+tchClass2Bits]
	crc := conv[tchClass1Bits+tchClass2Bits : tchClass1Bits+tchClass2Bits+tchCRCBits]

	type2 := make([]byte, 0, tchType2SpeechN)
	type2 = append(type2, class0...)
	type2 = append(type2, class1...)
	type2 = append(type2, class2...)
	frameA, frameB = type2ToSpeech(type2)

	crcOK = true
	want := crcTCHClass2(class2)
	for i := range want {
		if want[i] != (crc[i] & 1) {
			crcOK = false
			break
		}
	}
	return frameA, frameB, crcOK, metric, true
}
