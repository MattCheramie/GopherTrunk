package voice

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// aliasFragmentsFromInfo is lcFragments for a raw 9-octet talker-alias
// info block (whose octets 2..8 are alias text, not FLC address fields).
func aliasFragmentsFromInfo(t *testing.T, info []byte) [4][]byte {
	t.Helper()
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

// TestDecoderSurfacesTalkerAlias builds one superframe whose embedded LC
// is a talker-alias header and confirms the decoder surfaces the parsed
// fragment on VoiceSuperframe rather than misreading it as a group-voice LC.
func TestDecoderSurfacesTalkerAlias(t *testing.T) {
	info := dmr.AssembleTalkerAliasFragments(dmr.TalkerAliasFormatUTF8, "Truck 349")[0]
	aliasFrags := aliasFragmentsFromInfo(t, info)

	var frames [][]byte
	for f := 0; f < FramesPerSuperframe; f++ {
		frames = append(frames, mkFrame(f))
	}
	var stream []uint8
	stream = make([]uint8, 200) // leading filler
	for b := 0; b < BurstsPerSuperframe; b++ {
		sync := dmr.BSData.Dibits
		switch {
		case b == 0:
			sync = dmr.BSVoice.Dibits
		case b >= 1 && b <= 4:
			sync = embeddedSync(dmr.EMB{ColorCode: 1, LCSS: burstLCSS(b)}, aliasFrags[b-1])
		}
		base := b * FramesPerBurst
		stream = append(stream, makeVoiceBurst(frames[base:base+FramesPerBurst], sync)...)
	}
	stream = append(stream, make([]uint8, interleaveTail)...)

	d := NewDecoder()
	sfs := d.Process(stream, 0)
	if len(sfs) == 0 {
		t.Fatal("no superframe decoded")
	}
	sf := sfs[0]
	if !sf.HasTalkerAlias {
		t.Fatal("HasTalkerAlias not set on an alias-carrying superframe")
	}
	if !sf.TalkerAlias.Header || sf.TalkerAlias.Length != 9 ||
		sf.TalkerAlias.Format != dmr.TalkerAliasFormatUTF8 {
		t.Fatalf("surfaced alias fragment wrong: hdr=%v len=%d fmt=%d",
			sf.TalkerAlias.Header, sf.TalkerAlias.Length, sf.TalkerAlias.Format)
	}
}
