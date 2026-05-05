package dstar

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func TestHeaderRoundTrip(t *testing.T) {
	in := Header{
		Flag1: Flag1Data | Flag1EMR,
		Flag2: 0x12,
		Flag3: 0x34,
		RPT2:  "KD0AAA B",
		RPT1:  "KD0AAA A",
		UR:    "CQCQCQ  ",
		MY1:   "N0CALL  ",
		MY2:   "TEST",
		CRC:   0xCAFE,
	}
	bytes := AssembleHeader(in)
	if len(bytes) != 41 {
		t.Fatalf("AssembleHeader len = %d", len(bytes))
	}
	out, err := ParseHeader(bytes)
	if err != nil {
		t.Fatalf("ParseHeader: %v", err)
	}
	if !reflect.DeepEqual(out, in) {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", out, in)
	}
}

func TestHeaderShortStringPadsWithSpaces(t *testing.T) {
	in := Header{RPT1: "WB", MY1: "N", MY2: "X"}
	bytes := AssembleHeader(in)
	if string(bytes[11:19]) != "WB      " {
		t.Errorf("RPT1 padding: %q", string(bytes[11:19]))
	}
	if string(bytes[27:35]) != "N       " {
		t.Errorf("MY1 padding: %q", string(bytes[27:35]))
	}
	if string(bytes[35:39]) != "X   " {
		t.Errorf("MY2 padding: %q", string(bytes[35:39]))
	}
}

func TestParseHeaderWrongLength(t *testing.T) {
	if _, err := ParseHeader(make([]byte, 40)); err == nil {
		t.Error("expected error on len < 41")
	}
}

func TestHeaderFlagAccessors(t *testing.T) {
	cases := []struct {
		flag1 uint8
		emr   bool
		brk   bool
		data  bool
	}{
		{0x00, false, false, false},
		{Flag1EMR, true, false, false},
		{Flag1BreakIn, false, true, false},
		{Flag1Data, false, false, true},
		{Flag1EMR | Flag1Data, true, false, true},
	}
	for _, c := range cases {
		h := Header{Flag1: c.flag1}
		if h.IsEmergency() != c.emr ||
			h.IsBreakIn() != c.brk ||
			h.IsData() != c.data {
			t.Errorf("flag1=%#02x: emr=%v brk=%v data=%v",
				c.flag1, h.IsEmergency(), h.IsBreakIn(), h.IsData())
		}
	}
}

func TestHeaderIsGroupCall(t *testing.T) {
	cases := map[string]bool{
		"CQCQCQ  ": true,
		"CQCQCQ":   true, // already trimmed
		"/KD0AAA ": true, // /-prefix repeater routing
		"N0CALL  ": false,
		"        ": false,
	}
	for ur, want := range cases {
		got := (Header{UR: ur}).IsGroupCall()
		if got != want {
			t.Errorf("UR %q: IsGroupCall = %v, want %v", ur, got, want)
		}
	}
}

func TestComputeCRCKnownVector(t *testing.T) {
	// CRC-16-CCITT(0x1021, init 0xFFFF) over "123456789" is 0x29B1 by
	// the standard reference vector — sanity-check the implementation
	// against that.
	got := ComputeCRC([]byte("123456789"))
	if got != 0x29B1 {
		t.Errorf("CRC of '123456789' = %#04x, want 0x29B1", got)
	}
}

func TestComputeCRCRoundTripsHeader(t *testing.T) {
	h := Header{
		Flag1: Flag1EMR,
		RPT2:  "WB7XYZ B",
		RPT1:  "WB7XYZ A",
		UR:    "CQCQCQ  ",
		MY1:   "N0CALL  ",
		MY2:   "MOBI",
	}
	bytes := AssembleHeader(h)
	h.CRC = ComputeCRC(bytes[:39])
	bytes = AssembleHeader(h)
	out, err := ParseHeader(bytes)
	if err != nil {
		t.Fatal(err)
	}
	if want := ComputeCRC(bytes[:39]); out.CRC != want {
		t.Errorf("round-trip CRC = %#04x, want %#04x", out.CRC, want)
	}
}

func TestSyncBitsEncoding(t *testing.T) {
	bits := FrameSyncBitsSlice()
	if len(bits) != FrameSyncBits {
		t.Errorf("FrameSync len = %d, want %d", len(bits), FrameSyncBits)
	}
	for _, b := range bits {
		if b > 1 {
			t.Errorf("FrameSync contains non-bit %d", b)
		}
	}
	dataBits := SlowDataSyncBitsSlice()
	if len(dataBits) != SlowDataSyncBits {
		t.Errorf("SlowDataSync len = %d, want %d", len(dataBits), SlowDataSyncBits)
	}
	if reflect.DeepEqual(bits[:24], dataBits) {
		t.Error("FrameSync prefix and SlowDataSync are equal")
	}
}

func TestSyncDetectorExactMatch(t *testing.T) {
	pat := SlowDataSyncBitsSlice()
	det := NewSyncDetector(pat, 0)

	stream := make([]uint8, 50+len(pat)+5)
	copy(stream[50:], pat)
	hits, _ := det.Process(nil, stream, 0)
	if len(hits) != 1 || hits[0] != 50+len(pat)-1 {
		t.Errorf("hits = %v, want [%d]", hits, 50+len(pat)-1)
	}
}

func TestSyncDetectorTolerance(t *testing.T) {
	pat := SlowDataSyncBitsSlice()
	det := NewSyncDetector(pat, 1)
	const offset = 50
	stream := make([]uint8, offset+len(pat)+5)
	copy(stream[offset:], pat)
	stream[offset+5] ^= 1 // single bit error

	hits, _ := det.Process(nil, stream, 0)
	if len(hits) != 1 {
		t.Fatalf("hits = %v, want 1 (tolerance=1)", hits)
	}
}

func TestSyncDetectorRejectsZeroStream(t *testing.T) {
	pat := SlowDataSyncBitsSlice()
	det := NewSyncDetector(pat, 0)
	stream := make([]uint8, 4*len(pat))
	hits, _ := det.Process(nil, stream, 0)
	if len(hits) != 0 {
		t.Errorf("hits on zero stream = %v", hits)
	}
}

func TestControlChannelEmitsLockOnFirstHeader(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, FrequencyHz: 145_670_000})
	cc.Ingest(Header{
		RPT2: "KD0AAA B",
		RPT1: "KD0AAA A",
		UR:   "N0CALL  ",
		MY1:  "WB7XYZ  ",
	})

	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLocked {
			t.Fatalf("kind = %s, want cc.locked", ev.Kind)
		}
		ls, ok := ev.Payload.(LockState)
		if !ok {
			t.Fatalf("payload type = %T", ev.Payload)
		}
		if ls.FrequencyHz != 145_670_000 || ls.Repeater != "KD0AAA B" {
			t.Errorf("LockState = %+v", ls)
		}
	case <-time.After(time.Second):
		t.Fatal("no cc.locked")
	}
}

func TestControlChannelEmitsGrantOnGroupCall(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	fixed := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	cc := New(Options{
		Bus:         bus,
		SystemName:  "dstar-net",
		FrequencyHz: 145_670_000,
		Now:         func() time.Time { return fixed },
	})

	cc.Ingest(Header{
		Flag1: Flag1EMR,
		RPT2:  "KD0AAA B",
		UR:    "CQCQCQ  ",
		MY1:   "WB7XYZ  ",
	})

	<-sub.C // cc.locked
	ev := <-sub.C
	if ev.Kind != events.KindGrant {
		t.Fatalf("kind = %s, want grant", ev.Kind)
	}
	g := ev.Payload.(trunking.Grant)
	if g.Protocol != "dstar" {
		t.Errorf("Protocol = %q, want dstar", g.Protocol)
	}
	if g.System != "dstar-net" {
		t.Errorf("System = %q", g.System)
	}
	if g.FrequencyHz != 145_670_000 {
		t.Errorf("FrequencyHz = %d", g.FrequencyHz)
	}
	if !g.Emergency {
		t.Error("Emergency flag not propagated")
	}
	if !g.At.Equal(fixed) {
		t.Errorf("At = %v, want %v", g.At, fixed)
	}

	// Two distinct UR callsigns must hash to distinct GroupIDs.
	cqHash := hashCallsign("CQCQCQ")
	if g.GroupID != cqHash {
		t.Errorf("GroupID = %X, want %X", g.GroupID, cqHash)
	}
	if g.SourceID == 0 {
		t.Error("SourceID was zero for a non-empty MY1")
	}
}

func TestControlChannelSilentOnIndividualCall(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, FrequencyHz: 145_670_000})
	cc.Ingest(Header{
		RPT2: "KD0AAA B",
		UR:   "N0CALL  ", // explicit individual destination
		MY1:  "WB7XYZ  ",
	})

	// First event: cc.locked.
	if (<-sub.C).Kind != events.KindCCLocked {
		t.Fatal("missing cc.locked")
	}
	// No grant for a non-group-call.
	select {
	case ev := <-sub.C:
		t.Errorf("unexpected event for individual call: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestControlChannelSilentOnDataHeader(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, FrequencyHz: 145_670_000})
	cc.Ingest(Header{
		Flag1: Flag1Data,
		RPT2:  "KD0AAA B",
		UR:    "CQCQCQ  ",
	})
	select {
	case ev := <-sub.C:
		t.Errorf("unexpected event for data header: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestControlChannelMarkLost(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, FrequencyHz: 145_670_000})
	cc.Ingest(Header{RPT2: "KD0AAA B", UR: "N0CALL  ", MY1: "WB7XYZ  "})
	<-sub.C // cc.locked

	cc.MarkLost()
	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLost {
			t.Fatalf("kind = %s, want cc.lost", ev.Kind)
		}
		ls := ev.Payload.(LockState)
		if !strings.HasPrefix(ls.Repeater, "KD0AAA") {
			t.Errorf("LockState.Repeater = %q", ls.Repeater)
		}
	case <-time.After(time.Second):
		t.Fatal("no cc.lost")
	}

	// Second MarkLost is a no-op.
	cc.MarkLost()
	select {
	case ev := <-sub.C:
		t.Errorf("unexpected event after second MarkLost: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestControlChannelNoRepublishOnSameRepeater(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, FrequencyHz: 145_670_000})
	h := Header{RPT2: "KD0AAA B", UR: "N0CALL  ", MY1: "WB7XYZ  "}
	cc.Ingest(h)
	<-sub.C // first cc.locked

	cc.Ingest(h) // same repeater → no re-publish
	select {
	case ev := <-sub.C:
		t.Errorf("unexpected re-publish: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestHashCallsignDistinct(t *testing.T) {
	a := hashCallsign("WB7XYZ")
	b := hashCallsign("KD0AAA")
	if a == b {
		t.Errorf("distinct callsigns hashed to same value: %X", a)
	}
	if hashCallsign("") != 0 {
		t.Error("empty callsign should hash to 0")
	}
}
