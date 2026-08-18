// Command vulngate applies the repo's documented-acceptance policy to a
// govulncheck scan. ci.yml's vulncheck job states the policy: a known
// vulnerability in the dependency graph means upgrading the dep, substituting
// it, or accepting the risk WITH A DOCUMENTED JUSTIFICATION. Until this tool
// existed the third option had no mechanism, so a vulnerability with no fixed
// release anywhere (GO-2026-6115, DoS in rsc.io/pdf and every fork, hit via
// the operator-initiated `import` CLI's PDF parser) turned the required check
// red on every branch with no sanctioned way through.
//
// The gate is deliberately narrow:
//
//   - Only SYMBOL-LEVEL findings (govulncheck proved the code calls the
//     vulnerable function) are considered at all, mirroring govulncheck's own
//     exit-3 criterion.
//   - A finding passes only if its OSV ID appears in the acceptance registry
//     with a non-empty reason and an unexpired review-by date. Acceptance is
//     time-boxed on purpose: a permanently silenced vulnerability is the
//     failure mode this design refuses, so past review_by the entry stops
//     working and the check goes red until a human re-reviews.
//   - An empty or config-less scan stream fails: "the scanner produced
//     nothing" must never read as "no vulnerabilities".
//
// Usage: vulngate -findings <govulncheck-json-file> -accepted <registry.json>
// Exit codes mirror govulncheck: 0 pass, 3 blocking findings, 1 operational.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	findings := flag.String("findings", "", "file holding `govulncheck -format json` output")
	accepted := flag.String("accepted", "", "acceptance registry JSON file")
	flag.Parse()
	if *findings == "" || *accepted == "" {
		fmt.Fprintln(os.Stderr, "vulngate: -findings and -accepted are required")
		os.Exit(1)
	}

	scan, err := os.Open(*findings)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vulngate: %v\n", err)
		os.Exit(1)
	}
	defer scan.Close()
	reg, err := os.ReadFile(*accepted)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vulngate: %v\n", err)
		os.Exit(1)
	}

	report, code, err := Gate(scan, reg, time.Now())
	if err != nil {
		fmt.Fprintf(os.Stderr, "vulngate: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(report)
	os.Exit(code)
}
