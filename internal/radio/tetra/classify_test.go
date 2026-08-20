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

// TestClassifyPartiesIndividualSelfCall pins the fix for the "call source update
// tg==src" a TETRA operator reported: when the CMCE calling/transmitting party
// equals the addressed SSI, the PDU is addressed to the transmitting radio itself
// — an INDIVIDUAL call, not a group (you can't call yourself). It must be flagged
// Individual with no self-source, instead of surfaced as "group X, source X".
func TestClassifyPartiesIndividualSelfCall(t *testing.T) {
	cc := New(Options{FrequencyHz: 467_913_000})
	const radio = 1005499 // the reporter's ISSI
	var g VoiceGrant
	cc.classifyParties(&g,
		MACResource{Address: MACAddress{Type: addrSSI, SSI: radio}},
		CMCEMessage{Type: CMCETypeDConnect, CallIdentifier: 42, PartySSI: radio}, true)
	if !g.Individual {
		t.Errorf("self-addressed call not flagged Individual: %+v", g)
	}
	if g.SourceSSI == g.DestSSI {
		t.Errorf("individual call left with source==dest (%d): a radio cannot call itself", g.SourceSSI)
	}
	if g.SourceSSI != 0 {
		t.Errorf("SourceSSI = %d, want 0 (counterparty unknown from a self-addressed PDU)", g.SourceSSI)
	}
}

// TestCallSourceRestoredMidCall is the failing-first regression for the
// individual-call "missing SRC" symptom (the TMO private-call Caveat A): ETSI
// strips the source/destination SSIs out of the compressed mid-call signalling,
// so a D-CONNECT / follow-on grant that carries only the call identifier has no
// party field. The persistent call-identifier → source-ISSI binding learned at
// D-SETUP must restore the caller on that later PDU; before the binding the
// source is blank after the setup burst.
func TestCallSourceRestoredMidCall(t *testing.T) {
	cc := New(Options{FrequencyHz: 467_913_000})
	const (
		caller = 1005372 // calling party ISSI
		called = 1009999 // called party ISSI
		call   = 4242
	)

	// D-SETUP binds the call identifier to its calling party (source).
	cc.handleCMCE(
		MACResource{Address: MACAddress{Type: addrSSI, SSI: called}},
		CMCEMessage{Type: CMCETypeDSetup, CallIdentifier: call, PartySSI: caller})

	// A later PDU on the same call carries ONLY the call identifier (no party).
	// The source must be restored from the learned binding, not left blank.
	var g VoiceGrant
	cc.classifyParties(&g,
		MACResource{Address: MACAddress{Type: addrSSI, SSI: called}},
		CMCEMessage{Type: CMCETypeDConnect, CallIdentifier: call}, true)
	if g.SourceSSI != caller {
		t.Errorf("SourceSSI = %d, want %d (restored from the call's learned source)", g.SourceSSI, caller)
	}
	if g.DestSSI != called {
		t.Errorf("DestSSI = %d, want %d", g.DestSSI, called)
	}

	// Once the call is released, the binding is dropped: a later reuse of the same
	// call identifier does not resurrect the stale source.
	cc.forgetCall(call)
	var after VoiceGrant
	cc.classifyParties(&after,
		MACResource{Address: MACAddress{Type: addrSSI, SSI: 7000001}},
		CMCEMessage{Type: CMCETypeDConnect, CallIdentifier: call}, true)
	if after.SourceSSI != 0 {
		t.Errorf("SourceSSI = %d after release, want 0 (binding must be dropped)", after.SourceSSI)
	}
}

// TestBackfillCallSourceNeverSelfSources guards the backfill against surfacing a
// "dest X, source X": when the only learned source for a call equals the grant's
// destination, the source stays blank rather than being restored as itself.
func TestBackfillCallSourceNeverSelfSources(t *testing.T) {
	cc := New(Options{FrequencyHz: 467_913_000})
	const (
		radio = 1005499
		call  = 77
	)
	cc.rememberCallSource(call, radio)
	var g VoiceGrant
	cc.classifyParties(&g,
		MACResource{Address: MACAddress{Type: addrSSI, SSI: radio}},
		CMCEMessage{Type: CMCETypeDConnect, CallIdentifier: call}, true)
	if g.SourceSSI == g.DestSSI && g.SourceSSI != 0 {
		t.Errorf("backfill produced source==dest (%d): a radio cannot be its own source", g.SourceSSI)
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
