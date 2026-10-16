package nxdn

import (
	"math/rand"
	"testing"
)

func noiseDibits(r *rand.Rand, n int) []uint8 {
	d := make([]uint8, n)
	for i := range d {
		d[i] = uint8(r.Intn(4))
	}
	return d
}

func TestTrafficChannelDecodesVoiceFrames(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	stream := noiseDibits(r, 57)
	var sent [][4][]byte
	for i := 0; i < 3; i++ {
		p := randPayloads(r)
		sent = append(sent, p)
		f, err := EncodeAirVoiceFrame(p)
		if err != nil {
			t.Fatal(err)
		}
		stream = append(stream, f...)
	}
	stream = append(stream, noiseDibits(r, 40)...)

	var got []VCHFrame
	tc := NewTrafficChannel(func(fr []VCHFrame) { got = append(got, fr...) })
	// Feed in odd-sized chunks: framing must not depend on chunk edges.
	for i := 0; i < len(stream); i += 37 {
		tc.Process(stream[i:min(i+37, len(stream))], i)
	}
	if len(got) != 12 {
		t.Fatalf("decoded %d voice frames, want 12 (frames=%d lich_ok=%d)", len(got), tc.Frames, tc.LICHOK)
	}
	for k, f := range got {
		want := sent[k/4][k%4]
		for i := range want {
			if f.Payload[i] != want[i] {
				t.Fatalf("voice frame %d bit %d differs", k, i)
			}
		}
	}
}

// Once framed, the next FSW is accepted with up to airFSWLockTolerance
// symbol errors at exactly one frame later; a cold search allows 1.
func TestTrafficChannelFlywheel(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	f1, _ := EncodeAirVoiceFrame(randPayloads(r))
	f2, _ := EncodeAirVoiceFrame(randPayloads(r))
	for _, k := range []int{0, 4, 7} { // three FSW symbol errors in frame 2
		f2[k] ^= 2
	}
	stream := append(append(noiseDibits(r, 20), f1...), f2...)
	n := 0
	tc := NewTrafficChannel(func(fr []VCHFrame) { n += len(fr) })
	tc.Process(stream, 0)
	if n != 8 {
		t.Fatalf("locked: %d voice frames, want 8", n)
	}

	// The same damaged frame alone is not found by a cold search.
	n = 0
	tc = NewTrafficChannel(func(fr []VCHFrame) { n += len(fr) })
	tc.Process(append(noiseDibits(r, 20), f2...), 0)
	if n != 0 {
		t.Fatalf("cold search accepted an FSW with 3 errors (%d voice frames)", n)
	}
}

func TestTrafficChannelIgnoresNonVoiceAndBadLICH(t *testing.T) {
	r := rand.New(rand.NewSource(4))
	f, _ := EncodeAirVoiceFrame(randPayloads(r))
	// Flip the first LICH bit (RFCT MSB) after scrambling: parity fails.
	f[AirFSWDibits] ^= 2
	called := false
	tc := NewTrafficChannel(func([]VCHFrame) { called = true })
	tc.Process(append(noiseDibits(r, 12), f...), 0)
	if called || tc.Frames != 1 || tc.LICHOK != 0 {
		t.Fatalf("bad-LICH frame: called=%v frames=%d lich_ok=%d", called, tc.Frames, tc.LICHOK)
	}
}

func TestTrafficChannelReset(t *testing.T) {
	r := rand.New(rand.NewSource(5))
	f, _ := EncodeAirVoiceFrame(randPayloads(r))
	n := 0
	tc := NewTrafficChannel(func(fr []VCHFrame) { n += len(fr) })
	tc.Process(f[:100], 0) // mid-frame
	tc.Reset()
	tc.Process(append(noiseDibits(r, 15), f...), 100)
	if n != 4 {
		t.Fatalf("after Reset: %d voice frames, want 4", n)
	}
}
