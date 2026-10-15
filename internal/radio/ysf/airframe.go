package ysf

import "math/bits"

// This file decodes a real System Fusion frame as it arrives off air: the
// FICH (frame information channel) and the V/D mode 2 voice channels.
//
// Layout and coding follow MMDVMHost (YSFFICH.cpp, YSFPayload.cpp,
// YSFConvolution.cpp, CRC.cpp) and its YSF2DMR mode converter, read for
// protocol facts only. The tests pin them with literal on-air vectors from
// the reference encoder.
//
// On air the FICH is 100 DIBITS: four Golay(24,12) words (32 info bits +
// a CRC-16) behind a rate-1/2 K=5 convolutional code, interleaved over 200
// bits. (An earlier fich.go modelled 48 convolutionally coded bits in 100
// bits; its encoder and decoder shared the invented layout, so its tests
// passed while no real frame decoded. It was removed in issue #1261; the
// voice chain and the control channel both decode through DecodeAirFICH.)

// AirFICH is a decoded on-air FICH, with MMDVMHost's field names.
type AirFICH struct {
	FI   uint8 // frame indicator: 0 header, 1 communications, 2 terminator, 3 test
	CS   uint8 // callsign field size
	CM   uint8 // call mode: 0 group 1, 1 group 2, 3 individual
	BN   uint8 // block number
	BT   uint8 // block total
	FN   uint8 // frame number within the superframe
	FT   uint8 // frame total
	DT   uint8 // data type: 0 V/D mode 1, 1 data FR, 2 V/D mode 2, 3 voice FR
	MR   uint8 // message route: 0 direct, 1 not busy, 2 busy
	Dev  bool
	DGID uint8 // digital group ID
}

// FICH frame indicators and data types (MMDVMHost YSFDefines.h).
const (
	AirFIHeader     = 0
	AirFIComms      = 1
	AirFITerminator = 2

	AirDTVDMode1  = 0
	AirDTDataFR   = 1
	AirDTVDMode2  = 2
	AirDTVoiceFR  = 3
	airFICHDibits = 100
)

// airFICHInterleave maps the i-th received symbol pair to its first bit in
// the 200-bit FICH field: 20 rows of 5, each row 2 bits on, each column 40
// bits on (MMDVMHost YSFFICH.cpp INTERLEAVE_TABLE).
func airFICHInterleave(i int) int { return (i%5)*40 + (i/5)*2 }

// DecodeAirFICH decodes the 100 FICH dibits that follow a frame's sync.
// ok is false when a Golay word is beyond correction or the CRC fails.
func DecodeAirFICH(dibits []uint8) (f AirFICH, ok bool) {
	if len(dibits) < airFICHDibits {
		return AirFICH{}, false
	}
	bit := func(n int) uint8 { // MSB-first bit n of the 200-bit field
		d := dibits[n/2]
		if n%2 == 0 {
			return (d >> 1) & 1
		}
		return d & 1
	}
	var pairs [airFICHDibits][2]uint8
	for i := range pairs {
		n := airFICHInterleave(i)
		pairs[i] = [2]uint8{bit(n), bit(n + 1)}
	}
	dec := viterbiK5(pairs[:])
	var info [6]byte
	for w := 0; w < 4; w++ {
		var cw uint32
		for k := 0; k < 24; k++ {
			cw = cw<<1 | uint32(dec[w*24+k])
		}
		data, good := golay24Decode(cw)
		if !good {
			return AirFICH{}, false
		}
		// Two 12-bit words per three octets, as MMDVMHost packs them.
		switch w {
		case 0:
			info[0] = byte(data >> 4)
			info[1] = byte(data<<4) & 0xF0
		case 1:
			info[1] |= byte(data>>8) & 0x0F
			info[2] = byte(data)
		case 2:
			info[3] = byte(data >> 4)
			info[4] = byte(data<<4) & 0xF0
		case 3:
			info[4] |= byte(data>>8) & 0x0F
			info[5] = byte(data)
		}
	}
	if crc16GSM(info[:4]) != uint16(info[4])<<8|uint16(info[5]) {
		return AirFICH{}, false
	}
	return AirFICH{
		FI:   info[0] >> 6,
		CS:   (info[0] >> 4) & 3,
		CM:   (info[0] >> 2) & 3,
		BN:   info[0] & 3,
		BT:   info[1] >> 6,
		FN:   (info[1] >> 3) & 7,
		FT:   info[1] & 7,
		DT:   info[2] & 3,
		MR:   (info[2] >> 3) & 3,
		Dev:  info[2]&0x40 != 0,
		DGID: info[3] & 0x7F,
	}, true
}

// viterbiK5 hard-decision decodes the FICH's rate-1/2, K=5 convolutional
// code (outputs d^d3^d4 and d^d1^d2^d4, d1 the previous input bit) and
// returns the decoded input bits. The encoder starts and, after the four
// zero tail bits, ends in state 0.
func viterbiK5(pairs [][2]uint8) []uint8 {
	const states = 16 // state = last four inputs, d1 in bit 0
	const inf = 1 << 30
	metric := make([]int, states)
	for s := 1; s < states; s++ {
		metric[s] = inf
	}
	prev := make([][states]uint8, len(pairs)) // predecessor state
	in := make([][states]uint8, len(pairs))   // input bit taken
	next := make([]int, states)
	for t, p := range pairs {
		for s := range next {
			next[s] = inf
		}
		for s := 0; s < states; s++ {
			if metric[s] >= inf {
				continue
			}
			d1, d2, d3, d4 := uint8(s&1), uint8(s>>1&1), uint8(s>>2&1), uint8(s>>3&1)
			for d := uint8(0); d < 2; d++ {
				g1 := d ^ d3 ^ d4
				g2 := d ^ d1 ^ d2 ^ d4
				cost := metric[s] + int(g1^p[0]) + int(g2^p[1])
				ns := (s<<1 | int(d)) & (states - 1)
				if cost < next[ns] {
					next[ns] = cost
					prev[t][ns] = uint8(s)
					in[t][ns] = d
				}
			}
		}
		copy(metric, next)
	}
	out := make([]uint8, len(pairs))
	s := 0 // tail bits flush the encoder to state 0
	for t := len(pairs) - 1; t >= 0; t-- {
		out[t] = in[t][s]
		s = int(prev[t][s])
	}
	return out
}

// golay24Encode is the extended Golay(24,12) code MMDVMHost's
// CGolay24128 uses: data(12) ‖ parity(11) ‖ overall even parity(1),
// generator x^11+x^10+x^6+x^5+x^4+x^2+1.
func golay24Encode(data uint16) uint32 {
	const gen = 0xC75
	r := uint32(data&0xFFF) << 11
	for i := 22; i >= 11; i-- {
		if r&(1<<uint(i)) != 0 {
			r ^= gen << uint(i-11)
		}
	}
	cw := (uint32(data&0xFFF)<<11 | r) << 1
	if bits.OnesCount32(cw)%2 != 0 {
		cw |= 1
	}
	return cw
}

// golay24Decode returns the nearest codeword's data and whether it lies
// within the code's 3-error correction radius.
func golay24Decode(cw uint32) (uint16, bool) {
	cw &= 0xFFFFFF
	best, bestD := uint16(0), 25
	for d := uint16(0); d < 4096; d++ {
		if dist := bits.OnesCount32(golay24Encode(d) ^ cw); dist < bestD {
			best, bestD = d, dist
			if dist == 0 {
				break
			}
		}
	}
	return best, bestD <= 3
}

// crc16GSM is the FICH check (MMDVMHost CCRC::addCCITT162): polynomial
// 0x1021, initial 0, output inverted, MSB first.
func crc16GSM(b []byte) uint16 {
	var crc uint16
	for _, v := range b {
		crc ^= uint16(v) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = crc<<1 ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return ^crc
}

// VD2 voice channel layout (MMDVMHost processVDMode2Audio, YSF2DMR putYSF):
// after the sync and FICH, five 144-bit sections each hold a 40-bit DCH
// then a 104-bit VCH.
const (
	vd2VCHs      = 5
	vd2VCHBits   = 104
	vd2Section   = 144
	vd2FirstVCH  = 40
	PayloadBits  = PayloadDibits * 2
	VD2InfoBits  = 49
	vd2Triplets  = 27
	vd2PlainBits = 22
)

var vd2Whitening = [13]byte{0x93, 0xD7, 0x51, 0x21, 0x9C, 0x2F, 0x6C, 0xD0, 0xEF, 0x0F, 0xF8, 0x3D, 0xF1}

// vd2Interleave maps deinterleaved VCH bit i to its on-air position: 26
// columns of 4 (MMDVMHost INTERLEAVE_TABLE_26_4).
func vd2Interleave(i int) int { return (i%26)*4 + i/26 }

// VD2Frame is one decoded V/D mode 2 voice frame.
type VD2Frame struct {
	// Info is the 49-bit AMBE+2 payload, one bit per byte, in the order
	// the DMR AMBE+2 decoder hands its vocoder (C0 ‖ C1 ‖ C2 ‖ C3), so the
	// recorder's "ambe2-dmr" vocoder renders it unchanged.
	Info []byte
	// Corrected counts the repetition-coded triplets that disagreed.
	Corrected int
}

// DecodeVD2Voice extracts the five voice frames of a V/D mode 2 frame from
// its 360 payload dibits (the dibits after the sync and FICH).
func DecodeVD2Voice(payload []uint8) []VD2Frame {
	if len(payload) < PayloadDibits {
		return nil
	}
	bit := func(n int) uint8 {
		d := payload[n/2]
		if n%2 == 0 {
			return (d >> 1) & 1
		}
		return d & 1
	}
	out := make([]VD2Frame, 0, vd2VCHs)
	for j := 0; j < vd2VCHs; j++ {
		off := vd2FirstVCH + j*vd2Section
		var vch [vd2VCHBits]uint8
		for i := range vch {
			vch[i] = bit(off+vd2Interleave(i)) ^ (vd2Whitening[i/8] >> uint(7-i%8) & 1)
		}
		fr := VD2Frame{Info: make([]byte, 0, VD2InfoBits)}
		// The first 27 bits are sent three times each: majority vote.
		for k := 0; k < vd2Triplets; k++ {
			s := vch[3*k] + vch[3*k+1] + vch[3*k+2]
			if s == 1 || s == 2 {
				fr.Corrected++
			}
			fr.Info = append(fr.Info, b2u(s >= 2))
		}
		// Then 22 unprotected bits; bit 103 is padding.
		fr.Info = append(fr.Info, vch[3*vd2Triplets:3*vd2Triplets+vd2PlainBits]...)
		out = append(out, fr)
	}
	return out
}

func b2u(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

// AirFrame is one frame cut from the dibit stream at a sync word.
type AirFrame struct {
	// FICH is the decoded frame header; FICHOK is false when it did not
	// decode (CRC or Golay failure).
	FICH   AirFICH
	FICHOK bool
	// Payload holds the 360 dibits after the sync and FICH.
	Payload []uint8
	// SyncEnd is the absolute index of the sync word's last dibit.
	SyncEnd int
}

// AirFramer cuts a continuous dibit stream into frames at each sync word.
// It is stateful and not safe for concurrent use.
type AirFramer struct {
	det     *SyncDetector
	buf     []uint8 // dibits from bufBase on
	bufBase int
	pending []int // sync-end indices still waiting for their frame body
	hits    []int
}

// NewAirFramer returns a framer whose sync search tolerates tolerance
// mismatched dibits (2 is the package default).
func NewAirFramer(tolerance int) *AirFramer {
	return &AirFramer{det: NewSyncDetector(tolerance)}
}

// Process consumes dibits starting at absolute index baseIdx and returns
// every frame whose body is now complete.
func (f *AirFramer) Process(dibits []uint8, baseIdx int) []AirFrame {
	if len(f.buf) == 0 {
		f.bufBase = baseIdx
	}
	f.buf = append(f.buf, dibits...)
	f.hits, _ = f.det.Process(f.hits[:0], dibits, baseIdx)
	f.pending = append(f.pending, f.hits...)

	const body = FICHDibits + PayloadDibits
	var out []AirFrame
	keep := f.pending[:0]
	for _, end := range f.pending {
		start := end + 1 - f.bufBase
		if start < 0 {
			continue // fell out of the buffer
		}
		if start+body > len(f.buf) {
			keep = append(keep, end)
			continue
		}
		fich, ok := DecodeAirFICH(f.buf[start : start+FICHDibits])
		payload := append([]uint8(nil), f.buf[start+FICHDibits:start+body]...)
		out = append(out, AirFrame{FICH: fich, FICHOK: ok, Payload: payload, SyncEnd: end})
	}
	f.pending = keep

	// Keep only what a pending or future frame can still need.
	need := len(f.buf) - (FSWDibits + body)
	if len(f.pending) > 0 {
		if s := f.pending[0] + 1 - f.bufBase; s < need {
			need = s
		}
	}
	if need > 0 {
		f.buf = append(f.buf[:0], f.buf[need:]...)
		f.bufBase += need
	}
	return out
}

// EncodeVD2Voice is the inverse of DecodeVD2Voice, for synthetic fixtures:
// it lays five 49-bit AMBE+2 payloads (one bit per byte) into the 360
// payload dibits of a V/D mode 2 frame, DCH bits zero. The decoder is
// pinned against reference vectors independently of this encoder.
func EncodeVD2Voice(infos [vd2VCHs][]byte) []uint8 {
	payload := make([]uint8, PayloadDibits)
	for j, info := range infos {
		var vch [vd2VCHBits]uint8
		for k := 0; k < vd2Triplets; k++ {
			vch[3*k], vch[3*k+1], vch[3*k+2] = info[k]&1, info[k]&1, info[k]&1
		}
		for k := 0; k < vd2PlainBits; k++ {
			vch[3*vd2Triplets+k] = info[vd2Triplets+k] & 1
		}
		off := vd2FirstVCH + j*vd2Section
		for i, b := range vch {
			b ^= vd2Whitening[i/8] >> uint(7-i%8) & 1
			n := off + vd2Interleave(i)
			payload[n/2] |= b << uint(1-n%2)
		}
	}
	return payload
}
