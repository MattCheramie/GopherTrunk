package main

import (
	"regexp"
)

// rrCSVKind marks a parsedSystem that came from one of RadioReference's two
// native CSV exports. Neither describes a whole system: the talkgroup export
// (trs_tg_<sid>.csv) has no frequencies, and the sites export
// (trs_sites_<sid>.csv) has no talkgroups.
type rrCSVKind int

const (
	rrCSVNone rrCSVKind = iota
	rrCSVTalkgroups
	rrCSVSites
)

// rrExportNameRE matches the names RadioReference gives its exports. A
// browser's "trs_tg_7715 (1).csv" still matches: only the prefix is read.
var rrExportNameRE = regexp.MustCompile(`(?i)^trs_(?:tg|sites)_(\d+)`)

// rrExportSID returns the RadioReference system id in the first of names that
// is an export's file name (or stem), "" when none is.
func rrExportSID(names ...string) string {
	for _, n := range names {
		if m := rrExportNameRE.FindStringSubmatch(filenameStem(n)); m != nil {
			return m[1]
		}
	}
	return ""
}

// pairRRNativeCSVs joins a system's two native RadioReference CSV exports
// into ONE system when both are in the batch (#1403). Imported together they
// used to become two systems, one with talkgroups and no control channels,
// the other the reverse.
//
// A talkgroup export and a sites export are joined when their file names
// carry the same system id. Failing that, the batch's one remaining
// talkgroup export and one remaining sites export are joined, unless their
// file names name two different systems. Anything else (PDFs, CSV bundles,
// a lone export, three exports with no ids) is left as it was.
//
// The joined system takes the talkgroup export's place and name, and the
// sites export's sites.
func pairRRNativeCSVs(systems []parsedSystem) []parsedSystem {
	partner := make(map[int]int) // talkgroup index → sites index
	used := make(map[int]bool)
	for i, tg := range systems {
		if tg.rrCSV != rrCSVTalkgroups || tg.rrSID == "" {
			continue
		}
		for j, st := range systems {
			if st.rrCSV == rrCSVSites && !used[j] && st.rrSID == tg.rrSID {
				partner[i], used[j] = j, true
				break
			}
		}
	}
	var loneTG, loneSites []int
	for i, s := range systems {
		switch {
		case s.rrCSV == rrCSVTalkgroups:
			if _, ok := partner[i]; !ok {
				loneTG = append(loneTG, i)
			}
		case s.rrCSV == rrCSVSites && !used[i]:
			loneSites = append(loneSites, i)
		}
	}
	if len(loneTG) == 1 && len(loneSites) == 1 {
		tg, st := systems[loneTG[0]], systems[loneSites[0]]
		if tg.rrSID == "" || st.rrSID == "" || tg.rrSID == st.rrSID {
			partner[loneTG[0]], used[loneSites[0]] = loneSites[0], true
		}
	}
	if len(partner) == 0 {
		return systems
	}
	out := make([]parsedSystem, 0, len(systems)-len(partner))
	for i, s := range systems {
		if used[i] {
			continue
		}
		if j, ok := partner[i]; ok {
			st := systems[j]
			s.Sites = st.Sites
			if s.SysID == "" {
				s.SysID = st.SysID
			}
			if s.rrSID == "" {
				s.rrSID = st.rrSID
			}
			if s.SourcePath != "" && st.SourcePath != "" {
				s.SourcePath += " + " + st.SourcePath
			}
			s.rrCSV = rrCSVNone
		}
		out = append(out, s)
	}
	return out
}
