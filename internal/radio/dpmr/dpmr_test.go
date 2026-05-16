package dpmr

import (
	"reflect"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func TestCSBKByteRoundTrip(t *testing.T) {
	in := CSBK{
		Type:        MsgVoiceServiceAllocation,
		Flags:       FlagGroupCall | FlagEmergency,
		SourceID:    0x123456,
		DestID:      0xABCDEF,
		ServiceInfo: 0x42,
		Extra:       0x0123,
	}
	bytes := AssembleCSBK(in)
	if len(bytes) != 10 {
		t.Fatalf("AssembleCSBK len = %d, want 10", len(bytes))
	}
	out, err := ParseCSBK(bytes)
	if err != nil {
		t.Fatalf("ParseCSBK: %v", err)
	}
	if out != in {
		t.Errorf("round-trip = %+v, want %+v", out, in)
	}
}

func TestCSBKBitRoundTrip(t *testing.T) {
	in := CSBK{
		Type:     MsgIndividualVoiceAllocation,
		Flags:    FlagEncrypted,
		SourceID: 0x7FFFFF,
		DestID:   0x000001,
		Extra:    0xFFFF,
	}
	bits := CSBKBits(in)
	if len(bits) != 80 {
		t.Fatalf("CSBKBits len = %d, want 80", len(bits))
	}
	out, err := CSBKFromBits(bits)
	if err != nil {
		t.Fatalf("CSBKFromBits: %v", err)
	}
	if out != in {
		t.Errorf("bits round-trip = %+v, want %+v", out, in)
	}
}

func TestParseCSBKWrongLength(t *testing.T) {
	if _, err := ParseCSBK(make([]byte, 9)); err == nil {
		t.Error("expected error for len(info) < 10")
	}
}

func TestCSBKFlagAccessors(t *testing.T) {
	cases := []struct {
		flags       byte
		group, emer bool
		enc         bool
	}{
		{0x00, false, false, false},
		{FlagGroupCall, true, false, false},
		{FlagEmergency, false, true, false},
		{FlagEncrypted, false, false, true},
		{FlagGroupCall | FlagEmergency | FlagEncrypted, true, true, true},
	}
	for _, c := range cases {
		csbk := CSBK{Flags: c.flags}
		if csbk.IsGroup() != c.group ||
			csbk.IsEmergency() != c.emer ||
			csbk.IsEncrypted() != c.enc {
			t.Errorf("flags %#02x: group=%v emer=%v enc=%v",
				c.flags, csbk.IsGroup(), csbk.IsEmergency(), csbk.IsEncrypted())
		}
	}
}

func TestAsVoiceGrantExtractsFields(t *testing.T) {
	csbk := CSBK{
		Type:     MsgIndividualVoiceAllocation,
		Flags:    FlagEmergency | FlagEncrypted,
		SourceID: 0x100200,
		DestID:   0x300400,
		Extra:    1234,
	}
	g, ok := csbk.AsVoiceGrant()
	if !ok {
		t.Fatal("AsVoiceGrant returned !ok")
	}
	if g.SourceID != 0x100200 || g.DestID != 0x300400 || g.Channel != 1234 {
		t.Errorf("VoiceGrant fields = %+v", g)
	}
	if !g.Emergency || !g.Encrypted {
		t.Errorf("flags lost: %+v", g)
	}
	if g.Group {
		t.Error("individual call should not be marked Group")
	}
}

func TestAsVoiceGrantGroupForcesGroupFlag(t *testing.T) {
	csbk := CSBK{Type: MsgVoiceServiceAllocation, Extra: 7}
	g, ok := csbk.AsVoiceGrant()
	if !ok || !g.Group {
		t.Fatalf("group voice allocation: ok=%v group=%v", ok, g.Group)
	}
}

func TestAsVoiceGrantWrongType(t *testing.T) {
	csbk := CSBK{Type: MsgRegistrationRequest}
	if _, ok := csbk.AsVoiceGrant(); ok {
		t.Error("AsVoiceGrant returned ok for non-grant type")
	}
}

func TestAsSiteBroadcast(t *testing.T) {
	csbk := CSBK{
		Type:   MsgStandingServiceStatus,
		DestID: 0x000F00,
		Extra:  0xCAFE,
	}
	sb, ok := csbk.AsSiteBroadcast()
	if !ok {
		t.Fatal("AsSiteBroadcast returned !ok")
	}
	if sb.SystemID != 0x000F00 || sb.Status != 0xCAFE {
		t.Errorf("SiteBroadcast = %+v", sb)
	}
	other := CSBK{Type: MsgVoiceServiceAllocation}
	if _, ok := other.AsSiteBroadcast(); ok {
		t.Error("AsSiteBroadcast returned ok for non-broadcast type")
	}
}

func TestIsIdle(t *testing.T) {
	for _, tt := range []struct {
		t    MessageType
		want bool
	}{
		{MsgIdle, true},
		{MsgRelease, true},
		{MsgVoiceServiceAllocation, false},
		{MsgRegistrationRequest, false},
	} {
		if got := (CSBK{Type: tt.t}).IsIdle(); got != tt.want {
			t.Errorf("IsIdle(%s) = %v, want %v", tt.t, got, tt.want)
		}
	}
}

func TestSyncDibitsAndDetector(t *testing.T) {
	for name, dibits := range map[string][]uint8{
		"FS1": FS1Dibits(),
		"FS2": FS2Dibits(),
		"FS3": FS3Dibits(),
	} {
		if len(dibits) != SyncDibits {
			t.Errorf("%s len = %d, want %d", name, len(dibits), SyncDibits)
		}
		for _, d := range dibits {
			if d > 3 {
				t.Errorf("%s contains dibit %d", name, d)
			}
		}
	}
	if reflect.DeepEqual(FS1Dibits(), FS2Dibits()) {
		t.Error("FS1 and FS2 patterns are equal")
	}
	if reflect.DeepEqual(FS1Dibits(), FS3Dibits()) {
		t.Error("FS1 and FS3 patterns are equal")
	}
}

func TestSyncDetectorExactMatch(t *testing.T) {
	pat := FS3Dibits()
	det := NewSyncDetector(pat, 0)

	stream := make([]uint8, 30+len(pat)+5)
	copy(stream[30:], pat)
	hits, _ := det.Process(nil, stream, 0)
	if len(hits) != 1 {
		t.Fatalf("hits = %v, want exactly 1", hits)
	}
	if hits[0] != 30+len(pat)-1 {
		t.Errorf("hits[0] = %d, want %d", hits[0], 30+len(pat)-1)
	}
}

func TestSyncDetectorTolerance(t *testing.T) {
	pat := FS3Dibits()
	det := NewSyncDetector(pat, 1)

	const offset = 30
	stream := make([]uint8, offset+len(pat)+5)
	copy(stream[offset:], pat)
	stream[offset+5] = (stream[offset+5] + 1) & 0x3
	hits, _ := det.Process(nil, stream, 0)
	if len(hits) != 1 {
		t.Fatalf("hits = %v, want 1 (tolerance=1, one mismatch)", hits)
	}
}

func TestSyncDetectorRejectsNoise(t *testing.T) {
	pat := FS1Dibits()
	det := NewSyncDetector(pat, 0)
	stream := make([]uint8, 4*len(pat))
	hits, _ := det.Process(nil, stream, 0)
	if len(hits) != 0 {
		t.Errorf("hits on zero stream = %v", hits)
	}
}

func TestLinearBandPlan(t *testing.T) {
	bp := LinearBandPlan{BaseHz: 446_006_250, SpacingHz: 6_250, Offset: -1}
	hz, err := bp.Frequency(1)
	if err != nil {
		t.Fatal(err)
	}
	if hz != 446_006_250 {
		t.Errorf("ch1 = %d, want 446006250", hz)
	}
	hz, err = bp.Frequency(8)
	if err != nil {
		t.Fatal(err)
	}
	if hz != 446_006_250+7*6_250 {
		t.Errorf("ch8 = %d", hz)
	}
}

func TestLinearBandPlanRejectsZeroSpacing(t *testing.T) {
	bp := LinearBandPlan{BaseHz: 446_000_000}
	if _, err := bp.Frequency(1); err == nil {
		t.Error("expected error on zero SpacingHz")
	}
}

func TestTableBandPlan(t *testing.T) {
	bp := TableBandPlan{1: 446_006_250, 4: 446_025_000}
	if hz, err := bp.Frequency(1); err != nil || hz != 446_006_250 {
		t.Errorf("ch1 = %d/%v", hz, err)
	}
	if _, err := bp.Frequency(99); err == nil {
		t.Error("expected error on missing channel")
	}
}

// Test-helper: build a voice-allocation CSBK.
func voiceCSBK(typ MessageType, src, dst uint32, ch uint16) CSBK {
	return CSBK{Type: typ, SourceID: src, DestID: dst, Extra: ch}
}

func TestControlChannelEmitsLockOnSiteBroadcast(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, FrequencyHz: 446_018_750})
	cc.Ingest(CSBK{Type: MsgStandingServiceStatus, DestID: 0x123456, Extra: 0x42})

	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLocked {
			t.Fatalf("kind = %s, want cc.locked", ev.Kind)
		}
		ls, ok := ev.Payload.(LockState)
		if !ok || ls.FrequencyHz != 446_018_750 || ls.SystemID != 0x123456 {
			t.Errorf("payload = %+v", ev.Payload)
		}
	case <-time.After(time.Second):
		t.Fatal("no cc.locked event")
	}
}

func TestControlChannelEmitsGrant(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	bp := LinearBandPlan{BaseHz: 446_006_250, SpacingHz: 6_250, Offset: -1}
	fixed := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	cc := New(Options{
		Bus:         bus,
		SystemName:  "dpmr-sys",
		FrequencyHz: 446_018_750,
		Resolver:    bp,
		Now:         func() time.Time { return fixed },
	})

	cc.Ingest(voiceCSBK(MsgVoiceServiceAllocation, 0x100, 0x200, 4))

	// First event: cc.locked (synthesized when grant arrives without
	// a prior SiteBroadcast).
	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLocked {
			t.Fatalf("first event kind = %s, want cc.locked", ev.Kind)
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
		if g.Protocol != "dpmr" {
			t.Errorf("Protocol = %q, want dpmr", g.Protocol)
		}
		if g.System != "dpmr-sys" {
			t.Errorf("System = %q", g.System)
		}
		if g.GroupID != 0x200 || g.SourceID != 0x100 {
			t.Errorf("GroupID = %X, SourceID = %X", g.GroupID, g.SourceID)
		}
		if g.ChannelNum != 4 {
			t.Errorf("ChannelNum = %d", g.ChannelNum)
		}
		if g.FrequencyHz != 446_006_250+3*6_250 {
			t.Errorf("FrequencyHz = %d", g.FrequencyHz)
		}
		if !g.At.Equal(fixed) {
			t.Errorf("At = %v, want %v", g.At, fixed)
		}
	case <-time.After(time.Second):
		t.Fatal("no grant event")
	}
}

func TestControlChannelGrantNoResolverFallsBackToZero(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, FrequencyHz: 446_018_750})
	cc.Ingest(voiceCSBK(MsgVoiceServiceAllocation, 1, 2, 9))

	<-sub.C // cc.locked
	ev := <-sub.C
	g := ev.Payload.(trunking.Grant)
	if g.FrequencyHz != 0 {
		t.Errorf("FrequencyHz = %d, want 0 (no resolver)", g.FrequencyHz)
	}
	if g.ChannelNum != 9 {
		t.Errorf("ChannelNum = %d", g.ChannelNum)
	}
}

func TestControlChannelSilentOnIdle(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, FrequencyHz: 446_018_750})
	cc.Ingest(CSBK{Type: MsgIdle})
	cc.Ingest(CSBK{Type: MsgRelease})
	select {
	case ev := <-sub.C:
		t.Errorf("unexpected event on idle: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestControlChannelMarkLost(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, FrequencyHz: 446_018_750})
	cc.Ingest(CSBK{Type: MsgStandingServiceStatus, DestID: 0xAABB})
	<-sub.C // cc.locked

	cc.MarkLost()
	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLost {
			t.Fatalf("kind = %s, want cc.lost", ev.Kind)
		}
		ls, ok := ev.Payload.(LockState)
		if !ok || ls.SystemID != 0xAABB {
			t.Errorf("LockState = %+v", ev.Payload)
		}
	case <-time.After(time.Second):
		t.Fatal("no cc.lost event")
	}

	// Second MarkLost is a no-op.
	cc.MarkLost()
	select {
	case ev := <-sub.C:
		t.Errorf("unexpected event after second MarkLost: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestControlChannelNoRepublishOnSameLockState(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, FrequencyHz: 446_018_750})
	cc.Ingest(CSBK{Type: MsgStandingServiceStatus, DestID: 0xAABB})
	<-sub.C // first cc.locked
	cc.Ingest(CSBK{Type: MsgStandingServiceStatus, DestID: 0xAABB})
	select {
	case ev := <-sub.C:
		t.Errorf("unexpected re-publish: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}
