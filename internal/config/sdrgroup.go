package config

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
)

// SDRGroupConfig gangs several `role: wideband` SDRs into ONE logical wideband
// receiver that covers more spectrum than any single member can. Each member
// is an ordinary sdr.devices entry naming the group (`group: <name>`); the
// group carries the channel plan, and the daemon tiles the span with the
// members, each at its own centre frequency.
//
// The members' IQ is never combined. Every channel and every voice grant is
// decoded by the one member whose window covers it, so members with
// independent clocks (ordinary RTL-SDRs) work. ExpandSDRGroups turns a group
// into one plain wideband DeviceConfig per member; everything downstream (the
// wideband engines, the virtual voice taps, unplug recovery) runs on those
// unchanged.
type SDRGroupConfig struct {
	// Name identifies the group. Members reference it with
	// sdr.devices[].group, and the spectrum view lists it as group:<name>.
	Name string `yaml:"name"`
	// LowHz / HighHz optionally bound the span the members must cover. Leave
	// both zero to cover exactly the channel list's extent. Set them to the
	// system's whole voice band so trunked voice grants anywhere in it land on
	// a member (a grant is followed only where some member's window reaches).
	LowHz  uint32 `yaml:"low_hz"`
	HighHz uint32 `yaml:"high_hz"`
	// OverlapHz is how far adjacent members' windows overlap. 0 = auto: the
	// widest channel tap any of the group's systems needs (48 kHz, 144 kHz for
	// TETRA) plus 25 kHz, so a carrier near a seam sits wholly inside one
	// member.
	OverlapHz uint32 `yaml:"overlap_hz"`
	// VoiceTaps / SignallingTaps are allocated on EVERY member (see the
	// sdr.devices fields of the same name).
	VoiceTaps      int `yaml:"voice_taps"`
	SignallingTaps int `yaml:"signalling_taps"`
	// TunerStrategy applies to every member (see sdr.devices[].tuner_strategy).
	TunerStrategy string `yaml:"tuner_strategy"`
	// Channels and Conventional take the same entries as a wideband device's;
	// the planner assigns each to the member whose window covers it, the one
	// where it sits closest to centre when two overlap.
	Channels     []DeviceChannelConfig `yaml:"channels"`
	Conventional []ConvChannelConfig   `yaml:"conventional"`
}

// SDRGroupPlan is the planner's result for one group: where each member is
// tuned and what it decodes.
type SDRGroupPlan struct {
	Name string
	// LowHz / HighHz is the span the plan covers (the configured span, or the
	// channel extent).
	LowHz, HighHz uint32
	OverlapHz     uint32
	SampleRateHz  uint32
	// Members are sorted by centre frequency.
	Members []SDRGroupMemberPlan
	// Warnings are non-fatal plan notes (a channel left near a member's DC
	// spur, a member with nothing to do).
	Warnings []string
}

// SDRGroupMemberPlan is one member's share of a group plan.
type SDRGroupMemberPlan struct {
	Serial       string
	DeviceIndex  int // index into sdr.devices
	CenterFreqHz uint32
	Pinned       bool // center_freq_hz came from the member's own config
	// LowHz / HighHz is the member's usable window (centre ± the wideband
	// usable half-band).
	LowHz, HighHz uint32
	// OwnLowHz / OwnHighHz is the slice of the span this member owns in a
	// stitched view: its window cut at the midpoints of its overlaps with
	// its neighbours.
	OwnLowHz, OwnHighHz uint32
	// Channels lists every frequency assigned to this member (trunked and
	// conventional), ascending.
	Channels []uint32
}

// Covers reports whether f lies inside the member's usable window.
func (m SDRGroupMemberPlan) Covers(f uint32) bool { return f >= m.LowHz && f <= m.HighHz }

// MemberFor returns the member that should handle a signal centred on f with
// the given half-bandwidth: the one whose window holds [f-half, f+half] with
// f closest to its centre. ok is false when no single member covers it (the
// band straddles a seam wider than the overlap, or lies outside the plan).
func (p SDRGroupPlan) MemberFor(f, halfBW uint32) (SDRGroupMemberPlan, bool) {
	best, bestDist := -1, math.MaxFloat64
	for i, m := range p.Members {
		lo, hi := int64(f)-int64(halfBW), int64(f)+int64(halfBW)
		if lo < int64(m.LowHz) || hi > int64(m.HighHz) {
			continue
		}
		if d := math.Abs(float64(f) - float64(m.CenterFreqHz)); d < bestDist {
			best, bestDist = i, d
		}
	}
	if best < 0 {
		return SDRGroupMemberPlan{}, false
	}
	return p.Members[best], true
}

// Group-planner constants.
const (
	// sdrGroupNarrowTapHz / sdrGroupTETRATapHz are the channel tap widths the
	// wideband engine cuts (48 kHz for the narrowband family, 144 kHz TETRA).
	sdrGroupNarrowTapHz = 48_000
	sdrGroupTETRATapHz  = 144_000
	// sdrGroupOverlapMarginHz is added to the widest tap for the auto overlap.
	sdrGroupOverlapMarginHz = 25_000
	// sdrGroupDCGuardHz is how close (beyond the channel's own half-tap) a
	// channel may sit to an auto-planned centre before the planner nudges the
	// centre off it: an RTL-SDR's DC spur sits on the centre.
	sdrGroupDCGuardHz = 10_000
	// sdrGroupNudgeStepHz is the planner's DC-nudge search step.
	sdrGroupNudgeStepHz = 2_500
)

// sdrGroupRate is the IQ rate the planner assumes for every member.
func sdrGroupRate(c Config) uint32 {
	if c.SDR.SampleRate != 0 {
		return c.SDR.SampleRate
	}
	return 2_048_000 // sdr.DefaultSampleRateHz; repeated to avoid an import cycle
}

// sdrGroupUsableHalf is a member's usable half-band: the same rule
// validateWidebandDevice and wbvoice.VirtualTuner.CanTune apply.
func sdrGroupUsableHalf(rate uint32) float64 {
	return float64(rate) * (0.5 - widebandGuardFrac)
}

// ExpandSDRGroups returns sdr.devices with every group member replaced by a
// plain wideband device carrying its planned centre frequency, its share of
// the group's channels, and the group's per-member settings. Devices that are
// not group members come back unchanged and in place. The input is not
// modified. A group that cannot be planned is an error (all of them, joined).
func ExpandSDRGroups(c Config) ([]DeviceConfig, []SDRGroupPlan, error) {
	devs, plans, errs := expandSDRGroups(c)
	return devs, plans, errors.Join(errs...)
}

// WithSDRGroupsExpanded returns a copy of c whose sdr.devices has every group
// expanded (ExpandSDRGroups). The daemon builds from this copy.
func (c Config) WithSDRGroupsExpanded() (Config, []SDRGroupPlan, error) {
	devs, plans, err := ExpandSDRGroups(c)
	if err != nil {
		return c, nil, err
	}
	c.SDR.Devices = devs
	return c, plans, nil
}

// WidebandDevices is sdr.devices with groups expanded where they plan
// cleanly; a group that fails to plan leaves its members unexpanded (its
// errors are reported by validation). Cross-section checks that read the
// wideband channel lists use this, so a group's channels count.
func (c Config) WidebandDevices() []DeviceConfig {
	devs, _, _ := expandSDRGroups(c)
	return devs
}

// SDRGroupPlans returns the plan of every group that plans cleanly.
func (c Config) SDRGroupPlans() []SDRGroupPlan {
	_, plans, _ := expandSDRGroups(c)
	return plans
}

func expandSDRGroups(c Config) ([]DeviceConfig, []SDRGroupPlan, []error) {
	out := make([]DeviceConfig, len(c.SDR.Devices))
	copy(out, c.SDR.Devices)
	if len(c.SDR.Groups) == 0 {
		return out, nil, nil
	}
	var errs []error
	var plans []SDRGroupPlan
	rate := sdrGroupRate(c)
	for gi, g := range c.SDR.Groups {
		var members []int
		for di, d := range c.SDR.Devices {
			if strings.TrimSpace(d.Group) == g.Name && g.Name != "" {
				members = append(members, di)
			}
		}
		plan, err := PlanSDRGroup(gi, g, c.SDR.Devices, members, rate, c.Trunking.Systems)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		plans = append(plans, plan)
		for _, m := range plan.Members {
			out[m.DeviceIndex] = expandMember(c.SDR.Devices[m.DeviceIndex], g, m)
		}
	}
	return out, plans, errs
}

// expandMember builds the plain wideband device one member runs as.
func expandMember(d DeviceConfig, g SDRGroupConfig, m SDRGroupMemberPlan) DeviceConfig {
	d.Role = "wideband"
	d.CenterFreqHz = m.CenterFreqHz
	d.TunerStrategy = g.TunerStrategy
	d.VoiceTaps = g.VoiceTaps
	d.SignallingTaps = g.SignallingTaps
	mine := make(map[uint32]bool, len(m.Channels))
	for _, f := range m.Channels {
		mine[f] = true
	}
	d.Channels = nil
	for _, ch := range g.Channels {
		if mine[ch.FrequencyHz] {
			d.Channels = append(d.Channels, ch)
		}
	}
	d.Conventional = nil
	for _, ch := range g.Conventional {
		if mine[ch.FrequencyHz] {
			d.Conventional = append(d.Conventional, ch)
		}
	}
	return d
}

// sdrGroupChannel is one frequency the planner has to place.
type sdrGroupChannel struct {
	freq  uint32
	halfT float64 // half the tap width the engine cuts for it
}

// groupChannels lists the group's frequencies with their tap half-widths.
func groupChannels(g SDRGroupConfig, systems []SystemConfig) []sdrGroupChannel {
	proto := make(map[string]string, len(systems))
	for _, s := range systems {
		proto[s.Name] = strings.ToLower(strings.TrimSpace(s.Protocol))
	}
	var out []sdrGroupChannel
	for _, ch := range g.Channels {
		w := float64(sdrGroupNarrowTapHz)
		if proto[ch.System] == "tetra" {
			w = sdrGroupTETRATapHz
		}
		out = append(out, sdrGroupChannel{freq: ch.FrequencyHz, halfT: w / 2})
	}
	for _, ch := range g.Conventional {
		out = append(out, sdrGroupChannel{freq: ch.FrequencyHz, halfT: sdrGroupNarrowTapHz / 2})
	}
	return out
}

// groupMaxTapHz is the widest tap any of the group's channels needs.
func groupMaxTapHz(chs []sdrGroupChannel) float64 {
	w := float64(sdrGroupNarrowTapHz)
	for _, ch := range chs {
		if 2*ch.halfT > w {
			w = 2 * ch.halfT
		}
	}
	return w
}

// interval is a closed frequency interval in Hz (float64 to keep the
// arithmetic free of unsigned underflow).
type interval struct{ lo, hi float64 }

// PlanSDRGroup plans one group: it places the members' centre frequencies so
// their windows tile the span with the overlap, honouring members that pin
// center_freq_hz, then assigns every channel to the covering member where it
// sits closest to centre. idx is the group's index in sdr.groups (for error
// paths); members index into devices.
func PlanSDRGroup(idx int, g SDRGroupConfig, devices []DeviceConfig, members []int, rate uint32, systems []SystemConfig) (SDRGroupPlan, error) {
	path := fmt.Sprintf("sdr.groups[%d]", idx)
	if g.Name != "" {
		path = fmt.Sprintf("sdr.groups[%d] (%s)", idx, g.Name)
	}
	if len(members) == 0 {
		return SDRGroupPlan{}, fmt.Errorf("%s: no sdr.devices entry names this group (set `group: %s` on each member SDR)", path, g.Name)
	}
	chs := groupChannels(g, systems)
	half := sdrGroupUsableHalf(rate)
	width := 2 * half

	// The span to cover.
	lo, hi := float64(g.LowHz), float64(g.HighHz)
	if g.LowHz == 0 && g.HighHz == 0 {
		if len(chs) == 0 {
			return SDRGroupPlan{}, fmt.Errorf("%s: needs channels, conventional channels, or a low_hz/high_hz span to cover", path)
		}
		lo, hi = math.MaxFloat64, 0
		for _, ch := range chs {
			lo = math.Min(lo, float64(ch.freq)-ch.halfT)
			hi = math.Max(hi, float64(ch.freq)+ch.halfT)
		}
	} else if g.LowHz == 0 || g.HighHz == 0 || g.HighHz <= g.LowHz {
		return SDRGroupPlan{}, fmt.Errorf("%s: low_hz and high_hz must both be set, with high_hz > low_hz", path)
	}
	for _, ch := range chs {
		if float64(ch.freq) < lo || float64(ch.freq) > hi {
			return SDRGroupPlan{}, fmt.Errorf("%s: channel %d Hz lies outside the group span %d..%d Hz", path, ch.freq, uint32(lo), uint32(hi))
		}
	}

	overlap := float64(g.OverlapHz)
	if overlap == 0 {
		overlap = groupMaxTapHz(chs) + sdrGroupOverlapMarginHz
	}
	if overlap >= width/2 {
		return SDRGroupPlan{}, fmt.Errorf("%s: overlap_hz %.0f must be less than half a member's usable width (%.0f Hz at %d S/s)", path, overlap, width/2, rate)
	}
	// Pad the span by half the overlap so a centre nudged off a DC spur
	// (by at most overlap/2) still covers the span's edges.
	cover := interval{lo - overlap/2, hi + overlap/2}

	plan := SDRGroupPlan{
		Name: g.Name, LowHz: uint32(math.Max(lo, 0)), HighHz: uint32(hi),
		OverlapHz: uint32(overlap), SampleRateHz: rate,
	}

	// Pinned members first.
	type slot struct {
		dev    int
		center float64
		pinned bool
	}
	var slots []slot
	var free []int
	var pinned []interval
	for _, di := range members {
		if c := devices[di].CenterFreqHz; c != 0 {
			slots = append(slots, slot{dev: di, center: float64(c), pinned: true})
			pinned = append(pinned, interval{float64(c) - half, float64(c) + half})
		} else {
			free = append(free, di)
		}
	}

	// The parts of the span the pinned members leave uncovered. A gap that
	// borders a pinned window is widened by the overlap so the free member
	// placed there overlaps its pinned neighbour.
	gaps := subtractIntervals(cover, pinned)
	for i := range gaps {
		if gaps[i].lo > cover.lo {
			gaps[i].lo -= overlap
		}
		if gaps[i].hi < cover.hi {
			gaps[i].hi += overlap
		}
	}
	need := make([]int, len(gaps))
	total := 0
	for i, gp := range gaps {
		need[i] = windowsFor(gp.hi-gp.lo, width, overlap)
		total += need[i]
	}
	if total > len(free) {
		var parts []string
		for _, gp := range gaps {
			parts = append(parts, fmt.Sprintf("%.4f-%.4f MHz", gp.lo/1e6, gp.hi/1e6))
		}
		return SDRGroupPlan{}, fmt.Errorf(
			"%s: covering %.4f-%.4f MHz with %.0f kHz overlap needs %d more member(s) at %d S/s (each covers %.0f kHz) but only %d free member(s) are in the group; uncovered: %s — add members, raise sdr.sample_rate, or narrow the span",
			path, lo/1e6, hi/1e6, overlap/1e3, total, rate, width/1e3, len(free), strings.Join(parts, ", "))
	}
	// Spare members: no gaps (the pinned members cover everything) means the
	// spares tile the whole span redundantly; otherwise each spare goes to
	// the gap whose windows are most crowded.
	if len(gaps) == 0 && len(free) > 0 {
		gaps = []interval{cover}
		need = []int{0}
	}
	for spare := len(free) - total; spare > 0; spare-- {
		best, bestLoad := 0, -1.0
		for i, gp := range gaps {
			load := (gp.hi - gp.lo) / float64(need[i]+1)
			if need[i] == 0 {
				load = math.MaxFloat64
			}
			if load > bestLoad {
				best, bestLoad = i, load
			}
		}
		need[best]++
	}
	fi := 0
	for i, gp := range gaps {
		for _, c := range spreadCenters(gp, need[i], half) {
			slots = append(slots, slot{dev: free[fi], center: c})
			fi++
		}
	}

	// Nudge auto-placed centres off any channel's DC spur.
	for i := range slots {
		if slots[i].pinned {
			continue
		}
		if c, ok := nudgeOffDC(slots[i].center, chs, overlap/2); ok {
			slots[i].center = c
		} else {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf(
				"%s: member %s at %.4f MHz has a channel on its DC spur and no nearby centre avoids every channel; pin center_freq_hz to move it",
				path, devices[slots[i].dev].Serial, slots[i].center/1e6))
		}
	}

	sort.SliceStable(slots, func(a, b int) bool { return slots[a].center < slots[b].center })
	for _, s := range slots {
		if s.center-half < 0 {
			return SDRGroupPlan{}, fmt.Errorf("%s: member %s would tune below 0 Hz", path, devices[s.dev].Serial)
		}
		plan.Members = append(plan.Members, SDRGroupMemberPlan{
			Serial:       devices[s.dev].Serial,
			DeviceIndex:  s.dev,
			CenterFreqHz: uint32(math.Round(s.center)),
			Pinned:       s.pinned,
			LowHz:        uint32(math.Ceil(s.center - half)),
			HighHz:       uint32(math.Floor(s.center + half)),
		})
	}

	// Assign every channel to the covering member where it sits closest to
	// centre (and whose window holds its whole tap, when one does).
	for _, ch := range chs {
		best, bestScore := -1, math.MaxFloat64
		for i, m := range plan.Members {
			off := math.Abs(float64(ch.freq) - float64(m.CenterFreqHz))
			if off > half {
				continue
			}
			score := off
			if off+ch.halfT > half {
				score += width // tap clipped by the window edge: last resort
			}
			if score < bestScore {
				best, bestScore = i, score
			}
		}
		if best < 0 {
			return SDRGroupPlan{}, fmt.Errorf("%s: channel %d Hz is outside every member's window; pin the members' center_freq_hz so one covers it, or remove the pin", path, ch.freq)
		}
		plan.Members[best].Channels = append(plan.Members[best].Channels, ch.freq)
	}
	for i := range plan.Members {
		sort.Slice(plan.Members[i].Channels, func(a, b int) bool { return plan.Members[i].Channels[a] < plan.Members[i].Channels[b] })
	}

	// Stitch ownership: cut each window at the midpoint of its overlap with
	// the next member up.
	for i := range plan.Members {
		m := &plan.Members[i]
		m.OwnLowHz, m.OwnHighHz = m.LowHz, m.HighHz
		if i > 0 {
			prev := plan.Members[i-1]
			if prev.HighHz >= m.LowHz {
				m.OwnLowHz = uint32((uint64(prev.HighHz)+uint64(m.LowHz))/2) + 1
			}
		}
		if i+1 < len(plan.Members) {
			next := plan.Members[i+1]
			if m.HighHz >= next.LowHz {
				m.OwnHighHz = uint32((uint64(m.HighHz) + uint64(next.LowHz)) / 2)
			}
		}
		if len(m.Channels) == 0 && g.VoiceTaps == 0 && g.SignallingTaps == 0 && len(g.Channels)+len(g.Conventional) > 0 {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf(
				"%s: member %s (%.4f MHz) carries no channel and the group has no voice_taps, so it only feeds the spectrum view",
				path, m.Serial, float64(m.CenterFreqHz)/1e6))
		}
	}
	return plan, nil
}

// windowsFor is how many windows of the given width, overlapping by overlap,
// cover a length.
func windowsFor(length, width, overlap float64) int {
	if length <= width {
		return 1
	}
	return int(math.Ceil((length-width)/(width-overlap))) + 1
}

// spreadCenters places n window centres evenly so their windows cover gp:
// the first window's low edge on gp.lo, the last one's high edge on gp.hi.
// When gp is narrower than a window the centres spread over the range of
// centres that still cover all of it.
func spreadCenters(gp interval, n int, half float64) []float64 {
	if n <= 0 {
		return nil
	}
	first, last := gp.lo+half, gp.hi-half
	if n == 1 {
		return []float64{(first + last) / 2}
	}
	out := make([]float64, n)
	for i := range out {
		out[i] = first + float64(i)*(last-first)/float64(n-1)
	}
	return out
}

// nudgeOffDC moves c by up to maxShift so that no channel's tap lies within
// sdrGroupDCGuardHz of it, searching outward from c in sdrGroupNudgeStepHz
// steps. ok is false when no shift in range clears every channel.
func nudgeOffDC(c float64, chs []sdrGroupChannel, maxShift float64) (float64, bool) {
	clear := func(x float64) bool {
		for _, ch := range chs {
			if math.Abs(float64(ch.freq)-x) < ch.halfT+sdrGroupDCGuardHz {
				return false
			}
		}
		return true
	}
	if clear(c) {
		return c, true
	}
	for s := sdrGroupNudgeStepHz; float64(s) <= maxShift; s += sdrGroupNudgeStepHz {
		if clear(c + float64(s)) {
			return c + float64(s), true
		}
		if clear(c - float64(s)) {
			return c - float64(s), true
		}
	}
	return c, false
}

// subtractIntervals returns the parts of span no interval in cut covers, in
// ascending order.
func subtractIntervals(span interval, cut []interval) []interval {
	cs := append([]interval(nil), cut...)
	sort.Slice(cs, func(a, b int) bool { return cs[a].lo < cs[b].lo })
	var out []interval
	pos := span.lo
	for _, c := range cs {
		if c.hi <= pos {
			continue
		}
		if c.lo > pos {
			out = append(out, interval{pos, math.Min(c.lo, span.hi)})
		}
		pos = math.Max(pos, c.hi)
		if pos >= span.hi {
			break
		}
	}
	if pos < span.hi {
		out = append(out, interval{pos, span.hi})
	}
	// Drop slivers that are empty after clamping.
	keep := out[:0]
	for _, gp := range out {
		if gp.hi > gp.lo {
			keep = append(keep, gp)
		}
	}
	return keep
}

// validateSDRGroups checks sdr.groups and the group members, then validates
// each expanded member with the ordinary wideband rules.
func (c Config) validateSDRGroups() []error {
	var errs []error
	groupIdx := map[string]int{}
	for i, g := range c.SDR.Groups {
		path := fmt.Sprintf("sdr.groups[%d]", i)
		name := strings.TrimSpace(g.Name)
		switch {
		case name == "":
			errs = append(errs, fmt.Errorf("%s: name required", path))
			continue
		case name != g.Name:
			errs = append(errs, fmt.Errorf("%s: name %q has leading or trailing spaces", path, g.Name))
			continue
		case strings.ContainsAny(name, ": /"):
			errs = append(errs, fmt.Errorf("%s: name %q must not contain ':', '/' or spaces (it becomes the spectrum id group:%s)", path, name, name))
			continue
		}
		if prev, dup := groupIdx[name]; dup {
			errs = append(errs, fmt.Errorf("%s: duplicate group name %q (also sdr.groups[%d])", path, name, prev))
			continue
		}
		groupIdx[name] = i
		if g.VoiceTaps < 0 || g.SignallingTaps < 0 {
			errs = append(errs, fmt.Errorf("%s: voice_taps and signalling_taps must be >= 0", path))
		}
		switch g.TunerStrategy {
		case "", "auto", "ddc", "polyphase":
		default:
			errs = append(errs, fmt.Errorf("%s: tuner_strategy must be auto|ddc|polyphase, got %q", path, g.TunerStrategy))
		}
		// The channel rules (system exists, protocol supported, trunked
		// channel on a declared control channel, conventional field rules,
		// duplicates) are the wideband device's. Check them on the whole
		// list against a device wide enough to hold it, so an error names
		// the group's own channel index; the per-member span check runs on
		// the expanded members below.
		if err := validateGroupChannels(i, g, c.Trunking.Systems); err != nil {
			errs = append(errs, err)
		}
	}
	for i, d := range c.SDR.Devices {
		name := strings.TrimSpace(d.Group)
		if name == "" {
			continue
		}
		path := fmt.Sprintf("sdr.devices[%d]", i)
		if _, ok := groupIdx[name]; !ok {
			errs = append(errs, fmt.Errorf("%s: group %q is not declared in sdr.groups", path, name))
			continue
		}
		if d.Role != "wideband" {
			errs = append(errs, fmt.Errorf("%s: a member of sdr.groups %q must be role: wideband (got %q)", path, name, d.Role))
		}
		if d.Serial == "" {
			errs = append(errs, fmt.Errorf("%s: a group member needs a serial", path))
		}
		var set []string
		if len(d.Channels) > 0 {
			set = append(set, "channels")
		}
		if len(d.Conventional) > 0 {
			set = append(set, "conventional")
		}
		if d.VoiceTaps != 0 {
			set = append(set, "voice_taps")
		}
		if d.SignallingTaps != 0 {
			set = append(set, "signalling_taps")
		}
		if d.TunerStrategy != "" {
			set = append(set, "tuner_strategy")
		}
		if d.SampleRate != 0 {
			set = append(set, "sample_rate")
		}
		if len(set) > 0 {
			errs = append(errs, fmt.Errorf("%s: %s belong on sdr.groups %q, not on a member (members share the group's plan and sdr.sample_rate; a member may set only its own gain/ppm/bias_tee/center_freq_hz)",
				path, strings.Join(set, ", "), name))
		}
	}
	if len(errs) > 0 {
		return errs
	}
	devs, plans, perrs := expandSDRGroups(c)
	errs = append(errs, perrs...)
	for _, p := range plans {
		for _, m := range p.Members {
			if err := validateWidebandDevice(m.DeviceIndex, devs[m.DeviceIndex], c.SDR.SampleRate, c.Trunking.Systems); err != nil {
				errs = append(errs, fmt.Errorf("sdr.groups %q member: %w", p.Name, err))
			}
		}
	}
	return errs
}

// validateGroupChannels runs the wideband channel rules over a group's whole
// channel list. The device it builds is centred on the list and given a rate
// wide enough to hold it, so only the span check is skipped; the error paths
// are rewritten to the group's.
func validateGroupChannels(idx int, g SDRGroupConfig, systems []SystemConfig) error {
	if len(g.Channels) == 0 && len(g.Conventional) == 0 {
		return nil
	}
	lo, hi := uint32(math.MaxUint32), uint32(0)
	for _, ch := range g.Channels {
		lo, hi = min(lo, ch.FrequencyHz), max(hi, ch.FrequencyHz)
	}
	for _, ch := range g.Conventional {
		lo, hi = min(lo, ch.FrequencyHz), max(hi, ch.FrequencyHz)
	}
	center := uint32((uint64(lo) + uint64(hi)) / 2)
	if center == 0 {
		center = 1
	}
	span := float64(hi-lo) + 1
	rate := uint32(math.Min(math.Ceil(span/(1-2*widebandGuardFrac))+1_000, math.MaxUint32))
	const sentinel = -1
	d := DeviceConfig{
		Serial: "group", Role: "wideband", Group: g.Name, CenterFreqHz: center,
		Channels: g.Channels, Conventional: g.Conventional,
	}
	err := validateWidebandDevice(sentinel, d, rate, systems)
	if err == nil {
		return nil
	}
	msg := strings.ReplaceAll(err.Error(), fmt.Sprintf("sdr.devices[%d]", sentinel), fmt.Sprintf("sdr.groups[%d]", idx))
	return errors.New(msg)
}

// sdrGroupNamed reports whether name is an sdr.groups entry.
func (c Config) sdrGroupNamed(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	for _, g := range c.SDR.Groups {
		if g.Name == name {
			return true
		}
	}
	return false
}

// SDRGroupMemberSerials returns the serials of the sdr.devices entries that
// name the group, in config order.
func (c Config) SDRGroupMemberSerials(name string) []string {
	var out []string
	for _, d := range c.SDR.Devices {
		if strings.TrimSpace(d.Group) == name && name != "" {
			out = append(out, d.Serial)
		}
	}
	return out
}

// BasebandRecordEntries is baseband.record with every entry whose serial
// names an sdr.groups entry replaced by one entry per member (same dir,
// format and tap). An entry for a serial that is also listed explicitly keeps
// the explicit one.
func (c Config) BasebandRecordEntries() []BasebandRecordConfig {
	var out []BasebandRecordConfig
	explicit := map[string]bool{}
	for _, r := range c.Baseband.Record {
		if !c.sdrGroupNamed(r.Serial) {
			explicit[r.Serial] = true
		}
	}
	for _, r := range c.Baseband.Record {
		if !c.sdrGroupNamed(r.Serial) {
			out = append(out, r)
			continue
		}
		for _, s := range c.SDRGroupMemberSerials(strings.TrimSpace(r.Serial)) {
			if explicit[s] {
				continue
			}
			m := r
			m.Serial = s
			out = append(out, m)
		}
	}
	return out
}
