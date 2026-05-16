package mpt1327

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// makeCodeword builds an address codeword of the supplied kind with
// the given prefix / ident / lower-13-bit payload.
func makeCodeword(kind CodewordKind, prefix uint8, ident uint16, payload uint16) Codeword {
	function := (uint32(kind) & 0xF) << 13
	function |= uint32(payload) & 0x1FFF
	return Codeword{
		Type:     TypeAddress,
		Prefix:   prefix,
		Ident:    ident,
		Function: function,
	}
}

func TestCodewordAssembleParseRoundTrip(t *testing.T) {
	in := Codeword{
		Type:     TypeAddress,
		Prefix:   0x42,
		Ident:    0x1A1B,
		Function: (uint32(KindGoToChan) << 13) | 0x1234,
	}
	bytes := AssembleCodeword(in)
	if len(bytes) != 5 {
		t.Fatalf("AssembleCodeword = %d bytes", len(bytes))
	}
	got, err := ParseCodeword(bytes)
	if err != nil {
		t.Fatal(err)
	}
	if got != in {
		t.Errorf("round-trip = %+v, want %+v", got, in)
	}
}

func TestCodewordFromBitsRoundTrip(t *testing.T) {
	in := makeCodeword(KindAhoyChan, 0x55, 0x0ABC, 0x0DEF)
	bits := CodewordBits(in)
	if len(bits) != 38 {
		t.Fatalf("CodewordBits len = %d", len(bits))
	}
	got, err := CodewordFromBits(bits)
	if err != nil {
		t.Fatal(err)
	}
	if got != in {
		t.Errorf("round-trip = %+v", got)
	}
}

func TestCodewordFromBitsRejectsBadLength(t *testing.T) {
	if _, err := CodewordFromBits(make([]byte, 37)); err == nil {
		t.Error("expected error for 37 bits")
	}
}

func TestCodewordKindAndPayload(t *testing.T) {
	c := makeCodeword(KindGoToChan, 0x01, 0x100, 0x123)
	if c.Kind() != KindGoToChan {
		t.Errorf("Kind = %s, want GoToChannel", c.Kind())
	}
	if c.FunctionPayload() != 0x123 {
		t.Errorf("payload = %X", c.FunctionPayload())
	}
}

func TestIsAloha(t *testing.T) {
	if !makeCodeword(KindAloha, 0, 0, 0).IsAloha() {
		t.Error("ALH codeword not flagged as aloha")
	}
	if makeCodeword(KindGoToChan, 0, 0, 0).IsAloha() {
		t.Error("GTC codeword flagged as aloha")
	}
}

func TestAsGoToChannel(t *testing.T) {
	c := makeCodeword(KindGoToChan, 0x42, 0x1234, 0x0789)
	g, ok := c.AsGoToChannel()
	if !ok {
		t.Fatal("expected grant")
	}
	if g.Prefix != 0x42 || g.Ident != 0x1234 || g.Channel != 0x0789 {
		t.Errorf("grant = %+v", g)
	}

	// Aloha shouldn't masquerade as a grant.
	if _, ok := makeCodeword(KindAloha, 0, 0, 0).AsGoToChannel(); ok {
		t.Error("ALH reported a grant")
	}
	// Data codewords shouldn't either.
	if _, ok := (Codeword{Type: TypeData, Function: uint32(KindGoToChan) << 13}).AsGoToChannel(); ok {
		t.Error("data codeword reported a grant")
	}
}

func TestAsAhoyChannel(t *testing.T) {
	c := makeCodeword(KindAhoyChan, 0x07, 0x0050, 0x00CD)
	a, ok := c.AsAhoyChannel()
	if !ok {
		t.Fatal("expected AHYC")
	}
	if a.Prefix != 0x07 || a.Ident != 0x0050 || a.System != 0x00CD {
		t.Errorf("ahyc = %+v", a)
	}
}

func TestKindString(t *testing.T) {
	cases := map[CodewordKind]string{
		KindAloha:         "Aloha",
		KindGoToChan:      "GoToChannel",
		KindAhoyChan:      "AhoyChan",
		KindAck:           "Ack",
		CodewordKind(0xD): "CodewordKind(D)",
	}
	for k, want := range cases {
		if got := k.String(); got != want {
			t.Errorf("Kind(%X).String() = %s, want %s", uint8(k), got, want)
		}
	}
}

func TestCodewordTypeString(t *testing.T) {
	if TypeAddress.String() != "Address" || TypeData.String() != "Data" {
		t.Errorf("unexpected type strings: %s / %s", TypeAddress.String(), TypeData.String())
	}
}

func TestLinearBandPlan(t *testing.T) {
	bp := LinearBandPlan{BaseHz: 165_000_000, SpacingHz: 12_500, Offset: -1}
	if hz, _ := bp.Frequency(1); hz != 165_000_000 {
		t.Errorf("ch 1 → %d, want 165M", hz)
	}
	if hz, _ := bp.Frequency(100); hz != 165_000_000+99*12_500 {
		t.Errorf("ch 100 → %d", hz)
	}
	if _, err := (LinearBandPlan{BaseHz: 1, SpacingHz: 0}).Frequency(1); err == nil {
		t.Error("zero spacing should error")
	}
	if _, err := (LinearBandPlan{BaseHz: 1, SpacingHz: 12_500, Offset: -10}).Frequency(1); err == nil {
		t.Error("negative effective offset should error")
	}
}

func TestTableBandPlan(t *testing.T) {
	bp := TableBandPlan{1: 165_000_000, 5: 165_062_500}
	if hz, _ := bp.Frequency(1); hz != 165_000_000 {
		t.Errorf("ch 1 → %d", hz)
	}
	if _, err := bp.Frequency(99); err == nil {
		t.Error("missing channel should error")
	}
}

func TestControlChannelEmitsLockOnAloha(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		SystemName:  "TestMPT",
		FrequencyHz: 165_000_000,
	})
	cc.Ingest(makeCodeword(KindAloha, 0x42, 0, 0))

	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLocked {
			t.Fatalf("kind = %s", ev.Kind)
		}
		ls, ok := ev.Payload.(LockState)
		if !ok || ls.FrequencyHz != 165_000_000 || ls.Prefix != 0x42 {
			t.Errorf("lock state = %+v", ev.Payload)
		}
	case <-time.After(time.Second):
		t.Fatal("no event")
	}
}

func TestControlChannelEmitsLockOnAhoyChan(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "X", FrequencyHz: 1})
	cc.Ingest(makeCodeword(KindAhoyChan, 0x05, 0x100, 0x0ABC))

	ev := <-sub.C
	if ev.Kind != events.KindCCLocked {
		t.Fatalf("kind = %s", ev.Kind)
	}
	if ls, ok := ev.Payload.(LockState); !ok || ls.SystemID != 0x0ABC {
		t.Errorf("lock = %+v", ev.Payload)
	}
}

func TestControlChannelPublishesGrant(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		SystemName:  "TestMPT",
		FrequencyHz: 165_000_000,
		Resolver: LinearBandPlan{
			BaseHz: 165_000_000, SpacingHz: 12_500, Offset: -1,
		},
	})
	// Lock first.
	cc.Ingest(makeCodeword(KindAloha, 0x01, 0, 0))
	if ev := <-sub.C; ev.Kind != events.KindCCLocked {
		t.Fatalf("first event = %s, want cc.locked", ev.Kind)
	}
	// Now publish a GTC grant — channel 5.
	cc.Ingest(makeCodeword(KindGoToChan, 0x42, 0x1234, 5))

	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindGrant {
			t.Fatalf("kind = %s", ev.Kind)
		}
		g, ok := ev.Payload.(trunking.Grant)
		if !ok {
			t.Fatalf("payload = %T", ev.Payload)
		}
		if g.System != "TestMPT" || g.Protocol != "mpt1327" {
			t.Errorf("identity = %+v", g)
		}
		// GroupID = (prefix << 16) | ident = (0x42 << 16) | 0x1234.
		if g.GroupID != (0x42<<16)|0x1234 {
			t.Errorf("group = %X, want %X", g.GroupID, (0x42<<16)|0x1234)
		}
		if g.ChannelNum != 5 {
			t.Errorf("channel = %d", g.ChannelNum)
		}
		// LinearBandPlan: ch 5 with offset -1 → idx 4 → 165M + 50_000.
		if g.FrequencyHz != 165_050_000 {
			t.Errorf("freq = %d, want 165_050_000", g.FrequencyHz)
		}
	case <-time.After(time.Second):
		t.Fatal("no grant")
	}
}

func TestControlChannelGrantWithoutResolverHasZeroFreq(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "X", FrequencyHz: 1})
	cc.Ingest(makeCodeword(KindGoToChan, 0x10, 0x0050, 7))
	ev := <-sub.C
	g := ev.Payload.(trunking.Grant)
	if g.FrequencyHz != 0 {
		t.Errorf("freq = %d, want 0", g.FrequencyHz)
	}
	if g.ChannelNum != 7 {
		t.Errorf("channel = %d", g.ChannelNum)
	}
}

func TestControlChannelIgnoresDataCodewords(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "X", FrequencyHz: 1})
	cc.Ingest(Codeword{Type: TypeData, Function: uint32(KindGoToChan) << 13})

	select {
	case ev := <-sub.C:
		t.Errorf("data codeword produced an event: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestControlChannelMarkLost(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "X", FrequencyHz: 1})
	cc.Ingest(makeCodeword(KindAloha, 0x42, 0, 0))
	<-sub.C // cc.locked

	cc.MarkLost()
	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLost {
			t.Errorf("kind = %s, want cc.lost", ev.Kind)
		}
	case <-time.After(time.Second):
		t.Fatal("no cc.lost")
	}
}
