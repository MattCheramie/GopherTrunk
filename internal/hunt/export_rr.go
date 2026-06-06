package hunt

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// rrSubmitURL is the RadioReference "submit a new system" entry point.
// RadioReference has no public write API — new systems go through this web
// form, reviewed by Global administrators — so the submission package links
// here rather than POSTing anything.
const rrSubmitURL = "https://www.radioreference.com/db/submit/"

// writeRR renders a human-readable RadioReference.com submission package
// (Markdown) the operator can paste into the RR Submit form. When hints are
// supplied (from the optional read-only RR API duplicate check) they are
// surfaced at the top so the operator doesn't submit a duplicate.
func writeRR(w io.Writer, sys *DiscoveredSystem, hints []DuplicateHint) error {
	p := &errWriter{w: w}

	p.printf("# RadioReference submission: %s\n\n", sys.DisplayName())
	p.printf("_Discovered by GopherTrunk. Review every field before submitting; " +
		"a blind discovery cannot name talkgroups or confirm site geography._\n\n")

	// Duplicate warnings first.
	if len(hints) > 0 {
		p.printf("## ⚠️ Possible existing systems\n\n")
		p.printf("A RadioReference lookup found system(s) that may already cover this. " +
			"**Verify before submitting.**\n\n")
		sorted := append([]DuplicateHint(nil), hints...)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].Confidence > sorted[j].Confidence })
		for _, h := range sorted {
			p.printf("- SID **%d** — %s (matched on %s, confidence %.0f%%)\n",
				h.SID, nonEmpty(h.Name, "(unnamed)"), h.Reason, h.Confidence*100)
		}
		p.printf("\n")
	} else {
		p.printf("_No matching system was found in RadioReference (or the duplicate " +
			"check was skipped — supply an API key to enable it)._\n\n")
	}

	// Identity.
	p.printf("## System\n\n")
	p.printf("| Field | Value |\n|---|---|\n")
	p.printf("| Name | %s |\n", sys.DisplayName())
	p.printf("| Type / protocol | %s |\n", nonEmpty(sys.Protocol, "(unknown)"))
	if sys.WACN != 0 {
		p.printf("| WACN | %05X (hex) / %d (dec) |\n", sys.WACN, sys.WACN)
	}
	if sys.SystemID != 0 {
		p.printf("| System ID (SYSID) | %03X (hex) / %d (dec) |\n", sys.SystemID, sys.SystemID)
	}
	if sys.NAC != 0 {
		p.printf("| NAC | %03X (hex) |\n", sys.NAC)
	}
	for _, k := range identityKeys {
		if k == "WACN" || k == "SystemID" || k == "NAC" {
			continue
		}
		if v, ok := sys.Identity[k]; ok {
			p.printf("| %s | %v |\n", k, v)
		}
	}
	if sys.Location != "" {
		p.printf("| Location | %s |\n", sys.Location)
	}
	if sys.County != "" {
		p.printf("| County | %s |\n", sys.County)
	}
	if sys.State != "" {
		p.printf("| State | %s |\n", sys.State)
	}
	p.printf("| Identification confidence | %.0f%% |\n", sys.Confidence*100)
	p.printf("\n")

	// Sites + control channels.
	p.printf("## Sites & control channels\n\n")
	if len(sys.Sites) == 0 {
		p.printf("_No sites observed._\n\n")
	}
	for _, st := range sys.Sites {
		name := st.SiteName
		if name == "" {
			name = fmt.Sprintf("Site %d-%d", st.RFSS, st.SiteID)
		}
		p.printf("### %s (RFSS %d, Site %d)\n\n", name, st.RFSS, st.SiteID)
		if st.County != "" {
			p.printf("County: %s\n\n", st.County)
		}
		p.printf("Control channels (MHz):\n\n")
		for _, ch := range st.ControlChannels {
			flag := ""
			if ch.IsControl {
				flag = " (control)"
			}
			p.printf("- %s%s\n", formatMHz(ch.FrequencyHz), flag)
		}
		if len(st.Neighbors) > 0 {
			p.printf("\nAdjacent sites: ")
			refs := make([]string, 0, len(st.Neighbors))
			for _, n := range st.Neighbors {
				refs = append(refs, fmt.Sprintf("RFSS %d/Site %d", n.RFSS, n.Site))
			}
			p.printf("%s\n", strings.Join(refs, ", "))
		}
		p.printf("\n")
	}

	// Band plan.
	if len(sys.BandPlan) > 0 {
		p.printf("## Band plan\n\n")
		p.printf("| Channel ID | Base (MHz) | Spacing (kHz) | TX offset (MHz) |\n|---|---|---|---|\n")
		for _, e := range sys.BandPlan {
			p.printf("| %d | %s | %.4g | %.4g |\n",
				e.ChannelID,
				formatMHz(uint32(e.BaseHz)),
				float64(e.SpacingHz)/1000,
				float64(e.TxOffsetHz)/1_000_000)
		}
		p.printf("\n")
	}

	// Talkgroups.
	p.printf("## Observed talkgroups\n\n")
	if len(sys.Talkgroups) == 0 {
		p.printf("_None observed on the control channel during the hunt._\n\n")
	} else {
		p.printf("Observed live on the control channel — names/descriptions are unknown " +
			"and must be added during submission.\n\n")
		p.printf("| Decimal | Hex | Observed grants | Encrypted |\n|---|---|---|---|\n")
		for _, tg := range sys.Talkgroups {
			enc := ""
			if tg.Encrypted {
				enc = "yes"
			}
			p.printf("| %d | %s | %d | %s |\n", tg.Dec, tg.Hex, tg.Count, enc)
		}
		p.printf("\n")
	}

	p.printf("## Submit\n\n")
	p.printf("RadioReference has no public write API; new systems are added via the " +
		"web Submit form and reviewed by a database administrator:\n\n")
	p.printf("%s\n", rrSubmitURL)

	return p.err
}

func nonEmpty(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

// errWriter collects the first write error so the renderer stays linear.
type errWriter struct {
	w   io.Writer
	err error
}

func (e *errWriter) printf(format string, args ...any) {
	if e.err != nil {
		return
	}
	_, e.err = fmt.Fprintf(e.w, format, args...)
}
