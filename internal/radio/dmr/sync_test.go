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

// TestSyncPairsClosedUnderPolarityFlip documents the property the issue
// #264 fix relies on: applying the discriminator-polarity flip (every
// dibit + 2 mod 4) to a downlink voice/data sync word yields its
// data/voice twin. This is why a spectrum-inverted stream still
// produces sync hits (so the detector needs no change) AND why the
// sync match alone can't resolve the polarity (so the burst is decoded
// at both polarities downstream). MS-RC (the mobile reverse-channel
// sync) is the lone pattern with no twin and is not used on the
// downlink control channel.
func TestSyncPairsClosedUnderPolarityFlip(t *testing.T) {
	pairs := [][2]SyncPattern{
		{BSVoice, BSData},
		{MSVoice, MSData},
		{DMVoice1, DMData1},
		{DMVoice2, DMData2},
	}
	for _, pr := range pairs {
		var flipped [24]uint8
		for i, d := range pr[0].Dibits {
			flipped[i] = (d + PolarityFlip) & 3
		}
		if flipped != pr[1].Dibits {
			t.Errorf("%s flipped != %s", pr[0].Name, pr[1].Name)
		}
		// The flip is symmetric: flipping the twin recovers the first.
		for i, d := range pr[1].Dibits {
			flipped[i] = (d + PolarityFlip) & 3
		}
		if flipped != pr[0].Dibits {
			t.Errorf("%s flipped != %s", pr[1].Name, pr[0].Name)
		}
	}
}

// TestRotateBurstDibits pins the de-rotation helper: the polarity flip
// is self-inverse (applying it twice round-trips), and k=0 is a no-op.
func TestRotateBurstDibits(t *testing.T) {
	var orig Burst
	for i := range orig.Dibits {
		orig.Dibits[i] = uint8(i) & 3
	}
	// Flip then recover.
	flipped := orig
	RotateBurstDibits(&flipped, PolarityFlip)
	RotateBurstDibits(&flipped, PolarityFlip)
	if flipped.Dibits != orig.Dibits {
		t.Errorf("double polarity-flip did not round-trip")
	}
	// Rotation 0 is a no-op.
	noop := orig
	RotateBurstDibits(&noop, 0)
	if noop.Dibits != orig.Dibits {
		t.Errorf("rotate-0 mutated the burst")
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
