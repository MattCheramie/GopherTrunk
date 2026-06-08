// Package refcompare cross-checks GopherTrunk's P25 Phase 1 decode against an
// external reference decoder (OP25 / DSD-FME) run on the *same* IQ capture. It
// is the third leg of the measurement harness: the synthetic sweep says how
// far GopherTrunk is from the theoretical limit, and this says how far it is
// from another real decoder on identical input — which together localize a
// weak-decode gap. If GopherTrunk trails both theory and the reference the gap
// is GopherTrunk-specific; if GopherTrunk ≈ the reference and both trail theory
// the signal is inherently marginal and no demod change recovers it.
//
// The comparison is deliberately tool-agnostic: it diffs the set of decoded
// NACs (the most robust, format-independent signal a reference emits) and a
// coarse decoded-frame yield, modelled on internal/voice/calibrate's Result +
// pass-bar. The brittle part — invoking the external binary and scraping its
// output — lives in the skip-gated integration test that calls ParseNACs; the
// math here is pure and unit-tested.
package refcompare

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Report is the outcome of comparing GopherTrunk to a reference decoder on one
// capture. NAC sets are the primary signal; frame counts are a coarse yield
// proxy.
type Report struct {
	GTNACs     []uint16 // distinct NACs GopherTrunk decoded (sorted)
	RefNACs    []uint16 // distinct NACs the reference decoded (sorted)
	CommonNACs []uint16 // NACs both agreed on (sorted)
	GTOnly     []uint16 // NACs only GopherTrunk found
	RefOnly    []uint16 // NACs only the reference found
	GTFrames   int      // GopherTrunk decoded-frame yield
	RefFrames  int      // reference decoded-frame yield
	// Agreement is |common| / |union| of the NAC sets, in [0,1]. 1.0 means
	// the two decoders saw exactly the same systems.
	Agreement float64
	// Pass is true when Agreement ≥ the minAgreement bar and (when the
	// reference reported frames) GopherTrunk's yield is at least the required
	// fraction of the reference's.
	Pass  bool
	Notes string
}

// Compare diffs the two decoders' NAC sets and frame yields into a Report.
// minAgreement is the NAC-set overlap required to pass (e.g. 1.0 = identical
// systems); minYieldFrac is the fraction of the reference's decoded-frame
// count GopherTrunk must reach (e.g. 0.8). minYieldFrac is ignored when the
// reference reported no frames (frame scraping unavailable for that tool).
func Compare(gtNACs, refNACs []uint16, gtFrames, refFrames int, minAgreement, minYieldFrac float64) Report {
	gt := uniqueSorted(gtNACs)
	ref := uniqueSorted(refNACs)
	gtSet := toSet(gt)
	refSet := toSet(ref)

	var common, gtOnly, refOnly []uint16
	for _, n := range gt {
		if refSet[n] {
			common = append(common, n)
		} else {
			gtOnly = append(gtOnly, n)
		}
	}
	for _, n := range ref {
		if !gtSet[n] {
			refOnly = append(refOnly, n)
		}
	}

	union := len(common) + len(gtOnly) + len(refOnly)
	agreement := 1.0
	if union > 0 {
		agreement = float64(len(common)) / float64(union)
	}

	r := Report{
		GTNACs: gt, RefNACs: ref, CommonNACs: common,
		GTOnly: gtOnly, RefOnly: refOnly,
		GTFrames: gtFrames, RefFrames: refFrames,
		Agreement: agreement,
	}

	r.Pass = agreement >= minAgreement
	switch {
	case !r.Pass:
		r.Notes = fmt.Sprintf("NAC-set agreement %.2f < required %.2f (GT-only=%v, ref-only=%v)",
			agreement, minAgreement, gtOnly, refOnly)
	case refFrames > 0 && float64(gtFrames) < minYieldFrac*float64(refFrames):
		r.Pass = false
		r.Notes = fmt.Sprintf("frame yield %d < %.0f%% of reference %d", gtFrames, minYieldFrac*100, refFrames)
	default:
		r.Notes = fmt.Sprintf("agreement %.2f, GT/ref frames %d/%d", agreement, gtFrames, refFrames)
	}
	return r
}

// ParseNACs extracts every NAC value from a reference decoder's textual output
// using re, whose first capture group must hold the NAC as a hex ("0x167" or
// "167") or decimal string. A nil re uses DefaultNACPattern, which matches the
// common "NAC 0x167" / "nac=0x167" renderings OP25 and DSD-FME emit. Duplicate
// and unparseable matches are dropped.
func ParseNACs(output string, re *regexp.Regexp) []uint16 {
	if re == nil {
		re = DefaultNACPattern
	}
	var out []uint16
	for _, m := range re.FindAllStringSubmatch(output, -1) {
		if len(m) < 2 {
			continue
		}
		tok := strings.TrimSpace(m[1])
		base := 10
		if strings.HasPrefix(strings.ToLower(tok), "0x") {
			tok = tok[2:]
			base = 16
		}
		v, err := strconv.ParseUint(tok, base, 16)
		if err != nil {
			continue
		}
		out = append(out, uint16(v))
	}
	return out
}

// DefaultNACPattern matches "NAC 0x167", "nac=0x167", "NAC: 167" and similar,
// capturing the hex/decimal NAC token. Case-insensitive.
var DefaultNACPattern = regexp.MustCompile(`(?i)nac[\s:=]+((?:0x)?[0-9a-f]+)`)

func uniqueSorted(in []uint16) []uint16 {
	set := toSet(in)
	out := make([]uint16, 0, len(set))
	for n := range set {
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func toSet(in []uint16) map[uint16]bool {
	s := make(map[uint16]bool, len(in))
	for _, n := range in {
		s[n] = true
	}
	return s
}
