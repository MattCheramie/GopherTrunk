package tetra

import "testing"

// TestClassifyPartiesGroupAndIndividual pins the individual-vs-group
// classification. On real air a CMCE calling/transmitting party SSI is always an
// individual radio, and a grant carrying one is addressed under the group's GSSI;
// a later grant addressed to that radio (a D-CONNECT to the calling party, which
// has no party field) is individual-addressed, not a talkgroup grant.
func TestClassifyPartiesGroupAndIndividual(t *testing.T) {
	cc := New(Options{FrequencyHz: 467_913_000})
	const (
		gssi = 1020545 // talkgroup
		issi = 1005372 // radio ID
		call = 8675
	)

	// 1. Group D-SETUP: addressed under the GSSI, calling party present. GroupID is
	//    the GSSI, source is the party; the party is learned as an individual.
	var group VoiceGrant
	cc.classifyParties(&group,
		MACResource{Address: MACAddress{Type: addrSSI, SSI: gssi}},
		CMCEMessage{Type: CMCETypeDSetup, CallIdentifier: call, PartySSI: issi}, true)
	if group.DestSSI != gssi || group.SourceSSI != issi || group.Individual {
		t.Errorf("group grant = {dst:%d src:%d indiv:%v}, want {dst:%d src:%d indiv:false}",
			group.DestSSI, group.SourceSSI, group.Individual, gssi, issi)
	}
	if !cc.isIndividual(issi) {
		t.Errorf("party %d not learned as an individual", issi)
	}

	// 2. D-CONNECT addressed to the now-known individual, no party, no group mapped
	//    yet: flagged Individual so it is not surfaced as a talkgroup.
	var indiv VoiceGrant
	cc.classifyParties(&indiv,
		MACResource{Address: MACAddress{Type: addrSSI, SSI: issi}},
		CMCEMessage{Type: CMCETypeDConnect, CallIdentifier: 999}, true)
	if indiv.DestSSI != issi || !indiv.Individual {
		t.Errorf("individual-addressed grant = {dst:%d indiv:%v}, want {dst:%d indiv:true}",
			indiv.DestSSI, indiv.Individual, issi)
	}

	// 3. Once the call is mapped to its group, an individual-addressed grant for
	//    that call folds onto the group instead of standing alone.
	cc.rememberCall(call, gssi)
	var folded VoiceGrant
	cc.classifyParties(&folded,
		MACResource{Address: MACAddress{Type: addrSSI, SSI: issi}},
		CMCEMessage{Type: CMCETypeDConnect, CallIdentifier: call}, true)
	if folded.DestSSI != gssi || folded.SourceSSI != issi || folded.Individual {
		t.Errorf("folded grant = {dst:%d src:%d indiv:%v}, want {dst:%d src:%d indiv:false}",
			folded.DestSSI, folded.SourceSSI, folded.Individual, gssi, issi)
	}
}

// TestClassifyPartiesUnknownStaysGroup guards the conservative default: an SSI
// never seen as a party (and no CMCE party on this PDU) is left as a talkgroup —
// the MAC address type cannot distinguish a GSSI from an ISSI, so we never guess
// Individual without positive evidence.
func TestClassifyPartiesUnknownStaysGroup(t *testing.T) {
	cc := New(Options{FrequencyHz: 467_913_000})
	var g VoiceGrant
	cc.classifyParties(&g,
		MACResource{Address: MACAddress{Type: addrSSI, SSI: 1234567}},
		CMCEMessage{Type: CMCETypeDConnect, CallIdentifier: 1}, true)
	if g.DestSSI != 1234567 || g.Individual || g.SourceSSI != 0 {
		t.Errorf("unknown grant = {dst:%d src:%d indiv:%v}, want {dst:1234567 src:0 indiv:false}",
			g.DestSSI, g.SourceSSI, g.Individual)
	}
}
