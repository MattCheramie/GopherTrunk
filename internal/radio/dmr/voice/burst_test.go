package voice

import (
	"bytes"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
)

// mkFrame builds a deterministic, distinct 72-bit AMBE frame. The
// first six bits encode seed in binary so a frame-ordering bug is
// obvious in test output; the remainder is deterministic filler.
func mkFrame(seed int) []byte {
	f := make([]byte, AMBEFrameBits)
	for i := 0; i < 6; i++ {
		f[i] = byte((seed >> (5 - i)) & 1)
	}
	for i := 6; i < AMBEFrameBits; i++ {
		f[i] = byte((seed*3 + i*5) & 1)
	}
	return f
}

// bitsToDibits packs a bit slice (one bit per byte, MSB-first) into
// dibits.
func bitsToDibits(bits []byte) []uint8 {
	d := make([]uint8, len(bits)/2)
	for i := range d {
		d[i] = bits[2*i]<<1 | bits[2*i+1]
	}
	return d
}

// makeVoiceBurst assembles a 132-dibit voice burst from three 72-bit
// AMBE frames and a 24-dibit sync / embedded-signalling field.
func makeVoiceBurst(frames [][]byte, sync [24]uint8) []uint8 {
	var voiceBits []byte
	for _, f := range frames {
		voiceBits = append(voiceBits, f...)
	}
	burst := make([]uint8, 0, dmr.BurstDibits)
	burst = append(burst, bitsToDibits(voiceBits[:108])...)
	burst = append(burst, sync[:]...)
	burst = append(burst, bitsToDibits(voiceBits[108:])...)
	return burst
}

func TestVoiceBitsLength(t *testing.T) {
	var b dmr.Burst
	if got := len(VoiceBits(&b)); got != VoicePayloadBits {
		t.Errorf("VoiceBits length = %d, want %d", got, VoicePayloadBits)
	}
}

func TestAMBEFramesRoundTrip(t *testing.T) {
	in := [][]byte{mkFrame(1), mkFrame(2), mkFrame(3)}
	burstDibits := makeVoiceBurst(in, dmr.BSVoice.Dibits)

	var b dmr.Burst
	copy(b.Dibits[:], burstDibits)

	got := AMBEFrames(&b)
	for i := 0; i < FramesPerBurst; i++ {
		if !bytes.Equal(got[i], in[i]) {
			t.Errorf("frame %d round trip:\n got %v\nwant %v", i, got[i], in[i])
		}
	}
}

func TestAMBEFramesSkipSyncField(t *testing.T) {
	// The 24-dibit centre field must not leak into any AMBE frame:
	// fill it with a marker and confirm the extracted frames still
	// match the input regardless of the centre's contents.
	in := [][]byte{mkFrame(7), mkFrame(8), mkFrame(9)}
	var marker [24]uint8
	for i := range marker {
		marker[i] = 3
	}
	var b dmr.Burst
	copy(b.Dibits[:], makeVoiceBurst(in, marker))

	got := AMBEFrames(&b)
	for i := 0; i < FramesPerBurst; i++ {
		if !bytes.Equal(got[i], in[i]) {
			t.Errorf("frame %d: centre field leaked into payload", i)
		}
	}
}
