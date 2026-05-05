package dmr

import "testing"

func TestSyncPatternsAreDistinct(t *testing.T) {
	seen := map[uint64]string{}
	for _, p := range AllSyncs {
		if other, dup := seen[p.Hex]; dup {
			t.Errorf("hex collision: %s and %s share %X", p.Name, other, p.Hex)
		}
		seen[p.Hex] = p.Name
	}
}

func TestSyncDibitDecomposition(t *testing.T) {
	// Top dibit of 0x755FD7DF75F7 is 0x7>>2 = 01 (i.e. 0b01 = 1).
	p := BSVoice
	want := uint8((p.Hex >> 46) & 0x3)
	if p.Dibits[0] != want {
		t.Errorf("Dibits[0] = %d, want %d", p.Dibits[0], want)
	}
}

func TestSyncDetectorMatchesCleanSync(t *testing.T) {
	det := NewSyncDetector([]SyncPattern{BSVoice}, 0)
	stream := make([]uint8, 50)
	copy(stream[20:], BSVoice.Dibits[:])
	hits, _ := det.Process(nil, stream, 0)
	if len(hits) != 1 {
		t.Fatalf("hits = %v, want exactly 1", hits)
	}
	if hits[0].Index != 20+24-1 {
		t.Errorf("index = %d, want %d", hits[0].Index, 20+23)
	}
	if hits[0].Pattern.Name != "BS-Voice" {
		t.Errorf("name = %s, want BS-Voice", hits[0].Pattern.Name)
	}
}

func TestSyncDetectorTolerates2Errors(t *testing.T) {
	det := NewSyncDetector(nil, 2) // all patterns
	stream := make([]uint8, 50)
	copy(stream[10:], MSData.Dibits[:])
	stream[12] = (stream[12] + 1) % 4
	stream[20] = (stream[20] + 1) % 4
	hits, _ := det.Process(nil, stream, 0)
	if len(hits) != 1 || hits[0].Pattern.Name != "MS-Data" {
		t.Errorf("hits = %+v", hits)
	}
}

func TestSyncDetectorPicksBestMatch(t *testing.T) {
	det := NewSyncDetector(nil, 12) // very tolerant; force comparison
	stream := make([]uint8, 50)
	copy(stream[10:], BSVoice.Dibits[:])
	hits, _ := det.Process(nil, stream, 0)
	// With a clean BSVoice in the buffer, the detector should select it
	// even though many patterns fall within tolerance.
	if len(hits) == 0 || hits[0].Pattern.Name != "BS-Voice" {
		t.Errorf("hits = %+v, want first to be BS-Voice", hits)
	}
}
