package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/diag"
	"github.com/MattCheramie/GopherTrunk/internal/hunt"
	"github.com/MattCheramie/GopherTrunk/internal/radioreference"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// runHunt is the entry point for `gophertrunk hunt`. It maps a previously
// unknown/undocumented trunked system from one or more IQ captures (each
// centered on a suspected control channel), then exports the discovered system
// to standardized files plus a RadioReference.com submission package.
//
// This is the offline path: each -in capture is identified, decoded, and
// accumulated into one DiscoveredSystem. A live spectrum-sweep front-end that
// finds the candidate control channels on the air is the planned follow-on;
// today the operator supplies the captures (e.g. from `gophertrunk capture`).
func runHunt(args []string) {
	fs := flag.NewFlagSet("hunt", flag.ExitOnError)
	verboseFlag := fs.Bool("verbose-errors", false, "print full error chain + stack on failures")

	var inPaths repeatedString
	var freqs repeatedString
	fs.Var(&inPaths, "in", "raw IQ capture of a suspected control channel (repeatable)")
	fs.Var(&freqs, "freq", "nominal center frequency in Hz for the matching -in capture (repeatable, positional)")
	format := fs.String("format", "u8", "sample format: u8 | f32")
	sampleRate := fs.Float64("sample-rate", 2_400_000, "IQ sample rate in Hz")
	protocolFlag := fs.String("protocol", "", "force a protocol for every capture (default: auto-identify). One of p25,p25-phase2,dmr,dmr-tier2,nxdn,dpmr,edacs,motorola,ltr,mpt1327,tetra,ysf,dstar")
	autoTune := fs.Bool("auto-tune", false, "estimate the dominant carrier offset and tune it to 0 Hz before demod")
	conjugate := fs.Bool("conjugate", false, "conjugate IQ (negate Q) before channelization (spectrum-inverted front-end)")
	iqCorrect := fs.Bool("iq-correct", false, "apply blind I/Q-imbalance correction to the raw IQ before decimation")
	minConfidence := fs.Float64("min-confidence", 0.40, "skip auto-identified captures below this confidence (0..1)")

	name := fs.String("name", "", "system name (default: synthesized from identity)")
	state := fs.String("state", "", "US state (2-letter) — used in the RR submission package")
	county := fs.String("county", "", "county name — used in the RR submission package")
	location := fs.String("location", "", "free-form location (e.g. \"Phoenix, AZ\")")

	out := fs.String("out", "", "output directory (default: ./hunt-<timestamp>)")
	formats := fs.String("formats", "bundle,trunk-recorder,rr", "comma-separated export formats: bundle,trunk-recorder,rr")

	noRR := fs.Bool("no-rr", false, "skip the RadioReference duplicate check even if a key is configured")
	rrKey := fs.String("rr-key", "", "RadioReference API key (else GOPHERTRUNK_RR_KEY env). Enables the read-only duplicate check.")
	rrCountyID := fs.Int("rr-county-id", 0, "RadioReference county id (ctid) to scan for existing systems")
	var rrCheckSIDs repeatedString
	fs.Var(&rrCheckSIDs, "rr-check-sid", "RadioReference system id to compare against (repeatable)")

	commit := fs.Bool("commit", false, "merge the discovered system into config.yaml (like import-pdf)")
	configPath := fs.String("config", "config.yaml", "config.yaml path for -commit")
	csvDir := fs.String("csv-dir", "", "directory for generated talkgroup CSVs on -commit (default: alongside -config)")
	force := fs.Bool("force", false, "overwrite an existing system with the same name on -commit")
	dryRun := fs.Bool("dry-run", false, "with -commit, show what would change without writing")

	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `gophertrunk hunt — discover & map an unknown trunked system, then export it.

USAGE:
  gophertrunk hunt -in <capture> [-in <capture> …] [-format u8|f32] [-sample-rate Hz]
                  [-protocol <p>] [-out <dir>] [-formats bundle,trunk-recorder,rr]
                  [-name N] [-state XX] [-county C] [-commit -config config.yaml]

EXAMPLES:
  # Map an unknown P25 system from one control-channel capture and export everything
  gophertrunk hunt -in cc.cfile -format f32 -sample-rate 2400000 -state AZ -county Maricopa

  # Fold two sites of the same system into one map, auto-identifying each
  gophertrunk hunt -in site1.cfile -freq 851012500 -in site2.cfile -freq 853512500 \
                  -format f32 -sample-rate 2400000 -name "New County P25"

  # Discover and merge straight into config.yaml
  gophertrunk hunt -in cc.u8 -sample-rate 2400000 -commit -config ./config.yaml

FLAGS:`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)
	resolveVerbose(*verboseFlag, false)
	rep := newReporter("hunt")

	if len(inPaths) == 0 {
		fs.Usage()
		rep.Fatalf(2, "at least one -in capture is required")
	}
	if *sampleRate <= 0 {
		rep.Fatalf(2, "-sample-rate must be > 0")
	}
	sampleFormat, err := siglab.ParseSampleFormat(*format)
	if err != nil {
		rep.Fatal(2, err)
	}
	if len(freqs) != 0 && len(freqs) != len(inPaths) {
		rep.Fatalf(2, "-freq given %d times but -in given %d times — supply one -freq per -in or none", len(freqs), len(inPaths))
	}

	var proto trunking.Protocol
	if *protocolFlag != "" {
		proto, err = siglab.ParseProtocolCLI(*protocolFlag)
		if err != nil {
			rep.Fatal(2, err)
		}
	}

	// Parse the requested export formats up front so a typo fails fast.
	var outFormats []hunt.Format
	for _, f := range strings.Split(*formats, ",") {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		hf, ferr := hunt.ParseFormat(f)
		if ferr != nil {
			rep.Fatal(2, ferr)
		}
		outFormats = append(outFormats, hf)
	}
	if len(outFormats) == 0 {
		rep.Fatalf(2, "-formats listed no valid formats")
	}

	// Build the capture inputs.
	captures := make([]hunt.CaptureInput, 0, len(inPaths))
	for i, p := range inPaths {
		ci := hunt.CaptureInput{
			Path:         p,
			Format:       sampleFormat,
			SampleRateHz: *sampleRate,
			AutoTune:     *autoTune,
			Conjugate:    *conjugate,
			IQCorrect:    *iqCorrect,
			Protocol:     proto,
		}
		if len(freqs) == len(inPaths) {
			hz, perr := strconv.ParseUint(strings.TrimSpace(freqs[i]), 10, 32)
			if perr != nil {
				rep.Fatalf(2, "-freq[%d] %q: %v", i, freqs[i], perr)
			}
			ci.FrequencyHz = uint32(hz)
		}
		captures = append(captures, ci)
	}

	fmt.Fprintf(os.Stderr, "hunt: mapping %d capture(s)…\n", len(captures))
	sys, reports, derr := hunt.Discover(captures, hunt.DiscoverConfig{
		Name:          *name,
		State:         *state,
		County:        *county,
		Location:      *location,
		MinConfidence: *minConfidence,
	})
	if derr != nil {
		rep.Fatal(1, derr)
	}

	// Per-capture progress so the operator sees what locked vs. was skipped.
	for _, r := range reports {
		switch {
		case r.Error != "":
			fmt.Fprintf(os.Stderr, "hunt:   %s — ERROR: %s\n", r.Path, r.Error)
		case r.Skipped:
			fmt.Fprintf(os.Stderr, "hunt:   %s — skipped (%s)\n", r.Path, r.SkipReason)
		default:
			fmt.Fprintf(os.Stderr, "hunt:   %s — %s, locked=%v, +%d talkgroups\n",
				r.Path, r.Protocol, r.Locked, r.Talkgroups)
		}
	}
	if len(sys.Sites) == 0 && len(sys.Talkgroups) == 0 {
		rep.Fatalf(1, "no trunked control channel was decoded from the supplied capture(s)")
	}
	fmt.Fprintf(os.Stderr, "hunt: discovered %q — %d site(s), %d talkgroup(s)\n",
		sys.DisplayName(), len(sys.Sites), len(sys.Talkgroups))

	// Optional read-only RadioReference duplicate check. Failures here are
	// non-fatal: the export still happens, just without hints.
	var hints []hunt.DuplicateHint
	if !*noRR {
		hints = gatherRRHints(sys, rrOptions{
			key:       *rrKey,
			countyID:  *rrCountyID,
			checkSIDs: rrCheckSIDs,
		})
	}

	// Write the export files.
	outDir := *out
	if outDir == "" {
		outDir = fmt.Sprintf("hunt-%s", time.Now().Format("20060102-150405"))
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		rep.Fatal(1, fmt.Errorf("create out dir %s: %w", outDir, err))
	}
	base := slugName(sys.DisplayName())
	for _, hf := range outFormats {
		fname := filepath.Join(outDir, fmt.Sprintf("%s.%s", base, hf.FileExtension()))
		if hf == hunt.FormatRR {
			fname = filepath.Join(outDir, fmt.Sprintf("%s-radioreference.%s", base, hf.FileExtension()))
		}
		f, cerr := os.Create(fname)
		if cerr != nil {
			rep.Fatal(1, fmt.Errorf("create %s: %w", fname, cerr))
		}
		werr := hunt.Write(f, sys, hf, hints)
		cerr = f.Close()
		if werr != nil {
			rep.Fatal(1, fmt.Errorf("write %s: %w", fname, werr))
		}
		if cerr != nil {
			rep.Fatal(1, fmt.Errorf("close %s: %w", fname, cerr))
		}
		fmt.Fprintf(os.Stderr, "hunt: wrote %s (%s)\n", fname, hf)
	}

	// Optional: merge straight into config.yaml, reusing the importer's writer
	// so a discovery lands exactly like a PDF/CSV import would.
	if *commit {
		commitDiscovery(rep, sys, *configPath, *csvDir, *force, *dryRun)
	}
}

// commitDiscovery converts the discovery to the importer's parsedSystem and
// merges it into config.yaml via the shared mergeIntoConfig path.
func commitDiscovery(rep *diag.Reporter, sys *hunt.DiscoveredSystem, configPath, csvDir string, force, dryRun bool) {
	ps := discoveredToParsed(sys)
	res, err := mergeIntoConfig([]parsedSystem{ps}, mergeOptions{
		ConfigPath: configPath,
		CSVDir:     csvDir,
		Force:      force,
		DryRun:     dryRun,
	})
	if err != nil {
		rep.Fatal(1, fmt.Errorf("commit: %w", err))
	}
	for _, c := range res.Changes {
		fmt.Fprintf(os.Stderr, "hunt: %s\n", c)
	}
	if dryRun {
		fmt.Fprintln(os.Stderr, "hunt: dry-run — config.yaml not modified")
	}
}

// rrOptions carries the resolved RadioReference verify inputs.
type rrOptions struct {
	key       string
	countyID  int
	checkSIDs []string
}

// gatherRRHints runs the optional read-only RadioReference duplicate check.
// The API key comes from -rr-key or the GOPHERTRUNK_RR_KEY env var; username/
// password fall back to GOPHERTRUNK_RR_USER / GOPHERTRUNK_RR_PASS. With no key
// the check is skipped (a note is printed) and the export proceeds without
// hints. All RR errors are non-fatal.
func gatherRRHints(sys *hunt.DiscoveredSystem, opts rrOptions) []hunt.DuplicateHint {
	key := opts.key
	if key == "" {
		key = os.Getenv("GOPHERTRUNK_RR_KEY")
	}
	client, err := radioreference.NewClient(radioreference.Auth{
		AppKey:   key,
		Username: os.Getenv("GOPHERTRUNK_RR_USER"),
		Password: os.Getenv("GOPHERTRUNK_RR_PASS"),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "hunt: RadioReference duplicate check skipped (no API key configured — set -rr-key or GOPHERTRUNK_RR_KEY)")
		return nil
	}
	if opts.countyID == 0 && len(opts.checkSIDs) == 0 {
		fmt.Fprintln(os.Stderr, "hunt: RadioReference key present but no -rr-county-id or -rr-check-sid given; nothing to compare against")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Collect candidate existing systems: explicit SIDs first, then every
	// system registered in the given county (enriched with its identity).
	var existing []radioreference.System
	seen := map[int]bool{}
	addDetails := func(sid int) {
		if sid == 0 || seen[sid] {
			return
		}
		seen[sid] = true
		d, derr := client.GetTrsDetails(ctx, sid)
		if derr != nil {
			fmt.Fprintf(os.Stderr, "hunt: RadioReference getTrsDetails(%d): %v\n", sid, derr)
			return
		}
		existing = append(existing, d)
	}
	for _, s := range opts.checkSIDs {
		if sid, perr := strconv.Atoi(strings.TrimSpace(s)); perr == nil {
			addDetails(sid)
		}
	}
	if opts.countyID != 0 {
		briefs, cerr := client.GetCountyInfo(ctx, opts.countyID)
		if cerr != nil {
			fmt.Fprintf(os.Stderr, "hunt: RadioReference getCountyInfo(%d): %v\n", opts.countyID, cerr)
		}
		for _, b := range briefs {
			addDetails(b.SID)
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

	rrHints := radioreference.MatchAgainst(cand, existing)
	if len(rrHints) == 0 {
		fmt.Fprintf(os.Stderr, "hunt: RadioReference check found no existing match among %d system(s)\n", len(existing))
		return nil
	}
	out := make([]hunt.DuplicateHint, 0, len(rrHints))
	for _, h := range rrHints {
		fmt.Fprintf(os.Stderr, "hunt: possible duplicate — RR SID %d (%s): %s\n", h.SID, h.Name, h.Reason)
		out = append(out, hunt.DuplicateHint{SID: h.SID, Name: h.Name, Reason: h.Reason, Confidence: h.Confidence})
	}
	return out
}

// slugName lowercases a display name into a filesystem-safe stem.
func slugName(s string) string {
	var b strings.Builder
	prevHyphen := false
	for _, r := range strings.ToLower(s) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevHyphen = false
		default:
			if !prevHyphen && b.Len() > 0 {
				b.WriteByte('-')
				prevHyphen = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "discovered-system"
	}
	return out
}
