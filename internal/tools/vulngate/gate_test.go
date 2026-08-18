package main

import (
	"strings"
	"testing"
	"time"
)

// A stream shaped like the real `govulncheck -format json` output for the
// GO-2026-6115 finding that motivated this gate (trimmed to the fields the
// gate reads).
const streamWithFinding = `
{"config":{"protocol_version":"v1.0.0","scanner_name":"govulncheck"}}
{"progress":{"message":"Scanning your code..."}}
{"osv":{"id":"GO-2026-6115","summary":"Multiple denial of service vulnerabilities in rsc.io/pdf and forks"}}
{"finding":{"osv":"GO-2026-6115","trace":[{"module":"github.com/ledongthuc/pdf","version":"v0.0.0-20250511090121-5959a4027728","package":"github.com/ledongthuc/pdf","function":"Open"},{"module":"github.com/MattCheramie/GopherTrunk","package":"github.com/MattCheramie/GopherTrunk/cmd/gophertrunk","function":"extractPDFRows"}]}}
`

const cleanStream = `
{"config":{"protocol_version":"v1.0.0","scanner_name":"govulncheck"}}
{"progress":{"message":"Scanning your code..."}}
`

// Module-level only: govulncheck's plain run does not fail on these, so the
// gate must not either.
const moduleLevelStream = `
{"config":{"protocol_version":"v1.0.0"}}
{"osv":{"id":"GO-2026-9999","summary":"required but not called"}}
{"finding":{"osv":"GO-2026-9999","trace":[{"module":"example.com/dep","version":"v1.0.0"}]}}
`

const registryAccepting6115 = `{
  "accepted": [
    {
      "id": "GO-2026-6115",
      "reason": "DoS-only in the operator-initiated import CLI; no fixed release exists",
      "added": "2026-08-18",
      "review_by": "2026-11-18"
    }
  ]
}`

var during = time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)

func TestGateBlocksUnacceptedSymbolFinding(t *testing.T) {
	report, code, err := Gate(strings.NewReader(streamWithFinding), []byte(`{"accepted":[]}`), during)
	if err != nil {
		t.Fatalf("Gate: %v", err)
	}
	if code != 3 {
		t.Fatalf("exit code %d, want 3 (blocking finding)", code)
	}
	if !strings.Contains(report, "BLOCKED  GO-2026-6115") {
		t.Errorf("report %q does not name the blocked finding", report)
	}
}

func TestGatePassesAcceptedFinding(t *testing.T) {
	report, code, err := Gate(strings.NewReader(streamWithFinding), []byte(registryAccepting6115), during)
	if err != nil {
		t.Fatalf("Gate: %v", err)
	}
	if code != 0 {
		t.Fatalf("exit code %d, want 0 — the finding is accepted with justification:\n%s", code, report)
	}
	if !strings.Contains(report, "ACCEPTED GO-2026-6115") || !strings.Contains(report, "no fixed release exists") {
		t.Errorf("report %q must show the acceptance and its written reason", report)
	}
}

// A time-boxed acceptance that lapsed must go back to blocking: silenced
// forever is the failure mode the review_by field exists to prevent.
func TestGateFailsExpiredAcceptance(t *testing.T) {
	after := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	report, code, err := Gate(strings.NewReader(streamWithFinding), []byte(registryAccepting6115), after)
	if err != nil {
		t.Fatalf("Gate: %v", err)
	}
	if code != 3 {
		t.Fatalf("exit code %d, want 3 after review_by has passed:\n%s", code, report)
	}
	if !strings.Contains(report, "EXPIRED") {
		t.Errorf("report %q must say the acceptance expired", report)
	}
}

func TestGatePassesCleanScan(t *testing.T) {
	report, code, err := Gate(strings.NewReader(cleanStream), []byte(registryAccepting6115), during)
	if err != nil {
		t.Fatalf("Gate: %v", err)
	}
	if code != 0 {
		t.Fatalf("exit code %d, want 0 on a clean scan", code)
	}
	// A stale acceptance is surfaced so the registry gets pruned, not hoarded.
	if !strings.Contains(report, "NOTE     GO-2026-6115") {
		t.Errorf("report %q should note the no-longer-reported acceptance", report)
	}
}

func TestGateIgnoresModuleLevelFindings(t *testing.T) {
	_, code, err := Gate(strings.NewReader(moduleLevelStream), []byte(`{"accepted":[]}`), during)
	if err != nil {
		t.Fatalf("Gate: %v", err)
	}
	if code != 0 {
		t.Fatalf("exit code %d, want 0 — module-level findings do not fail the plain govulncheck run either", code)
	}
}

// An empty stream means the scanner never ran; that must never read as "no
// vulnerabilities".
func TestGateRefusesStreamWithoutConfig(t *testing.T) {
	if _, _, err := Gate(strings.NewReader(""), []byte(`{"accepted":[]}`), during); err == nil {
		t.Fatal("empty scan stream passed the gate")
	}
}

// An acceptance without a written reason or review date is not an acceptance.
func TestGateRejectsUnjustifiedRegistryEntry(t *testing.T) {
	for _, reg := range []string{
		`{"accepted":[{"id":"GO-2026-6115","reason":"","added":"2026-08-18","review_by":"2026-11-18"}]}`,
		`{"accepted":[{"id":"GO-2026-6115","reason":"because","added":"2026-08-18","review_by":""}]}`,
		`{"accepted":[{"id":"GO-2026-6115","reason":"because","added":"2026-08-18","review_by":"soonish"}]}`,
	} {
		if _, _, err := Gate(strings.NewReader(cleanStream), []byte(reg), during); err == nil {
			t.Errorf("registry %s passed validation", reg)
		}
	}
}
