package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"gopkg.in/yaml.v3"
)

// Patch is a sparse Config — every field is a pointer so callers can
// say "leave alone" with nil. The settings PATCH endpoint and the
// SIGHUP reload path both pipe operator edits through this shape.
//
// Only fields the daemon knows how to surface in the TUI / web UI
// settings panels are listed here; expand the struct as new editable
// knobs become available.
type Patch struct {
	// Log.
	LogLevel  *string
	LogFormat *string

	// API.
	APIHTTPAddr *string
	APIGRPCAddr *string
	APIAuthMode *string

	// Audio.
	AudioEnabled  *bool
	AudioDevice   *string
	AudioVolume   *float32
	AudioMuted    *bool
	AudioBufferMs *int

	// Recordings.
	RecordingsDir        *string
	RecordingsSampleRate *uint32
	RecordingsWriteRaw   *bool

	// Retention.
	RetentionCallLogDays *int
	RetentionFilesDays   *int
	RetentionInterval    *string

	// SDR (sample rate only; device list edits go through a separate
	// future endpoint because they're keyed by serial).
	SDRSampleRate *uint32

	// Scanner.
	ScannerScanMode          *string
	ScannerManualTuneEnabled *bool
	ScannerCCHuntEnabled     *bool
	ScannerCCHuntDwellMs     *int
	ScannerCCHuntBackoffMs   *int
	ScannerCCHuntMaxBackoff  *int

	// Storage.
	StoragePath        *string
	StorageCCCacheFile *string

	// Metrics.
	MetricsEnabled *bool
}

// IsEmpty reports whether every patch field is nil. Used to short-
// circuit the writer when an operator sends an empty body.
func (p Patch) IsEmpty() bool {
	return p == Patch{}
}

// Apply layers p onto cfg in place, returning the modified cfg.
// Fields with nil pointers are left untouched.
func (p Patch) Apply(cfg Config) Config {
	if p.LogLevel != nil {
		cfg.Log.Level = *p.LogLevel
	}
	if p.LogFormat != nil {
		cfg.Log.Format = *p.LogFormat
	}
	if p.APIHTTPAddr != nil {
		cfg.API.HTTPAddr = *p.APIHTTPAddr
	}
	if p.APIGRPCAddr != nil {
		cfg.API.GRPCAddr = *p.APIGRPCAddr
	}
	if p.APIAuthMode != nil {
		cfg.API.Auth.Mode = *p.APIAuthMode
	}
	if p.AudioEnabled != nil {
		cfg.Audio.Enabled = *p.AudioEnabled
	}
	if p.AudioDevice != nil {
		cfg.Audio.Device = *p.AudioDevice
	}
	if p.AudioVolume != nil {
		cfg.Audio.Volume = *p.AudioVolume
	}
	if p.AudioMuted != nil {
		cfg.Audio.Muted = *p.AudioMuted
	}
	if p.AudioBufferMs != nil {
		cfg.Audio.BufferMs = *p.AudioBufferMs
	}
	if p.RecordingsDir != nil {
		cfg.Recordings.Dir = *p.RecordingsDir
	}
	if p.RecordingsSampleRate != nil {
		cfg.Recordings.SampleRate = *p.RecordingsSampleRate
	}
	if p.RecordingsWriteRaw != nil {
		cfg.Recordings.WriteRaw = *p.RecordingsWriteRaw
	}
	if p.RetentionCallLogDays != nil {
		cfg.Retention.CallLogDays = *p.RetentionCallLogDays
	}
	if p.RetentionFilesDays != nil {
		cfg.Retention.FilesDays = *p.RetentionFilesDays
	}
	if p.RetentionInterval != nil {
		cfg.Retention.Interval = *p.RetentionInterval
	}
	if p.SDRSampleRate != nil {
		cfg.SDR.SampleRate = *p.SDRSampleRate
	}
	if p.ScannerScanMode != nil {
		cfg.Scanner.ScanMode = *p.ScannerScanMode
	}
	if p.ScannerManualTuneEnabled != nil {
		cfg.Scanner.ManualTuneEnabled = *p.ScannerManualTuneEnabled
	}
	if p.ScannerCCHuntEnabled != nil {
		cfg.Scanner.CCHunt.Enabled = *p.ScannerCCHuntEnabled
	}
	if p.ScannerCCHuntDwellMs != nil {
		cfg.Scanner.CCHunt.DwellMs = *p.ScannerCCHuntDwellMs
	}
	if p.ScannerCCHuntBackoffMs != nil {
		cfg.Scanner.CCHunt.BackoffMs = *p.ScannerCCHuntBackoffMs
	}
	if p.ScannerCCHuntMaxBackoff != nil {
		cfg.Scanner.CCHunt.MaxBackoffMs = *p.ScannerCCHuntMaxBackoff
	}
	if p.StoragePath != nil {
		cfg.Storage.Path = *p.StoragePath
	}
	if p.StorageCCCacheFile != nil {
		cfg.Storage.CCCacheFile = *p.StorageCCCacheFile
	}
	if p.MetricsEnabled != nil {
		cfg.Metrics.Enabled = *p.MetricsEnabled
	}
	return cfg
}

// patchPath is one key-path → yaml.Node mutation. The writer applies
// these in document order so blocks land in a predictable spot when a
// section needs to be created from scratch.
type patchPath struct {
	keys  []string
	value *yaml.Node
}

// patchEdits returns the list of node mutations needed to express p.
// Empty pointers are skipped.
func patchEdits(p Patch) []patchPath {
	var out []patchPath
	add := func(keys []string, v *yaml.Node) {
		out = append(out, patchPath{keys: keys, value: v})
	}
	scalarString := func(s string) *yaml.Node {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s}
	}
	scalarBool := func(b bool) *yaml.Node {
		v := "false"
		if b {
			v = "true"
		}
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: v}
	}
	scalarInt := func(n int64) *yaml.Node {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.FormatInt(n, 10)}
	}
	scalarFloat := func(f float64) *yaml.Node {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!float", Value: strconv.FormatFloat(f, 'g', -1, 32)}
	}

	if p.LogLevel != nil {
		add([]string{"log", "level"}, scalarString(*p.LogLevel))
	}
	if p.LogFormat != nil {
		add([]string{"log", "format"}, scalarString(*p.LogFormat))
	}
	if p.APIHTTPAddr != nil {
		add([]string{"api", "http_addr"}, scalarString(*p.APIHTTPAddr))
	}
	if p.APIGRPCAddr != nil {
		add([]string{"api", "grpc_addr"}, scalarString(*p.APIGRPCAddr))
	}
	if p.APIAuthMode != nil {
		add([]string{"api", "auth", "mode"}, scalarString(*p.APIAuthMode))
	}
	if p.AudioEnabled != nil {
		add([]string{"audio", "enabled"}, scalarBool(*p.AudioEnabled))
	}
	if p.AudioDevice != nil {
		add([]string{"audio", "device"}, scalarString(*p.AudioDevice))
	}
	if p.AudioVolume != nil {
		add([]string{"audio", "volume"}, scalarFloat(float64(*p.AudioVolume)))
	}
	if p.AudioMuted != nil {
		add([]string{"audio", "muted"}, scalarBool(*p.AudioMuted))
	}
	if p.AudioBufferMs != nil {
		add([]string{"audio", "buffer_ms"}, scalarInt(int64(*p.AudioBufferMs)))
	}
	if p.RecordingsDir != nil {
		add([]string{"recordings", "dir"}, scalarString(*p.RecordingsDir))
	}
	if p.RecordingsSampleRate != nil {
		add([]string{"recordings", "sample_rate"}, scalarInt(int64(*p.RecordingsSampleRate)))
	}
	if p.RecordingsWriteRaw != nil {
		add([]string{"recordings", "write_raw"}, scalarBool(*p.RecordingsWriteRaw))
	}
	if p.RetentionCallLogDays != nil {
		add([]string{"retention", "call_log_days"}, scalarInt(int64(*p.RetentionCallLogDays)))
	}
	if p.RetentionFilesDays != nil {
		add([]string{"retention", "files_days"}, scalarInt(int64(*p.RetentionFilesDays)))
	}
	if p.RetentionInterval != nil {
		add([]string{"retention", "interval"}, scalarString(*p.RetentionInterval))
	}
	if p.SDRSampleRate != nil {
		add([]string{"sdr", "sample_rate"}, scalarInt(int64(*p.SDRSampleRate)))
	}
	if p.ScannerScanMode != nil {
		add([]string{"scanner", "scan_mode"}, scalarString(*p.ScannerScanMode))
	}
	if p.ScannerManualTuneEnabled != nil {
		add([]string{"scanner", "manual_tune_enabled"}, scalarBool(*p.ScannerManualTuneEnabled))
	}
	if p.ScannerCCHuntEnabled != nil {
		add([]string{"scanner", "cc_hunt", "enabled"}, scalarBool(*p.ScannerCCHuntEnabled))
	}
	if p.ScannerCCHuntDwellMs != nil {
		add([]string{"scanner", "cc_hunt", "dwell_ms"}, scalarInt(int64(*p.ScannerCCHuntDwellMs)))
	}
	if p.ScannerCCHuntBackoffMs != nil {
		add([]string{"scanner", "cc_hunt", "backoff_ms"}, scalarInt(int64(*p.ScannerCCHuntBackoffMs)))
	}
	if p.ScannerCCHuntMaxBackoff != nil {
		add([]string{"scanner", "cc_hunt", "max_backoff_ms"}, scalarInt(int64(*p.ScannerCCHuntMaxBackoff)))
	}
	if p.StoragePath != nil {
		add([]string{"storage", "path"}, scalarString(*p.StoragePath))
	}
	if p.StorageCCCacheFile != nil {
		add([]string{"storage", "cc_cache_file"}, scalarString(*p.StorageCCCacheFile))
	}
	if p.MetricsEnabled != nil {
		add([]string{"metrics", "enabled"}, scalarBool(*p.MetricsEnabled))
	}
	return out
}

// Writer serialises in-place edits to a config.yaml so concurrent
// callers (settings PATCH, SIGHUP-driven reloads, import commits)
// can't tear each other's writes. The writer also enforces an
// mtime guard so externally-edited files aren't clobbered.
type Writer struct {
	mu         sync.Mutex
	path       string
	knownMtime int64
}

// NewWriter constructs a Writer bound to path. The path must point
// at an existing file; an empty path returns nil so callers can
// disable the live-edit surface gracefully.
func NewWriter(path string) (*Writer, error) {
	if path == "" {
		return nil, nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("config: writer: %w", err)
	}
	return &Writer{path: path, knownMtime: info.ModTime().UnixNano()}, nil
}

// Path returns the file the writer mutates.
func (w *Writer) Path() string {
	if w == nil {
		return ""
	}
	return w.path
}

// WritePatch loads the file, runs Patch.Apply against the parsed
// config, runs Validate, then mutates the underlying yaml.Node tree
// so unrelated content (comments, formatting, keys we don't know
// about) is preserved, and atomically writes the result back.
//
// Returns the patched Config so callers can hand it to in-process
// subsystems for hot-reload.
func (w *Writer) WritePatch(p Patch) (Config, error) {
	var zero Config
	if w == nil {
		return zero, errors.New("config: no writer wired (daemon started without a -config file)")
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	// mtime guard — refuse to clobber an external edit.
	info, err := os.Stat(w.path)
	if err != nil {
		return zero, fmt.Errorf("config: writer: stat: %w", err)
	}
	if info.ModTime().UnixNano() != w.knownMtime {
		return zero, fmt.Errorf("config: %s was modified externally; reload the daemon to pick up the new file before editing again", w.path)
	}

	data, err := os.ReadFile(w.path)
	if err != nil {
		return zero, fmt.Errorf("config: writer: read: %w", err)
	}

	// Parse current config for validation.
	var current Config
	if err := yaml.Unmarshal(data, &current); err != nil {
		return zero, fmt.Errorf("config: writer: parse: %w", err)
	}
	merged := p.Apply(current)
	if err := merged.Validate(); err != nil {
		return zero, fmt.Errorf("config: writer: validate: %w", err)
	}

	// Walk the yaml.Node tree and apply each patch edit.
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return zero, fmt.Errorf("config: writer: parse node: %w", err)
	}
	root := docRoot(&doc)
	if root == nil {
		root = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		doc.Kind = yaml.DocumentNode
		doc.Content = []*yaml.Node{root}
	}
	for _, edit := range patchEdits(p) {
		setMappingPath(root, edit.keys, edit.value)
	}

	// Re-encode and atomic-rename.
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return zero, fmt.Errorf("config: writer: encode: %w", err)
	}
	if err := enc.Close(); err != nil {
		return zero, err
	}
	if err := writeFileAtomic(w.path, buf.Bytes(), 0o644); err != nil {
		return zero, err
	}
	// Refresh the known mtime to the just-written file.
	if info, err := os.Stat(w.path); err == nil {
		w.knownMtime = info.ModTime().UnixNano()
	}
	return merged, nil
}

// docRoot returns the document's top-level mapping node, or nil for
// an empty / non-mapping document.
func docRoot(doc *yaml.Node) *yaml.Node {
	if doc == nil {
		return nil
	}
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		if doc.Content[0].Kind == yaml.MappingNode {
			return doc.Content[0]
		}
	}
	if doc.Kind == yaml.MappingNode {
		return doc
	}
	return nil
}

// setMappingPath walks `path` from root, creating intermediate
// mappings as needed, and replaces (or appends) the value at the
// final key.
func setMappingPath(root *yaml.Node, path []string, value *yaml.Node) {
	if len(path) == 0 {
		return
	}
	node := root
	for i, key := range path[:len(path)-1] {
		node = findOrAddMapping(node, key)
		_ = i
	}
	leaf := path[len(path)-1]
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == leaf {
			// Preserve the existing key node's comments / position
			// when replacing the value.
			node.Content[i+1] = value
			return
		}
	}
	node.Content = append(node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: leaf},
		value,
	)
}

// findOrAddMapping returns the child mapping node for `key`, creating
// it (as a block mapping) when absent.
func findOrAddMapping(parent *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(parent.Content); i += 2 {
		if parent.Content[i].Value == key {
			v := parent.Content[i+1]
			if v.Kind != yaml.MappingNode {
				v = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
				parent.Content[i+1] = v
			}
			return v
		}
	}
	kn := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	vn := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	parent.Content = append(parent.Content, kn, vn)
	return vn
}

// writeFileAtomic writes data to path through a same-directory temp
// file and an os.Rename. Atomic on POSIX filesystems.
func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("config: writer: tempfile: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("config: writer: write: %w", err)
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
