package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/terms"
)

// termsExempt reports whether the invoked command may run without the
// terms-of-service gate: reading the version, the help text, or the
// terms themselves must never require accepting them first.
func termsExempt(args []string) bool {
	if len(args) < 2 {
		return false // bare `gophertrunk` runs the daemon
	}
	switch args[1] {
	case "version", "--version", "-v", "help", "--help", "-h", "terms":
		return true
	}
	return false
}

// requireTermsAcceptance is the install/first-run gate: every real
// command checks for a recorded acknowledgment of the current
// TERMS_OF_SERVICE.md revision before doing anything else. Acceptance
// can come from the Windows installer's Terms page (it writes the same
// marker), a prior interactive acceptance, `gophertrunk terms accept`,
// or GOPHERTRUNK_ACCEPT_TERMS=1 for unattended installs. On a TTY with
// none of those, the terms are shown and acceptance is prompted for;
// off a TTY the process exits with EX_NOPERM and instructions.
func requireTermsAcceptance() {
	dir, dirErr := terms.DefaultDir()
	if dirErr == nil && terms.IsAccepted(dir) {
		return
	}
	if terms.EnvAccepted() {
		// Persist best-effort so later runs without the env var still
		// pass; the env var alone is sufficient for this run either way.
		if dirErr == nil {
			_ = terms.Record(dir, "env")
		}
		return
	}

	if !stdinIsTerminal() {
		fmt.Fprintf(os.Stderr,
			"gophertrunk: the Terms of Service have not been acknowledged on this machine.\n"+
				"Run `gophertrunk terms` to read them, then `gophertrunk terms accept` to\n"+
				"acknowledge, or set %s=1 for unattended installs.\n", terms.EnvVar)
		os.Exit(77) // EX_NOPERM
	}

	// Interactive first run (or first run after a terms revision).
	if dirErr == nil {
		if prior := terms.AcceptedVersion(dir); prior > 0 {
			fmt.Fprintf(os.Stderr,
				"The GopherTrunk Terms of Service have changed since you last accepted them (v%d -> v%d).\n\n",
				prior, terms.Version)
		}
	}
	fmt.Print(terms.Text)
	fmt.Fprint(os.Stderr, "\nDo you accept the GopherTrunk Terms of Service? [yes/no]: ")
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "yes", "y":
	default:
		fmt.Fprintln(os.Stderr, "gophertrunk: terms not accepted; exiting.")
		os.Exit(1)
	}
	if dirErr != nil {
		fmt.Fprintf(os.Stderr,
			"warning: cannot resolve a config directory to record acceptance (%v); you will be asked again next run\n", dirErr)
		return
	}
	if err := terms.Record(dir, "interactive"); err != nil {
		fmt.Fprintf(os.Stderr,
			"warning: could not record acceptance (%v); you will be asked again next run\n", err)
		return
	}
	fmt.Fprintf(os.Stderr, "Thanks - acceptance recorded at %s.\n\n", terms.MarkerPath(dir))
}

// runTerms implements `gophertrunk terms [show|status|accept]`.
func runTerms(args []string) {
	sub := "show"
	if len(args) > 0 {
		sub = args[0]
	}
	switch sub {
	case "show":
		fmt.Print(terms.Text)
	case "status":
		dir, err := terms.DefaultDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "terms: %v\n", err)
			os.Exit(1)
		}
		if v := terms.AcceptedVersion(dir); v >= terms.Version {
			fmt.Printf("accepted: terms v%d (marker: %s)\n", v, terms.MarkerPath(dir))
		} else if v > 0 {
			fmt.Printf("outdated: terms v%d accepted, current is v%d; run `gophertrunk terms accept`\n", v, terms.Version)
			os.Exit(1)
		} else {
			fmt.Printf("not accepted: run `gophertrunk terms accept` (marker would be: %s)\n", terms.MarkerPath(dir))
			os.Exit(1)
		}
	case "accept":
		// An explicit accept command IS the acknowledgment, so it works
		// non-interactively too (provisioning scripts, Dockerfiles).
		dir, err := terms.DefaultDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "terms: %v\n", err)
			os.Exit(1)
		}
		if err := terms.Record(dir, "command"); err != nil {
			fmt.Fprintf(os.Stderr, "terms: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("accepted: terms v%d recorded at %s\n", terms.Version, terms.MarkerPath(dir))
	default:
		fmt.Fprintln(os.Stderr, "usage: gophertrunk terms [show|status|accept]")
		os.Exit(2)
	}
}
