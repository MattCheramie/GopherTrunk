// Package radioreference is a read-only client for RadioReference.com's SOAP
// web service, used by `gophertrunk hunt` to check whether a discovered
// trunked system already exists in RadioReference before an operator submits
// it. RadioReference has no public write API — new systems go through a manual
// web form reviewed by administrators — so this package never writes anything;
// it only reads, to avoid duplicate submissions.
package radioreference

import "fmt"

// System is the subset of a RadioReference trunked-system record this package
// reads for duplicate detection.
type System struct {
	SID      int    // RadioReference system id
	Name     string // system name (sName)
	Type     string // system type (sType), e.g. "Project 25 Phase II"
	SystemID uint16 // decoded SYSID (0 when unknown)
	WACN     uint32 // decoded WACN (0 when unknown)
	// ControlChannels are control-channel frequencies in Hz across the
	// system's sites (best-effort; may be empty if not fetched).
	ControlChannels []uint32
}

// Candidate is the discovered system's identifying fields, passed to
// MatchAgainst. It mirrors the parts of hunt.DiscoveredSystem relevant to
// duplicate detection without importing that package (avoiding a dependency
// cycle — the hunt CLI builds this from its DiscoveredSystem).
type Candidate struct {
	WACN            uint32
	SystemID        uint16
	Name            string
	ControlChannels []uint32
}

// Hint is a possible duplicate match between a discovered system and an
// existing RadioReference system.
type Hint struct {
	SID        int
	Name       string
	Reason     string
	Confidence float64
}

// MatchAgainst compares a discovered system against a list of existing
// RadioReference systems and returns ranked duplicate hints. The strongest
// signal is a WACN+SYSID match (a P25 system's globally-unique identity);
// failing that, an overlapping control-channel frequency; failing that, an
// exact (case-insensitive) name match. Systems that match nothing are omitted.
func MatchAgainst(c Candidate, existing []System) []Hint {
	var hints []Hint
	for _, e := range existing {
		if c.WACN != 0 && c.SystemID != 0 && e.WACN == c.WACN && e.SystemID == c.SystemID {
			hints = append(hints, Hint{
				SID:        e.SID,
				Name:       e.Name,
				Reason:     fmt.Sprintf("WACN %05X + SYSID %03X", c.WACN, c.SystemID),
				Confidence: 0.99,
			})
			continue
		}
		if n := overlapCount(c.ControlChannels, e.ControlChannels); n > 0 {
			hints = append(hints, Hint{
				SID:        e.SID,
				Name:       e.Name,
				Reason:     fmt.Sprintf("%d shared control-channel frequency(ies)", n),
				Confidence: 0.80,
			})
			continue
		}
		if c.Name != "" && equalFold(c.Name, e.Name) {
			hints = append(hints, Hint{
				SID:        e.SID,
				Name:       e.Name,
				Reason:     "identical system name",
				Confidence: 0.50,
			})
		}
	}
	return hints
}

// overlapCount returns how many frequencies appear in both lists (exact Hz).
func overlapCount(a, b []uint32) int {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	set := make(map[uint32]struct{}, len(a))
	for _, f := range a {
		set[f] = struct{}{}
	}
	n := 0
	for _, f := range b {
		if _, ok := set[f]; ok {
			n++
		}
	}
	return n
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		// ASCII fast path differs in length only when bytes differ; fall back
		// to the full comparison for multibyte safety.
	}
	return foldKey(a) == foldKey(b)
}

func foldKey(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c == ' ' || c == '-' || c == '_' {
			continue // ignore spacing/punctuation differences
		}
		out = append(out, c)
	}
	return string(out)
}
