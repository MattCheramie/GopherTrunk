package phase2

import (
	"reflect"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func TestMACPDURoundTrip(t *testing.T) {
	in := MACPDU{
		Opcode:  OpGroupVoiceChannelGrant,
		Payload: []byte{0x00, 0x10, 0x05, 0x12, 0x34, 0x00, 0xAB, 0xCD},
	}
	bytes := AssembleMACPDU(in)
	if bytes[0] != byte(OpGroupVoiceChannelGrant) {
		t.Fatalf("opcode byte = %02X", bytes[0])
	}
	out, err := ParseMACPDU(bytes)
	if err != nil {
		t.Fatalf("ParseMACPDU: %v", err)
	}
	if out.Opcode != in.Opcode {
		t.Errorf("opcode = %s, want %s", out.Opcode, in.Opcode)
	}
	if !reflect.DeepEqual(out.Payload, in.Payload) {
		t.Errorf("payload = %v, want %v", out.Payload, in.Payload)
	}
}

func TestMACPDUParseEmpty(t *testing.T) {
	if _, err := ParseMACPDU(nil); err == nil {
		t.Fatal("expected error on empty info bytes")
	}
}

func TestMACPDUIsIdle(t *testing.T) {
	cases := map[Opcode]bool{
		OpMACIdle:                true,
		OpMACHangtime:            true,
		OpMACEnd:                 true,
		OpMACPTT:                 false,
		OpGroupVoiceChannelGrant: false,
	}
	for op, want := range cases {
		got := MACPDU{Opcode: op}.IsIdle()
		if got != want {
			t.Errorf("Opcode %s IsIdle = %v, want %v", op, got, want)
		}
	}
}

func TestAsGroupVoiceChannelGrantExtractsFields(t *testing.T) {
	// service options 0x84, channel ID 0x1, channel number 0x005,
	// group address 0x1234, source 0x00ABCD.
	pdu := MACPDU{
		Opcode:  OpGroupVoiceChannelGrant,
		Payload: []byte{0x84, 0x10, 0x05, 0x12, 0x34, 0x00, 0xAB, 0xCD},
	}
	g, ok := pdu.AsGroupVoiceChannelGrant()
	if !ok {
		t.Fatal("AsGroupVoiceChannelGrant returned !ok")
	}
	if g.ServiceOptions != 0x84 {
		t.Errorf("ServiceOptions = %02X, want 84", g.ServiceOptions)
	}
	if g.ChannelID != 0x1 {
		t.Errorf("ChannelID = %x, want 1", g.ChannelID)
	}
	if g.ChannelNumber != 0x005 {
		t.Errorf("ChannelNumber = %03x, want 005", g.ChannelNumber)
	}
	if g.GroupAddress != 0x1234 {
		t.Errorf("GroupAddress = %04X, want 1234", g.GroupAddress)
	}
	if g.SourceID != 0x00ABCD {
		t.Errorf("SourceID = %06X, want 00ABCD", g.SourceID)
	}
}

func TestAsGroupVoiceChannelGrantWrongOpcode(t *testing.T) {
	pdu := MACPDU{Opcode: OpMACPTT, Payload: make([]byte, 8)}
	if _, ok := pdu.AsGroupVoiceChannelGrant(); ok {
		t.Fatal("AsGroupVoiceChannelGrant returned ok for non-grant opcode")
	}
}

func TestAsGroupVoiceChannelGrantShortPayload(t *testing.T) {
	pdu := MACPDU{Opcode: OpGroupVoiceChannelGrant, Payload: []byte{0x00, 0x10}}
	if _, ok := pdu.AsGroupVoiceChannelGrant(); ok {
		t.Fatal("AsGroupVoiceChannelGrant returned ok for short payload")
	}
}

func TestSyncDibitsEncodeRoundTrip(t *testing.T) {
	out := OutboundSyncDibits()
	if len(out) != SyncDibits {
		t.Fatalf("len(OutboundSyncDibits) = %d, want %d", len(out), SyncDibits)
	}
	in := InboundSyncDibits()
	if len(in) != SyncDibits {
		t.Fatalf("len(InboundSyncDibits) = %d, want %d", len(in), SyncDibits)
	}
	// Patterns must differ.
	if reflect.DeepEqual(out, in) {
		t.Fatal("outbound and inbound sync patterns are equal")
	}
	for _, d := range out {
		if d > 3 {
			t.Fatalf("dibit %d out of range", d)
		}
	}
}

func TestSyncDetectorExactMatch(t *testing.T) {
	pat := OutboundSyncDibits()
	det := NewSyncDetector(pat, 0)

	// Place the sync at index 7, surrounded by zero dibits.
	stream := make([]uint8, 7+len(pat)+5)
	copy(stream[7:], pat)

	hits, _ := det.Process(nil, stream, 0)
	if len(hits) != 1 {
		t.Fatalf("hits = %v, want exactly one", hits)
	}
	// Detector reports the index of the LAST dibit of the pattern.
	want := 7 + len(pat) - 1
	if hits[0] != want {
		t.Errorf("hits[0] = %d, want %d", hits[0], want)
	}
}

func TestSyncDetectorTolerance(t *testing.T) {
	pat := OutboundSyncDibits()
	det := NewSyncDetector(pat, 1)

	// Place the pattern past the priming window so the comparison fires.
	const offset = 30
	stream := make([]uint8, offset+len(pat)+5)
	copy(stream[offset:], pat)
	stream[offset+3] = (stream[offset+3] + 1) & 0x3 // one dibit error

	hits, _ := det.Process(nil, stream, 0)
	if len(hits) != 1 {
		t.Fatalf("hits = %v, want exactly 1 (tolerance=1, one mismatch)", hits)
	}
	if hits[0] != offset+len(pat)-1 {
		t.Errorf("hits[0] = %d, want %d", hits[0], offset+len(pat)-1)
	}
}

func TestSyncDetectorRejectsBadStream(t *testing.T) {
	pat := OutboundSyncDibits()
	det := NewSyncDetector(pat, 0)

	stream := make([]uint8, 4*len(pat))
	for i := range stream {
		stream[i] = 0
	}
	hits, _ := det.Process(nil, stream, 0)
	if len(hits) != 0 {
		t.Errorf("hits = %v on zero stream", hits)
	}
}

func TestSlotTypeClassification(t *testing.T) {
	if !SlotTypeVoice4V.IsVoice() || !SlotTypeVoice2V.IsVoice() {
		t.Error("Voice4V/Voice2V should classify as voice")
	}
	if SlotTypeMACPTT.IsVoice() || SlotTypeMACIdle.IsVoice() {
		t.Error("MAC slots should not classify as voice")
	}
	if !SlotTypeMACPTT.IsMAC() || !SlotTypeMACEnd.IsMAC() ||
		!SlotTypeMACSignaling.IsMAC() {
		t.Error("MAC slots should classify as MAC")
	}
	if SlotTypeVoice4V.IsMAC() {
		t.Error("Voice slot should not classify as MAC")
	}
}

func grantPDU(tg uint16, src uint32, chanID uint8, chanNum uint16) MACPDU {
	chanField := (uint16(chanID&0xF) << 12) | (chanNum & 0x0FFF)
	return MACPDU{
		Opcode: OpGroupVoiceChannelGrant,
		Payload: []byte{
			0x00,
			byte(chanField >> 8), byte(chanField),
			byte(tg >> 8), byte(tg),
			byte(src >> 16), byte(src >> 8), byte(src),
		},
	}
}

func TestControlChannelEmitsLockAndGrant(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	fixed := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	cc := New(Options{
		Bus:         bus,
		SystemName:  "test-sys",
		FrequencyHz: 851_012_500,
		Now:         func() time.Time { return fixed },
	})

	cc.Ingest(grantPDU(0x1234, 0x00ABCD, 0x1, 0x005))

	// First event: cc.locked.
	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLocked {
			t.Fatalf("first event kind = %s, want cc.locked", ev.Kind)
		}
		ls, ok := ev.Payload.(LockState)
		if !ok {
			t.Fatalf("locked payload type = %T", ev.Payload)
		}
		if ls.FrequencyHz != 851_012_500 {
			t.Errorf("LockState.FrequencyHz = %d", ls.FrequencyHz)
		}
	case <-time.After(time.Second):
		t.Fatal("no cc.locked event")
	}

	// Second event: grant.
	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindGrant {
			t.Fatalf("second event kind = %s, want grant", ev.Kind)
		}
		g, ok := ev.Payload.(trunking.Grant)
		if !ok {
			t.Fatalf("grant payload type = %T", ev.Payload)
		}
		if g.Protocol != "p25-phase2" {
			t.Errorf("Protocol = %q, want p25-phase2", g.Protocol)
		}
		if g.System != "test-sys" {
			t.Errorf("System = %q", g.System)
		}
		if g.GroupID != 0x1234 {
			t.Errorf("GroupID = %X, want 1234", g.GroupID)
		}
		if g.SourceID != 0x00ABCD {
			t.Errorf("SourceID = %06X", g.SourceID)
		}
		if g.ChannelID != 0x1 || g.ChannelNum != 0x005 {
			t.Errorf("Channel = %d/%d", g.ChannelID, g.ChannelNum)
		}
		if !g.At.Equal(fixed) {
			t.Errorf("At = %v, want %v", g.At, fixed)
		}
	case <-time.After(time.Second):
		t.Fatal("no grant event")
	}
}

func TestControlChannelSilentOnIdle(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, FrequencyHz: 851_000_000})
	cc.Ingest(MACPDU{Opcode: OpMACIdle})
	cc.Ingest(MACPDU{Opcode: OpMACHangtime})

	select {
	case ev := <-sub.C:
		t.Errorf("unexpected event on idle PDUs: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestControlChannelMarkLost(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, FrequencyHz: 851_000_000})
	cc.Ingest(grantPDU(0x9999, 0x111111, 0, 0))

	// Drain cc.locked + grant.
	<-sub.C
	<-sub.C

	cc.MarkLost()
	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLost {
			t.Fatalf("kind = %s, want cc.lost", ev.Kind)
		}
		ls, ok := ev.Payload.(LockState)
		if !ok {
			t.Fatalf("lost payload type = %T", ev.Payload)
		}
		if ls.FrequencyHz != 851_000_000 {
			t.Errorf("LockState.FrequencyHz = %d", ls.FrequencyHz)
		}
	case <-time.After(time.Second):
		t.Fatal("no cc.lost event")
	}

	// Calling MarkLost again should be a no-op.
	cc.MarkLost()
	select {
	case ev := <-sub.C:
		t.Errorf("unexpected event after second MarkLost: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}
