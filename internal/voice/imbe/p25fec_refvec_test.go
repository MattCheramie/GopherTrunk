package imbe

import (
	"encoding/hex"
	"testing"
)

// Real-air-faithful reference vectors for the IMBE 4400 channel
// decode (TIA-102.BABA §7). Each `onair` is a 144-bit transmitted
// voice subframe exactly as a real P25 Phase 1 transmitter emits it,
// and `frame` is the 11-byte (88-bit) information frame a conformant
// decoder must recover from it.
//
// These were generated independently of GopherTrunk's own encoder,
// from the canonical mbelib / DSD reference convention (szechyjs
// mbelib imbe7200x4400.c + ecc.c + ecc_const.h, szechyjs dsd
// p25p1_const.h interleave schedule): Golay(23,12) with
// golayGenerator, Hamming(15,11) with hammingGenerator, the §7.4 PRBS
// scramble seeded from the 12 Golay data bits of u_0, and the §7.5
// interleave. Because they come from the reference algorithm rather
// than from EncodeFrameToChannel, they catch any convention drift
// between this package and real on-air P25 — the gap that left voice
// decoding garbled until the per-vector column order, descramble
// seed, and FEC generators were corrected (issue #489).
var p25RealAirVectors = []struct {
	onair string // 18 bytes (144 bits, MSB-first), hex
	frame string // 11 bytes (88 info bits, MSB-first), hex
}{
	{onair: "9e44a154b9d2ba52dfddf5e5ed1d376648fd", frame: "839e0cab56c353f634199a"},
	{onair: "ae6c7926cb9675a3b25c7b3811d9f1c19aa2", frame: "b350e1095681ef2f6fcaa1"},
	{onair: "84c6a99403ff81c826142c0390ec853359bc", frame: "89ec590eb56d85fe76c4c0"},
	{onair: "7a63c6589516e06fb5a78bb54d2b1617417d", frame: "01bdeb24c43914afbc78da"},
	{onair: "4c86c3a2fad2aa7135184fb92d361ae11748", frame: "0b90bcb6f55ac90826bf52"},
	{onair: "a09065eee16a174eea99c1c276a7c55fab11", frame: "88696c62f8467142e367ec"},
}

func TestDecodeChannelToFrameMatchesRealAirReference(t *testing.T) {
	for i, v := range p25RealAirVectors {
		packed, err := hex.DecodeString(v.onair)
		if err != nil {
			t.Fatalf("vector %d: bad onair hex: %v", i, err)
		}
		// Expand the 18 packed bytes into 144 one-bit-per-byte channel
		// bits (MSB-first), the form DecodeChannelToFrame consumes.
		channel := make([]byte, ChannelBits)
		for b := 0; b < ChannelBits; b++ {
			channel[b] = (packed[b/8] >> (7 - uint(b)%8)) & 1
		}
		gotFrame, errs, derr := DecodeChannelToFrame(channel)
		if derr != nil {
			t.Errorf("vector %d: decode error: %v", i, derr)
			continue
		}
		if errs != 0 {
			t.Errorf("vector %d: clean reference decoded with %d corrected errors, want 0", i, errs)
		}
		want, err := hex.DecodeString(v.frame)
		if err != nil {
			t.Fatalf("vector %d: bad frame hex: %v", i, err)
		}
		if len(gotFrame) != len(want) {
			t.Fatalf("vector %d: frame len %d, want %d", i, len(gotFrame), len(want))
		}
		for j := range want {
			if gotFrame[j] != want[j] {
				t.Fatalf("vector %d: frame mismatch\n got %x\nwant %x", i, gotFrame, want)
			}
		}
	}
}

// TestDecodeChannelToFrameCorrectsSingleOnAirError confirms the FEC
// recovers the reference frame after any single on-air bit error that
// lands in an FEC-protected vector (u_0..u_6) — the everyday
// weak-signal case the corrected Golay/Hamming generators must
// handle. A single on-air error becomes a single error in exactly one
// vector (the interleave is a permutation), which Golay(23,12,7) /
// Hamming(15,11,3) correct. Errors landing in the unprotected u_7
// vector are excluded.
func TestDecodeChannelToFrameCorrectsSingleOnAirError(t *testing.T) {
	// On-air positions that carry u_7 (vector bits 137..143) have no FEC.
	u7OnAir := map[int]bool{}
	for v := u7Offset; v < u7Offset+u7Bits; v++ {
		u7OnAir[imbeDeinterleave[v]] = true
	}
	for i, v := range p25RealAirVectors {
		packed, _ := hex.DecodeString(v.onair)
		want, _ := hex.DecodeString(v.frame)
		for flip := 0; flip < ChannelBits; flip++ {
			if u7OnAir[flip] {
				continue
			}
			channel := make([]byte, ChannelBits)
			for b := 0; b < ChannelBits; b++ {
				channel[b] = (packed[b/8] >> (7 - uint(b)%8)) & 1
			}
			channel[flip] ^= 1
			gotFrame, _, _ := DecodeChannelToFrame(channel)
			for j := range want {
				if gotFrame[j] != want[j] {
					t.Fatalf("vector %d flip %d: FEC failed to correct\n got %x\nwant %x",
						i, flip, gotFrame, want)
				}
			}
		}
	}
}
