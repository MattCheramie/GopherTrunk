package dmr

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// embeddedFragsFromInfo runs a 9-octet info block through the real
// embedded-LC BPTC encoder and splits it into the four 32-bit fragments
// bursts B–E carry — the inverse of ReassembleEmbeddedLCInfo.
func embeddedFragsFromInfo(t *testing.T, info []byte) [4][]byte {
	t.Helper()
	if len(info) < framing.EmbLCBits/8 {
		t.Fatalf("info too short: %d octets", len(info))
	}
	lc := make([]byte, framing.EmbLCBits)
	for i := 0; i < framing.EmbLCBits; i++ {
		lc[i] = (info[i/8] >> uint(7-(i%8))) & 1
	}
	ch := framing.EncodeEmbeddedLC(lc)
	var frags [4][]byte
	for i := range frags {
		frags[i] = ch[i*framing.EmbeddedFragmentBits : (i+1)*framing.EmbeddedFragmentBits]
	}
	return frags
}

// TestReassembleEmbeddedLCInfoCarriesAliasOctets confirms the raw-info
// accessor survives the BPTC round-trip and that ParseTalkerAliasFragment
// recovers a talker-alias header from the recovered octets — the octets
// the generic FLC view would misread as address fields.
func TestReassembleEmbeddedLCInfoCarriesAliasOctets(t *testing.T) {
	header := AssembleTalkerAliasFragments(TalkerAliasFormatUTF8, "Truck 349")[0]
	frags := embeddedFragsFromInfo(t, header)

	info, flc, ok := ReassembleEmbeddedLCInfo(frags)
	if !ok {
		t.Fatal("ReassembleEmbeddedLCInfo failed on a clean alias header")
	}
	if flc.FLCO != FLCOTalkerAlias {
		t.Fatalf("recovered FLCO: got %v want TalkerAlias header", flc.FLCO)
	}
	frag, ok := ParseTalkerAliasFragment(info)
	if !ok || !frag.Header {
		t.Fatalf("alias header not recovered from raw info: ok=%v hdr=%v", ok, frag.Header)
	}
	if frag.Length != 9 || frag.Format != TalkerAliasFormatUTF8 {
		t.Fatalf("recovered header fields wrong: len=%d fmt=%d", frag.Length, frag.Format)
	}
}
