package phase1

import "testing"

func TestSymbolToDibit(t *testing.T) {
	cases := map[int8]uint8{1: 0, 3: 1, -1: 2, -3: 3}
	for sym, want := range cases {
		if got := SymbolToDibit(sym); got != want {
			t.Errorf("SymbolToDibit(%d) = %d, want %d", sym, got, want)
		}
	}
}

func TestSyncDetectorMatchesCleanFSW(t *testing.T) {
	det := NewSyncDetector(0)
	stream := make([]uint8, 100)
	// Place an FSW at offset 50.
	copy(stream[50:], FrameSyncWord[:])
	hits, _ := det.Process(nil, stream, 0)
	if len(hits) != 1 || hits[0] != 50+24-1 {
		t.Errorf("hits = %v, want [%d]", hits, 50+24-1)
	}
}

func TestSyncDetectorTolerates2Errors(t *testing.T) {
	det := NewSyncDetector(2)
	stream := make([]uint8, 100)
	copy(stream[20:], FrameSyncWord[:])
	stream[22] = (stream[22] + 1) % 4
	stream[40] = (stream[40] + 1) % 4
	hits, _ := det.Process(nil, stream, 0)
	if len(hits) != 1 {
		t.Errorf("hits = %v, want exactly 1", hits)
	}
}

func TestSyncDetectorRejectsTooManyErrors(t *testing.T) {
	det := NewSyncDetector(1)
	stream := make([]uint8, 60)
	copy(stream[10:], FrameSyncWord[:])
	stream[12] = (stream[12] + 1) % 4
	stream[14] = (stream[14] + 1) % 4
	stream[16] = (stream[16] + 1) % 4
	hits, _ := det.Process(nil, stream, 0)
	if len(hits) != 0 {
		t.Errorf("hits = %v, want []", hits)
	}
}

// TestSyncDetectorMatchesAllRotations: the FSW search must accept the
// canonical pattern shifted by k ∈ {0, 1, 2, 3} mod 4 (the residual
// symbol-polarity / I-Q-swap ambiguity that can survive front-end
// processing) and return the rotation that matched so downstream
// parsing can invert it.
func TestSyncDetectorMatchesAllRotations(t *testing.T) {
	for k := uint8(0); k < 4; k++ {
		det := NewSyncDetector(0)
		stream := make([]uint8, 100)
		for i := range stream {
			stream[i] = 0
		}
		// Embed the FSW with k subtracted from each dibit so the
		// detector has to add k back via its rotation search.
		for i, d := range FrameSyncWord {
			stream[40+i] = (d + 4 - k) & 3
		}
		hits, rots, _ := det.ProcessWithRotation(nil, nil, stream, 0)
		if len(hits) != 1 {
			t.Errorf("k=%d hits = %v, want exactly 1", k, hits)
			continue
		}
		if rots[0] != k {
			t.Errorf("k=%d rots[0] = %d, want %d", k, rots[0], k)
		}
		if hits[0] != 40+24-1 {
			t.Errorf("k=%d hits[0] = %d, want %d", k, hits[0], 40+24-1)
		}
	}
}

// TestSyncDetectorPrefersRotationZero: when two rotations tie on
// mismatch count, k=0 must win so existing clean-fixture tests
// keep producing the same hit they always have.
func TestSyncDetectorPrefersRotationZero(t *testing.T) {
	det := NewSyncDetector(0)
	stream := make([]uint8, 80)
	copy(stream[30:], FrameSyncWord[:])
	_, rots, _ := det.ProcessWithRotation(nil, nil, stream, 0)
	if len(rots) != 1 {
		t.Fatalf("rots = %v, want length 1", rots)
	}
	if rots[0] != 0 {
		t.Errorf("rots[0] = %d, want 0 (clean FSW must bind rotation=0)", rots[0])
	}
}
