package tier3

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// NOTE: there are no real off-air MBC capture vectors, so these tests drive
// the assembler with synthetic blocks (header octets 2-9 carry the
// validated CSBK grant layout). They pin the multi-block state machine —
// header opens, continuation appends, LB=1 closes — not an unverified wire
// layout. See mbc.go for the honesty caveat on payload parsing / CRC.

// mbcGrantHeader builds an MBC header block (not last) carrying a channel
// grant in the CSBK payload octets.
func mbcGrantHeader(op CSBKOpcode, payload [8]byte) []byte {
	return AssembleCSBK(CSBK{LB: false, Opcode: op, Payload: payload})
}

// mbcBlock builds a raw 12-byte continuation/terminator block; last sets
// the LB bit that closes the message.
func mbcBlock(last bool) []byte {
	b := make([]byte, 12)
	if last {
		b[0] = 0x80
	}
	return b
}

func TestControlChannelAssemblesMBCGrant(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		SystemName:  "MBCSys",
		FrequencyHz: 851_000_000,
		Resolver:    TableBandPlan{8: 866_175_000},
		Now:         func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
	})

	// Header: TV_GRANT, LPCN 8/TS2, group 0x123456, src 0xABCDEF.
	header := mbcGrantHeader(OpTVGrant, [8]byte{0x00, 0x88, 0x12, 0x34, 0x56, 0xAB, 0xCD, 0xEF})
	cc.IngestBurst(burstWithInfoBlock(header), dmr.SlotType{ColorCode: 3, DataType: dmr.DTMBCHeader})
	// Continuation flagged LB=1 closes the message.
	cc.IngestBurst(burstWithInfoBlock(mbcBlock(true)), dmr.SlotType{ColorCode: 3, DataType: dmr.DTMBCContinuation})

	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindGrant {
				continue
			}
			g := ev.Payload.(trunking.Grant)
			if g.GroupID != 0x123456 || g.FrequencyHz != 866_175_000 || g.Timeslot != 2 {
				t.Errorf("grant = %+v", g)
			}
			return
		case <-deadline:
			t.Fatal("MBC did not assemble into a grant")
		}
	}
}

func TestControlChannelMBCContinuationWithoutHeaderDropped(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "S", FrequencyHz: 1, Resolver: TableBandPlan{1: 1_000_000}})
	// A lone terminator with no opening header has nothing to attach to.
	cc.IngestBurst(burstWithInfoBlock(mbcBlock(true)), dmr.SlotType{ColorCode: 1, DataType: dmr.DTMBCContinuation})

	select {
	case ev := <-sub.C:
		t.Errorf("unexpected event from headerless MBC continuation: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestControlChannelMBCHeaderResetsStalePartial(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus: bus, SystemName: "S", FrequencyHz: 1,
		Resolver: TableBandPlan{8: 866_175_000},
		Now:      func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
	})
	// First header never terminates; a second header on the same color code
	// must discard it. The closing continuation then dispatches the SECOND
	// header's grant (group 0xBBBBBB), not the first (0xAAAAAA).
	h1 := mbcGrantHeader(OpTVGrant, [8]byte{0x00, 0x88, 0xAA, 0xAA, 0xAA, 0x00, 0x00, 0x00})
	cc.IngestBurst(burstWithInfoBlock(h1), dmr.SlotType{ColorCode: 3, DataType: dmr.DTMBCHeader})
	h2 := mbcGrantHeader(OpTVGrant, [8]byte{0x00, 0x88, 0xBB, 0xBB, 0xBB, 0x00, 0x00, 0x00})
	cc.IngestBurst(burstWithInfoBlock(h2), dmr.SlotType{ColorCode: 3, DataType: dmr.DTMBCHeader})
	cc.IngestBurst(burstWithInfoBlock(mbcBlock(true)), dmr.SlotType{ColorCode: 3, DataType: dmr.DTMBCContinuation})

	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindGrant {
				continue
			}
			if g := ev.Payload.(trunking.Grant); g.GroupID != 0xBBBBBB {
				t.Errorf("group = %X, want BBBBBB (second header should win)", g.GroupID)
			}
			return
		case <-deadline:
			t.Fatal("no grant after header reset")
		}
	}
}

func TestControlChannelMBCBlockLimitGuard(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus: bus, SystemName: "S", FrequencyHz: 1,
		Resolver: TableBandPlan{8: 866_175_000},
		Now:      func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
	})
	header := mbcGrantHeader(OpTVGrant, [8]byte{0x00, 0x88, 0x12, 0x34, 0x56, 0x00, 0x00, 0x00})
	cc.IngestBurst(burstWithInfoBlock(header), dmr.SlotType{ColorCode: 3, DataType: dmr.DTMBCHeader})
	// Flood non-terminating continuations past the block cap — must not
	// panic and must not dispatch (no LB seen).
	for i := 0; i < mbcMaxBlocks+4; i++ {
		cc.IngestBurst(burstWithInfoBlock(mbcBlock(false)), dmr.SlotType{ColorCode: 3, DataType: dmr.DTMBCContinuation})
	}

	select {
	case ev := <-sub.C:
		t.Errorf("unexpected event from unterminated over-long MBC: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}
