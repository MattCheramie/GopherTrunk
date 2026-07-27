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

// grantMAC builds the MACResource that publishGrantFromMAC turns into a voice
// grant — the real on-air grant carrier (a MAC-RESOURCE channel allocation
// addressed to the group SSI), replacing the byte-aligned D-CONNECT the deleted
// AsVoiceGrant used to parse. The bit-accurate CMCE parse itself is covered by
// cmce_parse_test.go and the end-to-end MAC path by downlink_test.go.
func grantMAC(dst uint32, carrier uint16, slot uint8, enc bool) MACResource {
	return MACResource{
		Encrypted: enc,
		Address:   MACAddress{Type: addrSSI, SSI: dst},
		ChanAlloc: &ChannelAllocation{CarrierNumber: carrier, Timeslot: slot},
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
	for _, tc := range []struct {
		name   string
		dibits []uint8
		want   int
	}{
		{"nts1", NormalSyncDibits(), NormalTrainingBits / 2},
		{"nts2", NormalSyncDibits2(), NormalTrainingBits / 2},
		{"extended", ExtendedSyncDibits(), ExtendedTrainingBits / 2},
		{"sync", SyncTrainingDibits(), SyncTrainingBits / 2},
	} {
		if len(tc.dibits) != tc.want {
			t.Errorf("%s len = %d, want %d", tc.name, len(tc.dibits), tc.want)
		}
		for _, d := range tc.dibits {
			if d > 3 {
				t.Errorf("%s contains dibit %d", tc.name, d)
			}
		}
	}
	if reflect.DeepEqual(NormalSyncDibits(), ExtendedSyncDibits()) {
		t.Error("normal and extended sync patterns are equal")
	}
}

// TestTrainingSequencesMatchETSI guards the literal training-sequence
// bit values against the ETSI EN 300 392-2 §9.4.4.3 reference (the
// same values used by osmo-tetra). It is the regression that fails if
// anyone re-introduces a placeholder constant: the bytes are checked
// against an independent literal, not derived from the package vars.
func TestTrainingSequencesMatchETSI(t *testing.T) {
	want := map[string][]uint8{
		"NTS1": {1, 1, 0, 1, 0, 0, 0, 0, 1, 1, 1, 0, 1, 0, 0, 1, 1, 1, 0, 1, 0, 0},
		"NTS2": {0, 1, 1, 1, 1, 0, 1, 0, 0, 1, 0, 0, 0, 0, 1, 1, 0, 1, 1, 1, 1, 0},
		"ETS":  {1, 0, 0, 1, 1, 1, 0, 1, 0, 0, 0, 0, 1, 1, 1, 0, 1, 0, 0, 1, 1, 1, 0, 1, 0, 0, 1, 1, 1, 0},
		"STS":  {1, 1, 0, 0, 0, 0, 0, 1, 1, 0, 0, 1, 1, 1, 0, 0, 1, 1, 1, 0, 1, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 1, 1, 0, 0, 1, 1, 1},
	}
	got := map[string][]uint8{
		"NTS1": NormalTrainingSeq1, "NTS2": NormalTrainingSeq2,
		"ETS": ExtendedTrainingSeq, "STS": SyncTrainingSeq,
	}
	for name, w := range want {
		if !reflect.DeepEqual(got[name], w) {
			t.Errorf("%s = %v, want %v", name, got[name], w)
		}
	}
}

// TestTetraBitDibitRoundTrip checks the TETRA Gray bit↔dibit
// convention is self-inverse and matches the demod's labeling
// (00→0, 01→1, 11→2, 10→3).
func TestTetraBitDibitRoundTrip(t *testing.T) {
	cases := map[[2]uint8]uint8{{0, 0}: 0, {0, 1}: 1, {1, 1}: 2, {1, 0}: 3}
	for bits, want := range cases {
		got := TetraBitsToDibits([]uint8{bits[0], bits[1]})
		if len(got) != 1 || got[0] != want {
			t.Errorf("TetraBitsToDibits(%v) = %v, want [%d]", bits, got, want)
		}
		back := TetraDibitsToBits([]uint8{want})
		if back[0] != bits[0] || back[1] != bits[1] {
			t.Errorf("TetraDibitsToBits([%d]) = %v, want %v", want, back, bits)
		}
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

	// A MAC-RESOURCE grant for dest 0x00ABCD on carrier 200 / slot 1, encrypted,
	// with a D-SETUP CMCE PDU supplying the source SSI (0x000123), non-emergency.
	cc.publishGrantFromMAC(
		grantMAC(0x00ABCD, 200, 1, true),
		CMCEMessage{Type: CMCETypeDSetup, PartySSI: 0x000123},
		true,
	)

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
	cc.publishGrantFromMAC(grantMAC(3, 42, 0, false), CMCEMessage{}, false)

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
