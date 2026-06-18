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
	// Neighbours need corroborationMin (2) sightings before Snapshot surfaces
	// them, so each site is broadcast twice.
	m.ApplyAdjacentSite(AdjacentSiteStatusBroadcast{RFSS: 4, Site: 8, ChannelID: 1, ChannelNumber: 300})
	m.ApplyAdjacentSite(AdjacentSiteStatusBroadcast{RFSS: 4, Site: 9, ChannelID: 1, ChannelNumber: 301})
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

	// The adjacent-site broadcast is sent twice so it clears the
	// corroborationMin (2) gate and Snapshot surfaces the neighbour.
	base := 0
	for _, tsbk := range []TSBK{nsb, rfss, adj, adj} {
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
