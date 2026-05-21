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
