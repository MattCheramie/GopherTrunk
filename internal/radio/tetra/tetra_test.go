package tetra

import (
	"reflect"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func TestPDUByteRoundTrip(t *testing.T) {
	in := PDU{
		Disc:    DiscCMCE,
		Type:    uint8(CMCEDConnect),
		Payload: []byte{0xAB, 0xCD, 0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xE0, 0x12, 0x34},
	}
	bytes := AssemblePDU(in)
	out, err := ParsePDU(bytes)
	if err != nil {
		t.Fatalf("ParsePDU: %v", err)
	}
	if out.Disc != in.Disc || out.Type != in.Type {
		t.Errorf("header round-trip = %s/%X, want %s/%X",
			out.Disc, out.Type, in.Disc, in.Type)
	}
	if !reflect.DeepEqual(out.Payload, in.Payload) {
		t.Errorf("payload round-trip = %v, want %v", out.Payload, in.Payload)
	}
}

func TestPDUBitRoundTrip(t *testing.T) {
	in := PDU{
		Disc:    DiscMLE,
		Type:    uint8(MLESystemInfo),
		Payload: []byte{0x12, 0x34, 0x56, 0x78, 0x9A},
	}
	bits := PDUBits(in)
	if len(bits) != 8+5*8 {
		t.Fatalf("PDUBits len = %d, want 48", len(bits))
	}
	out, err := PDUFromBits(bits)
	if err != nil {
		t.Fatalf("PDUFromBits: %v", err)
	}
	if !reflect.DeepEqual(out, in) {
		t.Errorf("bits round-trip = %+v, want %+v", out, in)
	}
}

func TestParsePDUEmpty(t *testing.T) {
	if _, err := ParsePDU(nil); err == nil {
		t.Error("expected error on empty info")
	}
}

func TestPDUFromBitsTooShort(t *testing.T) {
	if _, err := PDUFromBits(make([]byte, 7)); err == nil {
		t.Error("expected error on <8 bits")
	}
}

func TestDiscriminatorClassification(t *testing.T) {
	cases := map[Discriminator]struct{ cmce, mle bool }{
		DiscMLE:        {false, true},
		DiscMLE | 0x1:  {false, true},
		DiscCMCE:       {true, false},
		DiscCMCE | 0x2: {true, false},
		DiscMM:         {false, false},
		DiscSDS:        {false, false},
	}
	for d, want := range cases {
		p := PDU{Disc: d}
		if p.IsCMCE() != want.cmce || p.IsMLE() != want.mle {
			t.Errorf("Disc %X: cmce=%v mle=%v, want %+v",
				uint8(d), p.IsCMCE(), p.IsMLE(), want)
		}
	}
}

// buildVoiceGrantPayload constructs the 11-byte D-CONNECT payload
// that AsVoiceGrant decodes.
func buildVoiceGrantPayload(cid uint16, src, dst uint32, carrier uint16, slot uint8, group, emer, enc bool) []byte {
	out := make([]byte, 11)
	cidField := (cid & 0x3FFF) << 2
	out[0] = byte(cidField >> 8)
	out[1] = byte(cidField & 0xFF)
	out[2] = byte((src >> 16) & 0xFF)
	out[3] = byte((src >> 8) & 0xFF)
	out[4] = byte(src & 0xFF)
	out[5] = byte((dst >> 16) & 0xFF)
	out[6] = byte((dst >> 8) & 0xFF)
	out[7] = byte(dst & 0xFF)
	var flags byte
	if group {
		flags |= 0x80
	}
	if emer {
		flags |= 0x40
	}
	if enc {
		flags |= 0x20
	}
	out[8] = flags
	carrierField := ((carrier & 0xFFF) << 4) | (uint16(slot&0x3) << 2)
	out[9] = byte(carrierField >> 8)
	out[10] = byte(carrierField & 0xFF)
	return out
}

func TestAsVoiceGrantExtractsFields(t *testing.T) {
	payload := buildVoiceGrantPayload(
		0x1234, 0x100200, 0x300400, 0x0ABC, 2,
		true, true, false,
	)
	pdu := PDU{Disc: DiscCMCE, Type: uint8(CMCEDConnect), Payload: payload}
	g, ok := pdu.AsVoiceGrant()
	if !ok {
		t.Fatal("AsVoiceGrant returned !ok")
	}
	if g.CallIdentifier != 0x1234 {
		t.Errorf("CallIdentifier = %X, want 1234", g.CallIdentifier)
	}
	if g.SourceSSI != 0x100200 || g.DestSSI != 0x300400 {
		t.Errorf("SSIs = %X / %X", g.SourceSSI, g.DestSSI)
	}
	if g.CarrierNumber != 0x0ABC {
		t.Errorf("CarrierNumber = %X, want ABC", g.CarrierNumber)
	}
	if g.Timeslot != 2 {
		t.Errorf("Timeslot = %d", g.Timeslot)
	}
	if !g.Group || !g.Emergency || g.Encrypted {
		t.Errorf("flags = %+v", g)
	}
}

func TestAsVoiceGrantTxGrantedAlsoMatches(t *testing.T) {
	payload := buildVoiceGrantPayload(0x10, 1, 2, 0x100, 0, false, false, true)
	pdu := PDU{Disc: DiscCMCE, Type: uint8(CMCEDTxGranted), Payload: payload}
	g, ok := pdu.AsVoiceGrant()
	if !ok || !g.Encrypted || g.CarrierNumber != 0x100 {
		t.Errorf("D-TX-GRANTED grant = %+v ok=%v", g, ok)
	}
}

func TestAsVoiceGrantWrongType(t *testing.T) {
	pdu := PDU{Disc: DiscCMCE, Type: uint8(CMCEDRelease), Payload: make([]byte, 11)}
	if _, ok := pdu.AsVoiceGrant(); ok {
		t.Error("AsVoiceGrant returned ok for D-RELEASE")
	}
}

func TestAsVoiceGrantWrongDisc(t *testing.T) {
	pdu := PDU{Disc: DiscMM, Type: uint8(CMCEDConnect), Payload: make([]byte, 11)}
	if _, ok := pdu.AsVoiceGrant(); ok {
		t.Error("AsVoiceGrant returned ok for non-CMCE Disc")
	}
}

func TestAsVoiceGrantShortPayload(t *testing.T) {
	pdu := PDU{Disc: DiscCMCE, Type: uint8(CMCEDConnect), Payload: make([]byte, 8)}
	if _, ok := pdu.AsVoiceGrant(); ok {
		t.Error("AsVoiceGrant returned ok for short payload")
	}
}

func TestAsRelease(t *testing.T) {
	payload := []byte{0x12, 0x38, 0x05} // CID = 0x048E (after >>2), cause = 5
	pdu := PDU{Disc: DiscCMCE, Type: uint8(CMCEDRelease), Payload: payload}
	r, ok := pdu.AsRelease()
	if !ok {
		t.Fatal("AsRelease returned !ok")
	}
	if r.CallIdentifier != 0x048E {
		t.Errorf("CallIdentifier = %X, want 048E", r.CallIdentifier)
	}
	if r.DisconnectCause != 5 {
		t.Errorf("DisconnectCause = %d", r.DisconnectCause)
	}
	other := PDU{Disc: DiscCMCE, Type: uint8(CMCEDConnect)}
	if _, ok := other.AsRelease(); ok {
		t.Error("AsRelease returned ok for non-RELEASE")
	}
}

// buildSysInfoPayload packs MCC/MNC/LA into the 5-byte payload
// AsSystemBroadcast decodes.
func buildSysInfoPayload(mcc, mnc, la uint16) []byte {
	out := make([]byte, 5)
	mcc &= 0x3FF
	mnc &= 0x3FFF
	la &= 0x3FFF
	out[0] = byte((mcc >> 2) & 0xFF)
	out[1] = byte((mcc&0x3)<<6) | byte((mnc>>8)&0x3F)
	out[2] = byte(mnc & 0xFF)
	out[3] = byte((la >> 6) & 0xFF)
	out[4] = byte((la & 0x3F) << 2)
	return out
}

func TestAsSystemBroadcast(t *testing.T) {
	payload := buildSysInfoPayload(234, 1234, 5678)
	pdu := PDU{Disc: DiscMLE, Type: uint8(MLESystemInfo), Payload: payload}
	sb, ok := pdu.AsSystemBroadcast()
	if !ok {
		t.Fatal("AsSystemBroadcast returned !ok")
	}
	if sb.MCC != 234 || sb.MNC != 1234 || sb.LocationArea != 5678 {
		t.Errorf("SysInfo = %+v, want MCC=234 MNC=1234 LA=5678", sb)
	}
	other := PDU{Disc: DiscCMCE, Type: uint8(CMCEDConnect)}
	if _, ok := other.AsSystemBroadcast(); ok {
		t.Error("AsSystemBroadcast returned ok for non-SYSINFO")
	}
}

func TestPDUIsIdle(t *testing.T) {
	cases := map[PDUType]bool{
		CMCEDTxCeased: true,
		CMCEDConnect:  false,
		CMCEDRelease:  false,
	}
	for typ, want := range cases {
		p := PDU{Disc: DiscCMCE, Type: uint8(typ)}
		if got := p.IsIdle(); got != want {
			t.Errorf("IsIdle(%X) = %v, want %v", uint8(typ), got, want)
		}
	}
	// Non-CMCE PDUs are never "idle" at the CMCE layer.
	if (PDU{Disc: DiscMM}).IsIdle() {
		t.Error("MM PDU classified as idle")
	}
}

func TestSyncDibits(t *testing.T) {
	for name, dibits := range map[string][]uint8{
		"normal":   NormalSyncDibits(),
		"extended": ExtendedSyncDibits(),
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
	if reflect.DeepEqual(NormalSyncDibits(), ExtendedSyncDibits()) {
		t.Error("normal and extended sync patterns are equal")
	}
}

func TestSyncDetectorExactMatch(t *testing.T) {
	pat := NormalSyncDibits()
	det := NewSyncDetector(pat, 0)
	stream := make([]uint8, 50+len(pat)+10)
	copy(stream[50:], pat)
	hits, _ := det.Process(nil, stream, 0)
	if len(hits) != 1 || hits[0] != 50+len(pat)-1 {
		t.Errorf("hits = %v, want [%d]", hits, 50+len(pat)-1)
	}
}

func TestSyncDetectorTolerance(t *testing.T) {
	pat := NormalSyncDibits()
	det := NewSyncDetector(pat, 1)
	const offset = 50
	stream := make([]uint8, offset+len(pat)+10)
	copy(stream[offset:], pat)
	stream[offset+9] = (stream[offset+9] + 1) & 0x3
	hits, _ := det.Process(nil, stream, 0)
	if len(hits) != 1 {
		t.Fatalf("hits = %v, want 1 (tolerance=1)", hits)
	}
}

func TestLinearBandPlan(t *testing.T) {
	bp := LinearBandPlan{BaseHz: 380_000_000, SpacingHz: 25_000, Offset: 0}
	hz, err := bp.Frequency(100)
	if err != nil {
		t.Fatal(err)
	}
	if hz != 380_000_000+100*25_000 {
		t.Errorf("ch100 = %d", hz)
	}
}

func TestLinearBandPlanRejectsZeroSpacing(t *testing.T) {
	bp := LinearBandPlan{BaseHz: 380_000_000}
	if _, err := bp.Frequency(1); err == nil {
		t.Error("expected error on zero SpacingHz")
	}
}

func TestLinearBandPlanRejectsNegativeIndex(t *testing.T) {
	bp := LinearBandPlan{BaseHz: 380_000_000, SpacingHz: 25_000, Offset: -10}
	if _, err := bp.Frequency(5); err == nil {
		t.Error("expected error on negative carrier+offset")
	}
}

func TestTableBandPlan(t *testing.T) {
	bp := TableBandPlan{0x100: 410_000_000, 0x200: 415_000_000}
	if hz, err := bp.Frequency(0x100); err != nil || hz != 410_000_000 {
		t.Errorf("0x100 = %d/%v", hz, err)
	}
	if _, err := bp.Frequency(0x999); err == nil {
		t.Error("expected error on missing carrier")
	}
}

func TestControlChannelEmitsLockOnSysInfo(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, FrequencyHz: 410_000_000})
	cc.Ingest(PDU{
		Disc:    DiscMLE,
		Type:    uint8(MLESystemInfo),
		Payload: buildSysInfoPayload(234, 1234, 5678),
	})

	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLocked {
			t.Fatalf("kind = %s, want cc.locked", ev.Kind)
		}
		ls, ok := ev.Payload.(LockState)
		if !ok || ls.MCC != 234 || ls.MNC != 1234 ||
			ls.LocationArea != 5678 || ls.FrequencyHz != 410_000_000 {
			t.Errorf("LockState = %+v", ev.Payload)
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

	bp := LinearBandPlan{BaseHz: 380_000_000, SpacingHz: 25_000, Offset: 0}
	fixed := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	cc := New(Options{
		Bus:         bus,
		SystemName:  "tetra-sys",
		FrequencyHz: 380_500_000,
		Resolver:    bp,
		Now:         func() time.Time { return fixed },
	})

	cc.Ingest(PDU{
		Disc: DiscCMCE,
		Type: uint8(CMCEDConnect),
		Payload: buildVoiceGrantPayload(
			0x55, 0x000123, 0x00ABCD, 200, 1,
			true, false, true,
		),
	})

	// First event: cc.locked synthesized when grant arrives.
	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLocked {
			t.Fatalf("first event = %s, want cc.locked", ev.Kind)
		}
	case <-time.After(time.Second):
		t.Fatal("no cc.locked")
	}

	// Second event: grant.
	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindGrant {
			t.Fatalf("second event = %s, want grant", ev.Kind)
		}
		g := ev.Payload.(trunking.Grant)
		if g.Protocol != "tetra" {
			t.Errorf("Protocol = %q, want tetra", g.Protocol)
		}
		if g.System != "tetra-sys" {
			t.Errorf("System = %q", g.System)
		}
		if g.GroupID != 0x00ABCD || g.SourceID != 0x000123 {
			t.Errorf("IDs = %X / %X", g.GroupID, g.SourceID)
		}
		if g.ChannelNum != 200 {
			t.Errorf("ChannelNum = %d", g.ChannelNum)
		}
		if g.FrequencyHz != 380_000_000+200*25_000 {
			t.Errorf("FrequencyHz = %d", g.FrequencyHz)
		}
		if !g.Encrypted || g.Emergency {
			t.Errorf("flags = enc=%v emer=%v", g.Encrypted, g.Emergency)
		}
		if !g.At.Equal(fixed) {
			t.Errorf("At = %v, want %v", g.At, fixed)
		}
	case <-time.After(time.Second):
		t.Fatal("no grant event")
	}
}

func TestControlChannelGrantWithoutResolver(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, FrequencyHz: 380_500_000})
	cc.Ingest(PDU{
		Disc:    DiscCMCE,
		Type:    uint8(CMCEDConnect),
		Payload: buildVoiceGrantPayload(1, 2, 3, 42, 0, false, false, false),
	})

	<-sub.C // cc.locked
	ev := <-sub.C
	g := ev.Payload.(trunking.Grant)
	if g.FrequencyHz != 0 {
		t.Errorf("FrequencyHz = %d, want 0 (no resolver)", g.FrequencyHz)
	}
	if g.ChannelNum != 42 {
		t.Errorf("ChannelNum = %d", g.ChannelNum)
	}
}

func TestControlChannelSilentOnIdle(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, FrequencyHz: 380_500_000})
	cc.Ingest(PDU{Disc: DiscCMCE, Type: uint8(CMCEDTxCeased)})

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

	cc := New(Options{Bus: bus, FrequencyHz: 380_500_000})
	cc.Ingest(PDU{
		Disc:    DiscMLE,
		Type:    uint8(MLESystemInfo),
		Payload: buildSysInfoPayload(234, 1234, 5678),
	})
	<-sub.C // cc.locked

	cc.MarkLost()
	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLost {
			t.Fatalf("kind = %s, want cc.lost", ev.Kind)
		}
		ls := ev.Payload.(LockState)
		if ls.MCC != 234 {
			t.Errorf("LockState.MCC = %d", ls.MCC)
		}
	case <-time.After(time.Second):
		t.Fatal("no cc.lost")
	}

	cc.MarkLost()
	select {
	case ev := <-sub.C:
		t.Errorf("unexpected event after second MarkLost: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestControlChannelNoRepublishOnSameSysInfo(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, FrequencyHz: 380_500_000})
	pdu := PDU{
		Disc:    DiscMLE,
		Type:    uint8(MLESystemInfo),
		Payload: buildSysInfoPayload(234, 1234, 5678),
	}
	cc.Ingest(pdu)
	<-sub.C
	cc.Ingest(pdu)
	select {
	case ev := <-sub.C:
		t.Errorf("unexpected re-publish: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}
