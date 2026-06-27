// Package tools registers the generic (cipher-agnostic) cryptolab tools —
// statistical triage, keyspace brute force, LFSR/keystream analysis, CRC
// parameter recovery, and analog voice descrambling — into the cryptolab
// registry. Each tool is a thin wrapper that parses its flags and calls the
// corresponding engine package. Subject-specific tools live separately under
// subjects/.
package tools

import (
	"flag"
	"fmt"
	"os"
)

// readFileArg parses a flag set that includes a required -in path and
// returns the file contents.
func readInput(path string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("-in is required")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return b, nil
}

// newFlagSet builds a ContinueOnError flag set so a parse error surfaces as
// a returned error (the CLI reports it) rather than os.Exit.
func newFlagSet(name string) *flag.FlagSet {
	return flag.NewFlagSet("cryptolab "+name, flag.ContinueOnError)
}
