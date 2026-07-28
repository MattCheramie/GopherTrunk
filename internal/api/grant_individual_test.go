package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestGrantIndividualDTO verifies the Individual flag survives the
// trunking.Grant → JSON DTO conversion, so a REST/SSE consumer can render a
// TETRA unit-to-unit call's radio ID as an individual rather than a phantom
// talkgroup. It is omitted from JSON for ordinary group grants.
func TestGrantIndividualDTO(t *testing.T) {
	indiv := trunking.Grant{
		System: "Sys", Protocol: "tetra",
		GroupID: 1005372, FrequencyHz: 467_912_500, Individual: true,
	}
	if dto := grantToDTO(indiv); !dto.Individual {
		t.Error("grantToDTO dropped Individual=true")
	}
	if b, _ := json.Marshal(grantToDTO(indiv)); !strings.Contains(string(b), `"individual":true`) {
		t.Errorf("individual grant JSON missing individual flag: %s", b)
	}

	group := trunking.Grant{
		System: "Sys", Protocol: "tetra",
		GroupID: 1020545, FrequencyHz: 467_912_500, // Individual defaults false
	}
	if dto := grantToDTO(group); dto.Individual {
		t.Error("grantToDTO set Individual on a group grant")
	}
	if b, _ := json.Marshal(grantToDTO(group)); strings.Contains(string(b), "individual") {
		t.Errorf("group grant JSON should omit the individual flag: %s", b)
	}
}
