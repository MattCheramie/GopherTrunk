package api

import (
	"encoding/json"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestEventToDTOAffiliationJSON pins the wire shape of the affiliation
// event so downstream consumers (Grafana, dashboards) don't break on a
// silent field-name change.
func TestEventToDTOAffiliationJSON(t *testing.T) {
	dto := eventToDTO(events.Event{
		Kind: events.KindAffiliation,
		Payload: trunking.Affiliation{
			System:            "MMR",
			Protocol:          "p25",
			SourceID:          0xABCDEF,
			GroupID:           0x1234,
			AnnouncementGroup: 0xAABB,
			Response:          trunking.AffiliationAccepted,
		},
	})
	body, err := json.Marshal(dto.Payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(body)
	want := `{"system":"MMR","protocol":"p25","source_id":11259375,"group_id":4660,"announcement_group":43707,"response":"accepted"}`
	if got != want {
		t.Errorf("affiliation JSON =\n  %s\nwant\n  %s", got, want)
	}
}

// TestEventToDTOUnitRegistrationJSON pins the wire shape of the
// registration event for the same reason.
func TestEventToDTOUnitRegistrationJSON(t *testing.T) {
	dto := eventToDTO(events.Event{
		Kind: events.KindUnitRegistration,
		Payload: trunking.UnitRegistration{
			System:   "MMR",
			Protocol: "p25",
			SourceID: 0x112233,
			WACN:     0xBEE08,
			SystemID: 0x534,
			Response: trunking.RegistrationAccepted,
		},
	})
	body, err := json.Marshal(dto.Payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(body)
	want := `{"system":"MMR","protocol":"p25","source_id":1122867,"wacn":781832,"system_id":1332,"response":"accepted"}`
	if got != want {
		t.Errorf("registration JSON =\n  %s\nwant\n  %s", got, want)
	}
}

// TestEventToDTOPatchJSON pins the wire shape of the patch event. The
// values mirror the report in issue #374 so this test doubles as the
// regression record for "CC Activity always shows super-group 0".
func TestEventToDTOPatchJSON(t *testing.T) {
	dto := eventToDTO(events.Event{
		Kind: events.KindPatch,
		Payload: trunking.Patch{
			System:     "MMR",
			Protocol:   "p25",
			SuperGroup: 32301,
			Members:    []uint32{32501},
			Vendor:     "motorola",
			Add:        true,
		},
	})
	body, err := json.Marshal(dto.Payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(body)
	want := `{"system":"MMR","protocol":"p25","super_group":32301,"members":[32501],"vendor":"motorola","add":true,"at":"0001-01-01T00:00:00Z"}`
	if got != want {
		t.Errorf("patch JSON =\n  %s\nwant\n  %s", got, want)
	}
}

// TestEventToDTODMRGrantObservedJSON pins the wire shape of the DMR
// grant-observed event so the web CC Activity panel reads stable snake_case
// fields rather than the raw PascalCase passthrough. (#638)
func TestEventToDTODMRGrantObservedJSON(t *testing.T) {
	dto := eventToDTO(events.Event{
		Kind: events.KindDMRGrantObserved,
		Payload: events.DMRGrantObserved{
			System:    "County T3",
			ColorCode: 1,
			LCN:       12,
			Timeslot:  1,
			GroupID:   1001,
			SourceID:  2002,
			CCFreqHz:  851_000_000,
		},
	})
	body, err := json.Marshal(dto.Payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(body)
	want := `{"system":"County T3","color_code":1,"lcn":12,"timeslot":1,"group_id":1001,"source_id":2002,"cc_freq_hz":851000000,"at":"0001-01-01T00:00:00Z"}`
	if got != want {
		t.Errorf("dmr grant observed JSON =\n  %s\nwant\n  %s", got, want)
	}
}

// TestEventToDTODMRBandPlanLearnedJSON pins the wire shape of the learned
// band-plan event (linear plan) the Systems / CC Activity panels render. (#638)
func TestEventToDTODMRBandPlanLearnedJSON(t *testing.T) {
	dto := eventToDTO(events.Event{
		Kind: events.KindDMRBandPlanLearned,
		Payload: events.DMRBandPlanLearned{
			System:     "County T3",
			BaseHz:     851_012_500,
			SpacingHz:  12_500,
			Offset:     1,
			NumPairs:   6,
			Confidence: 0.97,
			ResidualHz: 25,
		},
	})
	body, err := json.Marshal(dto.Payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(body)
	want := `{"system":"County T3","base_hz":851012500,"spacing_hz":12500,"offset":1,"num_pairs":6,"confidence":0.97,"residual_hz":25}`
	if got != want {
		t.Errorf("dmr band plan learned JSON =\n  %s\nwant\n  %s", got, want)
	}
}

// TestSystemToDTODMRBandPlan checks the active DMR band plan is surfaced on
// the systems API (linear and table forms) and omitted when absent. (#638)
func TestSystemToDTODMRBandPlan(t *testing.T) {
	t.Run("linear", func(t *testing.T) {
		dto := systemToDTO(trunking.System{
			Name:     "T3",
			Protocol: trunking.ProtocolDMR,
			DMRBandPlan: &trunking.DMRBandPlan{
				Linear: &trunking.DMRLinearBandPlan{BaseHz: 851_012_500, SpacingHz: 12_500, Offset: 1},
			},
		})
		if dto.DMRBandPlan == nil || dto.DMRBandPlan.Linear == nil {
			t.Fatalf("expected a linear band plan, got %+v", dto.DMRBandPlan)
		}
		if dto.DMRBandPlan.Linear.BaseHz != 851_012_500 || dto.DMRBandPlan.Linear.SpacingHz != 12_500 {
			t.Errorf("linear plan = %+v", dto.DMRBandPlan.Linear)
		}
	})
	t.Run("table", func(t *testing.T) {
		dto := systemToDTO(trunking.System{
			Name:     "T3",
			Protocol: trunking.ProtocolDMR,
			DMRBandPlan: &trunking.DMRBandPlan{
				Table: []trunking.DMRBandPlanTableEntry{{LCN: 1, FreqHz: 851_000_000}, {LCN: 2, FreqHz: 851_025_000}},
			},
		})
		if dto.DMRBandPlan == nil || len(dto.DMRBandPlan.Table) != 2 {
			t.Fatalf("expected a 2-entry table plan, got %+v", dto.DMRBandPlan)
		}
	})
	t.Run("absent", func(t *testing.T) {
		dto := systemToDTO(trunking.System{Name: "P25", Protocol: trunking.ProtocolP25})
		if dto.DMRBandPlan != nil {
			t.Errorf("expected nil band plan, got %+v", dto.DMRBandPlan)
		}
	})
}
