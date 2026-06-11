package tier3

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// burstWithCSBK packs a CSBK into a DMR burst's payload halves.
func burstWithCSBK(c CSBK) *dmr.Burst {
	info := AssembleCSBK(c)
	bits := make([]byte, 96)
	for i := 0; i < 96; i++ {
		bits[i] = (info[i>>3] >> uint(7-(i&7))) & 1
	}
	channel := framing.EncodeBPTC196_96(bits)
	var b dmr.Burst
	for i := 0; i < dmr.HalfPayloadDibits; i++ {
		b.Dibits[i] = (channel[2*i] << 1) | channel[2*i+1]
	}
	for i := 0; i < dmr.HalfPayloadDibits; i++ {
		b.Dibits[dmr.BurstDibits-dmr.HalfPayloadDibits+i] = (channel[2*(dmr.HalfPayloadDibits+i)] << 1) | channel[2*(dmr.HalfPayloadDibits+i)+1]
	}
	return &b
}

func TestControlChannelEmitsLockOnAloha(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := NewControlChannel(bus, nil, 851_000_000)
	c := CSBK{LB: true, Opcode: OpAloha, FID: 0, Payload: [8]byte{0x12, 0x08, 0x12, 0x34, 0, 0, 0, 0}}
	cc.IngestBurst(burstWithCSBK(c), dmr.SlotType{ColorCode: 1, DataType: dmr.DTCSBK})

	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLocked {
			t.Fatalf("kind = %s", ev.Kind)
		}
		ls, ok := ev.Payload.(LockState)
		if !ok {
			t.Fatalf("payload type %T", ev.Payload)
		}
		if ls.SystemID != 0x1234 || ls.ColorCode != 1 || ls.FrequencyHz != 851_000_000 {
			t.Errorf("payload = %+v", ls)
		}
	case <-time.After(time.Second):
		t.Fatal("no event")
	}
}

func TestControlChannelEmitsLockOnSysInfo(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := NewControlChannel(bus, nil, 851_000_000)
	c := CSBK{LB: true, Opcode: OpSysInfo, Payload: [8]byte{0xCA, 0xFE, 0x01, 0x05, 0xFF, 0, 0, 0}}
	cc.IngestBurst(burstWithCSBK(c), dmr.SlotType{ColorCode: 0xC, DataType: dmr.DTCSBK})

	select {
	case ev := <-sub.C:
		ls := ev.Payload.(LockState)
		if ls.SystemID != 0xCAFE {
			t.Errorf("sysid = %X, want CAFE", ls.SystemID)
		}
	case <-time.After(time.Second):
		t.Fatal("no event")
	}
}

func TestControlChannelIgnoresNonCSBK(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := NewControlChannel(bus, nil, 851_000_000)
	c := CSBK{LB: true, Opcode: OpAloha, Payload: [8]byte{0, 0x08, 0xCA, 0xFE, 0, 0, 0, 0}}
	cc.IngestBurst(burstWithCSBK(c), dmr.SlotType{ColorCode: 1, DataType: dmr.DTVoiceLCHeader})

	select {
	case ev := <-sub.C:
		t.Errorf("unexpected event: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestControlChannelPublishesTVGrant(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		SystemName:  "TestSys",
		FrequencyHz: 851_000_000,
		Resolver:    LinearBandPlan{BaseHz: 866_000_000, SpacingHz: 25_000, Offset: 1},
		Now:         func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
	})
	// Service options 0xC0 = encrypted+emergency, dst 0x123456, src 0xABCDEF,
	// byte 7 = 0x88 → TS2, LCN 8.
	csbk := CSBK{
		LB:      true,
		Opcode:  OpTVGrant,
		Payload: [8]byte{0xC0, 0x12, 0x34, 0x56, 0xAB, 0xCD, 0xEF, 0x88},
	}
	cc.IngestBurst(burstWithCSBK(csbk), dmr.SlotType{ColorCode: 3, DataType: dmr.DTCSBK})

	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindGrant {
				continue
			}
			g := ev.Payload.(trunking.Grant)
			if g.System != "TestSys" || g.Protocol != "dmr-tier3" {
				t.Errorf("identity = %s/%s", g.System, g.Protocol)
			}
			if g.GroupID != 0x123456 || g.SourceID != 0xABCDEF {
				t.Errorf("group/source = %X/%X", g.GroupID, g.SourceID)
			}
			if g.ChannelID != 3 {
				t.Errorf("ChannelID (color code) = %d, want 3", g.ChannelID)
			}
			if g.ChannelNum != 8 {
				t.Errorf("ChannelNum (LCN) = %d, want 8", g.ChannelNum)
			}
			// LinearBandPlan{base 866M, spacing 25k, offset 1}: LCN 8 = 866M + 7×25k = 866.175M.
			if g.FrequencyHz != 866_175_000 {
				t.Errorf("freq = %d, want 866_175_000", g.FrequencyHz)
			}
			if !g.Encrypted || !g.Emergency {
				t.Errorf("flags = enc=%v emer=%v, want both", g.Encrypted, g.Emergency)
			}
			// byte 7 bit 7 set → CSBK TS2 (0-based 1) → 1-based Timeslot 2.
			if g.Timeslot != 2 {
				t.Errorf("Timeslot = %d, want 2 (TS2)", g.Timeslot)
			}
			if g.At.Unix() != 1_700_000_000 {
				t.Errorf("At = %v, want injected Now", g.At)
			}
			return
		case <-deadline:
			t.Fatal("no grant event published")
		}
	}
}

func TestControlChannelGrantCarriesInterleavedFlag(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:              bus,
		SystemName:       "TestSys",
		Resolver:         TableBandPlan{8: 866_175_000},
		InterleavedVoice: true,
	})
	csbk := CSBK{LB: true, Opcode: OpTVGrant, Payload: [8]byte{0xC0, 0x12, 0x34, 0x56, 0xAB, 0xCD, 0xEF, 0x88}}
	cc.IngestBurst(burstWithCSBK(csbk), dmr.SlotType{ColorCode: 3, DataType: dmr.DTCSBK})

	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindGrant {
				continue
			}
			if g := ev.Payload.(trunking.Grant); !g.DMRInterleavedVoice {
				t.Errorf("DMRInterleavedVoice = false, want true when Options.InterleavedVoice is set")
			}
			return
		case <-deadline:
			t.Fatal("no grant event published")
		}
	}
}

func TestControlChannelPublishesPVGrant(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		SystemName:  "TestSys",
		FrequencyHz: 851_000_000,
		Resolver:    TableBandPlan{4: 462_550_000},
	})
	csbk := CSBK{
		LB:      true,
		Opcode:  OpPVGrant,
		Payload: [8]byte{0x00, 0x00, 0x12, 0x34, 0x00, 0xAB, 0xCD, 0x04},
	}
	cc.IngestBurst(burstWithCSBK(csbk), dmr.SlotType{ColorCode: 1, DataType: dmr.DTCSBK})

	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindGrant {
				continue
			}
			g := ev.Payload.(trunking.Grant)
			if g.GroupID != 0x001234 || g.SourceID != 0x00ABCD {
				t.Errorf("dst/src = %X/%X", g.GroupID, g.SourceID)
			}
			if g.FrequencyHz != 462_550_000 {
				t.Errorf("freq = %d, want 462_550_000", g.FrequencyHz)
			}
			// byte 7 bit 7 clear → CSBK TS1 (0-based 0) → 1-based Timeslot 1.
			if g.Timeslot != 1 {
				t.Errorf("Timeslot = %d, want 1 (TS1)", g.Timeslot)
			}
			return
		case <-deadline:
			t.Fatal("no PV grant event")
		}
	}
}

func TestControlChannelGrantWithoutResolverEmitsDecodeError(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	// New() with nil Resolver → grants emit decode-error stage=no-bandplan.
	cc := New(Options{Bus: bus, SystemName: "S", FrequencyHz: 1})
	csbk := CSBK{
		LB:      true,
		Opcode:  OpTVGrant,
		Payload: [8]byte{0x00, 0x00, 0x00, 0x42, 0x00, 0x00, 0x01, 0x05},
	}
	cc.IngestBurst(burstWithCSBK(csbk), dmr.SlotType{ColorCode: 1, DataType: dmr.DTCSBK})

	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				t.Fatalf("unexpected Grant without resolver: %+v", ev.Payload)
			}
			if ev.Kind != events.KindDecodeError {
				continue
			}
			de := ev.Payload.(events.DecodeError)
			if de.Protocol == "dmr-tier3" && de.Stage == "no-bandplan" {
				return
			}
		case <-deadline:
			t.Fatal("no decode-error stage=no-bandplan")
		}
	}
}

func TestControlChannelGrantOutsideBandPlanEmitsDecodeError(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		SystemName:  "S",
		FrequencyHz: 1,
		Resolver:    TableBandPlan{1: 1_000_000}, // only LCN 1 known
	})
	csbk := CSBK{
		LB:      true,
		Opcode:  OpTVGrant,
		Payload: [8]byte{0x00, 0x00, 0x00, 0x42, 0x00, 0x00, 0x01, 0x07}, // LCN 7
	}
	cc.IngestBurst(burstWithCSBK(csbk), dmr.SlotType{ColorCode: 1, DataType: dmr.DTCSBK})

	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				t.Fatalf("unexpected Grant for unknown LCN: %+v", ev.Payload)
			}
			if ev.Kind != events.KindDecodeError {
				continue
			}
			de := ev.Payload.(events.DecodeError)
			if de.Protocol == "dmr-tier3" && de.Stage == "no-bandplan" {
				return
			}
		case <-deadline:
			t.Fatal("no decode-error stage=no-bandplan")
		}
	}
}

func TestControlChannelMarkLost(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := NewControlChannel(bus, nil, 851_000_000)
	c := CSBK{LB: true, Opcode: OpSysInfo, Payload: [8]byte{0x42, 0x00, 0x01, 0x01, 0, 0, 0, 0}}
	cc.IngestBurst(burstWithCSBK(c), dmr.SlotType{ColorCode: 7, DataType: dmr.DTCSBK})
	<-sub.C

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

// collectGrantObserved drains the bus until it sees a KindDMRGrantObserved
// (returning it) or the deadline elapses. sawGrant/sawDecodeErr report
// whether a resolved grant / no-bandplan decode error were also seen.
func collectGrantObserved(t *testing.T, sub *events.Subscription) (obs events.DMRGrantObserved, sawObs, sawGrant, sawDecodeErr bool) {
	t.Helper()
	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			switch ev.Kind {
			case events.KindDMRGrantObserved:
				obs = ev.Payload.(events.DMRGrantObserved)
				sawObs = true
			case events.KindGrant:
				sawGrant = true
			case events.KindDecodeError:
				if de, ok := ev.Payload.(events.DecodeError); ok && de.Stage == events.StageNoBandPlan {
					sawDecodeErr = true
				}
			}
		case <-deadline:
			return
		}
	}
}

func TestControlChannelEmitsGrantObservedWithResolver(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		SystemName:  "ObsSys",
		FrequencyHz: 851_012_500,
		Resolver:    LinearBandPlan{BaseHz: 866_000_000, SpacingHz: 25_000, Offset: 1},
		Now:         func() time.Time { return time.Unix(1_700_000_001, 0).UTC() },
	})
	// dst 0x123456, src 0xABCDEF, byte 7 = 0x88 → TS2, LCN 8.
	csbk := CSBK{LB: true, Opcode: OpTVGrant, Payload: [8]byte{0x00, 0x12, 0x34, 0x56, 0xAB, 0xCD, 0xEF, 0x88}}
	cc.IngestBurst(burstWithCSBK(csbk), dmr.SlotType{ColorCode: 5, DataType: dmr.DTCSBK})

	obs, sawObs, sawGrant, _ := collectGrantObserved(t, sub)
	if !sawObs {
		t.Fatal("no KindDMRGrantObserved published")
	}
	if !sawGrant {
		t.Error("resolved grant (KindGrant) was not also published")
	}
	if obs.System != "ObsSys" || obs.ColorCode != 5 || obs.LCN != 8 || obs.Timeslot != 1 {
		t.Errorf("obs = %+v", obs)
	}
	if obs.GroupID != 0x123456 || obs.SourceID != 0xABCDEF {
		t.Errorf("group/source = %X/%X", obs.GroupID, obs.SourceID)
	}
	if obs.CCFreqHz != 851_012_500 {
		t.Errorf("CCFreqHz = %d, want 851_012_500", obs.CCFreqHz)
	}
	if obs.At.Unix() != 1_700_000_001 {
		t.Errorf("At = %v, want injected Now", obs.At)
	}
}

func TestControlChannelEmitsGrantObservedWithoutResolver(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	// nil Resolver: the LCN must still escape via KindDMRGrantObserved, and
	// the existing no-bandplan decode error must still fire.
	cc := New(Options{Bus: bus, SystemName: "ObsSys", FrequencyHz: 851_012_500})
	csbk := CSBK{LB: true, Opcode: OpTVGrant, Payload: [8]byte{0x00, 0x00, 0x10, 0x00, 0x00, 0x20, 0x00, 0x0B}}
	cc.IngestBurst(burstWithCSBK(csbk), dmr.SlotType{ColorCode: 2, DataType: dmr.DTCSBK})

	obs, sawObs, sawGrant, sawDecodeErr := collectGrantObserved(t, sub)
	if !sawObs {
		t.Fatal("no KindDMRGrantObserved published with nil resolver")
	}
	if obs.LCN != 0x0B {
		t.Errorf("LCN = %d, want 11", obs.LCN)
	}
	if sawGrant {
		t.Error("KindGrant published despite nil resolver")
	}
	if !sawDecodeErr {
		t.Error("no-bandplan decode error not published")
	}
}

func TestControlChannelSetResolverHotSwap(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	// Start with no resolver: the first grant drops with no-bandplan.
	cc := New(Options{Bus: bus, SystemName: "SwapSys", FrequencyHz: 851_012_500})
	if cc.HasResolver() {
		t.Fatal("HasResolver true before SetResolver")
	}
	csbk := CSBK{LB: true, Opcode: OpTVGrant, Payload: [8]byte{0x00, 0x00, 0x10, 0x00, 0x00, 0x20, 0x00, 0x08}}
	cc.IngestBurst(burstWithCSBK(csbk), dmr.SlotType{ColorCode: 1, DataType: dmr.DTCSBK})
	if _, _, sawGrant, sawDecodeErr := collectGrantObserved(t, sub); sawGrant || !sawDecodeErr {
		t.Fatalf("pre-swap: grant=%v decodeErr=%v, want grant=false decodeErr=true", sawGrant, sawDecodeErr)
	}

	// Hot-swap a learned resolver; the same grant now resolves.
	cc.SetResolver(LinearBandPlan{BaseHz: 866_000_000, SpacingHz: 25_000, Offset: 1})
	if !cc.HasResolver() {
		t.Fatal("HasResolver false after SetResolver")
	}
	cc.IngestBurst(burstWithCSBK(csbk), dmr.SlotType{ColorCode: 1, DataType: dmr.DTCSBK})
	_, _, sawGrant, _ := collectGrantObserved(t, sub)
	if !sawGrant {
		t.Fatal("post-swap: grant not resolved after SetResolver")
	}
}

func TestControlChannelSetResolverConcurrent(t *testing.T) {
	// Race detector: one goroutine ingests grants (reads resolver) while
	// another swaps it. go test -race must stay clean.
	bus := events.NewBus(256)
	defer bus.Close()
	sub := bus.Subscribe()
	go func() {
		for range sub.C { // drain so Publish never blocks the producer
		}
	}()

	cc := New(Options{Bus: bus, SystemName: "RaceSys", FrequencyHz: 851_012_500})
	csbk := CSBK{LB: true, Opcode: OpTVGrant, Payload: [8]byte{0x00, 0x00, 0x10, 0x00, 0x00, 0x20, 0x00, 0x08}}
	burst := burstWithCSBK(csbk)

	done := make(chan struct{})
	go func() {
		for i := 0; i < 2000; i++ {
			cc.IngestBurst(burst, dmr.SlotType{ColorCode: 1, DataType: dmr.DTCSBK})
		}
		close(done)
	}()
	for i := 0; i < 2000; i++ {
		cc.SetResolver(LinearBandPlan{BaseHz: 866_000_000, SpacingHz: 25_000, Offset: 1})
	}
	<-done
	sub.Close()
}
