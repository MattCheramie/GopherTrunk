package dpmr

import "math/bits"

// The dPMR traffic-channel air format, as transmitted.
//
// Everything here follows ETSI TS 102 658 V2.6.1 (clauses 4.2.2, 5.2.1.1,
// 6.1, 7 and 14.2.2) and was cross-checked against two decoders that
// receive dPMR on the air and were written independently of each other:
// dsdcc (the SDRangel DSD demodulator, dpmr.cpp) and DSD-FME
// (dpmr_voice.c). The three agree on every constant below. Nothing was
// ported: the tables are generated from the standard's rules.
//
// A voice transmission is
//
//	preamble | header frame (FS1) | superframe ... superframe | end frame (FS3)
//
// and a superframe is four 80 ms payload frames of 192 dibits (384 bits):
//
//	frame 1: FS2 (12 dibits) | CCH (36) | TCH (36) x 4
//	frame 2: CC  (12 dibits) | CCH (36) | TCH (36) x 4
//	frame 3: FS2             | CCH      | TCH x 4
//	frame 4: CC              | CCH      | TCH x 4
//
// FS2 is the frame sync, CC the 24-bit channel code, CCH the 72-bit control
// channel and each TCH one 72-bit vocoder frame. The frame layer in sync.go,
// process.go and csbk.go is an older model that matches none of this; it
// still serves the Mode 3 control-channel path and is not the air format
// (a dPMR beacon sends SYScast and message frames, not a "CSBK").
//
// Dibits follow table 14.1: 01 is +1050 Hz, 00 is +350 Hz, 10 is -350 Hz
// and 11 is -1050 Hz, so a dibit's high bit is the sign of its symbol and
// its low bit says whether it is an outer one. A receiver that sees the
// spectrum inverted flips every high bit.

// Frame synchronisation sequences (clause 6.1).
const (
	// AirFS1 opens the header frame of a voice or data transmission.
	AirFS1 uint64 = 0x57FF5F75D577
	// AirFS2 opens payload frames 1 and 3 of every superframe.
	AirFS2 uint32 = 0x5FF77D
	// AirFS3 opens the end frame.
	AirFS3 uint32 = 0x7DDFF5
	// AirFS4 opens a packet-data header frame. It is the symbol-wise
	// complement of FS1, so FS1 and FS4 cannot tell a receiver its polarity.
	AirFS4 uint64 = 0xFD55F5DF7FDD
)

// Field lengths of a payload frame, in dibits.
const (
	// AirSyncDibits is the length of FS2, FS3 and the channel code.
	AirSyncDibits = 12
	// AirCCHDibits is the control channel: 72 bits.
	AirCCHDibits = 36
	// AirFrameDibits is one 80 ms payload frame: 384 bits.
	AirFrameDibits = AirSyncDibits + AirCCHDibits + TCHFieldDibits

	// airPairDibits is an FS2 frame and the channel-code frame after it,
	// the unit the framer works in: FS2 recurs every 384 dibits.
	airPairDibits = 2 * AirFrameDibits
)

// AirFS2Dibits returns the 12 dibits of FS2, first on the air first.
func AirFS2Dibits() []uint8 { return wordToDibits(uint64(AirFS2), AirSyncDibits) }

// AirFS3Dibits returns the 12 dibits of FS3.
func AirFS3Dibits() []uint8 { return wordToDibits(uint64(AirFS3), AirSyncDibits) }

// AirFS1Dibits returns the 24 dibits of FS1.
func AirFS1Dibits() []uint8 { return wordToDibits(AirFS1, 2*AirSyncDibits) }

// AirFS4Dibits returns the 24 dibits of FS4.
func AirFS4Dibits() []uint8 { return wordToDibits(AirFS4, 2*AirSyncDibits) }

func wordToDibits(w uint64, n int) []uint8 {
	out := make([]uint8, n)
	for i := range out {
		out[i] = uint8(w>>uint(2*(n-1-i))) & 3
	}
	return out
}

// airSignWord packs the high (sign) bits of the first 12 dibits of d,
// first dibit in bit 11.
func airSignWord(d []uint8) uint16 {
	var w uint16
	for i := 0; i < AirSyncDibits; i++ {
		w = w<<1 | uint16(d[i]>>1&1)
	}
	return w
}

// airOuterCount counts the outer symbols among the first 12 dibits of d.
// Every sync and channel-code symbol is an outer one.
func airOuterCount(d []uint8) int {
	n := 0
	for i := 0; i < AirSyncDibits; i++ {
		n += int(d[i] & 1)
	}
	return n
}

var (
	airFS2Sign = airSignWord(AirFS2Dibits())
	airFS3Sign = airSignWord(AirFS3Dibits())
)

// airSyncDistance is the number of the 12 dibits of d that differ from the
// 24-bit sync word.
func airSyncDistance(d []uint8, word uint32) int {
	n := 0
	for i := 0; i < AirSyncDibits; i++ {
		if d[i] != uint8(word>>uint(2*(AirSyncDibits-1-i)))&3 {
			n++
		}
	}
	return n
}

// airScramble is the scrambling sequence of clause 7.3: x^9 + x^5 + 1 with
// the register preset to all ones, restarted for every scrambled field.
// 72 bits cover a CCH; a message field takes 120 and a beacon's three
// SYScasts, scrambled as one block, 216 (clause 11.7).
var airScramble = func() [3 * 72]uint8 {
	var s [9]uint8
	for i := range s {
		s[i] = 1
	}
	var out [3 * 72]uint8
	for i := range out {
		out[i] = s[0]
		fb := s[4] ^ s[0]
		copy(s[:], s[1:])
		s[8] = fb
	}
	return out
}()

// hamming128Parity returns the parity bits C3..C0 of a data byte under the
// shortened Hamming (12,8) code of clause 7.2: the remainder of the byte
// times x^4 divided by x^4 + x + 1.
func hamming128Parity(d uint8) uint8 {
	r := uint16(d) << 4
	for bit := 11; bit >= 4; bit-- {
		if r&(1<<uint(bit)) != 0 {
			r ^= 0x13 << uint(bit-4)
		}
	}
	return uint8(r & 0xF)
}

// hamming128Fix maps a non-zero syndrome to the bit of the 12-bit codeword
// (data byte in bits 11..4, parity in 3..0) that a single error there
// produces, or 0 for the three syndromes no single error gives.
var hamming128Fix = func() [16]uint16 {
	var t [16]uint16
	for k := 0; k < 8; k++ {
		t[hamming128Parity(1<<uint(k))] = 1 << uint(k+4)
	}
	for k := 0; k < 4; k++ {
		t[1<<uint(k)] = 1 << uint(k)
	}
	return t
}()

// hamming128Decode corrects up to one bit of a 12-bit codeword. It returns
// the data byte, the number of bits it changed (0 or 1) and whether the
// word was a codeword or one error away from one.
func hamming128Decode(cw uint16) (data uint8, fixed int, ok bool) {
	syn := hamming128Parity(uint8(cw>>4)) ^ uint8(cw&0xF)
	if syn != 0 {
		flip := hamming128Fix[syn]
		if flip == 0 {
			return uint8(cw >> 4), 0, false
		}
		cw ^= flip
		fixed = 1
	}
	return uint8(cw >> 4), fixed, true
}

// airCRC7 is the CCH checksum of clause 7.1: x^7 + x^3 + 1 over the bits
// given, register preset to zero.
func airCRC7(b []uint8) uint8 {
	var sr uint8
	for _, v := range b {
		if (sr>>6^v)&1 != 0 {
			sr = (sr<<1 ^ 0x09) & 0x7F
		} else {
			sr = sr << 1 & 0x7F
		}
	}
	return sr
}

// Communication modes carried in the CCH (table 5.55).
const (
	AirModeVoice         = 0 // voice, no user data in the slow-data field
	AirModeVoiceSlowData = 1 // voice with slow data
	AirModeData1         = 2 // data without FEC
	AirModeData2         = 3 // data with FEC
	AirModeData3         = 4 // packet data
	AirModeVoiceData     = 5 // voice with attached data
)

// AirCCH is the control channel of one payload frame (tables 5.5 to 5.8).
type AirCCH struct {
	// FrameNumber is 0..3 for frames 1..4 of the superframe.
	FrameNumber uint8
	// IDHalf is twelve bits of an address: the called ID's upper and lower
	// half in frames 1 and 2, the calling ID's in frames 3 and 4.
	IDHalf uint16
	// Mode is the 3-bit communication mode (AirMode*).
	Mode uint8
	// Version and Format are the 2-bit version and communication-format
	// fields; Emergency is the emergency-priority bit.
	Version   uint8
	Format    uint8
	Emergency bool
	// SlowData is the 18-bit slow-data field.
	SlowData uint32
	// Corrected is the number of bits the Hamming decode changed.
	Corrected int
}

// Voice reports whether the frame's four TCH blocks are vocoder frames.
func (c AirCCH) Voice() bool {
	return c.Mode == AirModeVoice || c.Mode == AirModeVoiceSlowData || c.Mode == AirModeVoiceData
}

// DecodeAirCCH decodes the 36 dibits of a control channel as received:
// descramble, de-interleave (12 x 6), Hamming (12,8) on each of the six
// blocks, CRC7 (clause 7.5). ok is false when a block is not correctable
// or the checksum fails.
func DecodeAirCCH(d []uint8) (cch AirCCH, ok bool) {
	if len(d) != AirCCHDibits {
		return AirCCH{}, false
	}
	// On-air bit k is bit k/6 of block k%6: the transmitter writes the six
	// blocks down the columns of a 12 x 6 matrix and sends it row by row.
	var block [6]uint16
	for k := 0; k < 2*AirCCHDibits; k++ {
		b := d[k/2] >> uint(1-k%2) & 1
		b ^= airScramble[k]
		block[k%6] |= uint16(b) << uint(11-k/6)
	}
	var info [48]uint8
	good := true
	for i, cw := range block {
		data, fixed, valid := hamming128Decode(cw)
		good = good && valid
		cch.Corrected += fixed
		for j := 0; j < 8; j++ {
			info[8*i+j] = data >> uint(7-j) & 1
		}
	}
	var crc uint8
	for _, b := range info[41:] {
		crc = crc<<1 | b
	}
	if !good || airCRC7(info[:41]) != crc {
		return cch, false
	}
	field := func(from, n int) uint32 {
		var v uint32
		for _, b := range info[from : from+n] {
			v = v<<1 | uint32(b)
		}
		return v
	}
	cch.FrameNumber = uint8(field(0, 2))
	cch.IDHalf = uint16(field(2, 12))
	cch.Mode = uint8(field(14, 3))
	cch.Version = uint8(field(17, 2))
	cch.Format = uint8(field(19, 2))
	cch.Emergency = info[21] == 1
	cch.SlowData = field(23, 18)
	return cch, true
}

// EncodeAirCCH is the inverse of DecodeAirCCH: the 36 dibits a transmitter
// sends for cch. Corrected is ignored.
func EncodeAirCCH(cch AirCCH) []uint8 {
	var info [48]uint8
	put := func(from, n int, v uint32) {
		for i := 0; i < n; i++ {
			info[from+i] = uint8(v>>uint(n-1-i)) & 1
		}
	}
	put(0, 2, uint32(cch.FrameNumber))
	put(2, 12, uint32(cch.IDHalf))
	put(14, 3, uint32(cch.Mode))
	put(17, 2, uint32(cch.Version))
	put(19, 2, uint32(cch.Format))
	if cch.Emergency {
		info[21] = 1
	}
	put(23, 18, cch.SlowData)
	put(41, 7, uint32(airCRC7(info[:41])))

	var block [6]uint16
	for i := range block {
		var data uint8
		for j := 0; j < 8; j++ {
			data = data<<1 | info[8*i+j]
		}
		block[i] = uint16(data)<<4 | uint16(hamming128Parity(data))
	}
	out := make([]uint8, AirCCHDibits)
	for k := 0; k < 2*AirCCHDibits; k++ {
		b := uint8(block[k%6]>>uint(11-k/6)) & 1
		b ^= airScramble[k]
		out[k/2] |= b << uint(1-k%2)
	}
	return out
}

// AirChannelCode reads the 24-bit channel code from its 12 dibits. Every
// one of its symbols is an outer one, so the code is the twelve sign bits;
// outer is how many of the twelve symbols were received as outer ones,
// which is 12 on a clean frame.
func AirChannelCode(d []uint8) (code uint32, outer int) {
	for i := 0; i < AirSyncDibits; i++ {
		code = code<<2 | uint32(d[i]&2) | 1
	}
	return code, airOuterCount(d)
}

// airInvert flips the sign bit of every dibit, undoing an inverted
// spectrum.
func airInvert(d []uint8) {
	for i := range d {
		d[i] ^= 2
	}
}

// airSignDistance is the number of sign bits in which the first 12 dibits
// of d differ from a sync word's.
func airSignDistance(d []uint8, sign uint16) int {
	return bits.OnesCount16(airSignWord(d) ^ sign)
}
