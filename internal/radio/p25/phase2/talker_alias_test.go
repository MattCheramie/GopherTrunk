package phase2

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func TestTalkerAliasFragmentRoundTrip(t *testing.T) {
	in := TalkerAliasFragment{
		SourceID: 0x00ABCD, BlockIndex: 2, BlockCount: 4, Data: []byte("NAME"),
	}
	got, ok := EncodeTalkerAliasFragment(in).AsTalkerAliasFragment()
	if !ok {
		t.Fatal("AsTalkerAliasFragment returned !ok")
	}
	if got.SourceID != in.SourceID || got.BlockIndex != in.BlockIndex ||
		got.BlockCount != in.BlockCount || string(got.Data) != string(in.Data) {
		t.Errorf("round-trip = %+v, want %+v", got, in)
	}
}

func frag(src uint32, idx, count uint8, data string) TalkerAliasFragment {
	return TalkerAliasFragment{SourceID: src, BlockIndex: idx, BlockCount: count, Data: []byte(data)}
}

func TestTalkerAliasAssemblerInOrder(t *testing.T) {
	a := NewTalkerAliasAssembler(nil)
	if _, _, done := a.Add(frag(1, 0, 3, "FIRE")); done {
		t.Fatal("block 0 of 3 should not complete")
	}
	if _, _, done := a.Add(frag(1, 1, 3, " ENG")); done {
		t.Fatal("block 1 of 3 should not complete")
	}
	alias, src, done := a.Add(frag(1, 2, 3, "INE 1"))
	if !done {
		t.Fatal("block 2 of 3 should complete the alias")
	}
	if src != 1 || alias != "FIRE ENGINE 1" {
		t.Errorf("completed alias = (%d, %q), want (1, %q)", src, alias, "FIRE ENGINE 1")
	}
}

func TestTalkerAliasAssemblerOutOfOrder(t *testing.T) {
	a := NewTalkerAliasAssembler(nil)
	a.Add(frag(9, 2, 3, "INE 1"))
	a.Add(frag(9, 0, 3, "FIRE"))
	alias, _, done := a.Add(frag(9, 1, 3, " ENG"))
	if !done || alias != "FIRE ENGINE 1" {
		t.Errorf("out-of-order assembly = (%q, %v), want (%q, true)", alias, done, "FIRE ENGINE 1")
	}
}

func TestTalkerAliasAssemblerEvictsStale(t *testing.T) {
	clk := time.Unix(1000, 0)
	a := NewTalkerAliasAssembler(func() time.Time { return clk })

	if _, _, done := a.Add(frag(0xAA, 0, 2, "AB")); done {
		t.Fatal("one block of two should not complete")
	}
	// Advance past the staleness window; the next Add must evict the
	// stale block 0 so block 1 alone does not complete.
	clk = clk.Add(aliasStaleAfter + time.Second)
	if _, _, done := a.Add(frag(0xAA, 1, 2, "CD")); done {
		t.Error("stale block 0 should have been evicted; alias must not complete")
	}
}

// --- Real Motorola FACCH-S alias (#376) ----------------------------

// macPDU builds a MACPDU as ParseMACPDU would yield it from the given
// post-FEC info bytes (opcode + MFID stripped for vendor opcodes).
func macPDU(t *testing.T, info ...byte) MACPDU {
	t.Helper()
	p, err := ParseMACPDU(info)
	if err != nil {
		t.Fatalf("ParseMACPDU: %v", err)
	}
	return p
}

// The golden bytes are the SDRTrunk MSG dumps from issue #376
// (Victorian MMR, TG 20208, RADIO:ISSI 781824.356.200062, SEQUENCE 0,
// 2 blocks). Trailing FACCH CRCs (CC8 / 979 / 9D7) are dropped — the
// assembler reads only the cipher fragments.
var (
	aliasHeaderMSG = []byte{0x91, 0x90, 0x11, 0x4E, 0xF0, 0x02, 0x01, 0x00, 0x06,
		0xBE, 0xE0, 0x01, 0x64, 0x03, 0x0D, 0x7E, 0x24}
	aliasData1MSG = []byte{0x95, 0x90, 0x11, 0x01, 0x04,
		0x4F, 0x6F, 0xF2, 0xFA, 0x9A, 0xC3, 0xEC, 0x34, 0x43, 0x2F, 0xA6, 0x3C}
	aliasData2MSG = []byte{0x95, 0x90, 0x11, 0x02, 0x0C,
		0x81, 0xC3, 0xC5, 0xD9, 0x6A, 0x96, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
)

func TestAsMotorolaAliasHeaderParsesDump(t *testing.T) {
	h, ok := macPDU(t, aliasHeaderMSG...).AsMotorolaAliasHeader()
	if !ok {
		t.Fatal("AsMotorolaAliasHeader returned !ok")
	}
	if h.TalkgroupID != 20208 {
		t.Errorf("TalkgroupID = %d, want 20208", h.TalkgroupID)
	}
	if h.BlockCount != 2 {
		t.Errorf("BlockCount = %d, want 2", h.BlockCount)
	}
	if h.Sequence != 0 {
		t.Errorf("Sequence = %d, want 0", h.Sequence)
	}
	// First fragment must begin with the SUID prefix BE E0 01 64 03 0D 7E.
	if len(h.Fragment) < 7 || h.Fragment[0] != 0xBE || h.Fragment[6] != 0x7E {
		t.Errorf("Fragment prefix = % x, want BE E0 01 64 03 0D 7E…", h.Fragment)
	}
}

func TestAsMotorolaAliasDataParsesDump(t *testing.T) {
	d, ok := macPDU(t, aliasData1MSG...).AsMotorolaAliasData()
	if !ok {
		t.Fatal("AsMotorolaAliasData returned !ok")
	}
	if d.BlockNumber != 1 {
		t.Errorf("BlockNumber = %d, want 1", d.BlockNumber)
	}
	if d.Sequence != 0 {
		t.Errorf("Sequence = %d, want 0", d.Sequence)
	}
}

// TestMotorolaAliasAssemblerYieldsRID feeds the golden header + both
// data blocks and asserts the source RID falls out of the reassembled
// message prefix (200062 = 0x030D7E), matching SDRTrunk's inline
// RADIO:ISSI 781824.356.200062. The decoded alias *string* is not
// asserted — the dump doesn't give the plaintext for this call and the
// fragment bit-packing is air-unverified (see talker_alias.go caveat).
func TestMotorolaAliasAssemblerYieldsRID(t *testing.T) {
	a := NewMotorolaAliasAssembler(nil)

	h, _ := macPDU(t, aliasHeaderMSG...).AsMotorolaAliasHeader()
	if _, _, done := a.AddHeader(h); done {
		t.Fatal("header alone (2 blocks pending) must not complete")
	}
	d1, _ := macPDU(t, aliasData1MSG...).AsMotorolaAliasData()
	if _, _, done := a.AddData(d1); done {
		t.Fatal("1 of 2 blocks must not complete")
	}
	d2, _ := macPDU(t, aliasData2MSG...).AsMotorolaAliasData()
	_, rid, done := a.AddData(d2)
	if !done {
		t.Fatal("both blocks present should complete the alias")
	}
	if rid != 200062 {
		t.Errorf("source RID = %d, want 200062", rid)
	}
}

func TestMotorolaAliasAssemblerWrongOpcodeRejected(t *testing.T) {
	// A 0x95 PDU before any header must not complete or panic.
	d, _ := macPDU(t, aliasData1MSG...).AsMotorolaAliasData()
	a := NewMotorolaAliasAssembler(nil)
	if _, _, done := a.AddData(d); done {
		t.Error("data before header must not complete")
	}
}

func TestControlChannelPublishesTalkerAlias(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "p2", FrequencyHz: 851_000_000})
	cc.Ingest(EncodeTalkerAliasFragment(frag(0x1234, 0, 2, "UNIT")))
	cc.Ingest(EncodeTalkerAliasFragment(frag(0x1234, 1, 2, "-7")))

	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindTalkerAlias {
				ta := ev.Payload.(trunking.TalkerAlias)
				if ta.SourceID != 0x1234 || ta.Alias != "UNIT-7" {
					t.Errorf("talker alias = (%#x, %q), want (0x1234, %q)",
						ta.SourceID, ta.Alias, "UNIT-7")
				}
				return
			}
		default:
			t.Fatal("no KindTalkerAlias event published")
		}
	}
}
