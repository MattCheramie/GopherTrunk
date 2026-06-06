package hunt

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// fakeResult builds a siglab.Result with a lock carrying the given identity
// fields and the given grant group ids.
func fakeResult(ccHz uint32, fields map[string]any, groups ...uint32) *siglab.Result {
	r := &siglab.Result{
		Locked: true,
		Lock:   &siglab.LockInfo{FrequencyHz: ccHz, Fields: fields},
	}
	for _, g := range groups {
		r.Grants = append(r.Grants, siglab.GrantRecord{GroupID: g})
	}
	return r
}

func TestAccumulate_IdentityAndTalkgroups(t *testing.T) {
	sys := &DiscoveredSystem{}
	at := time.Date(2026, 6, 6, 12, 0, 0, 0, time.UTC)

	Accumulate(sys, Observation{
		Protocol:   "p25",
		Confidence: 0.9,
		At:         at,
		Result: fakeResult(851012500, map[string]any{
			"NAC":      uint16(0x4D2),
			"WACN":     uint32(0xBEE99),
			"SystemID": uint16(0x49A),
			"RFSS":     uint8(1),
			"Site":     uint8(2),
		}, 1000, 1001, 1000),
	})

	if sys.Protocol != "p25" {
		t.Errorf("Protocol = %q, want p25", sys.Protocol)
	}
	if sys.WACN != 0xBEE99 {
		t.Errorf("WACN = %X, want BEE99", sys.WACN)
	}
	if sys.SystemID != 0x49A {
		t.Errorf("SystemID = %X, want 49A", sys.SystemID)
	}
	if sys.NAC != 0x4D2 {
		t.Errorf("NAC = %X, want 4D2", sys.NAC)
	}
	if len(sys.Sites) != 1 {
		t.Fatalf("len(Sites) = %d, want 1", len(sys.Sites))
	}
	st := sys.Sites[0]
	if st.RFSS != 1 || st.SiteID != 2 {
		t.Errorf("site = RFSS %d Site %d, want 1/2", st.RFSS, st.SiteID)
	}
	if len(st.ControlChannels) != 1 || st.ControlChannels[0].FrequencyHz != 851012500 {
		t.Errorf("control channels = %+v, want one @ 851012500", st.ControlChannels)
	}
	// 1000 was granted twice → one talkgroup with count 2; 1001 once.
	if len(sys.Talkgroups) != 2 {
		t.Fatalf("len(Talkgroups) = %d, want 2", len(sys.Talkgroups))
	}
	for _, tg := range sys.Talkgroups {
		if tg.Dec == 1000 && tg.Count != 2 {
			t.Errorf("tg 1000 count = %d, want 2", tg.Count)
		}
	}
}

func TestAccumulate_Idempotent_DedupesSites(t *testing.T) {
	sys := &DiscoveredSystem{}
	obs := Observation{
		Protocol:   "p25",
		Confidence: 0.8,
		Result: fakeResult(851012500, map[string]any{
			"RFSS": uint8(1), "Site": uint8(1),
		}, 2000),
	}
	Accumulate(sys, obs)
	Accumulate(sys, obs) // re-observe the same control channel

	if len(sys.Sites) != 1 {
		t.Fatalf("len(Sites) = %d, want 1 (deduped)", len(sys.Sites))
	}
	if len(sys.Sites[0].ControlChannels) != 1 {
		t.Fatalf("len(ControlChannels) = %d, want 1 (deduped by freq)", len(sys.Sites[0].ControlChannels))
	}
	if len(sys.Talkgroups) != 1 {
		t.Fatalf("len(Talkgroups) = %d, want 1", len(sys.Talkgroups))
	}
	if sys.Talkgroups[0].Count != 2 {
		t.Errorf("tg count = %d, want 2 (bumped on re-observe)", sys.Talkgroups[0].Count)
	}
}

func TestAccumulate_ConfidenceTakesMinimum(t *testing.T) {
	sys := &DiscoveredSystem{}
	Accumulate(sys, Observation{Protocol: "p25", Confidence: 0.9, Result: fakeResult(1, nil)})
	Accumulate(sys, Observation{Protocol: "p25", Confidence: 0.6, Result: fakeResult(2, nil)})
	if sys.Confidence != 0.6 {
		t.Errorf("Confidence = %v, want 0.6 (minimum across CCs)", sys.Confidence)
	}
}

func TestDisplayName_Synthesized(t *testing.T) {
	cases := []struct {
		sys  DiscoveredSystem
		want string
	}{
		{DiscoveredSystem{Name: "Real Name"}, "Real Name"},
		{DiscoveredSystem{Protocol: "p25", WACN: 0xBEE99, SystemID: 0x49A}, "Unknown-p25-BEE99-49A"},
		{DiscoveredSystem{Protocol: "dmr", SystemID: 0x10}, "Unknown-dmr-010"},
		{DiscoveredSystem{Protocol: "nxdn"}, "Unknown-nxdn"},
	}
	for _, c := range cases {
		if got := c.sys.DisplayName(); got != c.want {
			t.Errorf("DisplayName(%+v) = %q, want %q", c.sys, got, c.want)
		}
	}
}
