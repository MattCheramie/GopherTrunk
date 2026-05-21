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
