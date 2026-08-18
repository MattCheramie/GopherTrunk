package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// scanMessage is one object of govulncheck's streaming JSON output. Exactly
// one of the fields is set per message; unknown message kinds are ignored so
// a new govulncheck message type cannot break the gate.
type scanMessage struct {
	Config  *json.RawMessage `json:"config"`
	OSV     *osvEntry        `json:"osv"`
	Finding *finding         `json:"finding"`
}

type osvEntry struct {
	ID      string `json:"id"`
	Summary string `json:"summary"`
}

// finding is govulncheck's per-callsite result. A SYMBOL-LEVEL finding — the
// scanner proved the code calls the vulnerable function — carries a trace
// entry with a non-empty Function; module- and package-level findings do not,
// and govulncheck itself does not fail the plain run on those.
type finding struct {
	OSV   string       `json:"osv"`
	Trace []traceFrame `json:"trace"`
}

type traceFrame struct {
	Module   string `json:"module"`
	Version  string `json:"version"`
	Package  string `json:"package"`
	Function string `json:"function"`
}

func (f *finding) symbolLevel() bool {
	for _, t := range f.Trace {
		if t.Function != "" {
			return true
		}
	}
	return false
}

// registry is the checked-in acceptance file. Every entry is a risk the
// maintainer has explicitly accepted, in writing, for a bounded time.
type registry struct {
	Accepted []acceptance `json:"accepted"`
}

type acceptance struct {
	// ID is the Go vulnerability database ID, e.g. "GO-2026-6115".
	ID string `json:"id"`
	// Reason is the written justification. Empty is invalid: an acceptance
	// nobody bothered to justify is not an acceptance.
	Reason string `json:"reason"`
	// Added is informational, the date the entry landed (YYYY-MM-DD).
	Added string `json:"added"`
	// ReviewBy time-boxes the acceptance (YYYY-MM-DD). After this date the
	// entry stops applying and the gate fails until a human re-reviews —
	// the mechanism that keeps "accepted" from decaying into "forgotten".
	ReviewBy string `json:"review_by"`
}

// Gate reads a govulncheck JSON stream and the acceptance registry and
// returns the human-readable report plus the process exit code (0 pass,
// 3 blocking findings). Operational problems — unreadable stream, no config
// message, malformed registry — return an error instead: a broken scan must
// fail loudly rather than pass quietly.
func Gate(scan io.Reader, registryJSON []byte, now time.Time) (string, int, error) {
	var reg registry
	if err := json.Unmarshal(registryJSON, &reg); err != nil {
		return "", 0, fmt.Errorf("parse acceptance registry: %w", err)
	}
	accepted := make(map[string]acceptance, len(reg.Accepted))
	for _, a := range reg.Accepted {
		if a.ID == "" || strings.TrimSpace(a.Reason) == "" || a.ReviewBy == "" {
			return "", 0, fmt.Errorf("acceptance registry entry %+v: id, reason and review_by are all required", a)
		}
		if _, err := time.Parse("2006-01-02", a.ReviewBy); err != nil {
			return "", 0, fmt.Errorf("acceptance %s: bad review_by %q: %w", a.ID, a.ReviewBy, err)
		}
		accepted[a.ID] = a
	}

	dec := json.NewDecoder(scan)
	sawConfig := false
	summaries := map[string]string{}
	found := map[string]finding{} // one representative symbol-level finding per OSV id
	for {
		var m scanMessage
		if err := dec.Decode(&m); err == io.EOF {
			break
		} else if err != nil {
			return "", 0, fmt.Errorf("parse govulncheck stream: %w", err)
		}
		switch {
		case m.Config != nil:
			sawConfig = true
		case m.OSV != nil:
			summaries[m.OSV.ID] = m.OSV.Summary
		case m.Finding != nil && m.Finding.symbolLevel():
			if _, ok := found[m.Finding.OSV]; !ok {
				found[m.Finding.OSV] = *m.Finding
			}
		}
	}
	if !sawConfig {
		return "", 0, fmt.Errorf("scan stream carries no govulncheck config message — " +
			"the scan did not run; refusing to treat an empty stream as a clean result")
	}

	ids := make([]string, 0, len(found))
	for id := range found {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var b strings.Builder
	blocking := 0
	for _, id := range ids {
		f := found[id]
		mod := ""
		if len(f.Trace) > 0 {
			mod = f.Trace[0].Module + "@" + f.Trace[0].Version
		}
		a, ok := accepted[id]
		if !ok {
			blocking++
			fmt.Fprintf(&b, "BLOCKED  %s (%s): %s\n", id, mod, summaries[id])
			fmt.Fprintf(&b, "         not in the acceptance registry. Per ci.yml policy: upgrade the\n")
			fmt.Fprintf(&b, "         dependency, substitute it, or add an entry with a written\n")
			fmt.Fprintf(&b, "         justification and review_by date to vulncheck-accepted.json.\n")
			continue
		}
		reviewBy, _ := time.Parse("2006-01-02", a.ReviewBy)
		if now.After(reviewBy.Add(24 * time.Hour)) {
			blocking++
			fmt.Fprintf(&b, "EXPIRED  %s (%s): acceptance lapsed on %s and must be re-reviewed —\n", id, mod, a.ReviewBy)
			fmt.Fprintf(&b, "         re-check for a fixed release, then renew or remove the entry.\n")
			continue
		}
		fmt.Fprintf(&b, "ACCEPTED %s (%s) until %s: %s\n", id, mod, a.ReviewBy, a.Reason)
	}
	for id, a := range accepted {
		if _, ok := found[id]; !ok {
			fmt.Fprintf(&b, "NOTE     %s is in the acceptance registry but the scan no longer reports\n", id)
			fmt.Fprintf(&b, "         it (fixed, withdrawn, or the call was removed) — the entry added\n")
			fmt.Fprintf(&b, "         %s can likely be deleted.\n", a.Added)
		}
	}

	switch {
	case blocking > 0:
		fmt.Fprintf(&b, "vulncheck: %d blocking finding(s)\n", blocking)
		return b.String(), 3, nil
	case len(ids) > 0:
		fmt.Fprintf(&b, "vulncheck: all %d symbol-level finding(s) covered by documented acceptances\n", len(ids))
	default:
		fmt.Fprintf(&b, "vulncheck: no symbol-level findings\n")
	}
	return b.String(), 0, nil
}
