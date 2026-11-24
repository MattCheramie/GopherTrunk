package phase1

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// A hybrid site's Phase 1 control channel stamps each Phase 2 grant with the
// traffic channel's PN44 descramble seed, built from the WACN and System ID of
// its Network Status Broadcast and the NAC. Before that broadcast has been
// decoded it used to build the seed from zeros: a sequence no site sends,
// which the voice chain took for a real seed, so every call granted in the
// first seconds after lock decoded nothing (issue #1364). Such a grant now
// carries no seed, the next one carries the right one, and the site update the
// broadcast triggers carries the NAC the voice chain needs to finish the seed.
func TestControlChannelPhase2GrantSeedWaitsForNetworkStatus(t *testing.T) {
	const (
		nac         = 0x293
		wacn, sysID = 0xBEE00, 0x164
	)
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	identTSBK := TSBK{LB: false, Opcode: OpIdentifierUpdateTDMA}
	identTSBK.Payload = AssembleIdentifierUpdateTDMA(IdentifierUpdate{
		ChannelID: 1, BandwidthHz: 12_500, SpacingHz: 12_500, BaseHz: 851_000_000, AccessTDMA: true,
	})
	grantTSBK := TSBK{LB: true, Opcode: OpGroupVoiceChannelGrant, Payload: [8]byte{
		0x00, (1 << 4) | 0x00, 0x10, 0x12, 0x34, 0xAB, 0xCD, 0xEF,
	}}
	// LRA 1, WACN BEE00, System ID 164, channel 1-0x010, service class 0x70.
	nsbTSBK := TSBK{LB: true, Opcode: OpNetworkStatusBroadcast, Payload: [8]byte{
		0x01, 0xBE, 0xE0, 0x01, 0x64, 0x10, 0x10, 0x70,
	}}
	if n := ParseNetworkStatusBroadcast(nsbTSBK.Payload); n.WACN != wacn || n.SystemID != sysID {
		t.Fatalf("test NSB parses as WACN %#x System ID %#x", n.WACN, n.SystemID)
	}

	cc := New(Options{
		Bus: bus, SystemName: "MMR", FrequencyHz: 851_000_000,
		Now: func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
	})
	var stream []uint8
	feed := func(t TSBK, lead int) {
		s := buildLockedStreamWithTSBK(lead, nac, DUIDTrunkingSignaling, t)
		cc.Process(s, len(stream))
		stream = append(stream, s...)
	}
	nextGrant := func() trunking.Grant {
		deadline := time.After(time.Second)
		for {
			select {
			case ev := <-sub.C:
				if g, ok := ev.Payload.(trunking.Grant); ok && ev.Kind == events.KindGrant {
					return g
				}
			case <-deadline:
				t.Fatal("no grant event published")
			}
		}
	}

	feed(identTSBK, 10)
	feed(grantTSBK, 0)
	if g := nextGrant(); g.Protocol != "p25-phase2" || g.P25Phase2Decode.Seed != 0 {
		t.Fatalf("grant before the Network Status Broadcast: protocol %q seed %#x, want p25-phase2 with no seed",
			g.Protocol, g.P25Phase2Decode.Seed)
	}

	feed(nsbTSBK, 0)
	var nacSeen uint16
	deadline := time.After(time.Second)
	for nacSeen == 0 {
		select {
		case ev := <-sub.C:
			if su, ok := ev.Payload.(trunking.SiteUpdate); ok && ev.Kind == events.KindSiteUpdate && su.WACN == wacn {
				nacSeen = su.NAC
			}
		case <-deadline:
			t.Fatal("no site update naming the WACN after the Network Status Broadcast")
		}
	}
	if nacSeen != nac {
		t.Errorf("site update NAC %#x, want %#x", nacSeen, nac)
	}

	// A fresh grant (another group: a repeat of the first is deduplicated).
	grantTSBK.Payload[3], grantTSBK.Payload[4] = 0x56, 0x78
	feed(grantTSBK, 0)
	want := framing.PN44SeedFromIdentity(wacn, sysID, nac)
	if g := nextGrant(); g.P25Phase2Decode.Seed != want {
		t.Fatalf("grant after the Network Status Broadcast: seed %#x, want %#x", g.P25Phase2Decode.Seed, want)
	}
}
