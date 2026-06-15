package phase1

import (
	"bytes"
	"log/slog"
	"strings"
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

// TestAsMotorolaPatchGroupSingleMemberTriplicated pins the dedup
// against the live Mt Anakie observation: a Motorola patch with one
// active member encodes the same talkgroup ID in all three member
// slots, and the parser must report Patched = [ID] not [ID, ID, ID].
// Without dedup the duplicate triples flow through PatchRegistry,
// Grant.PatchedGroups, and the OpenMHz broadcaster's patch_list JSON
// — issue #275 retest log showed `members="[32501 32501 32501]"`
// for what was semantically a one-member patch.
func TestAsMotorolaPatchGroupSingleMemberTriplicated(t *testing.T) {
	// Mt Anakie capture: super-group 33601 (0x8341), member 32501 (0x7EF5)
	// triplicated.
	payload := [8]byte{0x83, 0x41, 0x7E, 0xF5, 0x7E, 0xF5, 0x7E, 0xF5}
	tsbk := TSBK{Opcode: OpMotorolaPatchGroupAdd, MFID: MFIDMotorola, Payload: payload}
	got, ok := tsbk.AsMotorolaPatchGroup()
	if !ok {
		t.Fatal("AsMotorolaPatchGroup returned !ok")
	}
	if got.SuperGroup != 0x8341 {
		t.Errorf("SuperGroup = %#x, want 0x8341", got.SuperGroup)
	}
	if len(got.Patched) != 1 || got.Patched[0] != 0x7EF5 {
		t.Errorf("Patched = %v, want [0x7EF5] (single logical member)", got.Patched)
	}
}

// TestAsMotorolaPatchGroupTwoDistinctOneRepeated covers the
// two-member case where the second slot is repeated as filler in the
// third: payload encodes ID-A, ID-B, ID-A — semantically two members.
func TestAsMotorolaPatchGroupTwoDistinctOneRepeated(t *testing.T) {
	payload := [8]byte{0x12, 0x34, 0x00, 0x0A, 0x00, 0x14, 0x00, 0x0A}
	tsbk := TSBK{Opcode: OpMotorolaPatchGroupAdd, MFID: MFIDMotorola, Payload: payload}
	got, ok := tsbk.AsMotorolaPatchGroup()
	if !ok {
		t.Fatal("AsMotorolaPatchGroup returned !ok")
	}
	if len(got.Patched) != 2 || got.Patched[0] != 10 || got.Patched[1] != 20 {
		t.Errorf("Patched = %v, want [10, 20] (dedup of [10, 20, 10])", got.Patched)
	}
}

// TestAsMotorolaPatchGroupThreeDistinct guards that the dedup does
// not collapse a genuine three-member patch where each slot carries a
// distinct talkgroup. Without this the dedup might over-fire.
func TestAsMotorolaPatchGroupThreeDistinct(t *testing.T) {
	payload := [8]byte{0x12, 0x34, 0x00, 0x0A, 0x00, 0x14, 0x00, 0x1E}
	tsbk := TSBK{Opcode: OpMotorolaPatchGroupAdd, MFID: MFIDMotorola, Payload: payload}
	got, ok := tsbk.AsMotorolaPatchGroup()
	if !ok {
		t.Fatal("AsMotorolaPatchGroup returned !ok")
	}
	want := []uint16{10, 20, 30}
	if len(got.Patched) != 3 {
		t.Fatalf("Patched = %v, want %v", got.Patched, want)
	}
	for i, w := range want {
		if got.Patched[i] != w {
			t.Errorf("Patched[%d] = %d, want %d", i, got.Patched[i], w)
		}
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

// TestDiagFirst pins the one-shot gating the CBD (issue #376) field
// diagnostics rely on: a key is reported once and then suppressed, and
// distinct keys are independent.
func TestDiagFirst(t *testing.T) {
	cc := New(Options{Bus: events.NewBus(1), SystemName: "S"})
	defer cc.bus.Close()
	if !cc.diagFirst(diagUnhandledTSBK | 0x42) {
		t.Fatal("first sight of a key must report true")
	}
	if cc.diagFirst(diagUnhandledTSBK | 0x42) {
		t.Error("second sight of the same key must report false")
	}
	if !cc.diagFirst(diagUnhandledTSBK | 0x43) {
		t.Error("a distinct key must report true")
	}
	// Namespace prefixes must keep an alias-source key from colliding
	// with an unhandled-opcode key of the same low bits.
	if !cc.diagFirst(diagAliasSrc | 0x42) {
		t.Error("alias-source key collided with unhandled-opcode key")
	}
}

// TestControlChannelCensusesUnhandledTSBK confirms an undecoded vendor
// opcode produces exactly one Info census line per distinct (MFID,opcode)
// — the signal a CBD-class field test needs to name an alias transport we
// don't yet decode (issue #376) — plus a capped run of raw-payload sample
// lines carrying the bytes, so the transport can be reverse-engineered.
func TestControlChannelCensusesUnhandledTSBK(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	bus := events.NewBus(16)
	defer bus.Close()
	cc := New(Options{Bus: bus, Log: log, SystemName: "S"})

	// An unknown Motorola opcode (not patch/delete/alias) seen more times
	// than the payload-sample cap, with a recognisable payload.
	unknown := TSBK{Opcode: 0x3F, MFID: MFIDMotorola,
		Payload: [8]byte{0xDE, 0xAD, 0xBE, 0xEF, 0x01, 0x02, 0x03, 0x04}}
	const fires = maxUnhandledSamples + 3
	for i := 0; i < fires; i++ {
		lead := 0
		if i == 0 {
			lead = 10 // establish lock on the first frame
		}
		cc.Process(buildLockedStreamWithTSBK(lead, 0x293, DUIDTrunkingSignaling, unknown), i<<20)
	}

	out := buf.String()
	// The summary message is a prefix of the payload message, so match the
	// closing quote slog adds around the (space-containing) msg value.
	if n := strings.Count(out, `msg="p25: unhandled tsbk"`); n != 1 {
		t.Errorf("unhandled-tsbk census lines = %d, want 1 (one per distinct opcode)", n)
	}
	if n := strings.Count(out, `msg="p25: unhandled tsbk payload"`); n != maxUnhandledSamples {
		t.Errorf("unhandled-tsbk payload lines = %d, want %d (capped)", n, maxUnhandledSamples)
	}
	if !strings.Contains(out, "deadbeef01020304") {
		t.Error("payload sample line missing the raw payload hex")
	}
	if !strings.Contains(out, "opcode=0x3F") {
		t.Error("diagnostic missing numeric opcode (Opcode.String mislabels vendor opcodes)")
	}
}

// TestAsMotorolaPatchGroupChannelGrant pins the decode against the real
// MMR/CBD field capture (issue #376): Motorola opcode 0x02 payload
// 401001006904e2cc is a patch-group voice channel grant — encrypted
// (service-options 0x40), channel (ID 1, number 1), super-group 105,
// source RID 320204 (0x04e2cc). A standard-MFID TSBK with the same
// opcode must not match (it would be a different message).
func TestAsMotorolaPatchGroupChannelGrant(t *testing.T) {
	payload := [8]byte{0x40, 0x10, 0x01, 0x00, 0x69, 0x04, 0xE2, 0xCC}
	tsbk := TSBK{Opcode: OpMotorolaPatchGroupChannelGrant, MFID: MFIDMotorola, Payload: payload}
	got, ok := tsbk.AsMotorolaPatchGroupChannelGrant()
	if !ok {
		t.Fatal("AsMotorolaPatchGroupChannelGrant returned !ok")
	}
	if got.ServiceOptions != 0x40 || got.ChannelID != 1 || got.ChannelNumber != 1 ||
		got.GroupAddress != 105 || got.SourceID != 0x04E2CC {
		t.Errorf("grant = %+v, want svc=0x40 chan=1.1 group=105 source=0x04e2cc", got)
	}
	tsbk.MFID = MFIDStandard
	if _, ok := tsbk.AsMotorolaPatchGroupChannelGrant(); ok {
		t.Error("AsMotorolaPatchGroupChannelGrant matched a standard-MFID TSBK")
	}
}

// TestAsMotorolaPatchGroupChannelGrantUpdate pins the decode against the
// real MMR/CBD capture (issue #376): Motorola opcode 0x03 payload
// 5090019131942710 carries two (channel, super-group) activity pairs —
// (ID 5, number 0x090, group 401) and (ID 3, number 0x194, group 10000).
func TestAsMotorolaPatchGroupChannelGrantUpdate(t *testing.T) {
	payload := [8]byte{0x50, 0x90, 0x01, 0x91, 0x31, 0x94, 0x27, 0x10}
	tsbk := TSBK{Opcode: OpMotorolaPatchGroupChannelGrantUpdate, MFID: MFIDMotorola, Payload: payload}
	got, ok := tsbk.AsMotorolaPatchGroupChannelGrantUpdate()
	if !ok {
		t.Fatal("AsMotorolaPatchGroupChannelGrantUpdate returned !ok")
	}
	if got.ChannelAID != 5 || got.ChannelANumber != 0x090 || got.GroupAddressA != 401 ||
		got.ChannelBID != 3 || got.ChannelBNumber != 0x194 || got.GroupAddressB != 10000 {
		t.Errorf("update = %+v, want A=(5.0x090,401) B=(3.0x194,10000)", got)
	}
	tsbk.MFID = MFIDStandard
	if _, ok := tsbk.AsMotorolaPatchGroupChannelGrantUpdate(); ok {
		t.Error("AsMotorolaPatchGroupChannelGrantUpdate matched a standard-MFID TSBK")
	}
}

// TestControlChannelDispatchesMotorolaPatchGroupGrant confirms a Motorola
// patch-group channel grant (opcode 0x02) resolves through the band plan
// and publishes a KindGrant carrying its source RID + encryption — the
// MMR/CBD calls that previously arrived src=0/enc=false (issue #376) — and
// that it is no longer logged as an unhandled TSBK.
func TestControlChannelDispatchesMotorolaPatchGroupGrant(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	identTSBK := TSBK{Opcode: OpIdentifierUpdate,
		Payload: AssembleIdentifierUpdate(IdentifierUpdate{
			ChannelID: 1, SpacingHz: 12_500, BaseHz: 851_000_000,
		})}
	// Real CBD opcode-0x02 payload: enc grant, channel (1, 1), group 105,
	// source 0x04e2cc.
	grantTSBK := TSBK{LB: true, Opcode: OpMotorolaPatchGroupChannelGrant, MFID: MFIDMotorola,
		Payload: [8]byte{0x40, 0x10, 0x01, 0x00, 0x69, 0x04, 0xE2, 0xCC}}

	stream1 := buildLockedStreamWithTSBK(10, 0x293, DUIDTrunkingSignaling, identTSBK)
	stream2 := buildLockedStreamWithTSBK(0, 0x293, DUIDTrunkingSignaling, grantTSBK)

	cc := New(Options{Bus: bus, Log: log, SystemName: "S", FrequencyHz: 851_000_000})
	cc.Process(stream1, 0)
	cc.Process(stream2, len(stream1))

	var got *trunking.Grant
	deadline := time.After(time.Second)
	for got == nil {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				g := ev.Payload.(trunking.Grant)
				got = &g
			}
		case <-deadline:
			t.Fatal("no grant event published for Motorola patch-group channel grant")
		}
	}
	if got.GroupID != 105 || got.SourceID != 0x04E2CC {
		t.Errorf("group/source = %d/%#x, want 105/0x04e2cc", got.GroupID, got.SourceID)
	}
	if got.FrequencyHz != 851_012_500 {
		t.Errorf("freq = %d, want 851_012_500", got.FrequencyHz)
	}
	if !got.Encrypted || got.Emergency {
		t.Errorf("flags = enc=%v emer=%v, want enc only", got.Encrypted, got.Emergency)
	}
	if strings.Contains(buf.String(), "p25: unhandled tsbk") {
		t.Error("Motorola patch-group channel grant was logged as unhandled")
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
