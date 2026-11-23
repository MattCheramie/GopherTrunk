package dpmr

// The coding of a Mode 3 beacon channel's two kinds of slot, as
// transmitted (ETSI TS 102 658 V2.6.1 clauses 4.2.2.5, 4.2.3.7, 5.2, 7.6,
// 11.5 and 11.7).
//
// A beacon downlink is continuous and synchronous: framesets of 528 bits
// (264 dibits, 110 ms), each a message slot and a SYScast slot.
//
//	FS1 | MI0 | CC | MI1 | SYC1 | SYC2 | SYC3 | FS1 | MI0 | CC | MI1 | ...
//	 48   120   24   120     72     72     72    48
//
// The message slot carries one 72-bit Message_Frame field (MI) twice,
// either side of the channel code. Each copy is 72 bits + CRC8
// (x^8 + x^2 + x + 1) = 80 bits, ten Hamming (12,8) blocks, the 12 x 10
// interleave and the clause 7.3 scrambler restarted for the copy.
//
// The SYScast slot carries three 17-bit fields. Each is 17 bits + CRC7
// (x^7 + x^3 + 1) = 24 bits, three Hamming (12,8) blocks, sent twice (six
// blocks) through the 12 x 6 interleave; the three 72-bit results are
// scrambled as one 216-bit block, then FS1 follows. A BS that has more to
// say than a message holds may send an appended-data frame in the SYScast
// slot instead (DI0 | CC | DI1, coded like MI and carrying no FS1).
//
// dsdcc (f4exb/dsdcc dpmr.cpp, processHIn) and DSD-FME decode the 120-bit
// field this way for a traffic-channel header; neither decodes a beacon.

// Beacon field lengths, in dibits.
const (
	// BeaconFS1Dibits is the 48-bit FS1 that opens each message slot.
	BeaconFS1Dibits = 2 * AirSyncDibits
	// BeaconMIDibits is one 120-bit message field.
	BeaconMIDibits = 60
	// BeaconSYScastDibits is the 216-bit SYScast block.
	BeaconSYScastDibits = 108
	// BeaconSlotDibits is one 264-bit slot.
	BeaconSlotDibits = 132
	// BeaconFramesetDibits is a message slot and a SYScast slot: 110 ms.
	BeaconFramesetDibits = 2 * BeaconSlotDibits
)

// airCRC8 is the message checksum of clause 7.1: x^8 + x^2 + x + 1 over
// the bits given, register preset to zero and no final inversion, as
// dsdcc's checkCRC8 divides it.
func airCRC8(b []uint8) uint8 {
	var sr uint8
	for _, v := range b {
		if (sr>>7^v)&1 != 0 {
			sr = sr<<1 ^ 0x07
		} else {
			sr <<= 1
		}
	}
	return sr
}

// DecodeBeaconMI decodes one 60-dibit message field as received:
// descramble, de-interleave (on-air bit k is bit k/10 of block k%10),
// Hamming (12,8) on each block, CRC8. bits are the 72 information bits,
// first transmitted first; corrected counts the bits the Hamming decode
// changed. ok is false when a block is uncorrectable or the CRC fails.
func DecodeBeaconMI(d []uint8) (bits [72]uint8, corrected int, ok bool) {
	if len(d) != BeaconMIDibits {
		return bits, 0, false
	}
	var block [10]uint16
	for k := 0; k < 2*BeaconMIDibits; k++ {
		b := d[k/2]>>uint(1-k%2)&1 ^ airScramble[k]
		block[k%10] |= uint16(b) << uint(11-k/10)
	}
	var info [80]uint8
	good := true
	for i, cw := range block {
		data, fixed, valid := hamming128Decode(cw)
		good = good && valid
		corrected += fixed
		for j := 0; j < 8; j++ {
			info[8*i+j] = data >> uint(7-j) & 1
		}
	}
	copy(bits[:], info[:72])
	var crc uint8
	for _, b := range info[72:] {
		crc = crc<<1 | b
	}
	return bits, corrected, good && airCRC8(info[:72]) == crc
}

// EncodeBeaconMI is the inverse of DecodeBeaconMI: the 60 dibits a beacon
// sends for 72 information bits.
func EncodeBeaconMI(bits [72]uint8) []uint8 {
	var info [80]uint8
	copy(info[:], bits[:])
	crc := airCRC8(bits[:])
	for i := 0; i < 8; i++ {
		info[72+i] = crc >> uint(7-i) & 1
	}
	var block [10]uint16
	for i := range block {
		var data uint8
		for j := 0; j < 8; j++ {
			data = data<<1 | info[8*i+j]
		}
		block[i] = uint16(data)<<4 | uint16(hamming128Parity(data))
	}
	out := make([]uint8, BeaconMIDibits)
	for k := 0; k < 2*BeaconMIDibits; k++ {
		b := uint8(block[k%10]>>uint(11-k/10))&1 ^ airScramble[k]
		out[k/2] |= b << uint(1-k%2)
	}
	return out
}

// SYScastField is one decoded 17-bit SYScast field.
type SYScastField struct {
	Bits uint32 // the 17 information bits, first transmitted in bit 16
	OK   bool   // a copy decoded with a valid CRC7
	// Corrected is the number of bits the Hamming decode changed in the
	// copy that was used.
	Corrected int
}

// DecodeBeaconSYScast decodes the 108-dibit SYScast block into its three
// fields. Each field is sent twice inside its 72 bits; the first copy
// whose blocks are correctable and whose CRC7 holds is used.
func DecodeBeaconSYScast(d []uint8) (fields [3]SYScastField) {
	if len(d) != BeaconSYScastDibits {
		return fields
	}
	for f := 0; f < 3; f++ {
		var block [6]uint16
		for k := 0; k < 72; k++ {
			n := 72*f + k
			b := d[n/2]>>uint(1-n%2)&1 ^ airScramble[n]
			block[k%6] |= uint16(b) << uint(11-k/6)
		}
		for c := 0; c < 2 && !fields[f].OK; c++ {
			var info [24]uint8
			good, fixedAll := true, 0
			for i := 0; i < 3; i++ {
				data, fixed, valid := hamming128Decode(block[3*c+i])
				good = good && valid
				fixedAll += fixed
				for j := 0; j < 8; j++ {
					info[8*i+j] = data >> uint(7-j) & 1
				}
			}
			var crc uint8
			for _, b := range info[17:] {
				crc = crc<<1 | b
			}
			if !good || airCRC7(info[:17]) != crc {
				continue
			}
			var v uint32
			for _, b := range info[:17] {
				v = v<<1 | uint32(b)
			}
			fields[f] = SYScastField{Bits: v, OK: true, Corrected: fixedAll}
		}
	}
	return fields
}

// EncodeBeaconSYScast is the inverse of DecodeBeaconSYScast: the 108
// dibits a beacon sends for three 17-bit fields (FS1 not included).
func EncodeBeaconSYScast(fields [3]uint32) []uint8 {
	out := make([]uint8, BeaconSYScastDibits)
	for f := 0; f < 3; f++ {
		var info [24]uint8
		for i := 0; i < 17; i++ {
			info[i] = uint8(fields[f]>>uint(16-i)) & 1
		}
		crc := airCRC7(info[:17])
		for i := 0; i < 7; i++ {
			info[17+i] = crc >> uint(6-i) & 1
		}
		var block [6]uint16
		for i := 0; i < 3; i++ {
			var data uint8
			for j := 0; j < 8; j++ {
				data = data<<1 | info[8*i+j]
			}
			block[i] = uint16(data)<<4 | uint16(hamming128Parity(data))
			block[i+3] = block[i]
		}
		for k := 0; k < 72; k++ {
			n := 72*f + k
			b := uint8(block[k%6]>>uint(11-k/6))&1 ^ airScramble[n]
			out[n/2] |= b << uint(1-n%2)
		}
	}
	return out
}

// beaconFS1Distance counts the dibits of the first 24 of d that differ
// from FS1.
func beaconFS1Distance(d []uint8) int {
	n := 0
	for i := 0; i < BeaconFS1Dibits; i++ {
		if d[i] != uint8(AirFS1>>uint(2*(BeaconFS1Dibits-1-i)))&3 {
			n++
		}
	}
	return n
}

// BeaconChannelCode0 is channel code number 0 of table 6.1, 57 5F 77.
const BeaconChannelCode0 uint32 = 0x575F77

// EncodeBeaconFrameset builds the 264 dibits of one frameset as a beacon
// sends it: the message twice either side of the channel code, the three
// SYScast fields, and the FS1 that opens the next message. A stream of
// these needs one FS1 in front of the first.
func EncodeBeaconFrameset(m BeaconMessage, channelCode uint32, syscast [3]uint32) []uint8 {
	mi := EncodeBeaconMI(m.Bits)
	out := make([]uint8, 0, BeaconFramesetDibits)
	out = append(out, mi...)
	out = append(out, wordToDibits(uint64(channelCode), AirSyncDibits)...)
	out = append(out, mi...)
	out = append(out, EncodeBeaconSYScast(syscast)...)
	return append(out, AirFS1Dibits()...)
}

// EncodeBeaconAppendedFrameset is a frameset whose SYScast slot carries an
// appended-data frame (clause 4.2.2.6): no SYScast and no FS1 before the
// next message.
func EncodeBeaconAppendedFrameset(m BeaconMessage, channelCode uint32, appended [72]uint8) []uint8 {
	mi := EncodeBeaconMI(m.Bits)
	di := EncodeBeaconMI(appended)
	cc := wordToDibits(uint64(channelCode), AirSyncDibits)
	out := make([]uint8, 0, BeaconFramesetDibits)
	out = append(out, mi...)
	out = append(out, cc...)
	out = append(out, mi...)
	out = append(out, di...)
	out = append(out, cc...)
	return append(out, di...)
}
