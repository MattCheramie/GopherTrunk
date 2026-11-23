package main

import (
	"fmt"
	"strings"

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
