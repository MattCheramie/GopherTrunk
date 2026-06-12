package dmr

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// No real off-air DMR Reverse Channel capture exists; these tests pin the
// 11-info / 21-parity framing and the embedded-field integration via synthetic
// round-trips. See rc.go for the capture-pending FEC posture.

func TestReverseChannelRoundTrip(t *testing.T) {
	rc := ReverseChannel{Info: 0x5A3, Parity: 0x1F2A3}
	frag := AssembleReverseChannel(rc)
	if len(frag) != framing.EmbeddedFragmentBits {
		t.Fatalf("fragment length = %d, want %d", len(frag), framing.EmbeddedFragmentBits)
	}
	got, ok := DecodeReverseChannel(frag)
	if !ok {
		t.Fatal("DecodeReverseChannel returned ok=false on a valid RC fragment")
	}
	if got != rc {
		t.Errorf("round trip = %+v, want %+v", got, rc)
	}
}

// TestReverseChannelInfoBitsMSBFirst confirms the 11-bit Info occupies the
// leading bits of the 32-bit field, MSB-first, ahead of the 21 parity bits.
func TestReverseChannelInfoBitsMSBFirst(t *testing.T) {
	frag := AssembleReverseChannel(ReverseChannel{Info: 1})
	if frag[RCInfoBits-1] != 1 {
		t.Errorf("Info LSB landed at bit %d = %d, want it set", RCInfoBits-1, frag[RCInfoBits-1])
	}
	for i := 0; i < RCInfoBits-1; i++ {
		if frag[i] != 0 {
			t.Errorf("unexpected set bit %d in the info region", i)
		}
	}
}

func TestReverseChannelRejectsNull(t *testing.T) {
	null := make([]byte, framing.EmbeddedFragmentBits) // all-zero = ETSI idle
	if _, ok := DecodeReverseChannel(null); ok {
		t.Error("DecodeReverseChannel accepted the all-zero null embedded message")
	}
}

func TestReverseChannelRejectsWrongLength(t *testing.T) {
	if _, ok := DecodeReverseChannel(make([]byte, 16)); ok {
		t.Error("DecodeReverseChannel accepted a 16-bit fragment")
	}
}

// TestReverseChannelThroughEmbeddedField confirms RC survives the same EMB
// framing split a burst's 48-bit centre field goes through: pack an EMB
// (LCSS=Single) + RC fragment, split it back out, and decode.
func TestReverseChannelThroughEmbeddedField(t *testing.T) {
	rc := ReverseChannel{Info: 0x2C7, Parity: 0x155AA}
	field := AssembleEmbeddedField(EMB{ColorCode: 0xB, LCSS: LCSSSingle}, AssembleReverseChannel(rc))
	emb, frag := SplitEmbeddedField(field)
	if emb.LCSS != LCSSSingle {
		t.Fatalf("LCSS = %d, want LCSSSingle", emb.LCSS)
	}
	got, ok := DecodeReverseChannel(frag)
	if !ok || got != rc {
		t.Errorf("RC through embedded field = %+v ok=%v, want %+v", got, ok, rc)
	}
}
