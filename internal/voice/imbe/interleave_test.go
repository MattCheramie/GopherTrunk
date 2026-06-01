package imbe

import (
	"math/rand"
	"testing"
)

// TestInterleavePermutationIsBijection pins that imbeDeinterleave
// maps each of the 144 vector-order positions to a distinct on-air
// position covering 0..143 exactly once. A non-bijective table
// would silently corrupt every voice frame, so this guards the
// transcribed DSD schedule against an editing slip. (The init()
// guard panics on the same condition at load; this surfaces it as a
// test failure with a useful message.)
func TestInterleavePermutationIsBijection(t *testing.T) {
	var seen [ChannelBits]bool
	for v, onAir := range imbeDeinterleave {
		if onAir < 0 || onAir >= ChannelBits {
			t.Fatalf("imbeDeinterleave[%d] = %d out of range [0,%d)", v, onAir, ChannelBits)
		}
		if seen[onAir] {
			t.Fatalf("on-air bit %d targeted by more than one vector bit", onAir)
		}
		seen[onAir] = true
	}
	for onAir, ok := range seen {
		if !ok {
			t.Errorf("on-air bit %d is never produced by the permutation", onAir)
		}
	}
}

// TestInterleaveDeinterleaveRoundTrip: Deinterleave is the exact
// inverse of Interleave in both directions, over random buffers.
func TestInterleaveDeinterleaveRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for trial := 0; trial < 64; trial++ {
		vec := make([]byte, ChannelBits)
		for i := range vec {
			vec[i] = byte(rng.Intn(2))
		}
		onAir, err := Interleave(vec)
		if err != nil {
			t.Fatalf("trial %d: Interleave: %v", trial, err)
		}
		got, err := Deinterleave(onAir)
		if err != nil {
			t.Fatalf("trial %d: Deinterleave: %v", trial, err)
		}
		for i := range vec {
			if got[i] != vec[i] {
				t.Fatalf("trial %d: Deinterleave(Interleave(x)) bit %d = %d, want %d", trial, i, got[i], vec[i])
			}
		}
		// And the other direction: Interleave(Deinterleave(x)) == x.
		back, err := Interleave(got)
		if err != nil {
			t.Fatalf("trial %d: Interleave(Deinterleave): %v", trial, err)
		}
		for i := range onAir {
			if back[i] != onAir[i] {
				t.Fatalf("trial %d: Interleave(Deinterleave(y)) bit %d = %d, want %d", trial, i, back[i], onAir[i])
			}
		}
	}
}

// TestInterleaveScattersContiguousVector: a contiguous run of set
// bits in vector order must land non-contiguously on-air. This
// catches an accidental identity table (the failure mode where the
// "fix" compiles, every round-trip test passes, and the codec still
// does nothing on real air). We set all of u_0's 23 channel bits
// and assert the resulting on-air positions are not a single
// contiguous block.
func TestInterleaveScattersContiguousVector(t *testing.T) {
	vec := make([]byte, ChannelBits)
	for i := u0Offset; i < u0Offset+u0Bits; i++ {
		vec[i] = 1
	}
	onAir, err := Interleave(vec)
	if err != nil {
		t.Fatalf("Interleave: %v", err)
	}
	// Collect the on-air positions of the set bits.
	var first, last, count int
	first = -1
	for i, b := range onAir {
		if b == 1 {
			if first < 0 {
				first = i
			}
			last = i
			count++
		}
	}
	if count != u0Bits {
		t.Fatalf("set-bit count after interleave = %d, want %d", count, u0Bits)
	}
	// If the table were the identity (or any block-preserving map),
	// the 23 set bits would occupy exactly 23 contiguous positions.
	if last-first+1 == u0Bits {
		t.Errorf("u_0's bits stayed contiguous on-air ([%d,%d]) — interleave is a no-op", first, last)
	}
}

// TestInterleaveRejectsWrongLength: both directions surface
// ErrChannelLength for anything other than 144 bits.
func TestInterleaveRejectsWrongLength(t *testing.T) {
	for _, n := range []int{0, ChannelBits - 1, ChannelBits + 1} {
		if _, err := Interleave(make([]byte, n)); err == nil {
			t.Errorf("Interleave(len=%d) returned no error", n)
		}
		if _, err := Deinterleave(make([]byte, n)); err == nil {
			t.Errorf("Deinterleave(len=%d) returned no error", n)
		}
	}
}

// TestEncodeFrameToChannelDecodes: the on-air encode helper round-
// trips through DecodeChannelToFrame with zero corrected errors on a
// clean burst — the encode/decode symmetry the LDU fixtures rely on.
func TestEncodeFrameToChannelDecodes(t *testing.T) {
	info := make([]byte, InfoBits)
	for i := range info {
		info[i] = byte((i*3 + 1) % 2)
	}
	onAir, err := EncodeFrameToChannel(info)
	if err != nil {
		t.Fatalf("EncodeFrameToChannel: %v", err)
	}
	if len(onAir) != ChannelBits {
		t.Fatalf("on-air length = %d, want %d", len(onAir), ChannelBits)
	}
	frame, errs, err := DecodeChannelToFrame(onAir)
	if err != nil {
		t.Fatalf("DecodeChannelToFrame: %v", err)
	}
	if errs != 0 {
		t.Errorf("errs = %d, want 0 on a clean on-air burst", errs)
	}
	want, err := PackInfoBitsToFrame(info)
	if err != nil {
		t.Fatalf("PackInfoBitsToFrame: %v", err)
	}
	for i := range want {
		if frame[i] != want[i] {
			t.Fatalf("byte %d: round-trip = %02x, want %02x", i, frame[i], want[i])
		}
	}
}
