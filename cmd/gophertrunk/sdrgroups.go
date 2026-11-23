package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
)

// sdr.groups support. The daemon builds from a config whose group members
// are expanded into plain role: wideband devices (config.WithSDRGroupsExpanded),
// so the engines, the virtual voice taps and unplug recovery run on them
// unchanged. What remains group-aware lives here: the startup plan log, the
// member lookups the hunt / capture / spectrum paths use, and the WARN that
// names the slice of the span a lost member takes dark.

// logSDRGroupPlans writes each group's plan as one INFO line per member, plus
// the planner's warnings, so the operator can see where every stick is tuned
// and which channels it decodes.
func (d *Daemon) logSDRGroupPlans() {
	for _, p := range d.sdrGroups {
		d.log.Info("daemon: sdr group planned",
			"group", p.Name,
			"span", fmt.Sprintf("%.4f-%.4f MHz", float64(p.LowHz)/1e6, float64(p.HighHz)/1e6),
			"members", len(p.Members),
			"overlap_hz", p.OverlapHz)
		for _, m := range p.Members {
			d.log.Info("daemon: sdr group member",
				"group", p.Name,
				"serial", m.Serial,
				"center_mhz", fmt.Sprintf("%.4f", float64(m.CenterFreqHz)/1e6),
				"pinned", m.Pinned,
				"covers", fmt.Sprintf("%.4f-%.4f MHz", float64(m.LowHz)/1e6, float64(m.HighHz)/1e6),
				"channels", formatMHzList(m.Channels))
		}
		for _, w := range p.Warnings {
			d.log.Warn("daemon: " + w)
			d.addWarning(w)
		}
	}
}

func formatMHzList(fs []uint32) string {
	if len(fs) == 0 {
		return "none"
	}
	parts := make([]string, len(fs))
	for i, f := range fs {
		parts[i] = fmt.Sprintf("%.4f", float64(f)/1e6)
	}
	return strings.Join(parts, ",")
}

// sdrGroupOf returns the group the serial is a member of, or "".
func (d *Daemon) sdrGroupOf(serial string) string {
	if _, p, ok := d.sdrGroupMember(serial); ok {
		return p.Name
	}
	return ""
}

// sdrGroupMember finds the serial in the group plans.
func (d *Daemon) sdrGroupMember(serial string) (config.SDRGroupMemberPlan, config.SDRGroupPlan, bool) {
	if serial == "" {
		return config.SDRGroupMemberPlan{}, config.SDRGroupPlan{}, false
	}
	for _, p := range d.sdrGroups {
		for _, m := range p.Members {
			if m.Serial == serial {
				return m, p, true
			}
		}
	}
	return config.SDRGroupMemberPlan{}, config.SDRGroupPlan{}, false
}

// sdrGroupPlan returns the named group's plan.
func (d *Daemon) sdrGroupPlan(name string) (config.SDRGroupPlan, bool) {
	for _, p := range d.sdrGroups {
		if p.Name == name {
			return p, true
		}
	}
	return config.SDRGroupPlan{}, false
}

// warnSDRGroupMemberLost names what a group member's dead IQ stream takes
// offline: the slice of the span it owned and the channels it decoded. The
// other members keep decoding; the retry loop recovers this one.
func (d *Daemon) warnSDRGroupMemberLost(serial string, err error) {
	m, p, ok := d.sdrGroupMember(serial)
	if !ok {
		return
	}
	d.log.Warn("daemon: sdr group member lost its IQ stream — its slice of the group's span is dark until it is reacquired; the other members keep decoding",
		"group", p.Name,
		"serial", serial,
		"dark", fmt.Sprintf("%.4f-%.4f MHz", float64(m.OwnLowHz)/1e6, float64(m.OwnHighHz)/1e6),
		"channels", formatMHzList(m.Channels),
		"err", err)
}

// sdrGroupPlanFile is the JSON written beside a group's baseband recordings:
// what each member file covers, so a tool can line the per-member captures
// up by frequency (they are separate streams on separate clocks, not one
// coherent capture).
type sdrGroupPlanFile struct {
	Group        string                  `json:"group"`
	WrittenAt    time.Time               `json:"written_at"`
	SampleRateHz uint32                  `json:"sample_rate_hz"`
	LowHz        uint32                  `json:"low_hz"`
	HighHz       uint32                  `json:"high_hz"`
	OverlapHz    uint32                  `json:"overlap_hz"`
	Members      []sdrGroupPlanFileEntry `json:"members"`
}

type sdrGroupPlanFileEntry struct {
	Index        int      `json:"index"`
	Serial       string   `json:"serial"`
	CenterFreqHz uint32   `json:"center_freq_hz"`
	Pinned       bool     `json:"pinned"`
	LowHz        uint32   `json:"low_hz"`
	HighHz       uint32   `json:"high_hz"`
	OwnLowHz     uint32   `json:"own_low_hz"`
	OwnHighHz    uint32   `json:"own_high_hz"`
	ChannelsHz   []uint32 `json:"channels_hz"`
}

func newSDRGroupPlanFile(p config.SDRGroupPlan, now time.Time) sdrGroupPlanFile {
	f := sdrGroupPlanFile{
		Group: p.Name, WrittenAt: now.UTC(), SampleRateHz: p.SampleRateHz,
		LowHz: p.LowHz, HighHz: p.HighHz, OverlapHz: p.OverlapHz,
	}
	for i, m := range p.Members {
		f.Members = append(f.Members, sdrGroupPlanFileEntry{
			Index: i, Serial: m.Serial, CenterFreqHz: m.CenterFreqHz, Pinned: m.Pinned,
			LowHz: m.LowHz, HighHz: m.HighHz, OwnLowHz: m.OwnLowHz, OwnHighHz: m.OwnHighHz,
			ChannelsHz: append([]uint32{}, m.Channels...),
		})
	}
	return f
}

// writeSDRGroupRecordPlans writes <dir>/<group>_plan_<UTC>.json for every
// baseband.record entry that names a group (wideband tap).
func (d *Daemon) writeSDRGroupRecordPlans(cfg config.Config, log *slog.Logger) {
	for _, r := range cfg.Baseband.Record {
		p, ok := d.sdrGroupPlan(strings.TrimSpace(r.Serial))
		if !ok || r.TapDDC() {
			continue
		}
		now := time.Now()
		body, err := json.MarshalIndent(newSDRGroupPlanFile(p, now), "", "  ")
		if err != nil {
			continue
		}
		if err := os.MkdirAll(r.Dir, 0o755); err != nil {
			log.Warn("baseband recording: cannot create dir for group plan", "group", p.Name, "dir", r.Dir, "err", err)
			continue
		}
		path := filepath.Join(r.Dir, fmt.Sprintf("%s_plan_%s.json", p.Name, now.UTC().Format("20060102T150405Z")))
		if err := os.WriteFile(path, append(body, '\n'), 0o644); err != nil {
			log.Warn("baseband recording: cannot write group plan", "group", p.Name, "path", path, "err", err)
			continue
		}
		log.Info("baseband recording: group members record separately; plan written beside them",
			"group", p.Name, "members", len(p.Members), "plan", path)
	}
}

// sdrGroupSpectrumPrefix prefixes a group's id in the spectrum / capture
// device lists ("group:<name>").
const sdrGroupSpectrumPrefix = "group:"

// resolveSDRGroupBand returns the member of the named group whose window
// holds the whole band [lo, hi], preferring the one whose centre is nearest
// the band's middle. name may carry the "group:" prefix. isGroup is false when
// name is not a group.
func resolveSDRGroupBand(plans []config.SDRGroupPlan, name string, lo, hi uint32) (serial string, isGroup bool, err error) {
	name = strings.TrimPrefix(name, sdrGroupSpectrumPrefix)
	var plan *config.SDRGroupPlan
	for i := range plans {
		if plans[i].Name == name {
			plan = &plans[i]
			break
		}
	}
	if plan == nil {
		return "", false, nil
	}
	mid := (uint64(lo) + uint64(hi)) / 2
	m, ok := plan.MemberFor(uint32(mid), uint32((uint64(hi)-uint64(lo)+1)/2))
	if ok {
		return m.Serial, true, nil
	}
	var touching []string
	for _, mm := range plan.Members {
		if hi >= mm.LowHz && lo <= mm.HighHz {
			touching = append(touching, fmt.Sprintf("%s (%.4f-%.4f MHz)", mm.Serial, float64(mm.LowHz)/1e6, float64(mm.HighHz)/1e6))
		}
	}
	band := fmt.Sprintf("%.4f-%.4f MHz", float64(lo)/1e6, float64(hi)/1e6)
	if len(touching) == 0 {
		return "", true, fmt.Errorf("%s is outside group %q (%.4f-%.4f MHz)", band, name, float64(plan.Members[0].LowHz)/1e6, float64(plan.Members[len(plan.Members)-1].HighHz)/1e6)
	}
	return "", true, fmt.Errorf("%s straddles the seam between members %s of group %q, and the members are separate streams on separate clocks — narrow the slice(s) so they fit one member, or capture each member separately",
		band, strings.Join(touching, " and "), name)
}

// ResolveCaptureGroup implements api.CaptureGroupResolver.
func (p *captureProvider) ResolveCaptureGroup(name string, lowHz, highHz uint32) (string, bool, error) {
	return resolveSDRGroupBand(p.sp.groups, name, lowHz, highHz)
}
