package voice

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
)

// No real off-air Reverse Channel capture exists; this drives a synthetic
// single-slot superframe whose burst B carries a single-fragment
// (LCSS=Single) embedded RC word and confirms the decoder surfaces it on the
// VoiceSuperframe. See dmr/rc.go for the capture-pending FEC posture.
func TestDecoderSurfacesReverseChannel(t *testing.T) {
	rc := dmr.ReverseChannel{Info: 0x3A5, Parity: 0x14B2C}
	rcSync := embeddedSync(dmr.EMB{ColorCode: 1, LCSS: dmr.LCSSSingle}, dmr.AssembleReverseChannel(rc))

	frames := make([][]byte, FramesPerSuperframe)
	for f := range frames {
		frames[f] = mkFrame(f)
	}
	var stream []uint8
	for b := 0; b < BurstsPerSuperframe; b++ {
		sync := dmr.BSData.Dibits
		switch b {
		case 0:
			sync = dmr.BSVoice.Dibits // burst A anchors the superframe
		case 1:
			sync = rcSync // burst B carries the embedded reverse channel
		}
		base := b * FramesPerBurst
		stream = append(stream, makeVoiceBurst(frames[base:base+FramesPerBurst], sync)...)
	}

	got := NewDecoder().Process(stream, 0)
	if len(got) != 1 {
		t.Fatalf("got %d superframes, want 1", len(got))
	}
	if !got[0].HasRC {
		t.Fatal("HasRC = false; want the burst-B reverse channel decoded")
	}
	if got[0].RC != rc {
		t.Errorf("RC = %+v, want %+v", got[0].RC, rc)
	}
}

// TestDecoderNoReverseChannelFromLC confirms an ordinary embedded Full LC
// (LCSS First/Cont/Cont/Last) never surfaces a bogus RC.
func TestDecoderNoReverseChannelFromLC(t *testing.T) {
	aLC := dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: 1001, SrcAddr: 55}
	bLC := dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: 2002, SrcAddr: 66}
	stream, _, _ := buildInterleavedStreamWithLC(t, aLC, bLC, 0)
	for _, sf := range NewInterleavedDecoder().Process(stream, 0) {
		if sf.HasRC {
			t.Errorf("HasRC = true on an LC-only superframe (phase %d); want false", sf.Phase)
		}
	}
}
