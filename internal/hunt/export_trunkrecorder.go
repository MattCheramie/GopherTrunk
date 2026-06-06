package hunt

import (
	"encoding/json"
	"io"
	"strings"
)

// trProtocol maps GopherTrunk protocol spellings onto trunk-recorder's
// `type` enum. trunk-recorder only models a few trunked types; protocols it
// doesn't support fall back to the GopherTrunk name so the stanza is still
// informative (the operator edits it).
func trProtocol(p string) string {
	switch strings.ToLower(p) {
	case "p25", "p25-phase2":
		return "p25"
	case "dmr", "dmr-tier2", "dmr-tier3":
		return "dmr"
	default:
		return strings.ToLower(p)
	}
}

// trSystem is the subset of a trunk-recorder system config GopherTrunk can
// populate from a discovery. Fields the operator must still supply (recorders,
// modulation tuning, talkgroupsFile path) are emitted with sensible defaults.
type trSystem struct {
	ShortName       string   `json:"shortName"`
	Type            string   `json:"type"`
	ControlChannels []int64  `json:"control_channels"`
	Modulation      string   `json:"modulation"`
	TalkgroupsFile  string   `json:"talkgroupsFile,omitempty"`
	Comment         string   `json:"comment,omitempty"`
	Sites           []string `json:"-"`
}

type trConfig struct {
	Systems []trSystem `json:"systems"`
}

// writeTrunkRecorder emits a trunk-recorder JSON config stanza for the
// discovered system. All discovered control channels (across sites) are
// flattened into the single `control_channels` array trunk-recorder expects
// per system; multi-site systems are noted in the comment.
func writeTrunkRecorder(w io.Writer, sys *DiscoveredSystem) error {
	var ccs []int64
	for _, st := range sys.Sites {
		for _, ch := range st.ControlChannels {
			if ch.IsControl {
				ccs = append(ccs, int64(ch.FrequencyHz))
			}
		}
	}
	mod := "qpsk"
	if trProtocol(sys.Protocol) == "p25" {
		mod = "qpsk"
	}
	comment := "Discovered by GopherTrunk hunt. Verify control channels and add recorders/talkgroupsFile before use."
	if len(sys.Sites) > 1 {
		comment += " Multi-site system: control_channels flattens all sites; split per-site if desired."
	}
	cfg := trConfig{Systems: []trSystem{{
		ShortName:       slug(sys.DisplayName()),
		Type:            trProtocol(sys.Protocol),
		ControlChannels: ccs,
		Modulation:      mod,
		TalkgroupsFile:  slug(sys.DisplayName()) + "-talkgroups.csv",
		Comment:         comment,
	}}}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(cfg)
}

// slug lowercases a name and replaces runs of non-alphanumeric characters
// with a single hyphen — a filesystem/shortName-safe identifier.
func slug(s string) string {
	var b strings.Builder
	prevHyphen := false
	for _, r := range strings.ToLower(s) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevHyphen = false
		default:
			if !prevHyphen && b.Len() > 0 {
				b.WriteByte('-')
				prevHyphen = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
