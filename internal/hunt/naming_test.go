package hunt

import (
	"bytes"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/survey"
)

func TestNameSignal_DecodedProtocol(t *testing.T) {
	ds := DetectedSignal{
		FreqHz:   851_012_500,
		Class:    survey.ClassTrunkControl,
		Trunking: &TrunkingRef{Protocol: "p25", Locked: true},
	}
	nameSignal(&ds)
	if ds.Name != "P25 Phase 1" {
		t.Errorf("Name = %q, want P25 Phase 1", ds.Name)
	}
	if !strings.Contains(ds.Service, "800 MHz") {
		t.Errorf("Service = %q, want the 800 MHz public-safety allocation", ds.Service)
	}
}

func TestNameSignal_UndecodedNamedByAllocation(t *testing.T) {
	// An analog carrier in the FRS/GMRS band: no decode, but the allocation
	// names the service and the modulation class is the fallback name.
	ds := DetectedSignal{FreqHz: 462_600_000, Class: survey.ClassNBFM, OccupiedBwHz: 12_500}
	nameSignal(&ds)
	if ds.Service != "FRS / GMRS" {
		t.Errorf("Service = %q, want FRS / GMRS", ds.Service)
	}
	if ds.Name == "" {
		t.Error("Name is empty; want at least the class fallback")
	}
}

func TestWidebandSignalRow(t *testing.T) {
	ds := widebandSignal(Candidate{FreqHz: 751_000_000, BwHz: 10_000_000, SNRDb: 30, IsWideband: true})
	if !ds.Wideband || ds.Class != survey.ClassWideband {
		t.Fatalf("row not marked wideband: %+v", ds)
	}
	if ds.OccupiedBwHz != 10_000_000 {
		t.Errorf("OccupiedBwHz = %d, want 10_000_000 (stitched width)", ds.OccupiedBwHz)
	}
	if !strings.Contains(ds.Name, "LTE") {
		t.Errorf("Name = %q, want an LTE/5G best guess", ds.Name)
	}
	if !strings.Contains(ds.Service, "Cellular 700") {
		t.Errorf("Service = %q, want the 700 MHz cellular allocation", ds.Service)
	}
	if ds.Trunking != nil {
		t.Error("a wideband row must not carry a trunking decode")
	}
}

func TestEncTypeName(t *testing.T) {
	if got := encTypeName("p25", 0x84); got != "AES-256" {
		t.Errorf("encTypeName(p25, 0x84) = %q, want AES-256", got)
	}
	if got := encTypeName("p25", 0); got != "" {
		t.Errorf("encTypeName(p25, 0) = %q, want empty (ALGID not surfaced)", got)
	}
	if got := encTypeName("dmr", 0x21); got != "RC4 (Enhanced Privacy)" {
		t.Errorf("encTypeName(dmr, 0x21) = %q, want RC4 (Enhanced Privacy)", got)
	}
}

func TestEncryptionFromGrants(t *testing.T) {
	enc, typ := encryptionFromGrants("p25", []siglab.GrantRecord{
		{Encrypted: false},
		{Encrypted: true, AlgorithmID: 0x84},
	})
	if !enc || typ != "AES-256" {
		t.Errorf("got (%v, %q), want (true, AES-256)", enc, typ)
	}
	// Encrypted but ALGID not surfaced ⇒ flagged, type empty.
	if enc, typ := encryptionFromGrants("p25", []siglab.GrantRecord{{Encrypted: true}}); !enc || typ != "" {
		t.Errorf("got (%v, %q), want (true, \"\")", enc, typ)
	}
	if enc, _ := encryptionFromGrants("p25", []siglab.GrantRecord{{Encrypted: false}}); enc {
		t.Error("clear grants reported encrypted")
	}
}

func TestSurveyCSVHasInventoryColumns(t *testing.T) {
	sv := &SignalSurvey{Signals: []DetectedSignal{
		{
			FreqHz: 851_012_500, Class: survey.ClassTrunkControl, OccupiedBwHz: 12_500,
			Name: "P25 Phase 1", Service: "800 MHz SMR/public safety", Purpose: "trunked land-mobile downlink",
			Encrypted: true, EncType: "AES-256",
			Trunking: &TrunkingRef{Protocol: "p25", Locked: true},
		},
		{
			FreqHz: 751_000_000, Class: survey.ClassWideband, OccupiedBwHz: 10_000_000,
			Name: "LTE/5G NR (10 MHz)", Service: "Cellular 700 MHz DL", Wideband: true,
		},
	}}
	var buf bytes.Buffer
	if err := WriteSurvey(&buf, sv, SurveyCSV); err != nil {
		t.Fatalf("WriteSurvey: %v", err)
	}
	out := buf.String()
	for _, col := range []string{"name", "service", "purpose", "encrypted", "enc_type", "wideband"} {
		if !strings.Contains(out, col) {
			t.Errorf("CSV header missing column %q", col)
		}
	}
	for _, want := range []string{"P25 Phase 1", "AES-256", "true", "LTE/5G NR (10 MHz)", "Cellular 700 MHz DL"} {
		if !strings.Contains(out, want) {
			t.Errorf("CSV missing value %q\n%s", want, out)
		}
	}
}
