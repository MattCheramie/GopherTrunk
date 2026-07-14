package gtbundle

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/version"
)

// WriterOptions configure a new bundle. Name is required; everything else has a
// sensible default.
type WriterOptions struct {
	Name          string
	CaptureIntent CaptureIntent
	Provenance    Provenance
	Title         string
	Notes         string
	// Now is an injectable clock so tests can produce deterministic archives.
	// When nil, time.Now is used. Its value stamps CreatedAt and every tar
	// header ModTime, making a repack byte-identical for a fixed input set.
	Now func() time.Time
}

// Writer assembles a .gtb.tar.gz. Files are streamed in with Add/AddFile/AddYAML
// as they are produced; MANIFEST.yaml and README.md are written by Close, which
// finalizes the archive. A Writer is not safe for concurrent use.
type Writer struct {
	gz   *gzip.Writer
	tw   *tar.Writer
	man  *Manifest
	top  string
	now  time.Time
	done bool
	// staged holds file bodies until Close so the manifest (which must list
	// every file with its checksum) can be written first, and the whole archive
	// emitted in deterministic role order.
	staged []stagedFile
}

type stagedFile struct {
	entry FileEntry
	body  []byte
}

// NewWriter wraps w in gzip+tar and prepares an in-memory manifest. The caller
// must Close the Writer to flush the archive. w is typically an *os.File.
func NewWriter(w io.Writer, opts WriterOptions) (*Writer, error) {
	name := SanitizeName(opts.Name)
	nowFn := opts.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	now := nowFn().UTC()

	gen := Generator{Tool: "gophertrunk", Version: version.Version, Commit: version.Commit}
	man := newManifest(name, opts.CaptureIntent, opts.Provenance, gen, now, opts.Title, opts.Notes)

	gz, err := gzip.NewWriterLevel(w, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	return &Writer{
		gz:  gz,
		tw:  tar.NewWriter(gz),
		man: man,
		top: name,
		now: now,
	}, nil
}

// Add stages src under the canonical path for role (topDir/roleDir/leaf),
// computing its sha256 and size and appending a manifest entry. leaf must be a
// single path component; producedBy records the tool/step that made the file.
// It returns the bundle-relative path.
func (w *Writer) Add(role Role, leaf string, src io.Reader, producedBy string) (string, error) {
	if w.done {
		return "", fmt.Errorf("gtbundle: writer already closed")
	}
	safe, err := safeLeaf(leaf)
	if err != nil {
		return "", err
	}
	body, err := io.ReadAll(src)
	if err != nil {
		return "", fmt.Errorf("gtbundle: read %s: %w", leaf, err)
	}
	sum := sha256.Sum256(body)
	rel := bundlePath(w.top, role, safe)
	entry := FileEntry{
		Path:       rel,
		Role:       role,
		MediaType:  role.mediaType(),
		SizeBytes:  int64(len(body)),
		SHA256:     hex.EncodeToString(sum[:]),
		ProducedBy: producedBy,
	}
	w.staged = append(w.staged, stagedFile{entry: entry, body: body})
	return rel, nil
}

// AddBytes is Add for an in-memory buffer.
func (w *Writer) AddBytes(role Role, leaf string, body []byte, producedBy string) (string, error) {
	return w.Add(role, leaf, bytes.NewReader(body), producedBy)
}

// AddYAML marshals v as two-space YAML and stages it. Use for models with yaml
// tags (siglab.Metadata, siglab.Result, cryptolab.Result).
func (w *Writer) AddYAML(role Role, leaf string, v any, producedBy string) (string, error) {
	raw, err := marshalYAML(v)
	if err != nil {
		return "", fmt.Errorf("gtbundle: marshal yaml %s: %w", leaf, err)
	}
	return w.AddBytes(role, leaf, raw, producedBy)
}

// AddJSON marshals v as indented JSON and stages it. Use for hunt.DiscoveredSystem
// (json-only tags) so snake_case is preserved.
func (w *Writer) AddJSON(role Role, leaf string, v any, producedBy string) (string, error) {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", fmt.Errorf("gtbundle: marshal json %s: %w", leaf, err)
	}
	raw = append(raw, '\n')
	return w.AddBytes(role, leaf, raw, producedBy)
}

// SetNote attaches a free-text note to the most recently added file entry (e.g.
// "signal wider than tune — recorded as a slice"). It is a no-op if nothing has
// been added yet.
func (w *Writer) SetNote(note string) {
	if n := len(w.staged); n > 0 {
		w.staged[n-1].entry.Note = note
	}
}

// Manifest returns the in-progress manifest header (files reflect what has been
// staged so far). The returned pointer is live; do not mutate Files.
func (w *Writer) Manifest() *Manifest {
	w.man.Files = w.stagedEntries()
	return w.man
}

func (w *Writer) stagedEntries() []FileEntry {
	out := make([]FileEntry, 0, len(w.staged))
	for _, s := range w.staged {
		out = append(out, s.entry)
	}
	return out
}

// Close writes MANIFEST.yaml and README.md, emits every staged file in
// deterministic role order, and flushes tar+gzip. It is safe to call once; a
// second call is a no-op.
func (w *Writer) Close() error {
	if w.done {
		return nil
	}
	w.done = true

	// Deterministic order: by canonical role rank, then by path.
	sort.SliceStable(w.staged, func(i, j int) bool {
		ri, rj := roleWriteRank(w.staged[i].entry.Role), roleWriteRank(w.staged[j].entry.Role)
		if ri != rj {
			return ri < rj
		}
		return w.staged[i].entry.Path < w.staged[j].entry.Path
	})

	// The manifest lists every staged file (readme/manifest excluded — they are
	// the index, not inventory).
	w.man.Files = w.stagedEntries()

	manRaw, err := MarshalManifest(w.man)
	if err != nil {
		return err
	}
	readmeRaw := renderReadme(w.man)

	// Root dir entry, then MANIFEST.yaml, README.md, then staged files.
	if err := w.writeDir(w.top + "/"); err != nil {
		return err
	}
	if err := w.writeMember(w.top+"/"+ManifestName, manRaw); err != nil {
		return err
	}
	if err := w.writeMember(w.top+"/"+ReadmeName, readmeRaw); err != nil {
		return err
	}
	// Track directories already emitted so tar carries explicit dir entries.
	seenDir := map[string]bool{}
	for _, s := range w.staged {
		if d := s.entry.Role.dir(); d != "" {
			dp := w.top + "/" + d + "/"
			if !seenDir[dp] {
				if err := w.writeDir(dp); err != nil {
					return err
				}
				seenDir[dp] = true
			}
		}
		if err := w.writeMember(s.entry.Path, s.body); err != nil {
			return err
		}
	}

	if err := w.tw.Close(); err != nil {
		return err
	}
	return w.gz.Close()
}

func (w *Writer) writeDir(name string) error {
	return w.tw.WriteHeader(&tar.Header{
		Name:     name,
		Typeflag: tar.TypeDir,
		Mode:     0o755,
		ModTime:  w.now,
	})
}

func (w *Writer) writeMember(name string, body []byte) error {
	if err := w.tw.WriteHeader(&tar.Header{
		Name:     name,
		Typeflag: tar.TypeReg,
		Mode:     0o644,
		Size:     int64(len(body)),
		ModTime:  w.now,
	}); err != nil {
		return err
	}
	_, err := w.tw.Write(body)
	return err
}
