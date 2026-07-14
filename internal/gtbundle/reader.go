package gtbundle

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/hunt"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"gopkg.in/yaml.v3"
)

// ErrChecksum is returned when a file's bytes do not match the sha256 recorded
// in the manifest.
var ErrChecksum = errors.New("gtbundle: sha256 mismatch")

// ErrFutureSchema is returned (wrapped, non-fatal) by ParseManifest/Open when a
// bundle's schema_version is newer than this build supports. The bundle is
// still usable best-effort.
var ErrFutureSchema = errors.New("gtbundle: manifest schema newer than supported")

// ErrNotFound is returned when a requested role/path is absent.
var ErrNotFound = errors.New("gtbundle: not found")

// Reader provides random access to a bundle. Because tar is sequential, the
// whole archive is read into memory on open; bundles are small (a manifest plus
// a handful of analysis files and one raw IQ), and the large IQ member can be
// streamed straight to a working file with ExtractRole/CaptureIQ. A Reader is
// read-only and safe for concurrent reads after construction.
type Reader struct {
	man    *Manifest
	blobs  map[string][]byte // archive member path -> bytes (excludes dirs)
	future bool              // schema_version newer than supported
}

// Open reads a .gtb.tar.gz from disk.
func Open(path string) (*Reader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return NewReader(f)
}

// NewReader reads a bundle from any stream.
func NewReader(r io.Reader) (*Reader, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("gtbundle: gunzip: %w", err)
	}
	defer gz.Close()

	blobs := map[string][]byte{}
	tr := tar.NewReader(gz)
	var manRaw []byte
	var manPath string
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("gtbundle: read tar: %w", err)
		}
		if hdr.Typeflag == tar.TypeDir {
			continue
		}
		clean, err := safeMemberPath(hdr.Name)
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("gtbundle: read member %s: %w", hdr.Name, err)
		}
		blobs[clean] = body
		if strings.HasSuffix(clean, "/"+ManifestName) || clean == ManifestName {
			manRaw, manPath = body, clean
		}
	}
	if manRaw == nil {
		return nil, fmt.Errorf("gtbundle: no %s in archive", ManifestName)
	}
	man, perr := ParseManifest(manRaw)
	future := false
	if perr != nil {
		if errors.Is(perr, ErrFutureSchema) && man != nil {
			future = true // tolerate, read best-effort
		} else {
			return nil, perr
		}
	}
	_ = manPath
	return &Reader{man: man, blobs: blobs, future: future}, nil
}

// Manifest returns the parsed manifest.
func (r *Reader) Manifest() *Manifest { return r.man }

// FutureSchema reports whether the bundle declares a schema newer than this
// build supports (it was still read best-effort).
func (r *Reader) FutureSchema() bool { return r.future }

// Bytes returns the verified bytes of the first file with role, or ErrNotFound.
// The sha256 is checked against the manifest; a mismatch returns ErrChecksum.
func (r *Reader) Bytes(role Role) ([]byte, *FileEntry, error) {
	e, ok := r.man.Find(role)
	if !ok {
		return nil, nil, fmt.Errorf("%w: role %s", ErrNotFound, role)
	}
	body, err := r.verified(e)
	if err != nil {
		return nil, nil, err
	}
	return body, &e, nil
}

// Open returns a reader over the verified bytes of the first file with role.
func (r *Reader) Open(role Role) (io.ReadCloser, *FileEntry, error) {
	body, e, err := r.Bytes(role)
	if err != nil {
		return nil, nil, err
	}
	return io.NopCloser(bytes.NewReader(body)), e, nil
}

// verified fetches a member's bytes and checks its size and sha256 against the
// manifest entry.
func (r *Reader) verified(e FileEntry) ([]byte, error) {
	body, ok := r.blobs[e.Path]
	if !ok {
		return nil, fmt.Errorf("%w: %s (listed in manifest, missing from archive)", ErrNotFound, e.Path)
	}
	if int64(len(body)) != e.SizeBytes {
		return nil, fmt.Errorf("%w: %s size %d != manifest %d", ErrChecksum, e.Path, len(body), e.SizeBytes)
	}
	sum := sha256.Sum256(body)
	if got := hex.EncodeToString(sum[:]); got != e.SHA256 {
		return nil, fmt.Errorf("%w: %s (%s != %s)", ErrChecksum, e.Path, got, e.SHA256)
	}
	return body, nil
}

// VerifyResult is one file's integrity outcome.
type VerifyResult struct {
	Path      string
	Role      Role
	OK        bool
	Untracked bool // present in archive but not in the manifest
	Err       error
}

// Verify checks every manifest file's size+sha256 and flags any archive member
// not listed in the manifest as Untracked (non-fatal). It returns one result
// per manifest file plus one per untracked member.
func (r *Reader) Verify() []VerifyResult {
	var out []VerifyResult
	listed := map[string]bool{}
	for _, e := range r.man.Files {
		listed[e.Path] = true
		_, err := r.verified(e)
		out = append(out, VerifyResult{Path: e.Path, Role: e.Role, OK: err == nil, Err: err})
	}
	for p := range r.blobs {
		if listed[p] {
			continue
		}
		if strings.HasSuffix(p, "/"+ManifestName) || p == ManifestName ||
			strings.HasSuffix(p, "/"+ReadmeName) || p == ReadmeName {
			continue // index files aren't inventory
		}
		out = append(out, VerifyResult{Path: p, Untracked: true, OK: true})
	}
	return out
}

// Extract writes the whole bundle tree under dstDir, verifying each listed file
// as it goes. Untracked members are extracted too (best-effort). It re-validates
// every member path against traversal.
func (r *Reader) Extract(dstDir string) error {
	for path, body := range r.blobs {
		if err := writeUnder(dstDir, path, body); err != nil {
			return err
		}
	}
	// Verify listed files after extraction so a tamper is still reported.
	for _, v := range r.Verify() {
		if v.Err != nil {
			return v.Err
		}
	}
	return nil
}

// ExtractRole writes just the files of one role under dstDir (flattened to
// their leaf names) and returns the written paths. Handy for handing one
// subtree onward — e.g. the cryptolab frames to a colleague.
func (r *Reader) ExtractRole(role Role, dstDir string) ([]string, error) {
	var written []string
	for _, e := range r.man.List(role) {
		body, err := r.verified(e)
		if err != nil {
			return written, err
		}
		leaf := filepath.Base(filepath.FromSlash(e.Path))
		full := filepath.Join(dstDir, leaf)
		if err := os.MkdirAll(dstDir, 0o755); err != nil {
			return written, err
		}
		if err := os.WriteFile(full, body, 0o644); err != nil {
			return written, err
		}
		written = append(written, full)
	}
	if len(written) == 0 {
		return nil, fmt.Errorf("%w: role %s", ErrNotFound, role)
	}
	return written, nil
}

// writeUnder safely writes body to dstDir/member, refusing any path that would
// escape dstDir.
func writeUnder(dstDir, member string, body []byte) error {
	clean, err := safeMemberPath(member)
	if err != nil {
		return err
	}
	full := filepath.Join(dstDir, filepath.FromSlash(clean))
	// Defense in depth: ensure the resolved path stays under dstDir.
	rel, err := filepath.Rel(dstDir, full)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("gtbundle: refusing to extract %q outside %q", member, dstDir)
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	return os.WriteFile(full, body, 0o644)
}

// --- typed accessors (verify + decode the reused models) ---

// CaptureMeta decodes the capture metadata sidecar (siglab.Metadata, YAML).
func (r *Reader) CaptureMeta() (*siglab.Metadata, error) {
	body, _, err := r.Bytes(RoleCaptureMeta)
	if err != nil {
		return nil, err
	}
	var m siglab.Metadata
	if err := yaml.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("gtbundle: decode capture meta: %w", err)
	}
	return &m, nil
}

// SiglabResult decodes the SigLab analysis result (siglab.Result, YAML).
func (r *Reader) SiglabResult() (*siglab.Result, error) {
	body, _, err := r.Bytes(RoleSiglabResult)
	if err != nil {
		return nil, err
	}
	var res siglab.Result
	if err := yaml.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("gtbundle: decode siglab result: %w", err)
	}
	return &res, nil
}

// MappingSystem decodes the discovered system map (hunt.DiscoveredSystem, JSON).
func (r *Reader) MappingSystem() (*hunt.DiscoveredSystem, error) {
	body, _, err := r.Bytes(RoleMappingSystem)
	if err != nil {
		return nil, err
	}
	var sys hunt.DiscoveredSystem
	if err := json.Unmarshal(body, &sys); err != nil {
		return nil, fmt.Errorf("gtbundle: decode mapping system: %w", err)
	}
	return &sys, nil
}

// MappingSurvey reconstructs a SignalSurvey from the survey NDJSON stream: one
// DetectedSignal per line. The System pointer is left nil (it lives in
// mapping/system.json); callers that need it use MappingSystem.
func (r *Reader) MappingSurvey() (*hunt.SignalSurvey, error) {
	body, _, err := r.Bytes(RoleMappingSurvey)
	if err != nil {
		return nil, err
	}
	sv := &hunt.SignalSurvey{}
	sc := bufio.NewScanner(bytes.NewReader(body))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		var ds hunt.DetectedSignal
		if err := json.Unmarshal(line, &ds); err != nil {
			return nil, fmt.Errorf("gtbundle: decode survey line: %w", err)
		}
		sv.Signals = append(sv.Signals, ds)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return sv, nil
}

// CryptolabFramesRaw returns the verified bytes of the cryptolab frames stream
// (cryptolab/frames.jsonl). Returns ErrNotFound when absent.
func (r *Reader) CryptolabFramesRaw() ([]byte, error) {
	body, _, err := r.Bytes(RoleCryptolabFrames)
	return body, err
}

// CryptolabResultRaw returns the verified bytes of the cryptolab result
// (cryptolab/assess.result.yaml). It is exposed as raw YAML rather than a typed
// accessor so this package needn't import internal/cryptolab; the caller (which
// already links cryptolab) decodes it. Returns ErrNotFound when absent.
func (r *Reader) CryptolabResultRaw() ([]byte, error) {
	body, _, err := r.Bytes(RoleCryptolabResult)
	return body, err
}

// CaptureIQ returns the verified raw-IQ bytes and its entry. For a large
// capture prefer ExtractRole(RoleCaptureIQ, dir) to stream it to a working file.
func (r *Reader) CaptureIQ() ([]byte, *FileEntry, error) {
	return r.Bytes(RoleCaptureIQ)
}
