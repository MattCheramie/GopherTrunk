package phase1

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func TestNetworkModelAccumulates(t *testing.T) {
	var m NetworkModel
	m.ApplyNetworkStatus(NetworkStatusBroadcast{WACN: 0xABCDE, SystemID: 0x123})
	m.ApplyRFSSStatus(RFSSStatusBroadcast{SystemID: 0x123, RFSS: 4, Site: 7, LRA: 9})
	m.ApplySecondaryControlChannel(SecondaryControlChannelBroadcast{
		ChannelAID: 1, ChannelANumber: 100, ChannelBID: 1, ChannelBNumber: 200})
	m.ApplyAdjacentSite(AdjacentSiteStatusBroadcast{RFSS: 4, Site: 8, ChannelID: 1, ChannelNumber: 300})
	m.ApplyAdjacentSite(AdjacentSiteStatusBroadcast{RFSS: 4, Site: 9, ChannelID: 1, ChannelNumber: 301})
	// Re-broadcast of site 8 must update in place, not duplicate.
	m.ApplyAdjacentSite(AdjacentSiteStatusBroadcast{RFSS: 4, Site: 8, ChannelID: 1, ChannelNumber: 305})

	cfg := m.Snapshot()
	if cfg.WACN != 0xABCDE || cfg.SystemID != 0x123 {
		t.Errorf("WACN/SystemID = %#x/%#x", cfg.WACN, cfg.SystemID)
	}
	if cfg.RFSS != 4 || cfg.Site != 7 || cfg.LRA != 9 {
		t.Errorf("RFSS/Site/LRA = %d/%d/%d", cfg.RFSS, cfg.Site, cfg.LRA)
	}
	if len(cfg.Secondary) != 2 {
		t.Errorf("Secondary = %v, want 2 channels", cfg.Secondary)
	}
	if len(cfg.Neighbors) != 2 {
		t.Fatalf("Neighbors = %v, want 2 (site 8 deduped)", cfg.Neighbors)
	}
	for _, n := range cfg.Neighbors {
		if n.Site == 8 && n.ChannelNumber != 305 {
			t.Errorf("site 8 neighbour not updated: %+v", n)
		}
	}
}

// TestControlChannelAccumulatesTopology drives status-broadcast TSBKs
// through the control channel and checks NetworkSnapshot reflects them.
func TestControlChannelAccumulatesTopology(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cc := New(Options{Bus: bus, SystemName: "S"})

	// NSB payload: LRA p0, WACN 0xABCDE (p1<<12|p2<<4|p3>>4), SystemID
	// 0x123 ((p3&0x0F)<<8|p4) — see ParseNetworkStatusBroadcast.
	nsb := TSBK{Opcode: OpNetworkStatusBroadcast,
		Payload: [8]byte{0x00, 0xAB, 0xCD, 0xE1, 0x23}}
	// RFSS payload: LRA p0, SystemID p1-2, RFSS p3, Site p4 —
	// see ParseRFSSStatusBroadcast.
	rfss := TSBK{Opcode: OpRFSSStatusBroadcast,
		Payload: [8]byte{9, 0x01, 0x23, 4, 7}}
	adj := TSBK{Opcode: OpAdjacentSiteStatusBroadcast,
		Payload: AssembleAdjacentSiteStatusBroadcast(AdjacentSiteStatusBroadcast{RFSS: 4, Site: 8, ChannelID: 1, ChannelNumber: 300})}

	base := 0
	for _, tsbk := range []TSBK{nsb, rfss, adj} {
		cc.Process(buildLockedStreamWithTSBK(10, 0x293, DUIDTrunkingSignaling, tsbk), base)
		base += 1 << 20
	}

	cfg := cc.NetworkSnapshot()
	if cfg.WACN != 0xABCDE || cfg.RFSS != 4 || cfg.Site != 7 {
		t.Errorf("snapshot = %+v, want WACN 0xABCDE / RFSS 4 / Site 7", cfg)
	}
	if len(cfg.Neighbors) != 1 || cfg.Neighbors[0].Site != 8 {
		t.Errorf("neighbours = %v, want one site-8 entry", cfg.Neighbors)
	}
}

// TestControlChannelDecodesBroadcastUnderVendorMFID drives the network/site
// broadcasts with the Motorola vendor MFID (0x90) — the form a real VHF site
// was seen emitting — and checks the standard broadcast opcodes are still
// decoded (WACN/SysID/RFSS/Site/neighbour surface) rather than dropped as
// unhandled vendor TSBKs, while a genuine Motorola vendor opcode (a patch)
// stays on the vendor path.
func TestControlChannelDecodesBroadcastUnderVendorMFID(t *testing.T) {
	cc := New(Options{Bus: events.NewBus(8), SystemName: "MotoVHF"})

	nsb := TSBK{Opcode: OpNetworkStatusBroadcast, MFID: MFIDMotorola,
		Payload: [8]byte{0x00, 0xAB, 0xCD, 0xE1, 0x23}}
	rfss := TSBK{Opcode: OpRFSSStatusBroadcast, MFID: MFIDMotorola,
		Payload: [8]byte{9, 0x01, 0x23, 4, 7}}
	adj := TSBK{Opcode: OpAdjacentSiteStatusBroadcast, MFID: MFIDMotorola,
		Payload: AssembleAdjacentSiteStatusBroadcast(AdjacentSiteStatusBroadcast{RFSS: 4, Site: 8, ChannelID: 1, ChannelNumber: 300})}

	base := 0
	for _, tsbk := range []TSBK{nsb, rfss, adj} {
		cc.Process(buildLockedStreamWithTSBK(10, 0x2C2, DUIDTrunkingSignaling, tsbk), base)
		base += 1 << 20
	}

	cfg := cc.NetworkSnapshot()
	if cfg.WACN != 0xABCDE || cfg.SystemID != 0x123 || cfg.RFSS != 4 || cfg.Site != 7 {
		t.Errorf("vendor-MFID broadcasts not decoded: %+v", cfg)
	}
	if len(cfg.Neighbors) != 1 || cfg.Neighbors[0].Site != 8 {
		t.Errorf("neighbour from vendor-MFID adjacent broadcast missing: %v", cfg.Neighbors)
	}
}

// TestControlChannelPublishesUnitToUnitRequest drives a standard-MFID
// Unit-to-Unit Answer Request (UU_ANS_REQ, opcode 0x05) through the control
// channel and checks a KindUnitToUnitRequest naming the calling/called units
// is published on the bus.
func TestControlChannelPublishesUnitToUnitRequest(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	cc := New(Options{Bus: bus, SystemName: "U2U"})

	utuar := TSBK{
		Opcode: OpUnitToUnitAnswerRequest,
		MFID:   MFIDStandard,
		Payload: AssembleUnitToUnitAnswerRequest(UnitToUnitAnswerRequest{
			ServiceOptions: 0x00, TargetID: 0x1234AB, SourceID: 0x00CDEF}),
	}
	cc.Process(buildLockedStreamWithTSBK(10, 0x705, DUIDTrunkingSignaling, utuar), 0)

	deadline := time.After(3 * time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindUnitToUnitRequest {
				continue
			}
			u, ok := ev.Payload.(trunking.UnitToUnitRequest)
			if !ok {
				t.Fatalf("payload is %T, want trunking.UnitToUnitRequest", ev.Payload)
			}
			if u.SourceID != 0x00CDEF || u.TargetID != 0x1234AB {
				t.Fatalf("src/target = %d/%d, want %d/%d", u.SourceID, u.TargetID, 0x00CDEF, 0x1234AB)
			}
			return
		case <-deadline:
			t.Fatal("no KindUnitToUnitRequest published within deadline")
		}
	}
}

// TestControlChannelIgnoresVendorUnitToUnitRequest checks that opcode 0x05 under
// a Motorola MFID is NOT decoded as the standard UU_ANS_REQ — under a vendor
// MFID the opcode is in the manufacturer's namespace, and decoding a Motorola
// 0x05 with the standard layout produced garbage IDs in the field.
func TestControlChannelIgnoresVendorUnitToUnitRequest(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	cc := New(Options{Bus: bus, SystemName: "U2U"})

	utuar := TSBK{
		Opcode: OpUnitToUnitAnswerRequest,
		MFID:   MFIDMotorola,
		Payload: AssembleUnitToUnitAnswerRequest(UnitToUnitAnswerRequest{
			ServiceOptions: 0x00, TargetID: 0x1234AB, SourceID: 0x00CDEF}),
	}
	cc.Process(buildLockedStreamWithTSBK(10, 0x705, DUIDTrunkingSignaling, utuar), 0)

	deadline := time.After(500 * time.Millisecond)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindUnitToUnitRequest {
				t.Fatal("vendor-MFID 0x05 must not publish a UU_ANS_REQ event")
			}
		case <-deadline:
			return // no UU_ANS_REQ event — correct
		}
	}
}

// TestMotorolaProbeOpcodesAreMapSafe drives the suspected-but-undecoded
// Motorola opcodes 0x05 and 0x09 (MFID 0x90) through the control channel and
// checks they inject NOTHING into the system map — no bus event and no
// neighbour / secondary-CC in the topology snapshot. They are captured for
// offline analysis only (logVendorProbe) until their layout is confirmed; this
// guards against anyone wiring a guessed layout into the map.
func TestMotorolaProbeOpcodesAreMapSafe(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	cc := New(Options{Bus: bus, SystemName: "Moto"})

	for _, op := range []Opcode{0x05, 0x09} {
		tsbk := TSBK{Opcode: op, MFID: MFIDMotorola,
			Payload: [8]byte{0x0C, 0x80, 0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC}}
		cc.Process(buildLockedStreamWithTSBK(10, 0x2C2, DUIDTrunkingSignaling, tsbk), 0)
	}

	// No grant/patch/unit-request/etc. should be published from these.
	deadline := time.After(300 * time.Millisecond)
	for {
		select {
		case ev := <-sub.C:
			switch ev.Kind {
			case events.KindCCLocked, events.KindSiteUpdate, events.KindDecodeError:
				// lock/site/decode bookkeeping is fine; only map-data events are forbidden
			default:
				t.Fatalf("Motorola probe opcode published a map event: %s", ev.Kind)
			}
		case <-deadline:
			cfg := cc.NetworkSnapshot()
			if len(cfg.Neighbors) != 0 || len(cfg.Secondary) != 0 {
				t.Fatalf("probe opcodes leaked into topology: %+v", cfg)
			}
			return
		}
	}
}

// TestControlChannelPublishesSiteUpdate drives an RFSS Status Broadcast
// through the control channel and checks a KindSiteUpdate naming the
// camped site (with the tuned control-channel frequency) is published
// on the bus (issue #698).
func TestControlChannelPublishesSiteUpdate(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	const ccHz = 420012500
	cc := New(Options{Bus: bus, SystemName: "MMR", FrequencyHz: ccHz})

	// NSB first so WACN/SystemID are populated, then RFSS to name the site.
	nsb := TSBK{Opcode: OpNetworkStatusBroadcast, Payload: [8]byte{0x00, 0xAB, 0xCD, 0xE1, 0x23}}
	rfss := TSBK{Opcode: OpRFSSStatusBroadcast, Payload: [8]byte{9, 0x01, 0x23, 4, 7}}
	base := 0
	for _, tsbk := range []TSBK{nsb, rfss} {
		cc.Process(buildLockedStreamWithTSBK(10, 0x293, DUIDTrunkingSignaling, tsbk), base)
		base += 1 << 20
	}

	deadline := time.After(3 * time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindSiteUpdate {
				continue
			}
			u, ok := ev.Payload.(trunking.SiteUpdate)
			if !ok {
				t.Fatalf("KindSiteUpdate payload is %T, want trunking.SiteUpdate", ev.Payload)
			}
			if u.System != "MMR" || u.RFSSID != 4 || u.SiteID != 7 {
				t.Fatalf("site update identity wrong: %+v", u)
			}
			if u.ControlChannelHz != ccHz {
				t.Fatalf("control_channel_hz = %d, want %d", u.ControlChannelHz, ccHz)
			}
			// WACN comes from the NSB; both the NSB and the RFSS Status
			// Broadcast carry SystemID 0x123 for these payloads.
			if u.WACN != 0xABCDE || u.SystemID != 0x123 {
				t.Fatalf("site update network ids wrong: %+v", u)
			}
			return
		case <-deadline:
			t.Fatal("no KindSiteUpdate published within deadline")
		}
	}
}
