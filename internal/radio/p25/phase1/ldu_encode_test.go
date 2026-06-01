package phase1

import (
	"bytes"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/voice/imbe"
)

func TestAssembleLDURoundTrip(t *testing.T) {
	var voice [LDUVoiceSubframeCount][]byte
	var want [LDUVoiceSubframeCount][]byte
	for s := range voice {
		info := make([]byte, imbe.InfoBits)
		for i := range info {
			info[i] = byte((s*7 + i*3) & 1)
		}
		// Build the real on-air burst (FEC + §7.4 scramble + §7.5
		// interleave) so the round-trip exercises the deinterleave
		// in ExtractVoiceFrames → DecodeChannelToFrame.
		onAir, err := imbe.EncodeFrameToChannel(info)
		if err != nil {
			t.Fatalf("EncodeFrameToChannel: %v", err)
		}
		voice[s] = onAir
		// The recovered frame must equal the original info bits packed
		// MSB-first — not a re-decode of the same bits, so a no-op
		// permutation can't make the test pass against itself.
		want[s], err = imbe.PackInfoBitsToFrame(info)
		if err != nil {
			t.Fatalf("PackInfoBitsToFrame: %v", err)
		}
	}
	var lces [LDULCESBlockCount][]byte
	var lsd [LDULSDBlockCount][]byte

	ldu, err := AssembleLDU(0x123, DUIDLogicalLink1, voice, lces, lsd)
	if err != nil {
		t.Fatalf("AssembleLDU: %v", err)
	}
	if len(ldu) != LDUTotalBits {
		t.Fatalf("AssembleLDU len = %d, want %d", len(ldu), LDUTotalBits)
	}

	frames, errs, err := ExtractVoiceFrames(ldu)
	if err != nil {
		t.Fatalf("ExtractVoiceFrames: %v", err)
	}
	if errs != 0 {
		t.Errorf("ExtractVoiceFrames corrected %d bits on a clean LDU, want 0", errs)
	}
	for i := range frames {
		if !bytes.Equal(frames[i], want[i]) {
			t.Errorf("subframe %d: extracted frame %x != original info frame %x", i, frames[i], want[i])
		}
	}
}
