package dmr

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

func TestEMBSplitAssembleRoundTrip(t *testing.T) {
	emb := EMB{ColorCode: 0xB, PI: true, LCSS: LCSSFirst}
	frag := make([]byte, framing.EmbeddedFragmentBits)
	for i := range frag {
		frag[i] = byte((i*7 + 3) & 1)
	}

	field := AssembleEmbeddedField(emb, frag)
	if len(field) != SyncBitCount {
		t.Fatalf("field length = %d, want %d", len(field), SyncBitCount)
	}
	gotEMB, gotFrag := SplitEmbeddedField(field)
	if gotEMB != emb {
		t.Errorf("EMB round trip = %+v, want %+v", gotEMB, emb)
	}
	for i := range frag {
		if gotFrag[i] != frag[i] {
			t.Fatalf("fragment bit %d mismatch", i)
		}
	}
}

// flcToFragments encodes an FLC into the four 32-bit embedded fragments a
// superframe's bursts B–E would carry.
func flcToFragments(t *testing.T, f FLC) [4][]byte {
	t.Helper()
	info := AssembleFLC(f) // 9 octets
	lc := make([]byte, framing.EmbLCBits)
	for i := 0; i < framing.EmbLCBits; i++ {
		lc[i] = (info[i/8] >> uint(7-(i%8))) & 1
	}
	channel := framing.EncodeEmbeddedLC(lc)
	var frags [4][]byte
	for i := range frags {
		frags[i] = channel[i*framing.EmbeddedFragmentBits : (i+1)*framing.EmbeddedFragmentBits]
	}
	return frags
}

func TestReassembleEmbeddedLC(t *testing.T) {
	want := FLC{
		FLCO:           FLCOGroupVoiceUser,
		FID:            0,
		ServiceOptions: 0x80, // emergency
		DstAddr:        0x00ABCD,
		SrcAddr:        0x123456,
	}
	frags := flcToFragments(t, want)

	got, ok := ReassembleEmbeddedLC(frags)
	if !ok {
		t.Fatal("ReassembleEmbeddedLC failed on a clean embedded LC")
	}
	if got.FLCO != want.FLCO || got.DstAddr != want.DstAddr || got.SrcAddr != want.SrcAddr {
		t.Errorf("reassembled FLC = %+v, want %+v", got, want)
	}
	gv, isGroup := got.AsGroupVoiceUser()
	if !isGroup || gv.GroupAddress != want.DstAddr || gv.SourceID != want.SrcAddr || !gv.Emergency {
		t.Errorf("group voice user = %+v (ok=%v), want group %X src %X emergency", gv, isGroup, want.DstAddr, want.SrcAddr)
	}
}

func TestReassembleEmbeddedLCRejectsCorruption(t *testing.T) {
	frags := flcToFragments(t, FLC{FLCO: FLCOGroupVoiceUser, DstAddr: 100, SrcAddr: 7})
	// Corrupt two bits in the first fragment's first row (on-air indices
	// 0 and 8 both map to matrix row 0) → uncorrectable → rejected.
	frags[0] = append([]byte(nil), frags[0]...)
	frags[0][0] ^= 1
	frags[0][8] ^= 1
	if _, ok := ReassembleEmbeddedLC(frags); ok {
		t.Error("corrupted embedded LC was accepted")
	}
}

func TestReassembleEmbeddedLCWrongLength(t *testing.T) {
	var frags [4][]byte
	frags[0] = make([]byte, framing.EmbeddedFragmentBits)
	// frags[1..3] left nil (wrong length) → must reject, not panic.
	if _, ok := ReassembleEmbeddedLC(frags); ok {
		t.Error("accepted fragments of the wrong length")
	}
}
