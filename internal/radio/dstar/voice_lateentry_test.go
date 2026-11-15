package dstar

import (
	"math/rand"
	"testing"
)

// lateEntryStream is nFrames DV frames of distinct random AMBE payloads
// (frame 0 carries the slow-data sync, as does every 21st), as bits.
func lateEntryStream(t *testing.T, r *rand.Rand, nFrames int) (bits []uint8, payloads [][]byte) {
	t.Helper()
	for f := 0; f < nFrames; f++ {
		info := make([]byte, 49)
		for i := range info {
			info[i] = byte(r.Intn(2))
		}
		payloads = append(payloads, info)
		voice, err := EncodeDVVoiceBits(info)
		if err != nil {
			t.Fatal(err)
		}
		bits = append(bits, EncodeDVFrame(voice, f, [3]byte{byte(r.Intn(256)), 'G', 'T'})...)
	}
	return bits, payloads
}

func collectFrames(bits []uint8, chunk int) [][]byte {
	var got [][]byte
	vc := NewVoiceChannel(func(f DVFrame) { got = append(got, f.Payload) })
	for i := 0; i < len(bits); i += chunk {
		vc.Process(bits[i:min(i+chunk, len(bits))], i)
	}
	return got
}

// A receiver that joins mid-cycle has already demodulated the frames
// before the first slow-data sync; they must be decoded, not dropped
// (issue #1261: up to 0.42 s of speech per join).
func TestVoiceChannelLateEntryDecodesFramesBeforeFirstSync(t *testing.T) {
	r := rand.New(rand.NewSource(5))
	bits, payloads := lateEntryStream(t, r, 60)
	for _, inv := range []bool{false, true} {
		in := append([]uint8(nil), bits...)
		if inv {
			in = invertBits(in)
		}
		// Join 3 frames and 17 bits into the transmission: frame 3 is cut,
		// frames 4..20 are whole and precede the first sync (frame 21).
		const cut = 3*DVFrameBits + 17
		for _, chunk := range []int{1, 97, 5000} {
			got := collectFrames(in[cut:], chunk)
			if len(got) != 56 {
				t.Fatalf("inverted=%v chunk %d: %d frames, want 56 (frames 4..59)", inv, chunk, len(got))
			}
			for i, g := range got {
				if string(g) != string(payloads[4+i]) {
					t.Fatalf("inverted=%v chunk %d: output %d is not frame %d", inv, chunk, i, 4+i)
				}
			}
		}
	}
}

// What precedes a transmission is not voice: noise, the header, the bits
// of a receiver still pulling in. None of it may be turned into frames.
func TestVoiceChannelLateEntryIgnoresNonVoice(t *testing.T) {
	r := rand.New(rand.NewSource(6))
	extra := 0
	for trial := 0; trial < 200; trial++ {
		junk := make([]uint8, retroHistoryBits+500)
		for i := range junk {
			junk[i] = uint8(r.Intn(2))
		}
		bits, _ := lateEntryStream(t, r, 5)
		got := collectFrames(append(junk, bits...), 4096)
		extra += len(got) - 5
	}
	// 20 junk frames per trial at ~2.6e-4 each, and the walk stops at the
	// first failure, so only the frame next to the sync can pass: ~0.05
	// expected over 200 trials.
	if extra > 1 {
		t.Fatalf("%d frames decoded from random bits ahead of 200 transmissions, want ≤ 1", extra)
	}
}

// A new transmission must not reach back across the end of the previous
// one, nor re-emit frames the cadence already delivered.
func TestVoiceChannelLateEntryStopsAtEndPatternAndEmittedFrames(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	first, p1 := lateEntryStream(t, r, 30)
	second, p2 := lateEntryStream(t, r, 30)
	// The second transmission is joined 2 frames in, straight after the
	// first one's end pattern.
	in := append(append(append([]uint8(nil), first...), EndSyncBitsSlice()...), second[2*DVFrameBits:]...)
	got := collectFrames(in, 333)
	want := append(append([][]byte(nil), p1...), p2[2:]...)
	if len(got) != len(want) {
		t.Fatalf("%d frames, want %d (30 + 28, each once)", len(got), len(want))
	}
	for i := range got {
		if string(got[i]) != string(want[i]) {
			t.Fatalf("output %d differs from the frame sent", i)
		}
	}
}
