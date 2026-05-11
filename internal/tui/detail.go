package tui

import (
	"fmt"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
)

// formatSystemDetail renders a SystemDTO as a labelled multi-line
// body for the read-only drill-in modal. Empty fields are skipped to
// keep the card compact across protocols (P25 systems carry WACN /
// SystemID / RFSS / Site; DMR / NXDN systems usually don't).
func formatSystemDetail(s client.SystemDTO) string {
	var lines []string
	lines = append(lines, "Name:     "+s.Name)
	lines = append(lines, "Protocol: "+s.Protocol)
	if len(s.ControlChannels) > 0 {
		ccs := make([]string, len(s.ControlChannels))
		for i, hz := range s.ControlChannels {
			ccs[i] = client.FormatFreqMHz(hz)
		}
		lines = append(lines, "Control:  "+strings.Join(ccs, ", "))
	} else {
		lines = append(lines, "Control:  (none configured)")
	}
	if s.WACN != 0 {
		lines = append(lines, fmt.Sprintf("WACN:     %X", s.WACN))
	}
	if s.SystemID != 0 {
		lines = append(lines, fmt.Sprintf("SystemID: %X", s.SystemID))
	}
	if s.RFSS != 0 || s.Site != 0 {
		lines = append(lines, fmt.Sprintf("RFSS/Site: %d / %d", s.RFSS, s.Site))
	}
	return strings.Join(lines, "\n")
}

// formatTalkgroupDetail renders a TalkgroupDTO as a labelled body.
// Optional fields are dropped when empty.
func formatTalkgroupDetail(tg client.TalkgroupDTO) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("ID:          %d", tg.ID))
	if tg.AlphaTag != "" {
		lines = append(lines, "Alpha:       "+tg.AlphaTag)
	}
	if tg.Description != "" {
		lines = append(lines, "Description: "+tg.Description)
	}
	if tg.Tag != "" {
		lines = append(lines, "Tag:         "+tg.Tag)
	}
	if tg.Group != "" {
		lines = append(lines, "Group:       "+tg.Group)
	}
	if tg.Mode != "" {
		lines = append(lines, "Mode:        "+tg.Mode)
	}
	lines = append(lines, fmt.Sprintf("Priority:    %d", tg.Priority))
	lock := "no"
	if tg.Lockout {
		lock = "yes"
	}
	lines = append(lines, "Lockout:     "+lock)
	return strings.Join(lines, "\n")
}
