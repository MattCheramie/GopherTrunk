package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/MattCheramie/GopherTrunk/internal/ekc"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25"
)

// runEKC is `gophertrunk ekc`: list, create and edit KFDtool key containers
// without starting the daemon (issue #1305; see ekc_edit.go). It never prints
// key material.
func runEKC(args []string) {
	rep := newReporter("ekc")
	if err := ekcMain(args, os.Stdout); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		// The reporter already prefixes "ekc:"; don't say it twice.
		rep.Fatal(1, errors.New(strings.TrimPrefix(err.Error(), "ekc: ")))
	}
}

// writeEKCListing prints one line per container key and the summary the
// daemon would log for it.
func writeEKCListing(w io.Writer, c *ekc.Container, keyset int) {
	load := ekcKeys(c, keyset)
	loaded := map[uint16]string{}
	for _, k := range load.keys {
		loaded[k.KeyID] = k.Algorithm
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ITEM\tNAME\tKEYSET\tSLN\tKEY ID\tALGORITHM\tTYPE\tBYTES\tLOADED AS")
	used := map[uint16]bool{}
	for _, k := range c.Keys {
		typ := "TEK"
		if k.KEK {
			typ = "KEK"
		}
		set := fmt.Sprintf("%d", k.KeysetID)
		if k.ActiveKeyset {
			set += " (active)"
		}
		as := "-"
		if alg, ok := loaded[uint16(k.KeyID)]; ok && !k.KEK && !used[uint16(k.KeyID)] && alg == ekcAlgorithm(k.AlgorithmID) &&
			(keyset == 0 || k.KeysetID == keyset) && (k.ActiveKeyset || !activeTwin(c, k, keyset)) {
			as = alg
			used[uint16(k.KeyID)] = true
		}
		fmt.Fprintf(tw, "%d\t%s\t%s\t%d\t%d (0x%X)\t%s\t%s\t%d\t%s\n", k.ID, k.Name, set, k.SLN, k.KeyID, k.KeyID,
			p25.FormatAlgorithm(uint8(k.AlgorithmID)), typ, len(k.Material), as)
	}
	_ = tw.Flush()
	fmt.Fprintf(w, "\n%d key(s) would load", len(load.keys))
	if sk := load.skipped(); sk != "" {
		fmt.Fprintf(w, "; skipped: %s", sk)
	}
	fmt.Fprintln(w)
}

// activeTwin reports whether another loadable key of an active keyset has
// k's key ID, i.e. whether k is the inactive copy ekcKeys passes over.
func activeTwin(c *ekc.Container, k ekc.Key, keyset int) bool {
	for _, o := range c.Keys {
		if o.KeyID == k.KeyID && o.ActiveKeyset && !o.KEK && ekcAlgorithm(o.AlgorithmID) != "" &&
			(keyset == 0 || o.KeysetID == keyset) {
			return true
		}
	}
	return false
}
