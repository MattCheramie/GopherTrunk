package trunking

import "testing"

// TestSyntheticCallFallsBackToGrantLabel pins that a conventional-scanner
// call whose talkgroup is not catalogued surfaces the channel's configured
// label as its alpha tag, and that a catalogue entry still wins over it.
func TestSyntheticCallFallsBackToGrantLabel(t *testing.T) {
	e, _, bus, _ := mkEngine(t, 1)
	defer bus.Close()

	e.HandleSyntheticCall(Grant{System: "scanner", GroupID: 0x80000001, GroupLabel: "Fire Dispatch"}, "dev-a")
	var got *ActiveCall
	for _, ac := range e.ActiveCalls() {
		if ac.Grant.GroupID == 0x80000001 {
			got = ac
		}
	}
	if got == nil || got.Talkgroup == nil || got.Talkgroup.AlphaTag != "Fire Dispatch" {
		t.Fatalf("uncatalogued synthetic call talkgroup = %+v, want alpha tag %q", got, "Fire Dispatch")
	}
	if e.talkgroups.Lookup(0x80000001) != nil {
		t.Error("label fallback must not add a record to the catalogue")
	}

	e.talkgroups.Add(&TalkGroup{ID: 0x80000002, AlphaTag: "Catalogue Name", Scan: true, Stream: true, Record: true})
	e.HandleSyntheticCall(Grant{System: "scanner", GroupID: 0x80000002, GroupLabel: "Fire Dispatch"}, "dev-b")
	for _, ac := range e.ActiveCalls() {
		if ac.Grant.GroupID == 0x80000002 && ac.Talkgroup.AlphaTag != "Catalogue Name" {
			t.Errorf("catalogue entry should win, got %q", ac.Talkgroup.AlphaTag)
		}
	}
}
