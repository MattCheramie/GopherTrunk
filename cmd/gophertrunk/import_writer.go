package main

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"gopkg.in/yaml.v3"
)

// mergeOptions controls the behaviour of mergeIntoConfig.
type mergeOptions struct {
	ConfigPath string
	CSVDir     string
	Force      bool // overwrite an existing system with the same name
	DryRun     bool // produce buffers but don't write to disk
}

// mergeResult is the outcome of a merge — the YAML bytes that would be
// written, plus the talkgroup CSV bytes per system, plus a list of human-
// readable change descriptions for --dry-run output.
type mergeResult struct {
	ConfigYAML []byte
	CSVs       []csvOutput
	Changes    []string
}

type csvOutput struct {
	Path string
	Data []byte
}

// mergeIntoConfig is the atomic merge entry point. It validates the
// merged config in-memory BEFORE producing any output, so a malformed
// PDF or schema mismatch never corrupts the user's config file.
//
// Order of operations:
//  1. Load existing config.yaml as a structured Config and as a *yaml.Node.
//  2. Append each parsed system to the Config struct; call Validate().
//  3. Append matching mapping nodes to the Node tree (preserves comments).
//  4. Encode the Node tree to a buffer; re-decode + Validate again.
//  5. Produce talkgroup CSVs in memory.
//  6. If !DryRun: write each CSV via atomic-rename, then the YAML.
func mergeIntoConfig(systems []parsedSystem, opts mergeOptions) (mergeResult, error) {
	var res mergeResult

	// Load existing config (allow missing file → empty config).
	var existingBytes []byte
	if b, err := os.ReadFile(opts.ConfigPath); err == nil {
		existingBytes = b
	} else if !errors.Is(err, os.ErrNotExist) {
		return res, fmt.Errorf("import-pdf: read %s: %w", opts.ConfigPath, err)
	}

	// Decode into struct for schema validation.
	var cfg config.Config
	if len(existingBytes) > 0 {
		if err := yaml.Unmarshal(existingBytes, &cfg); err != nil {
			return res, fmt.Errorf("import-pdf: parse %s: %w", opts.ConfigPath, err)
		}
	}

	// Track existing system names for collision detection.
	existing := map[string]int{}
	for i, s := range cfg.Trunking.Systems {
		existing[strings.ToLower(s.Name)] = i
	}

	csvDir := opts.CSVDir
	if csvDir == "" {
		csvDir = filepath.Dir(opts.ConfigPath)
		if csvDir == "" || csvDir == "." {
			abs, _ := filepath.Abs(opts.ConfigPath)
			csvDir = filepath.Dir(abs)
		}
	}

	// Build the new SystemConfig entries + CSVs.
	newSystems := make([]config.SystemConfig, 0, len(systems))
	for _, sys := range systems {
		slug := buildSlug(sys.Name, sys.SysID)
		csvPath := filepath.Join(csvDir, "talkgroups-"+slug+".csv")
		entry := config.SystemConfig{
			Name:            sys.Name,
			Protocol:        sys.Protocol,
			ControlChannels: collectControlChannels(sys),
			TalkgroupFile:   csvPath,
		}
		newSystems = append(newSystems, entry)

		// Generate CSV.
		csvBytes, err := buildTalkgroupCSV(sys)
		if err != nil {
			return res, fmt.Errorf("import-pdf: build CSV for %q: %w", sys.Name, err)
		}
		res.CSVs = append(res.CSVs, csvOutput{Path: csvPath, Data: csvBytes})

		// Collision check.
		if idx, ok := existing[strings.ToLower(sys.Name)]; ok {
			if !opts.Force {
				return res, fmt.Errorf("import-pdf: system %q already exists in %s (use --force to overwrite)", sys.Name, opts.ConfigPath)
			}
			cfg.Trunking.Systems[idx] = entry
			res.Changes = append(res.Changes, fmt.Sprintf("overwrite system %q (%d CCs, %d talkgroups → %s)", sys.Name, len(entry.ControlChannels), len(sys.Talkgroups), csvPath))
		} else {
			cfg.Trunking.Systems = append(cfg.Trunking.Systems, entry)
			res.Changes = append(res.Changes, fmt.Sprintf("add system %q (%d CCs, %d talkgroups → %s)", sys.Name, len(entry.ControlChannels), len(sys.Talkgroups), csvPath))
		}
	}

	// Validate the in-memory merged config before touching the file.
	if err := cfg.Validate(); err != nil {
		return res, fmt.Errorf("import-pdf: merged config fails validation: %w", err)
	}

	// Comment-preserving merge into the *yaml.Node tree.
	yamlBytes, err := mergeYAMLNodes(existingBytes, systems, newSystems, opts.Force)
	if err != nil {
		return res, err
	}

	// Re-decode + re-validate to assert the comment-preserving merge
	// didn't drift from the struct path.
	var roundTrip config.Config
	if err := yaml.Unmarshal(yamlBytes, &roundTrip); err != nil {
		return res, fmt.Errorf("import-pdf: round-trip parse: %w", err)
	}
	if err := roundTrip.Validate(); err != nil {
		return res, fmt.Errorf("import-pdf: round-trip validate: %w", err)
	}
	res.ConfigYAML = yamlBytes

	if opts.DryRun {
		return res, nil
	}

	// Write CSVs first (atomic rename), then the config.
	if err := os.MkdirAll(csvDir, 0o755); err != nil {
		return res, fmt.Errorf("import-pdf: mkdir %s: %w", csvDir, err)
	}
	for _, c := range res.CSVs {
		if err := writeAtomic(c.Path, c.Data, 0o644); err != nil {
			return res, err
		}
	}
	if err := writeAtomic(opts.ConfigPath, res.ConfigYAML, 0o644); err != nil {
		return res, err
	}
	return res, nil
}

// collectControlChannels flattens every Include=true site's CC-capable
// frequencies into a single slice, deduplicated and sorted.
func collectControlChannels(sys parsedSystem) []uint32 {
	seen := map[uint32]bool{}
	for _, site := range sys.Sites {
		if !site.Include {
			continue
		}
		for _, f := range site.Frequencies {
			if f.ControlChannel {
				seen[f.Hz] = true
			}
		}
	}
	out := make([]uint32, 0, len(seen))
	for hz := range seen {
		out = append(out, hz)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// buildSlug produces a filesystem-safe identifier for the per-system
// talkgroup CSV. Slug = lowercase alphanumeric of system name + "-" +
// lowercase WACN.SysID (last 8 chars). The WACN.SysID component makes
// the slug collision-resistant across systems with similar names.
func buildSlug(name, sysid string) string {
	var b strings.Builder
	lastWasAlnum := false
	for _, r := range strings.ToLower(name) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastWasAlnum = true
		case lastWasAlnum:
			b.WriteByte('-')
			lastWasAlnum = false
		}
	}
	slug := strings.TrimRight(b.String(), "-")
	if sysid != "" {
		slug += "-" + strings.ToLower(sysid)
	}
	return slug
}

// buildTalkgroupCSV produces the Trunk-Recorder-style CSV that the
// daemon's internal/trunking.TalkgroupDB.LoadCSV understands. Column
// order matches the loader's case-insensitive lookups.
func buildTalkgroupCSV(sys parsedSystem) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	headers := []string{"Decimal", "Hex", "Mode", "Alpha Tag", "Description", "Tag", "Group", "Priority", "Lockout", "Scan"}
	if err := w.Write(headers); err != nil {
		return nil, err
	}
	for _, tg := range sys.Talkgroups {
		priority := ""
		if tg.Priority > 0 {
			priority = strconv.Itoa(tg.Priority)
		}
		lockout := ""
		if tg.Lockout {
			lockout = "Y"
		}
		scan := "Y"
		if !tg.Scan {
			scan = "N"
		}
		row := []string{
			strconv.FormatUint(uint64(tg.Dec), 10),
			tg.Hex,
			tg.Mode,
			tg.AlphaTag,
			tg.Description,
			tg.Tag,
			tg.Group,
			priority,
			lockout,
			scan,
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// mergeYAMLNodes does the comment-preserving merge: decode → walk →
// append/replace mapping nodes in trunking.systems → re-encode.
//
// If the existing file is empty or missing trunking.systems, we synthesize
// the structure from scratch.
func mergeYAMLNodes(existing []byte, parsed []parsedSystem, newEntries []config.SystemConfig, force bool) ([]byte, error) {
	var doc yaml.Node
	if len(existing) > 0 {
		if err := yaml.Unmarshal(existing, &doc); err != nil {
			return nil, fmt.Errorf("import-pdf: yaml decode: %w", err)
		}
	}
	if doc.Kind == 0 {
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}}
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, errors.New("import-pdf: config root is not a mapping")
	}
	root := doc.Content[0]

	trunkingNode := findOrCreateMapping(root, "trunking")
	systemsNode := findOrCreateSequence(trunkingNode, "systems")

	// Build name → existing-index map for replacement.
	existingIdx := map[string]int{}
	for i, n := range systemsNode.Content {
		if name := mapStringValue(n, "name"); name != "" {
			existingIdx[strings.ToLower(name)] = i
		}
	}

	for i, sys := range parsed {
		entry := newEntries[i]
		node := buildSystemMappingNode(entry)
		if idx, ok := existingIdx[strings.ToLower(sys.Name)]; ok {
			if !force {
				return nil, fmt.Errorf("import-pdf: system %q already exists (use --force)", sys.Name)
			}
			systemsNode.Content[idx] = node
		} else {
			systemsNode.Content = append(systemsNode.Content, node)
			existingIdx[strings.ToLower(sys.Name)] = len(systemsNode.Content) - 1
		}
	}

	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return nil, fmt.Errorf("import-pdf: yaml encode: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// findOrCreateMapping returns the child mapping node for `key` under
// `parent`, creating an empty one (with key) if absent.
func findOrCreateMapping(parent *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(parent.Content); i += 2 {
		if parent.Content[i].Value == key {
			if parent.Content[i+1].Kind != yaml.MappingNode {
				// Reuse but reshape to a mapping.
				parent.Content[i+1] = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			}
			return parent.Content[i+1]
		}
	}
	kn := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	vn := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	parent.Content = append(parent.Content, kn, vn)
	return vn
}

// findOrCreateSequence is the sequence-valued sibling of findOrCreateMapping.
// Forces block style so appended entries render as readable YAML.
func findOrCreateSequence(parent *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(parent.Content); i += 2 {
		if parent.Content[i].Value == key {
			v := parent.Content[i+1]
			if v.Kind != yaml.SequenceNode {
				v = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
				parent.Content[i+1] = v
			}
			v.Style = 0 // force block style
			return v
		}
	}
	kn := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	vn := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Style: 0}
	parent.Content = append(parent.Content, kn, vn)
	return vn
}

func mapStringValue(m *yaml.Node, key string) string {
	if m.Kind != yaml.MappingNode {
		return ""
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1].Value
		}
	}
	return ""
}

// buildSystemMappingNode constructs a fresh mapping node for one
// trunking.systems[] entry. Keys are emitted in a stable order
// (name, protocol, control_channels, talkgroup_file).
func buildSystemMappingNode(s config.SystemConfig) *yaml.Node {
	m := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	addKV(m, "name", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s.Name})
	addKV(m, "protocol", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s.Protocol})

	ccSeq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Style: 0}
	for _, hz := range s.ControlChannels {
		ccSeq.Content = append(ccSeq.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!int",
			Value: strconv.FormatUint(uint64(hz), 10),
		})
	}
	addKV(m, "control_channels", ccSeq)
	addKV(m, "talkgroup_file", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s.TalkgroupFile})
	return m
}

func addKV(m *yaml.Node, key string, v *yaml.Node) {
	m.Content = append(m.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		v,
	)
}

// writeAtomic writes b to path via a same-directory <path>.tmp file
// and an os.Rename. Atomic on POSIX filesystems.
func writeAtomic(path string, b []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("import-pdf: tempfile %s: %w", path, err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // best-effort cleanup on error path
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return fmt.Errorf("import-pdf: write %s: %w", tmpPath, err)
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("import-pdf: rename %s → %s: %w", tmpPath, path, err)
	}
	return nil
}

// renderDryRun produces the human-readable diff-like summary for
// --dry-run mode. Writes to w.
func renderDryRun(w io.Writer, res mergeResult, configPath string) {
	fmt.Fprintln(w, "import-pdf: dry-run — no files written")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Changes:")
	for _, c := range res.Changes {
		fmt.Fprintf(w, "  - %s\n", c)
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Would write %s (%d bytes)\n", configPath, len(res.ConfigYAML))
	for _, c := range res.CSVs {
		fmt.Fprintf(w, "Would write %s (%d bytes)\n", c.Path, len(c.Data))
	}
}
