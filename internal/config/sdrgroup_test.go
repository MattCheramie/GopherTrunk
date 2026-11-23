package config

import (
	"reflect"
	"strings"
	"testing"
)

const groupTestRate = 2_400_000

// groupHalf is a member's usable half-window at groupTestRate (1.08 MHz).
var groupHalf = sdrGroupUsableHalf(groupTestRate)

func groupConfig(g SDRGroupConfig, members ...DeviceConfig) Config {
	var c Config
	c.SDR.SampleRate = groupTestRate
	c.SDR.Groups = []SDRGroupConfig{g}
	c.SDR.Devices = members
	c.Trunking.Systems = []SystemConfig{
		{Name: "P25", Protocol: "p25", ControlChannels: []uint32{851_012_500, 853_500_000, 856_987_500}},
		{Name: "TET", Protocol: "tetra", ControlChannels: []uint32{390_100_000, 392_700_000}},
	}
	return c
}

func member(serial string) DeviceConfig {
	return DeviceConfig{Serial: serial, Role: "wideband", Group: "south"}
}

// checkCovers fails unless every frequency in [lo, hi] lies inside some
// member's window.
func checkCovers(t *testing.T, p SDRGroupPlan, lo, hi uint32) {
	t.Helper()
	for f := lo; f <= hi; f += 1_000 {
		ok := false
		for _, m := range p.Members {
			if m.Covers(f) {
				ok = true
				break
			}
		}
		if !ok {
			t.Fatalf("%d Hz is not covered by any member: %+v", f, p.Members)
		}
		if hi-f < 1_000 {
			break
		}
	}
}

func TestPlanSDRGroupTilesSpanWithOverlap(t *testing.T) {
	for n := 1; n <= 4; n++ {
		var members []DeviceConfig
		for i := 0; i < n; i++ {
			members = append(members, member(string(rune('A'+i))))
		}
		// The widest span n members can cover with the auto overlap.
		overlap := float64(sdrGroupNarrowTapHz + sdrGroupOverlapMarginHz)
		span := float64(n)*2*groupHalf - float64(n-1)*overlap - overlap - 1_000
		lo := uint32(851_000_000)
		hi := lo + uint32(span)
		c := groupConfig(SDRGroupConfig{Name: "south", LowHz: lo, HighHz: hi}, members...)
		_, plans, err := ExpandSDRGroups(c)
		if err != nil {
			t.Fatalf("n=%d: %v", n, err)
		}
		p := plans[0]
		if len(p.Members) != n {
			t.Fatalf("n=%d: %d members planned", n, len(p.Members))
		}
		checkCovers(t, p, lo, hi)
		for i := 1; i < n; i++ {
			got := float64(p.Members[i-1].HighHz) - float64(p.Members[i].LowHz)
			if got < overlap-2 {
				t.Errorf("n=%d: members %d/%d overlap %.0f Hz, want >= %.0f", n, i-1, i, got, overlap)
			}
			if p.Members[i-1].OwnHighHz+1 != p.Members[i].OwnLowHz {
				t.Errorf("n=%d: stitch slices %d/%d don't abut: %d / %d", n, i-1, i, p.Members[i-1].OwnHighHz, p.Members[i].OwnLowHz)
			}
		}
		// One more kHz of span than n members can hold must fail and name
		// what is uncovered.
		c.SDR.Groups[0].HighHz = hi + uint32(2*groupHalf)
		if _, _, err := ExpandSDRGroups(c); err == nil || !strings.Contains(err.Error(), "uncovered") {
			t.Errorf("n=%d: over-wide span: err = %v, want an uncovered-range error", n, err)
		}
	}
}

func TestPlanSDRGroupHonoursPinnedMember(t *testing.T) {
	pinned := member("B")
	pinned.CenterFreqHz = 853_500_000
	c := groupConfig(SDRGroupConfig{Name: "south", LowHz: 851_000_000, HighHz: 856_000_000},
		member("A"), pinned, member("C"))
	_, plans, err := ExpandSDRGroups(c)
	if err != nil {
		t.Fatal(err)
	}
	p := plans[0]
	var found bool
	for _, m := range p.Members {
		if m.Serial == "B" {
			found = true
			if !m.Pinned || m.CenterFreqHz != 853_500_000 {
				t.Errorf("pinned member moved: %+v", m)
			}
		}
	}
	if !found {
		t.Fatal("pinned member missing from plan")
	}
	checkCovers(t, p, 851_000_000, 856_000_000)
}

func TestPlanSDRGroupAssignsOverlapChannelToNearerCentre(t *testing.T) {
	c := groupConfig(SDRGroupConfig{Name: "south", LowHz: 851_000_000, HighHz: 855_000_000},
		member("A"), member("B"))
	_, plans, err := ExpandSDRGroups(c)
	if err != nil {
		t.Fatal(err)
	}
	a, b := plans[0].Members[0], plans[0].Members[1]
	// A frequency inside both windows, nearer B's centre.
	f := b.LowHz + 10_000
	if !a.Covers(f) || !b.Covers(f) {
		t.Fatalf("test frequency %d not in the overlap of %+v and %+v", f, a, b)
	}
	c.SDR.Groups[0].Conventional = []ConvChannelConfig{{FrequencyHz: f}}
	devs, plans, err := ExpandSDRGroups(c)
	if err != nil {
		t.Fatal(err)
	}
	near, far := plans[0].Members[1], plans[0].Members[0]
	if d1, d0 := absDiff(f, near.CenterFreqHz), absDiff(f, far.CenterFreqHz); d1 > d0 {
		near, far = far, near
	}
	if got := devs[near.DeviceIndex].Conventional; len(got) != 1 || got[0].FrequencyHz != f {
		t.Errorf("nearer member %s got %+v, want the channel", near.Serial, got)
	}
	if got := devs[far.DeviceIndex].Conventional; len(got) != 0 {
		t.Errorf("farther member %s got %+v, want nothing", far.Serial, got)
	}
}

func absDiff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

func TestPlanSDRGroupTETRAOverlapHoldsA144kTap(t *testing.T) {
	c := groupConfig(SDRGroupConfig{Name: "south",
		Channels: []DeviceChannelConfig{{FrequencyHz: 390_100_000, System: "TET"}, {FrequencyHz: 392_700_000, System: "TET"}}},
		member("A"), member("B"))
	devs, plans, err := ExpandSDRGroups(c)
	if err != nil {
		t.Fatal(err)
	}
	p := plans[0]
	if p.OverlapHz < sdrGroupTETRATapHz {
		t.Fatalf("auto overlap %d Hz is narrower than a TETRA tap", p.OverlapHz)
	}
	a, b := p.Members[0], p.Members[1]
	if float64(a.HighHz)-float64(b.LowHz) < sdrGroupTETRATapHz {
		t.Errorf("members overlap only %d Hz", a.HighHz-b.LowHz)
	}
	// Every TETRA carrier's whole 144 kHz tap sits inside its member.
	for _, m := range p.Members {
		for _, ch := range devs[m.DeviceIndex].Channels {
			if ch.FrequencyHz-sdrGroupTETRATapHz/2 < m.LowHz || ch.FrequencyHz+sdrGroupTETRATapHz/2 > m.HighHz {
				t.Errorf("TETRA tap at %d clipped by member %+v", ch.FrequencyHz, m)
			}
		}
	}
}

func TestPlanSDRGroupNudgesCentreOffDC(t *testing.T) {
	// One member, one channel: the naive centre is the channel itself.
	c := groupConfig(SDRGroupConfig{Name: "south",
		Channels: []DeviceChannelConfig{{FrequencyHz: 853_500_000, System: "P25"}}}, member("A"))
	_, plans, err := ExpandSDRGroups(c)
	if err != nil {
		t.Fatal(err)
	}
	m := plans[0].Members[0]
	if d := absDiff(m.CenterFreqHz, 853_500_000); d < sdrGroupNarrowTapHz/2+sdrGroupDCGuardHz {
		t.Errorf("centre %d is %d Hz from the channel; want it nudged off the DC spur", m.CenterFreqHz, d)
	}
	if !m.Covers(853_500_000) {
		t.Errorf("nudged member no longer covers its channel: %+v", m)
	}
	if len(plans[0].Warnings) != 0 {
		t.Errorf("unexpected warnings: %v", plans[0].Warnings)
	}
}

func TestPlanSDRGroupTooFewMembers(t *testing.T) {
	c := groupConfig(SDRGroupConfig{Name: "south", LowHz: 851_000_000, HighHz: 857_000_000}, member("A"))
	_, _, err := ExpandSDRGroups(c)
	if err == nil || !strings.Contains(err.Error(), "uncovered: ") || !strings.Contains(err.Error(), "MHz") {
		t.Fatalf("err = %v, want it to name the uncovered range", err)
	}
}

func TestExpandSDRGroupsLeavesInputUntouched(t *testing.T) {
	c := groupConfig(SDRGroupConfig{Name: "south", VoiceTaps: 2,
		Channels: []DeviceChannelConfig{{FrequencyHz: 851_012_500, System: "P25"}, {FrequencyHz: 853_500_000, System: "P25"}}},
		member("A"), DeviceConfig{Serial: "V", Role: "voice"}, member("B"))
	before := cloneDevices(c.SDR.Devices)
	devs, plans, err := ExpandSDRGroups(c)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(c.SDR.Devices, before) {
		t.Fatal("ExpandSDRGroups modified its input")
	}
	if !reflect.DeepEqual(devs[1], before[1]) {
		t.Errorf("non-member device changed: %+v", devs[1])
	}
	total := 0
	for _, m := range plans[0].Members {
		d := devs[m.DeviceIndex]
		if d.CenterFreqHz != m.CenterFreqHz || d.VoiceTaps != 2 || d.Role != "wideband" {
			t.Errorf("expanded member %+v does not match plan %+v", d, m)
		}
		total += len(d.Channels)
	}
	if total != 2 {
		t.Errorf("%d channels assigned, want 2", total)
	}
	// The expanded config validates as plain wideband devices.
	if errs := c.ValidateAll(); len(errs) != 0 {
		t.Fatalf("valid group config rejected: %v", errs)
	}
}

func cloneDevices(in []DeviceConfig) []DeviceConfig {
	out := make([]DeviceConfig, len(in))
	for i, d := range in {
		d.Channels = append([]DeviceChannelConfig(nil), d.Channels...)
		d.Conventional = append([]ConvChannelConfig(nil), d.Conventional...)
		out[i] = d
	}
	return out
}

func TestValidateSDRGroups(t *testing.T) {
	ok := SDRGroupConfig{Name: "south",
		Channels: []DeviceChannelConfig{{FrequencyHz: 851_012_500, System: "P25"}, {FrequencyHz: 856_987_500, System: "P25"}}}
	for name, tc := range map[string]struct {
		g       SDRGroupConfig
		members []DeviceConfig
		want    string
	}{
		"unknown group":       {ok, []DeviceConfig{member("A"), {Serial: "B", Role: "wideband", Group: "north"}}, `group "north" is not declared`},
		"member not wideband": {ok, []DeviceConfig{member("A"), {Serial: "B", Role: "voice", Group: "south"}}, "must be role: wideband"},
		"member channels": {ok, []DeviceConfig{member("A"), func() DeviceConfig {
			d := member("B")
			d.Channels = []DeviceChannelConfig{{FrequencyHz: 851_012_500, System: "P25"}}
			d.VoiceTaps = 1
			return d
		}()}, "channels, voice_taps belong on sdr.groups"},
		"no members":  {ok, []DeviceConfig{{Serial: "X", Role: "voice"}}, "no sdr.devices entry names this group"},
		"bad name":    {SDRGroupConfig{Name: "a b"}, nil, "must not contain"},
		"unknown sys": {SDRGroupConfig{Name: "south", Channels: []DeviceChannelConfig{{FrequencyHz: 851_012_500, System: "nope"}}}, []DeviceConfig{member("A")}, "sdr.groups[0].channels[0]: system \"nope\""},
		"not a cc":    {SDRGroupConfig{Name: "south", Channels: []DeviceChannelConfig{{FrequencyHz: 852_000_000, System: "P25"}}}, []DeviceConfig{member("A")}, "sdr.groups[0].channels[0]: frequency_hz 852000000 does not match"},
		"too wide":    {ok, []DeviceConfig{member("A")}, "uncovered"},
	} {
		c := groupConfig(tc.g, tc.members...)
		var msgs []string
		for _, e := range c.ValidateAll() {
			msgs = append(msgs, e.Error())
		}
		if !strings.Contains(strings.Join(msgs, "\n"), tc.want) {
			t.Errorf("%s: errors %q, want one mentioning %q", name, msgs, tc.want)
		}
	}
}

func TestSDRGroupMemberForSeam(t *testing.T) {
	c := groupConfig(SDRGroupConfig{Name: "south", LowHz: 851_000_000, HighHz: 855_000_000}, member("A"), member("B"))
	_, plans, err := ExpandSDRGroups(c)
	if err != nil {
		t.Fatal(err)
	}
	p := plans[0]
	a := p.Members[0]
	if m, ok := p.MemberFor(a.CenterFreqHz+100_000, 50_000); !ok || m.Serial != a.Serial {
		t.Errorf("MemberFor near A's centre = %+v, %v", m, ok)
	}
	// A band wider than any member's window fits nowhere.
	if _, ok := p.MemberFor(a.HighHz, uint32(groupHalf)); ok {
		t.Error("MemberFor accepted a band wider than a member")
	}
}
