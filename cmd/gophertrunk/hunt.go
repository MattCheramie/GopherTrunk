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
// unknown/undocumented trunked system, then exports the discovered system to
// standardized files plus a RadioReference.com submission package.
//
// Two modes share the same identify→decode→map→export pipeline:
//   - Offline (-in): each capture is identified, decoded, and accumulated.
//   - Live (-serial): a wideband spectrum sweep across operator-given -band(s)
//     (or an explicit -candidates list) finds carriers on the air, then each is
//     identified, decoded, and accumulated.
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

	// Live-mode flags (no -in): sweep an SDR across operator-given band(s) or
	// probe an explicit candidate list.
	serial := fs.String("serial", "", "SDR serial to sweep for a live hunt (omit -in). Empty + no -in ⇒ error")
	surveyMode := fs.Bool("survey", false, "signal-survey mode: classify every detected carrier (analog/digital/paging/trunking) and decode the conventional ones, not just trunking control channels (live only)")
	var bands repeatedString
	fs.Var(&bands, "band", "frequency band to sweep as low:high in MHz (repeatable; live mode)")
	candidatesFlag := fs.String("candidates", "", "comma-separated control-channel frequencies in MHz to probe directly (skips the sweep)")
	noSweep := fs.Bool("no-sweep", false, "with -candidates, probe only the listed frequencies (no spectrum sweep)")
	sweepDwell := fs.Duration("sweep-dwell", 0, "accumulation time per sweep step (e.g. 200ms; 0 ⇒ one frame)")
	peakThresholdDb := fs.Float64("peak-threshold-db", 10, "minimum dB above the noise floor for a carrier to count (live sweep)")
	minSpacingHz := fs.Uint("min-spacing", 6250, "minimum Hz between detected carriers (live sweep)")
	fftSize := fs.Int("fft-size", 4096, "FFT size per sweep step (power of two)")
	dwellSeconds := fs.Float64("dwell-seconds", 3, "IQ seconds captured per candidate for identify+decode (live mode)")
	gain := fs.Int("gain", -1, "SDR gain in tenths of dB for live mode (-1 = automatic)")
	ppm := fs.Int("ppm", 0, "SDR frequency correction in PPM for live mode")

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

  # LIVE: sweep the 851-869 MHz band on an SDR and map whatever it finds
  gophertrunk hunt -serial 00000001 -sample-rate 2400000 -band 851:869 -state AZ

  # LIVE: probe a known control-channel list directly (no sweep)
  gophertrunk hunt -serial 00000001 -sample-rate 2400000 -no-sweep -candidates 851.0125,853.5125

  # SURVEY: sweep a band and classify+decode every signal (analog, paging, trunking)
  gophertrunk hunt -survey -serial 00000001 -sample-rate 2400000 -band 460:470

FLAGS:`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)
	resolveVerbose(*verboseFlag, false)
	rep := newReporter("hunt")

	live := len(inPaths) == 0
	if live && *serial == "" {
		fs.Usage()
		rep.Fatalf(2, "supply -in <capture> for an offline hunt, or -serial <sdr> with -band/-candidates for a live hunt")
	}
	if *sampleRate <= 0 {
		rep.Fatalf(2, "-sample-rate must be > 0")
	}
	sampleFormat, err := siglab.ParseSampleFormat(*format)
	if err != nil {
		rep.Fatal(2, err)
	}
	if !live && len(freqs) != 0 && len(freqs) != len(inPaths) {
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

	var (
		sys     *hunt.DiscoveredSystem
		reports []hunt.CaptureReport
	)
	if live {
		sys, reports = runHuntLive(rep, huntLiveParams{
			serial:          *serial,
			survey:          *surveyMode,
			bands:           []string(bands),
			candidatesMHz:   *candidatesFlag,
			noSweep:         *noSweep,
			sampleRateHz:    *sampleRate,
			protocol:        proto,
			fftSize:         *fftSize,
			sweepDwell:      *sweepDwell,
			peakThresholdDb: *peakThresholdDb,
			minSpacingHz:    uint32(*minSpacingHz),
			dwellSeconds:    *dwellSeconds,
			autoTune:        *autoTune,
			gain:            *gain,
			ppm:             *ppm,
			name:            *name,
			state:           *state,
			county:          *county,
			location:        *location,
			minConfidence:   *minConfidence,
		})
	} else {
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
		var derr error
		sys, reports, derr = hunt.Discover(captures, hunt.DiscoverConfig{
			Name:          *name,
			State:         *state,
			County:        *county,
			Location:      *location,
			MinConfidence: *minConfidence,
		})
		if derr != nil {
			rep.Fatal(1, derr)
		}
	}

	finishHunt(rep, sys, reports, huntExportParams{
		surveyMode: *surveyMode,
		outFormats: outFormats,
		outDir:     *out,
		noRR:       *noRR,
		rr:         rrOptions{key: *rrKey, countyID: *rrCountyID, checkSIDs: rrCheckSIDs},
		commit:     *commit,
		configPath: *configPath,
		csvDir:     *csvDir,
		force:      *force,
		dryRun:     *dryRun,
	})
}

// huntExportParams carries the post-discovery export/RR/commit options shared
// by the offline and live hunt paths.
type huntExportParams struct {
	surveyMode bool
	outFormats []hunt.Format
	outDir     string
	noRR       bool
	rr         rrOptions
	commit     bool
	configPath string
	csvDir     string
	force      bool
	dryRun     bool
}

// finishHunt prints the per-candidate reports, runs the optional RadioReference
// duplicate check, writes the export files, and optionally commits the
// discovery into config.yaml. Shared by offline and live hunts.
func finishHunt(rep *diag.Reporter, sys *hunt.DiscoveredSystem, reports []hunt.CaptureReport, p huntExportParams) {
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
	if sys == nil || (len(sys.Sites) == 0 && len(sys.Talkgroups) == 0) {
		// A survey can legitimately find no trunked system (only analog/paging/
		// unclassified carriers); the inventory was already printed, so exit
		// cleanly rather than treating "no system" as a hunt failure.
		if p.surveyMode {
			fmt.Fprintln(os.Stderr, "hunt: survey complete — no trunked system to export")
			return
		}
		rep.Fatalf(1, "no trunked control channel was decoded")
	}
	fmt.Fprintf(os.Stderr, "hunt: discovered %q — %d site(s), %d talkgroup(s)\n",
		sys.DisplayName(), len(sys.Sites), len(sys.Talkgroups))

	// Optional read-only RadioReference duplicate check. Non-fatal: the export
	// still happens, just without hints.
	var hints []hunt.DuplicateHint
	if !p.noRR {
		hints = gatherRRHints(sys, p.rr)
	}

	outDir := p.outDir
	if outDir == "" {
		outDir = fmt.Sprintf("hunt-%s", time.Now().Format("20060102-150405"))
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		rep.Fatal(1, fmt.Errorf("create out dir %s: %w", outDir, err))
	}
	base := slugName(sys.DisplayName())
	for _, hf := range p.outFormats {
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

	if p.commit {
		commitDiscovery(rep, sys, p.configPath, p.csvDir, p.force, p.dryRun)
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
