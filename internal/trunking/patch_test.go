package trunking

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

func TestPatchRegistry(t *testing.T) {
	r := NewPatchRegistry()
	if r.MembersOf(100) != nil {
		t.Error("MembersOf on an empty registry should be nil")
	}
	r.Apply(PatchGroup{SuperGroup: 100, Members: []uint32{10, 20, 30}, Vendor: "motorola"})

	got := r.MembersOf(100)
	if len(got) != 3 || got[0] != 10 || got[2] != 30 {
		t.Errorf("MembersOf(100) = %v, want [10 20 30]", got)
	}
	if r.MembersOf(999) != nil {
		t.Error("MembersOf for an unknown super-group should be nil")
	}
	if len(r.Active()) != 1 {
		t.Errorf("Active() len = %d, want 1", len(r.Active()))
	}
	// MembersOf must return a defensive copy.
	got[0] = 999
	if r.MembersOf(100)[0] != 10 {
		t.Error("MembersOf did not return a defensive copy")
	}
	r.Delete(100)
	if r.MembersOf(100) != nil {
		t.Error("MembersOf after Delete should be nil")
	}
}

// TestEngineScanModeListAllowsPatchedMember confirms a grant on a
// patch super-group passes the scan-list gate when a member talkgroup
// is scanned, and the call carries the member list.
func TestEngineScanModeListAllowsPatchedMember(t *testing.T) {
	e, _, bus, _ := mkEngine(t, 1)
	defer bus.Close()
	e.SetScanMode(ScanModeList)
	// Super-group 500 is not scanned; member 78 is.
	e.talkgroups.Add(&TalkGroup{ID: 500, AlphaTag: "SUPER", Scan: false})
	e.talkgroups.Add(&TalkGroup{ID: 78, AlphaTag: "MEMBER", Scan: true})
	e.handlePatch(Patch{SuperGroup: 500, Members: []uint32{78}, Vendor: "motorola", Add: true})

	sub := bus.Subscribe()
	defer sub.Close()
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 500, FrequencyHz: 1_000_000})

	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCallStart {
				cs := ev.Payload.(CallStart)
				if len(cs.Grant.PatchedGroups) != 1 || cs.Grant.PatchedGroups[0] != 78 {
					t.Errorf("CallStart PatchedGroups = %v, want [78]", cs.Grant.PatchedGroups)
				}
				return
			}
		case <-deadline:
			t.Fatal("no CallStart for the patched-member grant")
		}
	}
}

// TestEngineScanModeListDropsPatchWithNoScannedMember confirms a
// patched super-group is still dropped when neither it nor any member
// is scanned.
func TestEngineScanModeListDropsPatchWithNoScannedMember(t *testing.T) {
	e, _, bus, tuners := mkEngine(t, 1)
	defer bus.Close()
	e.SetScanMode(ScanModeList)
	e.talkgroups.Add(&TalkGroup{ID: 78, AlphaTag: "MEMBER", Scan: false})
	e.handlePatch(Patch{SuperGroup: 500, Members: []uint32{78}, Add: true})

	sub := bus.Subscribe()
	defer sub.Close()
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 500, FrequencyHz: 1_000_000})

	select {
	case ev := <-sub.C:
		t.Errorf("unexpected event for an all-off-list patch: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
	if got := tuners[0].tuned(); len(got) != 0 {
		t.Errorf("off-list patched grant should not retune; got %v", got)
	}
}

func TestEnginePatchDeleteRemoves(t *testing.T) {
	e, _, bus, _ := mkEngine(t, 1)
	defer bus.Close()
	e.handlePatch(Patch{SuperGroup: 500, Members: []uint32{78}, Add: true})
	if len(e.Patches()) != 1 {
		t.Fatalf("after Add, Patches() len = %d, want 1", len(e.Patches()))
	}
	e.handlePatch(Patch{SuperGroup: 500, Add: false})
	if len(e.Patches()) != 0 {
		t.Errorf("after cancel, Patches() len = %d, want 0", len(e.Patches()))
	}
}
