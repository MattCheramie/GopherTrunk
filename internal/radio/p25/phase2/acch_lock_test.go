//go:build integration

package phase2_test

// Lock diagnostic: the receiver stops finding the outbound sync partway
// through a capture SDRtrunk decodes end to end. This asks whether the sync is
// genuinely gone from the dibit stream or merely rotated — real air is
// differentially decoded, so a residual carrier offset rotates every dibit by
// a constant 0..3, and the superframe decoder searches the rotations at a
// stricter tolerance than the canonical one.

import (
	"os"
	"testing"
)

func syncDibitsLocal() []uint8 {
	const magic uint64 = 0x575D57F7FF
	out := make([]uint8, 20)
	for i := range out {
		out[i] = uint8(magic >> (38 - 2*i) & 3)
	}
	return out
}

func TestACCHLockDiagnostic(t *testing.T) {
	path := os.Getenv("GT_P2_WAV")
	if path == "" {
		t.Skip("set GT_P2_WAV")
	}
	iq, rate := readWavIQ(t, path)
	var stream []uint8
	probeDibits(iq, rate, func(d []uint8, _ int) { stream = append(stream, d...) })
	t.Logf("stream=%d dibits (%.1f superframes)", len(stream), float64(len(stream))/2160)

	base := syncDibitsLocal()
	// Per rotation and tolerance: how many sync hits, and where the last one is.
	for rot := 0; rot < 4; rot++ {
		pat := make([]uint8, 20)
		for i, p := range base {
			pat[i] = (p + uint8(rot)) & 3
		}
		for _, tol := range []int{2, 4, 6} {
			hits, last := 0, -1
			for i := 0; i+20 <= len(stream); i++ {
				miss := 0
				for k := 0; k < 20; k++ {
					if stream[i+k] != pat[k] {
						miss++
						if miss > tol {
							break
						}
					}
				}
				if miss <= tol {
					hits++
					last = i
					i += 19
				}
			}
			if hits > 0 {
				t.Logf("rot=%d tol=%d: hits=%3d last_at=%6d (%.0f%% into stream)",
					rot, tol, hits, last, 100*float64(last)/float64(len(stream)))
			}
		}
	}

	// Is the stream still carrying *something* late on, or has the demod died?
	// A dead demod produces a degenerate dibit histogram; live air is ~uniform.
	for _, seg := range [][2]int{{0, 3600}, {3600, 12000}, {12000, len(stream)}} {
		var h [4]int
		for _, d := range stream[seg[0]:min(seg[1], len(stream))] {
			h[d&3]++
		}
		t.Logf("dibit histogram %6d..%6d: %v", seg[0], seg[1], h)
	}
}
