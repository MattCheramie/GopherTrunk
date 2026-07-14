package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/diag"
	"github.com/MattCheramie/GopherTrunk/internal/gtbundle"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// runBundle dispatches `gophertrunk bundle <subcommand>`. The GopherTrunk
// Bundle (.gtb.tar.gz) is the single-file case container that packages a
// capture together with its logs, signal analysis, crypto analysis, and
// site/network mapping. This is distinct from hunt's CSV "import bundle" (that
// CSV is stored *inside* a GopherTrunk Bundle as a mapping-export artifact).
func runBundle(args []string) {
	if len(args) == 0 {
		printBundleUsage(os.Stderr)
		os.Exit(2)
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "pack":
		runBundlePack(rest)
	case "info", "ls":
		runBundleInfo(rest)
	case "verify":
		runBundleVerify(rest)
	case "extract":
		runBundleExtract(rest)
	case "add":
		runBundleAdd(rest)
	case "commit", "import":
		runBundleCommit(rest)
	case "help", "-h", "--help":
		printBundleUsage(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "bundle: unknown subcommand %q\n\n", sub)
		printBundleUsage(os.Stderr)
		os.Exit(2)
	}
}

func printBundleUsage(w *os.File) {
	fmt.Fprint(w, `gophertrunk bundle — build and inspect GopherTrunk Bundles (.gtb.tar.gz).

A GopherTrunk Bundle packages one capture-to-analysis case in a single sharable
archive: the raw IQ capture, logs, SigLab signal analysis, CryptoLab crypto
analysis, and the hunt site/network mapping, indexed by MANIFEST.yaml.

SUBCOMMANDS:
  pack      assemble a bundle from a capture and/or analysis files
  info|ls   print a bundle's manifest summary
  verify    recompute and check every file's sha256
  extract   unpack a whole bundle, or one role's files, to a directory
  add       append an artifact to an existing bundle
  commit    merge the bundle's discovered system into config.yaml

Run "gophertrunk bundle <subcommand> -h" for flags.
`)
}

// --- pack ---

func runBundlePack(args []string) {
	fs := flag.NewFlagSet("bundle pack", flag.ExitOnError)
	out := fs.String("out", "", "output bundle path (.gtb.tar.gz) (required)")
	name := fs.String("name", "", "bundle name / top-level dir (default: derived from -out)")
	intent := fs.String("intent", "general", "capture intent: cc-map | crypto | survey | general")
	title := fs.String("title", "", "human title for the case")
	source := fs.String("source", "", "free-text capture provenance")
	notes := fs.String("notes-text", "", "free-text notes stored in the manifest")

	capture := fs.String("capture", "", "raw IQ capture (.cfile/.bin) → capture-iq")
	meta := fs.String("meta", "", "capture metadata sidecar (json/yaml) → capture-meta")
	slice := fs.String("slice", "", "narrowband DDC slice (.wav) → capture-slice")
	events := fs.String("events", "", "session event log (jsonl) → log-events")
	messages := fs.String("messages", "", "human decoded-message log → log-messages")
	mappingSystem := fs.String("mapping-system", "", "discovered system (json) → mapping-system")
	survey := fs.String("survey", "", "survey stream (jsonl) → mapping-survey")
	mappingExport := multiFlag{}
	fs.Var(&mappingExport, "mapping-export", "a hunt export file (csv/json/md) → mapping-export (repeatable)")
	siglabResult := fs.String("siglab-result", "", "SigLab result (yaml/json) → siglab-result")
	siglabEvents := fs.String("siglab-events", "", "SigLab events (jsonl) → siglab-events")
	cryptoFrames := fs.String("cryptolab-frames", "", "crypto frames (jsonl) → cryptolab-frames")
	cryptoResult := fs.String("cryptolab-result", "", "CryptoLab result (yaml/json) → cryptolab-result")
	notesFile := fs.String("notes", "", "notes.md file → notes")
	fromDir := fs.String("from-dir", "", "pack a pre-laid-out directory (capture/, logs/, mapping/, …)")
	sigmf := fs.Bool("sigmf", false, "emit a SigMF .sigmf-meta sidecar from -meta (interop with SigMF tooling)")
	_ = fs.Parse(args)

	rep := newReporter("bundle pack")
	if *out == "" {
		rep.Fatalf(2, "-out is required")
	}
	nm := *name
	if nm == "" {
		nm = strings.TrimSuffix(filepath.Base(*out), gtbundle.Ext)
		nm = strings.TrimSuffix(nm, ".tar.gz")
		nm = strings.TrimSuffix(nm, ".gtb")
	}

	f, err := os.Create(*out)
	if err != nil {
		rep.Fatal(1, err)
	}
	defer f.Close()

	w, err := gtbundle.NewWriter(f, gtbundle.WriterOptions{
		Name:          nm,
		CaptureIntent: gtbundle.CaptureIntent(*intent),
		Title:         *title,
		Notes:         *notes,
		Provenance:    gtbundle.Provenance{Source: *source},
	})
	if err != nil {
		rep.Fatal(1, err)
	}

	added := 0
	add := func(role gtbundle.Role, path, producedBy string) {
		if path == "" {
			return
		}
		src, oerr := os.Open(path)
		if oerr != nil {
			rep.Fatal(1, oerr)
		}
		defer src.Close()
		if _, aerr := w.Add(role, filepath.Base(path), src, producedBy); aerr != nil {
			rep.Fatal(1, fmt.Errorf("add %s: %w", path, aerr))
		}
		added++
	}

	if *fromDir != "" {
		added += packFromDir(rep, w, *fromDir)
	}
	add(gtbundle.RoleCaptureIQ, *capture, "gophertrunk capture")
	add(gtbundle.RoleCaptureMeta, *meta, "gophertrunk capture")
	if *sigmf && *meta != "" {
		m, lerr := siglab.LoadMetadata(*meta)
		if lerr != nil {
			rep.Fatal(1, fmt.Errorf("sigmf: load metadata: %w", lerr))
		}
		if doc, ok, serr := gtbundle.MetadataToSigMF(m); serr != nil {
			rep.Fatal(1, fmt.Errorf("sigmf: %w", serr))
		} else if ok {
			if _, aerr := w.AddBytes(gtbundle.RoleCaptureSigMF, stemOf(baseName(*capture))+".sigmf-meta", doc, "gophertrunk bundle"); aerr != nil {
				rep.Fatal(1, aerr)
			}
			added++
		} else {
			fmt.Fprintln(os.Stderr, "bundle: -sigmf skipped (capture format has no SigMF datatype equivalent)")
		}
	}
	add(gtbundle.RoleCaptureSlice, *slice, "gophertrunk bundle")
	add(gtbundle.RoleLogEvents, *events, "daemon")
	add(gtbundle.RoleLogMessages, *messages, "daemon")
	add(gtbundle.RoleMappingSystem, *mappingSystem, "hunt")
	add(gtbundle.RoleMappingSurvey, *survey, "hunt")
	for _, me := range mappingExport {
		add(gtbundle.RoleMappingExport, me, "hunt")
	}
	add(gtbundle.RoleSiglabResult, *siglabResult, "siglab")
	add(gtbundle.RoleSiglabEvents, *siglabEvents, "siglab")
	add(gtbundle.RoleCryptolabFrames, *cryptoFrames, "cryptolab")
	add(gtbundle.RoleCryptolabResult, *cryptoResult, "cryptolab")
	add(gtbundle.RoleNotes, *notesFile, "operator")

	if added == 0 {
		rep.Fatalf(2, "nothing to pack — give at least one artifact (e.g. -capture) or -from-dir")
	}
	if err := w.Close(); err != nil {
		rep.Fatal(1, err)
	}
	fmt.Printf("bundle: wrote %s (%d files)\n", *out, added)
}

// packFromDir walks a laid-out directory and adds every file under a recognized
// subdirectory to the writer, inferring the role from the subdir + filename. It
// returns the count added. Unrecognized files are skipped with a note.
func packFromDir(rep *diag.Reporter, w *gtbundle.Writer, dir string) int {
	added := 0
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, rerr := filepath.Rel(dir, path)
		if rerr != nil {
			return rerr
		}
		role, ok := roleForRelPath(filepath.ToSlash(rel))
		if !ok {
			fmt.Fprintf(os.Stderr, "bundle: skipping unrecognized %s\n", rel)
			return nil
		}
		src, oerr := os.Open(path)
		if oerr != nil {
			return oerr
		}
		defer src.Close()
		if _, aerr := w.Add(role, filepath.Base(path), src, "from-dir"); aerr != nil {
			return aerr
		}
		added++
		return nil
	})
	if err != nil {
		rep.Fatal(1, fmt.Errorf("from-dir: %w", err))
	}
	return added
}

// roleForRelPath infers a role from a directory-relative path inside a laid-out
// case tree.
func roleForRelPath(rel string) (gtbundle.Role, bool) {
	base := strings.ToLower(filepath.Base(rel))
	switch {
	case strings.HasPrefix(rel, "capture/") && strings.HasSuffix(base, ".wav"):
		return gtbundle.RoleCaptureSlice, true
	case strings.HasPrefix(rel, "capture/") && (strings.Contains(base, "meta")):
		return gtbundle.RoleCaptureMeta, true
	case strings.HasPrefix(rel, "capture/"):
		return gtbundle.RoleCaptureIQ, true
	case strings.HasPrefix(rel, "logs/") && strings.HasSuffix(base, ".jsonl"):
		return gtbundle.RoleLogEvents, true
	case strings.HasPrefix(rel, "logs/"):
		return gtbundle.RoleLogMessages, true
	case strings.HasPrefix(rel, "mapping/") && base == "system.json":
		return gtbundle.RoleMappingSystem, true
	case strings.HasPrefix(rel, "mapping/") && strings.HasSuffix(base, ".jsonl"):
		return gtbundle.RoleMappingSurvey, true
	case strings.HasPrefix(rel, "mapping/"):
		return gtbundle.RoleMappingExport, true
	case strings.HasPrefix(rel, "siglab/") && strings.HasSuffix(base, ".jsonl"):
		return gtbundle.RoleSiglabEvents, true
	case strings.HasPrefix(rel, "siglab/"):
		return gtbundle.RoleSiglabResult, true
	case strings.HasPrefix(rel, "cryptolab/") && strings.HasSuffix(base, ".jsonl"):
		return gtbundle.RoleCryptolabFrames, true
	case strings.HasPrefix(rel, "cryptolab/"):
		return gtbundle.RoleCryptolabResult, true
	case base == "notes.md":
		return gtbundle.RoleNotes, true
	default:
		return "", false
	}
}

// --- info ---

func runBundleInfo(args []string) {
	fs := flag.NewFlagSet("bundle info", flag.ExitOnError)
	in := fs.String("in", "", "bundle path (.gtb.tar.gz) (required)")
	showFiles := fs.Bool("files", false, "list the full file inventory")
	_ = fs.Parse(args)

	rep := newReporter("bundle info")
	if *in == "" {
		rep.Fatalf(2, "-in is required")
	}
	r, err := gtbundle.Open(*in)
	if err != nil {
		rep.Fatal(1, err)
	}
	m := r.Manifest()
	if r.FutureSchema() {
		fmt.Fprintf(os.Stderr, "bundle: warning — schema_version %d is newer than this build supports (%d); reading best-effort\n",
			m.SchemaVersion, gtbundle.SchemaVersion)
	}
	fmt.Printf("name:     %s\n", m.Name)
	if m.Title != "" {
		fmt.Printf("title:    %s\n", m.Title)
	}
	fmt.Printf("intent:   %s\n", m.CaptureIntent)
	fmt.Printf("created:  %s\n", m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	fmt.Printf("by:       %s %s\n", m.Generator.Tool, m.Generator.Version)
	if m.Provenance.Source != "" {
		fmt.Printf("source:   %s\n", m.Provenance.Source)
	}
	if m.Provenance.Protocol != "" {
		fmt.Printf("protocol: %s\n", m.Provenance.Protocol)
	}
	fmt.Printf("files:    %d\n", len(m.Files))
	if *showFiles {
		for _, f := range m.Files {
			fmt.Printf("  %-18s %-40s %10d  %s\n", f.Role, f.Path, f.SizeBytes, f.ProducedBy)
		}
	}
}

// --- verify ---

func runBundleVerify(args []string) {
	fs := flag.NewFlagSet("bundle verify", flag.ExitOnError)
	in := fs.String("in", "", "bundle path (.gtb.tar.gz) (required)")
	quiet := fs.Bool("quiet", false, "print nothing on success")
	_ = fs.Parse(args)

	rep := newReporter("bundle verify")
	if *in == "" {
		rep.Fatalf(2, "-in is required")
	}
	r, err := gtbundle.Open(*in)
	if err != nil {
		rep.Fatal(1, err)
	}
	bad, untracked := 0, 0
	for _, v := range r.Verify() {
		switch {
		case v.Untracked:
			untracked++
			if !*quiet {
				fmt.Printf("untracked  %s\n", v.Path)
			}
		case !v.OK:
			bad++
			fmt.Fprintf(os.Stderr, "FAIL       %s: %v\n", v.Path, v.Err)
		case !*quiet:
			fmt.Printf("ok         %s\n", v.Path)
		}
	}
	if bad > 0 {
		rep.Fatalf(1, "%d file(s) failed verification", bad)
	}
	if !*quiet {
		fmt.Printf("bundle: OK (%d untracked)\n", untracked)
	}
}

// --- extract ---

func runBundleExtract(args []string) {
	fs := flag.NewFlagSet("bundle extract", flag.ExitOnError)
	in := fs.String("in", "", "bundle path (.gtb.tar.gz) (required)")
	out := fs.String("out", ".", "destination directory")
	role := fs.String("role", "", "extract only this role's files (flattened to leaf names)")
	_ = fs.Parse(args)

	rep := newReporter("bundle extract")
	if *in == "" {
		rep.Fatalf(2, "-in is required")
	}
	r, err := gtbundle.Open(*in)
	if err != nil {
		rep.Fatal(1, err)
	}
	if *role != "" {
		written, werr := r.ExtractRole(gtbundle.Role(*role), *out)
		if werr != nil {
			rep.Fatal(1, werr)
		}
		for _, p := range written {
			fmt.Println(p)
		}
		return
	}
	if err := r.Extract(*out); err != nil {
		rep.Fatal(1, err)
	}
	fmt.Printf("bundle: extracted %s → %s\n", *in, *out)
}

// --- add ---

func runBundleAdd(args []string) {
	fs := flag.NewFlagSet("bundle add", flag.ExitOnError)
	in := fs.String("in", "", "bundle path to rewrite in place (required)")
	role := fs.String("role", "", "role for the new file (required)")
	file := fs.String("file", "", "file to add (required)")
	producedBy := fs.String("produced-by", "", "tool/step that produced the file")
	_ = fs.Parse(args)

	rep := newReporter("bundle add")
	if *in == "" || *role == "" || *file == "" {
		rep.Fatalf(2, "-in, -role and -file are required")
	}
	if err := bundleAddFile(*in, gtbundle.Role(*role), *file, *producedBy); err != nil {
		rep.Fatal(1, err)
	}
	fmt.Printf("bundle: added %s (%s) → %s\n", filepath.Base(*file), *role, *in)
}

// bundleAddFile appends one artifact to an existing bundle. tar.gz cannot be
// appended in place, so it reads the bundle, re-packs it plus the new file into
// a temp archive, and renames over the original.
func bundleAddFile(bundlePath string, role gtbundle.Role, file, producedBy string) error {
	r, err := gtbundle.Open(bundlePath)
	if err != nil {
		return err
	}
	dir := filepath.Dir(bundlePath)
	staging, err := os.MkdirTemp(dir, ".gtb-add-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	if err := r.Extract(staging); err != nil {
		return err
	}
	// The extracted tree lives under staging/<name>/…; copy the new file into
	// the role's directory, then re-pack from that laid-out tree.
	caseDir := filepath.Join(staging, r.Manifest().Name)
	dst := filepath.Join(caseDir, roleSubdir(role), filepath.Base(file))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	body, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dst, body, 0o644); err != nil {
		return err
	}
	_ = producedBy // producedBy is recorded via role inference on repack

	tmpOut, err := os.CreateTemp(dir, ".gtb-out-*")
	if err != nil {
		return err
	}
	tmpName := tmpOut.Name()
	m := r.Manifest()
	w, err := gtbundle.NewWriter(tmpOut, gtbundle.WriterOptions{
		Name:          m.Name,
		CaptureIntent: m.CaptureIntent,
		Title:         m.Title,
		Notes:         m.Notes,
		Provenance:    m.Provenance,
	})
	if err != nil {
		tmpOut.Close()
		os.Remove(tmpName)
		return err
	}
	if n := packFromDirSilent(w, caseDir); n == 0 {
		w.Close()
		tmpOut.Close()
		os.Remove(tmpName)
		return fmt.Errorf("bundle add: nothing to repack")
	}
	if err := w.Close(); err != nil {
		tmpOut.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmpOut.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, bundlePath)
}

// roleSubdir returns the on-disk subdirectory a role extracts into.
func roleSubdir(role gtbundle.Role) string {
	switch role {
	case gtbundle.RoleCaptureIQ, gtbundle.RoleCaptureSlice, gtbundle.RoleCaptureMeta, gtbundle.RoleCaptureSigMF:
		return "capture"
	case gtbundle.RoleLogEvents, gtbundle.RoleLogMessages:
		return "logs"
	case gtbundle.RoleMappingSystem, gtbundle.RoleMappingSurvey, gtbundle.RoleMappingExport:
		return "mapping"
	case gtbundle.RoleSiglabResult, gtbundle.RoleSiglabEvents:
		return "siglab"
	case gtbundle.RoleCryptolabFrames, gtbundle.RoleCryptolabResult:
		return "cryptolab"
	default:
		return "."
	}
}

// packFromDirSilent is packFromDir without a reporter (used on the add repack
// path), returning the count added.
func packFromDirSilent(w *gtbundle.Writer, dir string) int {
	added := 0
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, rerr := filepath.Rel(dir, path)
		if rerr != nil {
			return rerr
		}
		role, ok := roleForRelPath(filepath.ToSlash(rel))
		if !ok {
			return nil
		}
		src, oerr := os.Open(path)
		if oerr != nil {
			return oerr
		}
		defer src.Close()
		if _, aerr := w.Add(role, filepath.Base(path), src, "repack"); aerr == nil {
			added++
		}
		return nil
	})
	return added
}

// --- commit ---

func runBundleCommit(args []string) {
	fs := flag.NewFlagSet("bundle commit", flag.ExitOnError)
	in := fs.String("in", "", "bundle path (.gtb.tar.gz) (required)")
	configPath := fs.String("config", "config.yaml", "config.yaml to merge into")
	csvDir := fs.String("csv-dir", "", "directory for generated talkgroup CSVs")
	force := fs.Bool("force", false, "overwrite conflicting entries")
	dryRun := fs.Bool("dry-run", false, "report changes without writing config.yaml")
	markEncrypted := fs.Bool("mark-encrypted", true, "fold a confirmed CryptoLab encryption verdict onto talkgroups")
	_ = fs.Parse(args)

	rep := newReporter("bundle commit")
	if *in == "" {
		rep.Fatalf(2, "-in is required")
	}
	r, err := gtbundle.Open(*in)
	if err != nil {
		rep.Fatal(1, err)
	}
	sys, err := r.MappingSystem()
	if err != nil {
		rep.Fatal(1, fmt.Errorf("bundle has no mapping-system to commit: %w", err))
	}
	if *markEncrypted {
		if n := enrichFromCryptoVerdict(r, sys); n > 0 {
			fmt.Fprintf(os.Stderr, "bundle: folded CryptoLab verdict onto %d talkgroup(s)\n", n)
		}
	}
	commitDiscovery(rep, sys, *configPath, *csvDir, *force, *dryRun)
}

// multiFlag collects a repeatable string flag.
type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}
