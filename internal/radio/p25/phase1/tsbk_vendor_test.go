package phase1

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func TestAsMotorolaPatchGroupRoundTrip(t *testing.T) {
	in := MotorolaPatchGroup{SuperGroup: 0x1234, Patched: []uint16{10, 20}}
	tsbk := TSBK{Opcode: OpMotorolaPatchGroupAdd, MFID: MFIDMotorola,
		Payload: AssembleMotorolaPatchGroup(in)}
	got, ok := tsbk.AsMotorolaPatchGroup()
	if !ok {
		t.Fatal("AsMotorolaPatchGroup returned !ok")
	}
	if got.SuperGroup != in.SuperGroup || len(got.Patched) != 2 ||
		got.Patched[0] != 10 || got.Patched[1] != 20 {
		t.Errorf("round-trip = %+v, want %+v", got, in)
	}
	// A standard-MFID TSBK with the same opcode must not match.
	tsbk.MFID = MFIDStandard
	if _, ok := tsbk.AsMotorolaPatchGroup(); ok {
		t.Error("AsMotorolaPatchGroup matched a standard-MFID TSBK")
	}
}

func TestAsMotorolaPatchDelete(t *testing.T) {
	tsbk := TSBK{Opcode: OpMotorolaPatchGroupDelete, MFID: MFIDMotorola,
		Payload: [8]byte{0x05, 0x55}}
	super, ok := tsbk.AsMotorolaPatchDelete()
	if !ok || super != 0x0555 {
		t.Errorf("AsMotorolaPatchDelete = (%#x, %v), want (0x555, true)", super, ok)
	}
}

func TestAsHarrisRegroupRoundTrip(t *testing.T) {
	in := HarrisRegroup{RegroupGroup: 0x0777, TargetID: 0x00BEEF}
	tsbk := TSBK{Opcode: OpHarrisRegroup, MFID: MFIDHarris, Payload: AssembleHarrisRegroup(in)}
	got, ok := tsbk.AsHarrisRegroup()
	if !ok || got != in {
		t.Errorf("AsHarrisRegroup = (%+v, %v), want %+v", got, ok, in)
	}
}

func TestAsTalkerAliasFragmentRoundTrip(t *testing.T) {
	in := TalkerAliasFragment{SourceID: 0x00ABCD, BlockIndex: 1, BlockCount: 3, Data: []byte("ABC")}
	tsbk := TSBK{Opcode: OpVendorTalkerAlias, MFID: MFIDMotorola,
		Payload: AssembleTalkerAliasFragment(in)}
	got, ok := tsbk.AsTalkerAliasFragment()
	if !ok {
		t.Fatal("AsTalkerAliasFragment returned !ok")
	}
	if got.SourceID != in.SourceID || got.BlockIndex != in.BlockIndex ||
		got.BlockCount != in.BlockCount || string(got.Data) != "ABC" {
		t.Errorf("round-trip = %+v, want %+v", got, in)
	}
}

func TestTalkerAliasAssembler(t *testing.T) {
	a := NewTalkerAliasAssembler(nil)
	if _, _, done := a.Add(TalkerAliasFragment{SourceID: 1, BlockIndex: 2, BlockCount: 3, Data: []byte("INE 1")}); done {
		t.Fatal("one of three blocks should not complete")
	}
	a.Add(TalkerAliasFragment{SourceID: 1, BlockIndex: 0, BlockCount: 3, Data: []byte("FIRE")})
	alias, src, done := a.Add(TalkerAliasFragment{SourceID: 1, BlockIndex: 1, BlockCount: 3, Data: []byte(" ENG")})
	if !done || src != 1 || alias != "FIRE ENGINE 1" {
		t.Errorf("assembly = (%q, %d, %v), want (%q, 1, true)", alias, src, done, "FIRE ENGINE 1")
	}
}

func TestTalkerAliasAssemblerEvictsStale(t *testing.T) {
	clk := time.Unix(1000, 0)
	a := NewTalkerAliasAssembler(func() time.Time { return clk })
	a.Add(TalkerAliasFragment{SourceID: 0xAA, BlockIndex: 0, BlockCount: 2, Data: []byte("AB")})
	clk = clk.Add(aliasStaleAfter + time.Second)
	if _, _, done := a.Add(TalkerAliasFragment{SourceID: 0xAA, BlockIndex: 1, BlockCount: 2, Data: []byte("CD")}); done {
		t.Error("stale block 0 should have been evicted; alias must not complete")
	}
}

func TestControlChannelPublishesMotorolaPatch(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "S"})
	add := TSBK{Opcode: OpMotorolaPatchGroupAdd, MFID: MFIDMotorola,
		Payload: AssembleMotorolaPatchGroup(MotorolaPatchGroup{SuperGroup: 0x1234, Patched: []uint16{10, 20}})}
	del := TSBK{Opcode: OpMotorolaPatchGroupDelete, MFID: MFIDMotorola,
		Payload: [8]byte{0x12, 0x34}}
	cc.Process(buildLockedStreamWithTSBK(10, 0x293, DUIDTrunkingSignaling, add), 0)
	cc.Process(buildLockedStreamWithTSBK(0, 0x293, DUIDTrunkingSignaling, del), 1<<20)

	var patches []trunking.Patch
	deadline := time.After(time.Second)
	for len(patches) < 2 {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindPatch {
				patches = append(patches, ev.Payload.(trunking.Patch))
			}
		case <-deadline:
			t.Fatalf("got %d patch events, want 2", len(patches))
		}
	}
	if !patches[0].Add || patches[0].SuperGroup != 0x1234 || len(patches[0].Members) != 2 {
		t.Errorf("add patch = %+v", patches[0])
	}
	if patches[1].Add || patches[1].SuperGroup != 0x1234 {
		t.Errorf("delete patch = %+v, want Add false / super 0x1234", patches[1])
	}
}

func TestControlChannelPublishesTalkerAlias(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "S"})
	f0 := TSBK{Opcode: OpVendorTalkerAlias, MFID: MFIDMotorola,
		Payload: AssembleTalkerAliasFragment(TalkerAliasFragment{SourceID: 0x123, BlockIndex: 0, BlockCount: 2, Data: []byte("UN")})}
	f1 := TSBK{Opcode: OpVendorTalkerAlias, MFID: MFIDMotorola,
		Payload: AssembleTalkerAliasFragment(TalkerAliasFragment{SourceID: 0x123, BlockIndex: 1, BlockCount: 2, Data: []byte("IT")})}
	cc.Process(buildLockedStreamWithTSBK(10, 0x293, DUIDTrunkingSignaling, f0), 0)
	cc.Process(buildLockedStreamWithTSBK(0, 0x293, DUIDTrunkingSignaling, f1), 1<<20)

	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindTalkerAlias {
				ta := ev.Payload.(trunking.TalkerAlias)
				if ta.SourceID != 0x123 || ta.Alias != "UNIT" {
					t.Errorf("talker alias = (%#x, %q), want (0x123, %q)", ta.SourceID, ta.Alias, "UNIT")
				}
				return
			}
		case <-deadline:
			t.Fatal("no KindTalkerAlias event")
		}
	}
}
