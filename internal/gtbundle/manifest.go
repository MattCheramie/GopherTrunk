package gtbundle

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// SchemaVersion is the on-disk manifest schema version. Readers accept any
	// bundle whose schema_version is <= this value; a higher version is read
	// best-effort with a warning rather than rejected (see reader.go).
	SchemaVersion = 1
	// FormatID identifies the archive family; it is written into the manifest's
	// `format` field so a consumer can tell a GopherTrunk Bundle from any other
	// tar.gz at a glance.
	FormatID = "gophertrunk-bundle"
	// ManifestName / ReadmeName are the fixed filenames at the bundle root.
	ManifestName = "MANIFEST.yaml"
	ReadmeName   = "README.md"
	// Ext is the conventional archive extension.
	Ext = ".gtb.tar.gz"
)

// CaptureIntent classifies why a case was collected. It documents the
// operator's purpose and drives the default narrowband-slice length when a
// bundle is produced from a live capture.
type CaptureIntent string

const (
	IntentCCMap   CaptureIntent = "cc-map"  // control-channel mapping / hunt
	IntentCrypto  CaptureIntent = "crypto"  // encryption assessment
	IntentSurvey  CaptureIntent = "survey"  // wideband classification survey
	IntentGeneral CaptureIntent = "general" // unspecified / general purpose
)

// Valid reports whether i is a recognized intent.
func (i CaptureIntent) Valid() bool {
	switch i {
	case IntentCCMap, IntentCrypto, IntentSurvey, IntentGeneral:
		return true
	default:
		return false
	}
}

// Manifest is the machine-and-human index stored at <name>/MANIFEST.yaml. It is
// YAML with snake_case keys. Unknown top-level keys and unknown file roles are
// tolerated on read so a newer bundle still opens in an older binary.
type Manifest struct {
	SchemaVersion int           `yaml:"schema_version"`
	Format        string        `yaml:"format"`
	Name          string        `yaml:"name"`
	CaptureIntent CaptureIntent `yaml:"capture_intent"`
	CreatedAt     time.Time     `yaml:"created_at"`

	Generator  Generator  `yaml:"generator"`
	Provenance Provenance `yaml:"provenance"`

	Title string `yaml:"title,omitempty"`
	Notes string `yaml:"notes,omitempty"`

	Files []FileEntry `yaml:"files"`
}

// Generator records the tool that produced the bundle.
type Generator struct {
	Tool    string `yaml:"tool"`
	Version string `yaml:"version"`
	Commit  string `yaml:"commit,omitempty"`
}

// Provenance is the capture-side context: where, when, and how the RF was
// collected. Every field is optional — a bundle packed from loose files may
// know little about the capture — but recording what is known makes a shared
// case reproducible. Frequencies are integer Hz; timestamps are UTC RFC3339.
type Provenance struct {
	Source       string    `yaml:"source,omitempty"`
	Operator     string    `yaml:"operator,omitempty"`
	CenterFreqHz uint32    `yaml:"center_freq_hz,omitempty"`
	SampleRateHz float64   `yaml:"sample_rate_hz,omitempty"`
	Protocol     string    `yaml:"protocol,omitempty"`
	Location     string    `yaml:"location,omitempty"`
	CapturedAt   time.Time `yaml:"captured_at,omitempty"`
	DeviceDriver string    `yaml:"device_driver,omitempty"`
	DeviceSerial string    `yaml:"device_serial,omitempty"`
}

// FileEntry is one archived artifact. Path is bundle-relative (including the
// top-level dir) and always uses forward slashes, e.g.
// "mesa-p25/capture/cc.cfile". SHA256 is lowercase hex over the file bytes.
type FileEntry struct {
	Path       string `yaml:"path"`
	Role       Role   `yaml:"role"`
	MediaType  string `yaml:"media_type"`
	SizeBytes  int64  `yaml:"size_bytes"`
	SHA256     string `yaml:"sha256"`
	ProducedBy string `yaml:"produced_by,omitempty"`
	Note       string `yaml:"note,omitempty"`
}

// newManifest builds an in-memory manifest header from writer options and a
// generator stamp. Files are appended as artifacts are written.
func newManifest(name string, intent CaptureIntent, prov Provenance, gen Generator, createdAt time.Time, title, notes string) *Manifest {
	if !intent.Valid() {
		intent = IntentGeneral
	}
	return &Manifest{
		SchemaVersion: SchemaVersion,
		Format:        FormatID,
		Name:          name,
		CaptureIntent: intent,
		CreatedAt:     createdAt.UTC(),
		Generator:     gen,
		Provenance:    prov,
		Title:         title,
		Notes:         notes,
		Files:         nil,
	}
}

// MarshalManifest renders m as YAML with two-space indentation, matching the
// repo's siglab/cryptolab YAML export style.
func MarshalManifest(m *Manifest) ([]byte, error) {
	return marshalYAML(m)
}

// ParseManifest decodes a MANIFEST.yaml. It validates the format id and that
// the schema version is not from the future beyond what this build supports;
// an unsupported-but-higher version is returned with a non-nil *Manifest and a
// non-fatal ErrFutureSchema so the caller can warn and proceed.
func ParseManifest(raw []byte) (*Manifest, error) {
	var m Manifest
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("gtbundle: parse manifest: %w", err)
	}
	if m.Format != FormatID {
		return nil, fmt.Errorf("gtbundle: not a %s manifest (format=%q)", FormatID, m.Format)
	}
	if m.SchemaVersion > SchemaVersion {
		return &m, fmt.Errorf("%w: manifest schema_version %d newer than supported %d",
			ErrFutureSchema, m.SchemaVersion, SchemaVersion)
	}
	return &m, nil
}

// Find returns the first file entry with the given role, or ok=false.
func (m *Manifest) Find(r Role) (FileEntry, bool) {
	for _, f := range m.Files {
		if f.Role == r {
			return f, true
		}
	}
	return FileEntry{}, false
}

// List returns every file entry with the given role, in manifest order.
func (m *Manifest) List(r Role) []FileEntry {
	var out []FileEntry
	for _, f := range m.Files {
		if f.Role == r {
			out = append(out, f)
		}
	}
	return out
}

// marshalYAML is the shared two-space YAML encoder used for the manifest and
// any YAML artifact the Writer produces.
func marshalYAML(v any) ([]byte, error) {
	var b yamlBuffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// yamlBuffer is a tiny bytes.Buffer alias kept local so marshalYAML has no
// import beyond what the file already needs; it satisfies io.Writer.
type yamlBuffer struct{ buf []byte }

func (b *yamlBuffer) Write(p []byte) (int, error) { b.buf = append(b.buf, p...); return len(p), nil }
func (b *yamlBuffer) Bytes() []byte               { return b.buf }
