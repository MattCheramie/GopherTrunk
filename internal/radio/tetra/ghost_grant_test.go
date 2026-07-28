package tetra

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// macResourceGrant builds a MAC-RESOURCE type-1 block addressed by SSI, carrying
// a channel-allocation element (carrier/timeslot) and the given TM-SDU (an LLC
// PDU) — the exact layout ParseMACResource reads (EN 300 392-2 §21.4.3.1 /
// §21.5.2). ULDL=1 and monitoring-pattern=1 keep parseChanAlloc out of its
// augmented / frame-18 branches so the TM-SDU begins right after the element.
func macResourceGrant(ssi uint32, carrier uint16, timeslot uint8, tmsdu []byte) []byte {
	w := &cmceBitWriter{}
	w.u(uint64(MACPDUResource), 2) // MAC PDU type
	w.u(0, 1)                      // fill bit
	w.u(0, 1)                      // grant position
	w.u(0, 2)                      // encryption mode (0 = clear)
	w.u(0, 1)                      // random access
	w.u(1, 6)                      // length indication (not 0x3f/0x3e)
	w.u(uint64(addrSSI), 3)        // address type = SSI
	w.u(uint64(ssi), 24)           // SSI
	w.u(0, 1)                      // power control absent
	w.u(0, 1)                      // slot granting absent
	w.u(1, 1)                      // channel allocation present
	w.u(0, 2)                      // allocation type
	w.u(uint64(timeslot), 4)       // assigned timeslot
	w.u(1, 2)                      // up/downlink (nonzero → skip augmented branch)
	w.u(0, 1)                      // CLCH permission
	w.u(0, 1)                      // cell change
	w.u(uint64(carrier), 12)       // main carrier number
	w.u(0, 1)                      // extended carrier absent
	w.u(1, 2)                      // monitoring pattern (nonzero → no frame-18 field)
	return append(w.bits, tmsdu...)
}

// cmceReleaseTMSDU / cmceSetupTMSDU wrap a CMCE PDU in a BL-UDATA LLC PDU — the
// TM-SDU shape a real MAC-RESOURCE delivers.
func cmceReleaseTMSDU(callID uint16, cause uint8) []byte {
	c := &cmceBitWriter{}
	c.header(CMCETypeDRelease)
	c.u(uint64(callID), 14)
	c.u(uint64(cause), 5)
	return append(bl(llcBLUDATA, 0), c.bits...)
}

func cmceSetupTMSDU(callID uint16) []byte {
	c := &cmceBitWriter{}
	c.header(CMCETypeDSetup)
	c.u(uint64(callID), 14)
	c.u(0, 4+1+1+8+2+1) // mandatory fields through transmission-request permission
	c.u(1, 4)           // call priority (non-emergency)
	c.u(0, 1)           // O-bit clear (no optional elements)
	return append(bl(llcBLUDATA, 0), c.bits...)
}

func drainKinds(sub *events.Subscription) map[events.Kind]int {
	counts := map[events.Kind]int{}
	for {
		select {
		case ev := <-sub.C:
			counts[ev.Kind]++
		default:
			return counts
		}
	}
}

// TestIngestMACSuppressesGrantOnRelease is the ghost-call regression. A single
// MAC-RESOURCE that carries both a channel-allocation element and a CMCE
// D-RELEASE must NOT publish a voice grant (which spawns a zero-byte ghost call)
// — only the release. Before the teardown gate, ingestMAC published a grant for
// every ChanAlloc regardless of the CMCE type.
func TestIngestMACSuppressesGrantOnRelease(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "Sys", FrequencyHz: 467_913_000})
	block := macResourceGrant(0x0F5670, 2716, 1, cmceReleaseTMSDU(0x1234, 13))
	cc.ingestMAC(block)

	counts := drainKinds(sub)
	if counts[events.KindGrant] != 0 {
		t.Errorf("published %d grant(s) for a D-RELEASE-bearing MAC-RESOURCE, want 0 (ghost call)", counts[events.KindGrant])
	}
	if counts[events.KindCallRelease] == 0 {
		t.Error("did not publish the D-RELEASE")
	}
}

// TestIngestMACPublishesGrantOnSetup guards that the teardown gate is
// teardown-only: a MAC-RESOURCE carrying a channel allocation and a CMCE D-SETUP
// still publishes a voice grant.
func TestIngestMACPublishesGrantOnSetup(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "Sys", FrequencyHz: 467_913_000})
	block := macResourceGrant(0x0F5670, 2716, 1, cmceSetupTMSDU(0x1234))
	cc.ingestMAC(block)

	var grant *trunking.Grant
	for drained := false; !drained; {
		select {
		case ev := <-sub.C:
			if g, ok := ev.Payload.(trunking.Grant); ok && ev.Kind == events.KindGrant {
				gg := g
				grant = &gg
			}
		default:
			drained = true
		}
	}
	if grant == nil {
		t.Fatal("no grant published for a D-SETUP-bearing MAC-RESOURCE")
	}
	if grant.GroupID != 0x0F5670 {
		t.Errorf("grant GroupID = %#x, want 0x0F5670", grant.GroupID)
	}
}
