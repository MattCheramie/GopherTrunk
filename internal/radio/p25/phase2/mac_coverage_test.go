package phase2

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func TestAsGroupAffiliationResponse(t *testing.T) {
	in := GroupAffiliationResponse{
		Response: 2, AnnouncementGroup: 0x00FF, GroupAddress: 0x1234, TargetID: 0x00BEEF,
	}
	got, ok := EncodeGroupAffiliationResponse(in).AsGroupAffiliationResponse()
	if !ok {
		t.Fatal("AsGroupAffiliationResponse returned !ok")
	}
	if got != in {
		t.Errorf("round-trip = %+v, want %+v", got, in)
	}
}

func TestAsUnitRegistrationResponse(t *testing.T) {
	in := UnitRegistrationResponse{
		Response: 1, WACN: 0xABCDE, SystemID: 0x123, SourceID: 0x00BEEF,
	}
	got, ok := EncodeUnitRegistrationResponse(in).AsUnitRegistrationResponse()
	if !ok {
		t.Fatal("AsUnitRegistrationResponse returned !ok")
	}
	if got != in {
		t.Errorf("round-trip = %+v, want %+v", got, in)
	}
}

func TestAsUnitToUnitGrantAcceptsUpdateOpcode(t *testing.T) {
	pdu := u2uPDU(0x00BEEF, 0x00ABCD, 0x2, 0x123)
	pdu.Opcode = OpUnitToUnitVoiceChannelGrantUpdate
	if _, ok := pdu.AsUnitToUnitVoiceChannelGrant(); !ok {
		t.Error("AsUnitToUnitVoiceChannelGrant rejected the grant-update opcode")
	}
}

func TestAsMotorolaPatchDelete(t *testing.T) {
	pdu := MACPDU{Opcode: OpMotorolaPatchDelete, MFID: MFIDMotorola, Payload: []byte{0x12, 0x34}}
	super, ok := pdu.AsMotorolaPatchDelete()
	if !ok || super != 0x1234 {
		t.Errorf("AsMotorolaPatchDelete = (%#x, %v), want (0x1234, true)", super, ok)
	}
	// Wrong MFID must not match.
	pdu.MFID = MFIDHarris
	if _, ok := pdu.AsMotorolaPatchDelete(); ok {
		t.Error("AsMotorolaPatchDelete matched a non-Motorola MFID")
	}
}

func TestNewOpcodesAreKnown(t *testing.T) {
	for _, o := range []Opcode{
		OpUnitToUnitVoiceChannelGrantUpdate, OpGroupAffiliationResponse,
		OpUnitRegistrationResponse, OpMotorolaPatchDelete,
	} {
		if !o.IsKnown() {
			t.Errorf("opcode %#x should be known", uint8(o))
		}
	}
}

// TestControlChannelEmitsAffiliationAndRegistration confirms the Phase 2
// control channel publishes the identity events Phase 1 already does.
func TestControlChannelEmitsAffiliationAndRegistration(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "p2", FrequencyHz: 851_000_000})
	cc.Ingest(EncodeGroupAffiliationResponse(GroupAffiliationResponse{
		Response: 0, GroupAddress: 0x1234, TargetID: 0x00ABCD,
	}))
	cc.Ingest(EncodeUnitRegistrationResponse(UnitRegistrationResponse{
		Response: 0, WACN: 0xABCDE, SystemID: 0x123, SourceID: 0x00BEEF,
	}))

	var sawAff, sawReg bool
	for {
		select {
		case ev := <-sub.C:
			switch ev.Kind {
			case events.KindAffiliation:
				if a := ev.Payload.(trunking.Affiliation); a.GroupID == 0x1234 && a.SourceID == 0x00ABCD {
					sawAff = true
				}
			case events.KindUnitRegistration:
				if r := ev.Payload.(trunking.UnitRegistration); r.SourceID == 0x00BEEF && r.WACN == 0xABCDE {
					sawReg = true
				}
			}
		default:
			if !sawAff {
				t.Error("no KindAffiliation event")
			}
			if !sawReg {
				t.Error("no KindUnitRegistration event")
			}
			return
		}
	}
}

// TestControlChannelStampsSiteIdentity confirms that once the Phase 2
// control channel has decoded a Network Status Broadcast (WACN / System
// ID / Color Code) and an RFSS Status Broadcast (RFSS / Site), it (a)
// publishes a KindSiteUpdate so the site surfaces in GET /api/v1/sites,
// and (b) stamps rfss_id / site_id / nac onto the grant, affiliation,
// and registration events (issue #698 addendum).
func TestControlChannelStampsSiteIdentity(t *testing.T) {
	bus := events.NewBus(32)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "p2", FrequencyHz: 851_000_000})

	// NSB: LRA=0x07, WACN=0xABCDE, SystemID=0x123, ColorCode(=NAC)=0x293.
	cc.Ingest(MACPDU{Opcode: OpNetworkStatusBroadcastUpdate,
		Payload: []byte{0x07, 0xAB, 0xCD, 0xE1, 0x23, 0x29, 0x30, 0x00, 0x00}})
	// RFSS Status: RFSS=5, Site=9.
	cc.Ingest(MACPDU{Opcode: OpRFSSStatusBroadcastUpdate,
		Payload: []byte{0x07, 0x01, 0x23, 0x05, 0x09, 0x00, 0x00}})

	cc.Ingest(grantPDU(0x1234, 0x00ABCD, 0x1, 0x002))
	cc.Ingest(EncodeGroupAffiliationResponse(GroupAffiliationResponse{
		Response: 0, GroupAddress: 0x1234, TargetID: 0x00ABCD,
	}))
	cc.Ingest(EncodeUnitRegistrationResponse(UnitRegistrationResponse{
		Response: 0, WACN: 0xABCDE, SystemID: 0x123, SourceID: 0x00BEEF,
	}))

	var sawSite, sawGrant, sawAff, sawReg bool
	for {
		select {
		case ev := <-sub.C:
			switch ev.Kind {
			case events.KindSiteUpdate:
				s := ev.Payload.(trunking.SiteUpdate)
				if s.RFSSID == 5 && s.SiteID == 9 && s.WACN == 0xABCDE &&
					s.SystemID == 0x123 && s.ControlChannelHz == 851_000_000 {
					sawSite = true
				}
			case events.KindGrant:
				g := ev.Payload.(trunking.Grant)
				if g.RFSSID == 5 && g.SiteID == 9 && g.NAC == 0x293 {
					sawGrant = true
				}
			case events.KindAffiliation:
				a := ev.Payload.(trunking.Affiliation)
				if a.RFSSID == 5 && a.SiteID == 9 && a.NAC == 0x293 {
					sawAff = true
				}
			case events.KindUnitRegistration:
				r := ev.Payload.(trunking.UnitRegistration)
				if r.RFSSID == 5 && r.SiteID == 9 && r.NAC == 0x293 {
					sawReg = true
				}
			}
		default:
			if !sawSite {
				t.Error("no KindSiteUpdate event carrying the decoded site")
			}
			if !sawGrant {
				t.Error("grant did not carry rfss_id/site_id/nac")
			}
			if !sawAff {
				t.Error("affiliation did not carry rfss_id/site_id/nac")
			}
			if !sawReg {
				t.Error("registration did not carry rfss_id/site_id/nac")
			}
			return
		}
	}
}

// TestControlChannelPublishesNSBOnlySiteUpdate confirms that a Network
// Status Broadcast alone — with no RFSS Status Broadcast — publishes a
// KindSiteUpdate carrying the decoded WACN / System ID. On Phase 2 the
// WACN and System ID come ONLY from the NSB, so a system that emits the
// NSB but seldom/never emits an RFSS Status Broadcast must still surface
// its identity the instant the NSB lands (mirroring Phase 1). Regression
// for the propagation defect where the NSB branch stored netWACN/netSysID
// but never published, and publishSiteUpdate gated on RFSS/Site != 0.
func TestControlChannelPublishesNSBOnlySiteUpdate(t *testing.T) {
	bus := events.NewBus(32)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "p2", FrequencyHz: 851_000_000})

	// NSB only — WACN=0xABCDE, SystemID=0x123, ColorCode=0x293. No RFSS_STS.
	cc.Ingest(MACPDU{Opcode: OpNetworkStatusBroadcastUpdate,
		Payload: []byte{0x07, 0xAB, 0xCD, 0xE1, 0x23, 0x29, 0x30, 0x00, 0x00}})

	var sawWACN bool
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindSiteUpdate {
				s := ev.Payload.(trunking.SiteUpdate)
				if s.WACN == 0xABCDE && s.SystemID == 0x123 &&
					s.ControlChannelHz == 851_000_000 {
					sawWACN = true
				}
			}
		default:
			if !sawWACN {
				t.Fatal("NSB-only ingest published no KindSiteUpdate carrying WACN/SystemID")
			}
			return
		}
	}
}

func TestControlChannelPatchDelete(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "p2", FrequencyHz: 851_000_000})
	cc.Ingest(MACPDU{Opcode: OpMotorolaPatchDelete, MFID: MFIDMotorola, Payload: []byte{0x05, 0x55}})

	p := nextPatch(t, sub)
	if p.SuperGroup != 0x0555 || p.Add {
		t.Errorf("patch delete = %+v, want super 0x555 / Add false", p)
	}
}
