package api

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestGrantTimeslotMapping verifies the DMR timeslot survives the
// trunking.Grant → JSON DTO and → protobuf conversions the API layer
// performs, so SSE/REST and gRPC consumers can distinguish a carrier's
// two concurrent calls.
func TestGrantTimeslotMapping(t *testing.T) {
	g := trunking.Grant{
		System:      "Sys",
		Protocol:    "dmr-tier3",
		GroupID:     0x123456,
		SourceID:    0xABCDEF,
		FrequencyHz: 866_175_000,
		ChannelID:   3,
		ChannelNum:  8,
		Timeslot:    2,
	}

	if dto := grantToDTO(g); dto.Timeslot != 2 {
		t.Errorf("grantToDTO Timeslot = %d, want 2", dto.Timeslot)
	}
	if pb := grantToPB(g); pb.GetTimeslot() != 2 {
		t.Errorf("grantToPB Timeslot = %d, want 2", pb.GetTimeslot())
	}

	// A non-slotted grant (P25 Phase 1) leaves Timeslot zero, which the
	// DTO omits from JSON (omitempty) and the PB carries as 0.
	clear := trunking.Grant{System: "Sys", Protocol: "p25", FrequencyHz: 851_000_000}
	if dto := grantToDTO(clear); dto.Timeslot != 0 {
		t.Errorf("grantToDTO Timeslot = %d, want 0 for non-slotted grant", dto.Timeslot)
	}
	if pb := grantToPB(clear); pb.GetTimeslot() != 0 {
		t.Errorf("grantToPB Timeslot = %d, want 0 for non-slotted grant", pb.GetTimeslot())
	}
}
