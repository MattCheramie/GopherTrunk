package tetra

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestParseDAttachDetachGroupIdentityLiteral pins the D-ATTACH/DETACH GROUP
// IDENTITY layout with a LITERAL bit vector hand-assembled from the field
// order tetra-kit implements (mm.cc parseDAttachDetachGroupIdentity +
// mm_elements.cc parseType34Elements / parseGroupIdentityDownlink) — not from
// this package's own parser (the #764/#771 self-consistent trap). It also
// proves an unmodelled type-3/4 element is skipped by its own length
// indicator without desyncing the walk.
func TestParseDAttachDetachGroupIdentityLiteral(t *testing.T) {
	pdu := "001" + "1010" + // PD=MM, type=D-ATTACH/DETACH GROUP IDENTITY
		"1" + "0" + "0" + // report request=1, ack requested=0, mode=amendment
		"1" + // O-bit
		// Unknown element (id 0011 security downlink), 10-bit body — must be
		// skipped by its length indicator.
		"1" + "0011" + "00000001010" + "1111000011" +
		// Group identity downlink element (id 0111), value = count(6) + 2 elements.
		"1" + "0111" + "00001011011" + // LI = 6 + 32 + 53 = 91
		"000010" + // number of elements = 2
		// Element A — attachment: lifetime=2, class of usage=3, addr type 00,
		// GSSI = 1020529 (0x0F9271).
		"0" + "10" + "011" + "00" + "000011111001001001110001" +
		// Element B — detachment: reason=1, addr type 01 (GSSI + extension),
		// GSSI = 1020530 (0x0F9272), MCC=250, MNC=13.
		"1" + "01" + "01" + "000011111001001001110010" + "0011111010" + "00000000001101" +
		"0" // M-bit: no more elements

	ad, ok := ParseDAttachDetachGroupIdentity(bitsOf(t, pdu))
	if !ok {
		t.Fatal("ParseDAttachDetachGroupIdentity returned ok=false on a valid PDU")
	}
	if !ad.ReportRequest || ad.AckRequested || ad.DetachAllAttach {
		t.Errorf("header flags = %+v, want report=true ack=false mode=amendment", ad)
	}
	if len(ad.Groups) != 2 {
		t.Fatalf("decoded %d group elements, want 2", len(ad.Groups))
	}
	a := ad.Groups[0]
	if a.Detach || a.Lifetime != 2 || a.ClassOfUsage != 3 || a.GSSI != 1020529 ||
		a.HasExtension || a.HasVGSSI {
		t.Errorf("attach element = %+v, want lifetime=2 class=3 gssi=1020529 no ext", a)
	}
	b := ad.Groups[1]
	if !b.Detach || b.DetachReason != 1 || b.GSSI != 1020530 ||
		!b.HasExtension || b.MCC != 250 || b.MNC != 13 || b.HasVGSSI {
		t.Errorf("detach element = %+v, want reason=1 gssi=1020530 mcc=250 mnc=13", b)
	}
}

// TestParseMMPDUTypeRejectsNonMM: the 3-bit protocol discriminator gates MM
// parsing — a CMCE or MLE TL-SDU must not read as an MM PDU.
func TestParseMMPDUTypeRejectsNonMM(t *testing.T) {
	if _, ok := ParseMMPDUType(bitsOf(t, "010"+"1010"+"0000")); ok {
		t.Error("ParseMMPDUType accepted a CMCE TL-SDU")
	}
	if _, ok := ParseDAttachDetachGroupIdentity(bitsOf(t, "001"+"0101"+"000000")); ok {
		t.Error("ParseDAttachDetachGroupIdentity accepted a D-LOCATION-UPDATE-ACCEPT")
	}
	if got, ok := ParseMMPDUType(bitsOf(t, "001"+"0101"+"0")); !ok || got != MMDLocationUpdateAccept {
		t.Errorf("ParseMMPDUType = %v/%v, want D-LOCATION-UPDATE-ACCEPT/true", got, ok)
	}
}

// TestHandleMMPublishesAffiliation: a decoded group attachment publishes a
// KindAffiliation (ISSI→GSSI, protocol tetra) — the TETRA analogue of the P25
// Group Affiliation Response feed — while a detachment only logs.
func TestHandleMMPublishesAffiliation(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "Sys", FrequencyHz: 467_912_500})

	pdu := "001" + "1010" + "0" + "0" + "0" + "1" +
		"1" + "0111" + "00000100110" + // LI = 6 + 32 = 38
		"000001" + // one element
		"0" + "01" + "010" + "00" + "000011111001001001110001" + // attach GSSI 1020529
		"0"
	cc.handleMM(MACResource{Address: MACAddress{SSI: 1005611}}, bitsOf(t, pdu))

	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindAffiliation {
			t.Fatalf("published %s, want affiliation", ev.Kind)
		}
		aff, ok := ev.Payload.(trunking.Affiliation)
		if !ok {
			t.Fatalf("payload is %T, want trunking.Affiliation", ev.Payload)
		}
		if aff.System != "Sys" || aff.Protocol != "tetra" ||
			aff.SourceID != 1005611 || aff.GroupID != 1020529 ||
			aff.Response != trunking.AffiliationAccepted {
			t.Errorf("affiliation = %+v, want Sys/tetra issi=1005611 gssi=1020529 accepted", aff)
		}
	default:
		t.Fatal("no KindAffiliation event published for a group attachment")
	}

	// A detachment logs but publishes no affiliation.
	det := "001" + "1010" + "0" + "0" + "0" + "1" +
		"1" + "0111" + "00000100011" + // LI = 6 + 29 = 35
		"000001" +
		"1" + "00" + "00" + "000011111001001001110001" + // detach GSSI 1020529
		"0"
	cc.handleMM(MACResource{Address: MACAddress{SSI: 1005611}}, bitsOf(t, det))
	select {
	case ev := <-sub.C:
		t.Fatalf("detachment published %s, want no event", ev.Kind)
	default:
	}
}
