package framing

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestInterleaveRoundTrip(t *testing.T) {
	in := make([]byte, bptcN)
	for i := range in {
		in[i] = byte(i & 1)
	}
	channel := InterleaveBPTC(in)
	back := DeinterleaveBPTC(channel)
	if !bytes.Equal(back, in) {
		t.Errorf("interleave round-trip mismatch")
	}
}

func TestInterleaveIsAPermutation(t *testing.T) {
	// All 196 indices must appear exactly once.
	seen := [bptcN]bool{}
	for i := 0; i < bptcN; i++ {
		seen[(i*181)%bptcN] = true
	}
	for i, ok := range seen {
		if !ok {
			t.Fatalf("interleave drops index %d", i)
		}
	}
}

func TestBPTCEncodeDecodeRoundTrip(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	for trial := 0; trial < 50; trial++ {
		info := make([]byte, 96)
		for i := range info {
			info[i] = byte(r.Intn(2))
		}
		channel := EncodeBPTC196_96(info)
		decoded, errs := DecodeBPTC196_96(channel)
		if errs != 0 {
			t.Fatalf("clean decode reported %d errors", errs)
		}
		if !bytes.Equal(decoded, info) {
			t.Fatalf("trial %d: decode mismatch", trial)
		}
	}
}

func TestBPTCCorrectsSingleBitErrors(t *testing.T) {
	r := rand.New(rand.NewSource(11))
	info := make([]byte, 96)
	for i := range info {
		info[i] = byte(r.Intn(2))
	}
	channel := EncodeBPTC196_96(info)
	for bit := 0; bit < bptcN; bit++ {
		corrupted := append([]byte(nil), channel...)
		corrupted[bit] ^= 1
		decoded, errs := DecodeBPTC196_96(corrupted)
		if errs == -1 {
			t.Errorf("bit %d: decoder reported failure", bit)
			continue
		}
		if !bytes.Equal(decoded, info) {
			t.Errorf("bit %d: decode failed to recover info", bit)
		}
	}
}

// bptcInfoIndices lists, in output order, the deinterleaved-stream indices
// that hold the 96 BPTC(196,96) information bits — verbatim from the
// reference MMDVM CBPTC19696 decodeExtractData/encodeExtractData (row 0
// cols 3..10 = deInter 4..11, then rows 1..8 cols 0..10). It is the literal
// ETSI Annex B mapping and is intentionally expressed independently of
// bptc.go's [13][15] matrix wiring so the golden test below cross-checks
// that wiring against the spec.
func bptcInfoIndices() []int {
	idx := make([]int, 0, 96)
	for a := 4; a <= 11; a++ { // row 0, cols 3..10
		idx = append(idx, a)
	}
	for _, base := range []int{16, 31, 46, 61, 76, 91, 106, 121} { // rows 1..8
		for a := base; a < base+11; a++ {
			idx = append(idx, a)
		}
	}
	return idx
}

// refEncodeBPTC196_96 is an independent BPTC(196,96) encoder built straight
// from the flat deinterleaved-stream indices of ETSI Annex B / MMDVM
// CBPTC19696: place info at bptcInfoIndices, Hamming(15,11) parity on each
// data row, Hamming(13,9) parity on each column, R bit at index 0, then
// interleave channel[(a*181) mod 196] = deInter[a]. It shares only the
// (separately MMDVM-verified) Hamming helpers with bptc.go, so agreement
// pins the on-air layout rather than merely round-tripping the encoder.
func refEncodeBPTC196_96(info []byte) []byte {
	deInter := make([]byte, bptcN)
	for i, a := range bptcInfoIndices() {
		deInter[a] = info[i] & 1
	}
	for r := 0; r < 9; r++ { // row parity
		base := r*15 + 1
		var d uint16
		for c := 0; c < 11; c++ {
			d |= uint16(deInter[base+c]) << uint(c)
		}
		cw := HammingEncode15_11(d)
		for j := 0; j < 4; j++ {
			deInter[base+11+j] = byte((cw >> uint(j)) & 1)
		}
	}
	for c := 0; c < 15; c++ { // column parity
		var d uint16
		for r := 0; r < 9; r++ {
			d |= uint16(deInter[c+1+r*15]) << uint(r)
		}
		cw := HammingEncode13_9(d)
		for j := 0; j < 4; j++ {
			deInter[c+1+(9+j)*15] = byte((cw >> uint(j)) & 1)
		}
	}
	channel := make([]byte, bptcN)
	for a := 0; a < bptcN; a++ {
		channel[(a*181)%bptcN] = deInter[a]
	}
	return channel
}

// TestBPTCCanonicalLayoutGolden is the cross-check that bptc.go matches the
// real DMR on-air bit layout (not just its own encoder). It builds a real
// Voice LC Header info block — 9 FLC octets + RS(12,9) parity (seeded with
// RS129SeedVoiceLCHeader) — encodes it with the spec-literal reference
// encoder, and asserts that:
//   - bptc.go's EncodeBPTC196_96 produces the identical 196 on-air bits;
//   - DecodeBPTC196_96 recovers the info with zero corrections;
//   - the independent RS(12,9) parity (which depends on the exact info-bit
//     order) verifies — catching any bit-ordering regression that a pure
//     round-trip would miss;
//   - the recovered FLC octets match, and single-bit on-air errors are
//     still corrected on the canonical layout.
func TestBPTCCanonicalLayoutGolden(t *testing.T) {
	// Group-voice FLC: FLCO=0x00, FID=0x00, SvcOpts=0x00, dst=0x000123,
	// src=0x456789 (the same tuple the Tier II fixtures use).
	flc := []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x23, 0x45, 0x67, 0x89}
	var data [9]byte
	copy(data[:], flc)
	cw := EncodeRS12_9(data)
	for i := 0; i < 3; i++ {
		cw[9+i] ^= RS129SeedVoiceLCHeader[i]
	}
	info := cw[:] // 12 octets = 9 FLC + 3 RS parity
	bits := make([]byte, 96)
	for i := 0; i < 96; i++ {
		bits[i] = (info[i>>3] >> uint(7-(i&7))) & 1
	}

	refChannel := refEncodeBPTC196_96(bits)
	if got := EncodeBPTC196_96(bits); !bytes.Equal(got, refChannel) {
		t.Fatalf("EncodeBPTC196_96 disagrees with the spec-literal reference layout")
	}

	decoded, errs := DecodeBPTC196_96(refChannel)
	if errs != 0 {
		t.Fatalf("clean canonical decode reported %d errors", errs)
	}
	if !bytes.Equal(decoded, bits) {
		t.Fatalf("canonical decode did not recover the info bits")
	}

	// Repack and verify the RS(12,9) parity — independent of BPTC and
	// sensitive to info-bit ordering.
	var rec [12]byte
	for i := 0; i < 96; i++ {
		if decoded[i]&1 != 0 {
			rec[i>>3] |= 1 << uint(7-(i&7))
		}
	}
	if !VerifyRS12_9(rec[:], RS129SeedVoiceLCHeader) {
		t.Fatalf("recovered info failed RS(12,9) parity — info-bit ordering is wrong")
	}
	if !bytes.Equal(rec[:9], flc) {
		t.Fatalf("recovered FLC octets = % x, want % x", rec[:9], flc)
	}

	// Single-bit error correction on the canonical on-air vector.
	for bit := 0; bit < bptcN; bit++ {
		corrupted := append([]byte(nil), refChannel...)
		corrupted[bit] ^= 1
		got, e := DecodeBPTC196_96(corrupted)
		if e == -1 || !bytes.Equal(got, bits) {
			t.Errorf("bit %d: canonical decode failed to correct (errs=%d)", bit, e)
		}
	}
}

func TestBPTCAllZeroAndAllOne(t *testing.T) {
	for _, fill := range []byte{0, 1} {
		info := make([]byte, 96)
		for i := range info {
			info[i] = fill
		}
		channel := EncodeBPTC196_96(info)
		decoded, errs := DecodeBPTC196_96(channel)
		if errs != 0 {
			t.Errorf("fill=%d: errs=%d", fill, errs)
		}
		if !bytes.Equal(decoded, info) {
			t.Errorf("fill=%d: round-trip mismatch", fill)
		}
	}
}
