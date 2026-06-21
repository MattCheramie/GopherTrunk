package phase1

import (
	"log/slog"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestControlChannelTopologySnapshot drives identity / band-plan / secondary /
// neighbour state into a control channel and checks TopologySnapshot maps it,
// resolving each channel's downlink frequency through the band plan. This is
// the single builder both the siglab engine and the live ccdecoder pipeline use.
func TestControlChannelTopologySnapshot(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Main", FrequencyHz: 851_100_000})
	cc.lastNAC = 0x2C1
	cc.bandPlan.Apply(IdentifierUpdate{ChannelID: 1, BaseHz: 851_000_000, SpacingHz: 12_500, BandwidthHz: 12_500, TxOffsetHz: -45_000_000})
	cc.netModel.ApplyNetworkStatus(NetworkStatusBroadcast{WACN: 0xBEE99, SystemID: 0x49A, ChannelID: 1, ChannelNumber: 8})
	cc.netModel.ApplyRFSSStatus(RFSSStatusBroadcast{SystemID: 0x49A, RFSS: 1, Site: 2, LRA: 0x0C, ChannelID: 1, ChannelNumber: 8})
	cc.netModel.ApplySecondaryControlChannel(SecondaryControlChannelBroadcast{ChannelAID: 1, ChannelANumber: 100})
	cc.netModel.ApplyAdjacentSite(AdjacentSiteStatusBroadcast{RFSS: 1, Site: 3, ChannelID: 1, ChannelNumber: 200})

	snap := cc.TopologySnapshot()
	if snap == nil {
		t.Fatal("TopologySnapshot returned nil for a populated control channel")
	}
	if snap.Protocol != "p25" || snap.SystemName != "Main" || snap.NAC != 0x2C1 {
		t.Errorf("metadata = proto %q name %q nac %X", snap.Protocol, snap.SystemName, snap.NAC)
	}
	if snap.WACN != 0xBEE99 || snap.SystemID != 0x49A || snap.RFSS != 1 || snap.Site != 2 || snap.LRA != 0x0C {
		t.Errorf("identity = %+v", snap)
	}
	// Primary CC 1-8 resolves to 851_000_000 + 8*12_500 = 851_100_000.
	if snap.PrimaryCC == nil || snap.PrimaryCC.FrequencyHz != 851_100_000 {
		t.Errorf("PrimaryCC = %+v, want freq 851100000", snap.PrimaryCC)
	}
	// Secondary 1-100 → 852_250_000; neighbour 1-200 → 853_500_000.
	if len(snap.Secondary) != 1 || snap.Secondary[0].FrequencyHz != 852_250_000 {
		t.Errorf("Secondary = %+v", snap.Secondary)
	}
	if len(snap.Neighbors) != 1 || snap.Neighbors[0].FrequencyHz != 853_500_000 {
		t.Errorf("Neighbors = %+v", snap.Neighbors)
	}
	if len(snap.BandPlan) != 1 || snap.BandPlan[0].BaseHz != 851_000_000 {
		t.Errorf("BandPlan = %+v", snap.BandPlan)
	}
}

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

func TestNetworkModelPrimaryControlChannel(t *testing.T) {
	var m NetworkModel
	// The camped site advertises its primary CC (2-1620) in repeated RFSS/
	// Network status broadcasts; a single corrupt-but-CRC-passing frame names a
	// different channel and must not win the majority vote.
	for i := 0; i < 3; i++ {
		m.ApplyRFSSStatus(RFSSStatusBroadcast{SystemID: 0x2C2, RFSS: 1, Site: 1, ChannelID: 2, ChannelNumber: 1620})
		m.ApplyNetworkStatus(NetworkStatusBroadcast{WACN: 0xBEE00, SystemID: 0x2C2, ChannelID: 2, ChannelNumber: 1620})
	}
	m.ApplyRFSSStatus(RFSSStatusBroadcast{SystemID: 0x2C2, RFSS: 1, Site: 1, ChannelID: 7, ChannelNumber: 9})

	cfg := m.Snapshot()
	if cfg.PrimaryCC != (Channel{ChannelID: 2, ChannelNumber: 1620}) {
		t.Errorf("PrimaryCC = %+v, want {2 1620}", cfg.PrimaryCC)
	}
}

func TestNetworkModelPrimaryControlChannelZeroIgnored(t *testing.T) {
	var m NetworkModel
	// A 0-0 channel is the "no channel" sentinel and must never be voted, so an
	// unseen primary CC stays the zero Channel.
	m.ApplyRFSSStatus(RFSSStatusBroadcast{SystemID: 0x2C2, RFSS: 1, Site: 1, ChannelID: 0, ChannelNumber: 0})
	if cc := m.Snapshot().PrimaryCC; cc != (Channel{}) {
		t.Errorf("PrimaryCC = %+v, want zero", cc)
	}
}

func TestNetworkModelSecondaryControlChannelExplicit(t *testing.T) {
	var m NetworkModel
	// The explicit SCCB (0x29) records only its transmit (downlink)
	// channel — the one a receiver tunes to.
	m.ApplySecondaryControlChannelExplicit(SecondaryControlChannelBroadcastExplicit{
		TxChannelID: 1, TxChannelNumber: 100,
		RxChannelID: 1, RxChannelNumber: 200,
	})
	cfg := m.Snapshot()
	if len(cfg.Secondary) != 1 {
		t.Fatalf("Secondary = %v, want 1 (downlink only)", cfg.Secondary)
	}
	if cfg.Secondary[0] != (Channel{1, 100}) {
		t.Errorf("Secondary[0] = %+v, want {1 100}", cfg.Secondary[0])
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
	// Explicit Secondary Control Channel Broadcast (0x29): its transmit
	// (downlink) channel must surface in the topology's secondary list.
	sccbExp := TSBK{Opcode: OpSecondaryControlChannelExpl,
		Payload: AssembleSecondaryControlChannelBroadcastExplicit(SecondaryControlChannelBroadcastExplicit{
			RFSS: 4, Site: 7, TxChannelID: 1, TxChannelNumber: 0x123, RxChannelID: 1, RxChannelNumber: 0x456})}

	base := 0
	for _, tsbk := range []TSBK{nsb, rfss, adj, sccbExp} {
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
	if len(cfg.Secondary) != 1 || cfg.Secondary[0] != (Channel{1, 0x123}) {
		t.Errorf("secondary = %v, want one downlink {1 0x123} from SCCB_EXP", cfg.Secondary)
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

// TestMotorolaProbeOpcodesAreMapSafe drives the named-but-not-field-decoded
// Motorola opcodes Traffic Channel ID (0x05) and System Loading (0x09), MFID
// 0x90, through the control channel and checks they inject NOTHING into the
// system map — no bus event and no neighbour / secondary-CC in the topology
// snapshot. Neither reference decoder (SDRtrunk / OP25) field-decodes them, so
// they are captured for the record only (logVendorProbe); this guards against
// anyone wiring a guessed layout into the map.
func TestMotorolaProbeOpcodesAreMapSafe(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	cc := New(Options{Bus: bus, SystemName: "Moto"})

	for _, op := range []Opcode{OpMotorolaTrafficChannelID, OpMotorolaSystemLoading} {
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
