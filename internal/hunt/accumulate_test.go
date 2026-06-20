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

// TestAccumulate_SkipsIndividualGrants: a unit-to-unit / telephone grant
// carries a 24-bit destination unit in GroupID (the field-reported bogus
// 140957), not a talkgroup. It must NOT become a talkgroup — only the group
// grant (TG 100) does — while still being recorded as a site voice channel.
func TestAccumulate_SkipsIndividualGrants(t *testing.T) {
	sys := &DiscoveredSystem{}
	r := fakeResult(851012500, map[string]any{"RFSS": uint8(1), "Site": uint8(2)})
	r.Grants = []siglab.GrantRecord{
		{GroupID: 100, FrequencyHz: 851_500_000},                      // group call → talkgroup
		{GroupID: 140957, FrequencyHz: 851_975_000, Individual: true}, // unit-to-unit target → not a TG
	}
	Accumulate(sys, Observation{Protocol: "p25", Confidence: 0.9, Result: r})

	if len(sys.Talkgroups) != 1 || sys.Talkgroups[0].Dec != 100 {
		t.Fatalf("talkgroups = %+v, want exactly [100] (no bogus 140957)", sys.Talkgroups)
	}
	// The individual call's frequency is still a real site voice channel.
	vc := sys.Sites[0].VoiceChannels
	if len(vc) != 2 {
		t.Errorf("site voice channels = %+v, want both grant freqs recorded", vc)
	}
}

// TestAccumulate_RecordsZeroRFSSSite: a single-RFSS system (RFSS=0, Site=0)
// that has decoded identity (WACN/SystemID) must place its site at (0,0) and
// record RFSS/Site=0 in the Identity map rather than dropping them as "unknown".
func TestAccumulate_RecordsZeroRFSSSite(t *testing.T) {
	sys := &DiscoveredSystem{}
	r := fakeResult(450125000, nil, 204)
	r.Topology = &siglab.TopologySnapshot{
		WACN:     0xBEE99,
		SystemID: 0x49A,
		RFSS:     0,
		Site:     0,
	}
	Accumulate(sys, Observation{Protocol: "p25", Confidence: 0.9, Result: r})

	if sys.RFSS() != 0 || sys.SiteNum() != 0 {
		t.Errorf("RFSS/Site = %d/%d, want 0/0", sys.RFSS(), sys.SiteNum())
	}
	if _, ok := sys.Identity["RFSS"]; !ok {
		t.Error("RFSS should be recorded (present) even when zero, once identity is known")
	}
	if _, ok := sys.Identity["Site"]; !ok {
		t.Error("Site should be recorded (present) even when zero, once identity is known")
	}
	if len(sys.Sites) != 1 || sys.Sites[0].RFSS != 0 || sys.Sites[0].SiteID != 0 {
		t.Errorf("sites = %+v, want one site at (0,0)", sys.Sites)
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

func TestAccumulate_FoldsTopology(t *testing.T) {
	sys := &DiscoveredSystem{}
	r := fakeResult(851012500, map[string]any{"NAC": uint16(0x293)}, 1000)
	r.Topology = &siglab.TopologySnapshot{
		WACN:     0xBEE99,
		SystemID: 0x49A,
		RFSS:     1,
		Site:     2,
		Neighbors: []siglab.NeighborRef{
			{RFSS: 1, Site: 3},
			{RFSS: 1, Site: 4},
		},
		BandPlan: []siglab.BandPlanSlot{
			{ChannelID: 1, BaseHz: 851_000_000, SpacingHz: 12_500},
		},
	}
	Accumulate(sys, Observation{Protocol: "p25", Confidence: 0.9, Result: r})

	if sys.WACN != 0xBEE99 || sys.SystemID != 0x49A {
		t.Errorf("identity = %X/%X, want BEE99/49A", sys.WACN, sys.SystemID)
	}
	if len(sys.Sites) != 1 {
		t.Fatalf("len(Sites) = %d, want 1", len(sys.Sites))
	}
	st := sys.Sites[0]
	if st.RFSS != 1 || st.SiteID != 2 {
		t.Errorf("camped site = %d/%d, want 1/2", st.RFSS, st.SiteID)
	}
	if len(st.ControlChannels) != 1 || st.ControlChannels[0].FrequencyHz != 851012500 {
		t.Errorf("control channels = %+v", st.ControlChannels)
	}
	if len(st.Neighbors) != 2 {
		t.Errorf("neighbors = %+v, want 2", st.Neighbors)
	}
	if len(sys.BandPlan) != 1 || sys.BandPlan[0].BaseHz != 851_000_000 {
		t.Errorf("band plan = %+v, want one entry base 851M", sys.BandPlan)
	}
	// Talkgroup from the grant still lands alongside the topology.
	if len(sys.Talkgroups) != 1 || sys.Talkgroups[0].Dec != 1000 {
		t.Errorf("talkgroups = %+v, want [1000]", sys.Talkgroups)
	}
}

func TestResolveNeighborFreqs(t *testing.T) {
	sys := &DiscoveredSystem{}
	r := fakeResult(851012500, map[string]any{"NAC": uint16(0x293)}, 1000)
	r.Topology = &siglab.TopologySnapshot{
		RFSS: 1, Site: 2,
		Neighbors: []siglab.NeighborRef{
			{RFSS: 1, Site: 3, ChannelID: 1, ChannelNumber: 10}, // resolvable
			{RFSS: 1, Site: 4}, // no coords ⇒ unresolved
		},
		BandPlan: []siglab.BandPlanSlot{
			{ChannelID: 1, BaseHz: 851_000_000, SpacingHz: 12_500},
		},
	}
	Accumulate(sys, Observation{Protocol: "p25", Confidence: 0.9, Result: r})
	sys.sortAll() // resolution runs at finish

	nb := sys.Sites[0].Neighbors
	// Sorted by (RFSS, Site): Site 3 first, then Site 4.
	if len(nb) != 2 {
		t.Fatalf("neighbors = %+v, want 2", nb)
	}
	if got := nb[0].FrequencyHz; got != 851_125_000 { // 851M + 10*12.5k
		t.Errorf("neighbor site 3 freq = %d, want 851125000", got)
	}
	if nb[1].FrequencyHz != 0 {
		t.Errorf("neighbor site 4 (no coords) should stay unresolved, got %d", nb[1].FrequencyHz)
	}
}

// TestNeighbor_DecoderResolvedFreqSurfaces covers the protocol-generic path:
// a decoder that resolves its neighbour frequency itself (DMR/EDACS/Motorola set
// TopoNeighborRef.FrequencyHz; there is no advertised band plan) must have that
// frequency carried through to the discovered system.
func TestNeighbor_DecoderResolvedFreqSurfaces(t *testing.T) {
	sys := &DiscoveredSystem{}
	r := fakeResult(440_000_000, map[string]any{"ColorCode": uint16(1)}, 100)
	r.Topology = &siglab.TopologySnapshot{
		RFSS: 0, Site: 2,
		Neighbors: []siglab.NeighborRef{
			{Site: 5, ChannelNumber: 7, FrequencyHz: 441_250_000}, // decoder-resolved
		},
	}
	Accumulate(sys, Observation{Protocol: "dmr", Confidence: 0.9, Result: r})
	sys.sortAll()
	if got := sys.Sites[0].Neighbors[0].FrequencyHz; got != 441_250_000 {
		t.Errorf("decoder-resolved neighbor freq = %d, want 441250000", got)
	}
}

// TestNeighbor_FreqBackfilledAcrossObservations: the first sighting of a
// neighbour had no resolver (freq 0); a later sighting resolved it. The merge
// keeps the identity but backfills the frequency.
func TestNeighbor_FreqBackfilledAcrossObservations(t *testing.T) {
	sys := &DiscoveredSystem{}
	mk := func(freq uint32) *siglab.Result {
		r := fakeResult(440_000_000, map[string]any{"ColorCode": uint16(1)}, 100)
		r.Topology = &siglab.TopologySnapshot{
			Site:      2,
			Neighbors: []siglab.NeighborRef{{Site: 5, ChannelNumber: 7, FrequencyHz: freq}},
		}
		return r
	}
	Accumulate(sys, Observation{Protocol: "dmr", Confidence: 0.9, Result: mk(0)})
	Accumulate(sys, Observation{Protocol: "dmr", Confidence: 0.9, Result: mk(441_250_000)})
	sys.sortAll()
	nb := sys.Sites[0].Neighbors
	if len(nb) != 1 {
		t.Fatalf("neighbors = %+v, want 1 (deduped)", nb)
	}
	if nb[0].FrequencyHz != 441_250_000 {
		t.Errorf("neighbor freq not backfilled: got %d", nb[0].FrequencyHz)
	}
}

func TestResolveNeighborFreqs_NoBandPlan(t *testing.T) {
	sys := &DiscoveredSystem{
		Sites: []DiscoveredSite{{
			RFSS: 1, SiteID: 2,
			Neighbors: []NeighborRef{{RFSS: 1, Site: 3, ChannelID: 1, ChannelNumber: 10}},
		}},
	}
	sys.sortAll()
	if sys.Sites[0].Neighbors[0].FrequencyHz != 0 {
		t.Errorf("no band plan ⇒ neighbor must stay unresolved")
	}
}

func TestAccumulate_TopologyDedupesAcrossObservations(t *testing.T) {
	sys := &DiscoveredSystem{}
	mk := func() Observation {
		r := fakeResult(851012500, nil)
		r.Topology = &siglab.TopologySnapshot{
			RFSS: 1, Site: 1,
			Neighbors: []siglab.NeighborRef{{RFSS: 1, Site: 2}},
			BandPlan:  []siglab.BandPlanSlot{{ChannelID: 1, BaseHz: 851_000_000, SpacingHz: 12_500}},
		}
		return Observation{Protocol: "p25", Confidence: 0.8, Result: r}
	}
	Accumulate(sys, mk())
	Accumulate(sys, mk())

	if len(sys.Sites) != 1 || len(sys.Sites[0].Neighbors) != 1 {
		t.Errorf("neighbors should dedupe: %+v", sys.Sites)
	}
	if len(sys.BandPlan) != 1 {
		t.Errorf("band plan should dedupe by channel id: %+v", sys.BandPlan)
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
