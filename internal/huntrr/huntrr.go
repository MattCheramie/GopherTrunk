// Package huntrr cross-references a discovered trunked system against
// RadioReference: duplicate-system hints plus a frequency/talkgroup diff
// (offsets and items not yet in RR). It bridges the hunt package (which must
// stay free of the radioreference import) and the radioreference client, so the
// CLI and the daemon/web run identical logic. The diff math itself lives in
// hunt (DiffAgainstRR, over primitive slices); this package only orchestrates
// the RR lookups and flattening.
package huntrr

import (
	"context"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/hunt"
	"github.com/MattCheramie/GopherTrunk/internal/radioreference"
)

// diffMinConfidence is the match confidence above which fetching the full RR
// system for a detailed diff is worthwhile (the WACN+SYSID / control-overlap
// tiers of radioreference.MatchAgainst).
const diffMinConfidence = 0.80

// Result is the RadioReference cross-reference for one discovered system.
type Result struct {
	Hints    []hunt.DuplicateHint // ranked possible existing systems
	Diff     *hunt.RRDiff         // freq/talkgroup diff vs the strongest match, or nil
	Compared int                  // how many existing RR systems were compared against
}

// Gather collects candidate existing systems (the explicit checkSIDs first,
// then every system registered in countyID when non-zero), matches the
// discovered system against them, and — for the strongest confident match —
// fetches its full detail and diffs frequencies/talkgroups. Every RR call is
// best-effort: a per-call error is passed to onErr (which may be nil) and that
// source is skipped, never aborting the whole gather.
func Gather(ctx context.Context, client *radioreference.Client, sys *hunt.DiscoveredSystem, countyID int, checkSIDs []int, onErr func(error)) Result {
	report := func(err error) {
		if onErr != nil && err != nil {
			onErr(err)
		}
	}

	var existing []radioreference.System
	seen := map[int]bool{}
	add := func(sid int) {
		if sid == 0 || seen[sid] {
			return
		}
		seen[sid] = true
		d, err := client.GetTrsDetails(ctx, sid)
		if err != nil {
			report(fmt.Errorf("getTrsDetails(%d): %w", sid, err))
			return
		}
		existing = append(existing, d)
	}
	for _, sid := range checkSIDs {
		add(sid)
	}
	if countyID != 0 {
		briefs, err := client.GetCountyInfo(ctx, countyID)
		if err != nil {
			report(fmt.Errorf("getCountyInfo(%d): %w", countyID, err))
		}
		for _, b := range briefs {
			add(b.SID)
		}
	}

	cand := radioreference.Candidate{
		WACN:     sys.WACN,
		SystemID: sys.SystemID,
		Name:     sys.DisplayName(),
	}
	for _, st := range sys.Sites {
		for _, ch := range st.ControlChannels {
			if ch.IsControl {
				cand.ControlChannels = append(cand.ControlChannels, ch.FrequencyHz)
			}
		}
	}

	hints := radioreference.MatchAgainst(cand, existing)
	res := Result{Compared: len(existing)}
	for _, h := range hints {
		res.Hints = append(res.Hints, hunt.DuplicateHint{SID: h.SID, Name: h.Name, Reason: h.Reason, Confidence: h.Confidence})
	}
	res.Diff = buildDiff(ctx, client, sys, hints, report)
	return res
}

// buildDiff fetches the strongest confident hint's full RR system and diffs the
// discovered system against it. Returns nil on any error, when no hint clears
// diffMinConfidence, or when the diff is empty.
func buildDiff(ctx context.Context, client *radioreference.Client, sys *hunt.DiscoveredSystem, hints []radioreference.Hint, report func(error)) *hunt.RRDiff {
	var top radioreference.Hint
	for _, h := range hints {
		if h.SID != 0 && h.Confidence >= diffMinConfidence && h.Confidence > top.Confidence {
			top = h
		}
	}
	if top.SID == 0 {
		return nil
	}
	fs, err := client.GetFullSystem(ctx, top.SID)
	if err != nil {
		report(fmt.Errorf("getFullSystem(%d): %w", top.SID, err))
		return nil
	}
	var rrFreqs []uint32
	for _, st := range fs.Sites {
		rrFreqs = append(rrFreqs, st.Frequencies...)
	}
	rrTGs := make([]uint32, 0, len(fs.Talkgroups))
	for _, tg := range fs.Talkgroups {
		rrTGs = append(rrTGs, tg.Dec)
	}
	name := fs.Name
	if name == "" {
		name = top.Name
	}
	d := hunt.DiffAgainstRR(sys, top.SID, name, rrFreqs, rrTGs)
	if d.Empty() {
		return nil
	}
	return &d
}
