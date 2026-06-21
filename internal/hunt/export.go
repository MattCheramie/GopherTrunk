package hunt

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// Format selects an export encoding for a DiscoveredSystem.
type Format int

const (
	// FormatBundle is GopherTrunk's multi-section CSV import bundle (the exact
	// reverse of cmd/gophertrunk/import_csv.go parseCSVStream — it round-trips
	// straight back into config.yaml via `import-pdf -csv`).
	FormatBundle Format = iota
	// FormatTrunkRecorder is a trunk-recorder JSON system config stanza.
	FormatTrunkRecorder
	// FormatRR is a human-readable RadioReference.com submission package.
	FormatRR
	// FormatSummary is the human-readable P25 network-configuration / activity
	// summary report (GopherTrunk's equivalent of SDRtrunk's network monitor).
	FormatSummary
)

// String renders the format as the value operators type on the CLI.
func (f Format) String() string {
	switch f {
	case FormatTrunkRecorder:
		return "trunk-recorder"
	case FormatRR:
		return "rr"
	case FormatSummary:
		return "summary"
	default:
		return "bundle"
	}
}

// FileExtension is the conventional extension for a format's output file.
func (f Format) FileExtension() string {
	switch f {
	case FormatTrunkRecorder:
		return "json"
	case FormatRR:
		return "md"
	case FormatSummary:
		return "txt"
	default:
		return "csv"
	}
}

// ParseFormat maps a -formats list value to a Format.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "bundle", "csv", "import":
		return FormatBundle, nil
	case "trunk-recorder", "trunkrecorder", "tr", "trunk_recorder":
		return FormatTrunkRecorder, nil
	case "rr", "radioreference", "submission":
		return FormatRR, nil
	case "summary", "txt", "report":
		return FormatSummary, nil
	default:
		return FormatBundle, fmt.Errorf("hunt: unknown export format %q (want bundle|trunk-recorder|rr|summary)", s)
	}
}

// Write serializes sys to w in the requested format. RRHints, when non-empty,
// are rendered into the RR submission package (ignored by the other formats).
func Write(w io.Writer, sys *DiscoveredSystem, f Format, hints []DuplicateHint) error {
	return WriteWithRRDiff(w, sys, f, hints, nil)
}

// WriteWithRRDiff is Write with an optional cross-reference diff against an
// existing RadioReference system, rendered into the RR submission package
// (ignored by the other formats). diff == nil is identical to Write.
func WriteWithRRDiff(w io.Writer, sys *DiscoveredSystem, f Format, hints []DuplicateHint, diff *RRDiff) error {
	sys.sortAll()
	switch f {
	case FormatBundle:
		return writeBundle(w, sys)
	case FormatTrunkRecorder:
		return writeTrunkRecorder(w, sys)
	case FormatRR:
		return writeRR(w, sys, hints, diff)
	case FormatSummary:
		return writeSummary(w, sys)
	default:
		return fmt.Errorf("hunt: unsupported format %d", f)
	}
}

// FreqOffset is a discovered frequency that matched a RadioReference frequency
// within tolerance but not exactly — likely a tuning/PPM offset worth checking.
type FreqOffset struct {
	DiscoveredHz uint32 `json:"discovered_hz"`
	RRHz         uint32 `json:"rr_hz"`
	DeltaHz      int64  `json:"delta_hz"` // discovered - RR
}

// RRDiff is the cross-reference of a discovered system against the matched
// RadioReference system: frequencies/talkgroups that differ or are new. It
// upgrades the RR step from "is this a duplicate?" to "what's new or off here?".
type RRDiff struct {
	SID               int          `json:"sid"`
	Name              string       `json:"name"`
	FreqOffsets       []FreqOffset `json:"freq_offsets,omitempty"`
	FreqsNotInRR      []uint32     `json:"freqs_not_in_rr,omitempty"`
	TalkgroupsNotInRR []uint32     `json:"talkgroups_not_in_rr,omitempty"`
}

// Empty reports whether the diff found nothing worth reporting.
func (d *RRDiff) Empty() bool {
	return d == nil || (len(d.FreqOffsets) == 0 && len(d.FreqsNotInRR) == 0 && len(d.TalkgroupsNotInRR) == 0)
}

// rrFreqToleranceHz is how far a discovered control/voice frequency may sit from
// a RadioReference frequency and still be considered the same channel (PPM drift
// + RR rounding). 5 kHz stays below the tight 6.25 kHz channel raster so adjacent
// channels aren't paired.
const rrFreqToleranceHz = 5_000

// DiffAgainstRR compares the system's discovered control/secondary frequencies
// and talkgroups against a flattened RadioReference frequency/talkgroup list
// (the CLI flattens FullSystem into these slices, keeping this package free of
// the radioreference import). Frequencies are paired greedily by nearest unused
// RR frequency: an exact pair is silent, a pair within rrFreqToleranceHz is an
// offset, and anything farther (or unmatched) is "not in RR".
func DiffAgainstRR(sys *DiscoveredSystem, sid int, name string, rrFreqs, rrTGs []uint32) RRDiff {
	diff := RRDiff{SID: sid, Name: name}

	// Discovered frequencies: control channels + secondary control channels.
	seen := map[uint32]struct{}{}
	var disc []uint32
	add := func(hz uint32) {
		if hz == 0 {
			return
		}
		if _, ok := seen[hz]; !ok {
			seen[hz] = struct{}{}
			disc = append(disc, hz)
		}
	}
	for _, st := range sys.Sites {
		for _, ch := range st.ControlChannels {
			if ch.IsControl {
				add(ch.FrequencyHz)
			}
		}
		for _, s := range st.Secondary {
			add(s)
		}
	}
	sort.Slice(disc, func(i, j int) bool { return disc[i] < disc[j] })

	rr := append([]uint32(nil), rrFreqs...)
	sort.Slice(rr, func(i, j int) bool { return rr[i] < rr[j] })
	used := make([]bool, len(rr))
	for _, d := range disc {
		best := -1
		var bestAbs int64 = -1
		var bestDelta int64
		for i, f := range rr {
			if used[i] {
				continue
			}
			delta := int64(d) - int64(f)
			ad := delta
			if ad < 0 {
				ad = -ad
			}
			if best < 0 || ad < bestAbs {
				best, bestAbs, bestDelta = i, ad, delta
			}
		}
		switch {
		case best < 0 || bestAbs > rrFreqToleranceHz:
			diff.FreqsNotInRR = append(diff.FreqsNotInRR, d)
		case bestAbs == 0:
			used[best] = true // exact match — nothing to report
		default:
			used[best] = true
			diff.FreqOffsets = append(diff.FreqOffsets, FreqOffset{DiscoveredHz: d, RRHz: rr[best], DeltaHz: bestDelta})
		}
	}

	rrtg := make(map[uint32]struct{}, len(rrTGs))
	for _, tg := range rrTGs {
		rrtg[tg] = struct{}{}
	}
	for _, tg := range sys.Talkgroups {
		if _, ok := rrtg[tg.Dec]; !ok {
			diff.TalkgroupsNotInRR = append(diff.TalkgroupsNotInRR, tg.Dec)
		}
	}
	return diff
}

// DuplicateHint is a possible pre-existing RadioReference system match,
// surfaced by an optional read-only RR API lookup. It is rendered into the RR
// submission package so an operator doesn't submit a duplicate.
type DuplicateHint struct {
	SID        int     `json:"sid"`        // RadioReference system id
	Name       string  `json:"name"`       // existing system name
	Reason     string  `json:"reason"`     // why it matched (WACN+SYSID, overlapping CC, name/county)
	Confidence float64 `json:"confidence"` // 0..1
}
