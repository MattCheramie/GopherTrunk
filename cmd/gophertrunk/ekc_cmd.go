package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/ekc"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25"
)

// runEKC is `gophertrunk ekc`: open a KFDtool key container and list what it
// holds and what the daemon would load from it, without starting the daemon
// (issue #1305). It never prints key material.
func runEKC(args []string) {
	fs := flag.NewFlagSet("ekc", flag.ExitOnError)
	in := fs.String("in", "", "the KFDtool .ekc file")
	passwordEnv := fs.String("password-env", "", "read the password from this environment variable instead of asking")
	keyset := fs.Int("keyset", 0, "show what encryption_key_files would load with this keyset (0 = all keysets)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `gophertrunk ekc — list the keys in a KFDtool encrypted key container (.ekc).

USAGE:
  gophertrunk ekc -in keys.ekc [-password-env VAR] [-keyset N]

Asks for the container's password (or reads it from -password-env), then
lists every key: keyset, SLN, key ID, algorithm, and whether a system with
this file under encryption_key_files would load it. Key material is never
printed.

FLAGS:`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)
	rep := newReporter("ekc")
	if *in == "" || fs.NArg() > 0 {
		fs.Usage()
		rep.Fatalf(2, "-in is required")
	}
	c, err := openKeyFile("", config.EncryptionKeyFileConfig{Path: *in, PasswordEnv: *passwordEnv})
	if err != nil {
		// The reporter already prefixes "ekc:"; don't say it twice.
		rep.Fatal(1, errors.New(strings.TrimPrefix(err.Error(), "ekc: ")))
	}
	writeEKCListing(os.Stdout, c, *keyset)
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
	fmt.Fprintln(tw, "NAME\tKEYSET\tSLN\tKEY ID\tALGORITHM\tTYPE\tBYTES\tLOADED AS")
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
		fmt.Fprintf(tw, "%s\t%s\t%d\t%d (0x%X)\t%s\t%s\t%d\t%s\n", k.Name, set, k.SLN, k.KeyID, k.KeyID,
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
