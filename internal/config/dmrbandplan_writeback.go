package config

import (
	"bytes"
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// WriteDMRLearnedBandPlan persists a learned DMR Tier III linear band
// plan into the named trunking system, preserving comments and unrelated
// content the way WritePatch does. It is used by the DMR LCN autoconfig
// learner so a plan discovered at runtime survives a restart. The system
// must already exist in the file and must not already carry a
// dmr_band_plan (the learner only runs when none is configured); both
// conditions are enforced so the writeback never clobbers an operator's
// hand-authored plan. The mtime guard refuses to overwrite an external
// edit, matching WritePatch.
func (w *Writer) WriteDMRLearnedBandPlan(systemName string, lin DMRLinearBandPlanConfig) error {
	if w == nil {
		return fmt.Errorf("config: no writer wired (daemon started without a -config file)")
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	info, err := os.Stat(w.path)
	if err != nil {
		return fmt.Errorf("config: writer: stat: %w", err)
	}
	if info.ModTime().UnixNano() != w.knownMtime {
		return fmt.Errorf("config: %s was modified externally; reload the daemon before persisting a learned band plan", w.path)
	}

	data, err := os.ReadFile(w.path)
	if err != nil {
		return fmt.Errorf("config: writer: read: %w", err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("config: writer: parse node: %w", err)
	}
	root := docRoot(&doc)
	if root == nil {
		return fmt.Errorf("config: writer: empty config document")
	}

	sysNode, err := findSystemNode(root, systemName)
	if err != nil {
		return err
	}
	if mappingChild(sysNode, "dmr_band_plan") != nil {
		return fmt.Errorf("config: system %q already has a dmr_band_plan; refusing to overwrite", systemName)
	}
	setMappingPath(sysNode, []string{"dmr_band_plan", "linear", "base_hz"}, scalarUint(uint64(lin.BaseHz)))
	setMappingPath(sysNode, []string{"dmr_band_plan", "linear", "spacing_hz"}, scalarUint(uint64(lin.SpacingHz)))
	setMappingPath(sysNode, []string{"dmr_band_plan", "linear", "offset"}, scalarInt(int64(lin.Offset)))

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return fmt.Errorf("config: writer: encode: %w", err)
	}
	if err := enc.Close(); err != nil {
		return err
	}

	// Validate the merged result before committing.
	var merged Config
	if err := yaml.Unmarshal(buf.Bytes(), &merged); err != nil {
		return fmt.Errorf("config: writer: re-parse: %w", err)
	}
	if err := merged.Validate(); err != nil {
		return fmt.Errorf("config: writer: validate: %w", err)
	}

	if err := writeFileAtomic(w.path, buf.Bytes(), 0o644); err != nil {
		return err
	}
	if info, err := os.Stat(w.path); err == nil {
		w.knownMtime = info.ModTime().UnixNano()
	}
	return nil
}

// findSystemNode walks root → trunking → systems[] and returns the
// mapping node whose `name` equals systemName.
func findSystemNode(root *yaml.Node, systemName string) (*yaml.Node, error) {
	trunking := mappingChild(root, "trunking")
	if trunking == nil {
		return nil, fmt.Errorf("config: no trunking section to persist band plan into")
	}
	systems := mappingChild(trunking, "systems")
	if systems == nil || systems.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("config: no trunking.systems sequence")
	}
	for _, el := range systems.Content {
		if el.Kind != yaml.MappingNode {
			continue
		}
		name := mappingChild(el, "name")
		if name != nil && name.Value == systemName {
			return el, nil
		}
	}
	return nil, fmt.Errorf("config: system %q not found in %s", systemName, "trunking.systems")
}

// mappingChild returns the value node for key in a mapping node, or nil.
func mappingChild(m *yaml.Node, key string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

func scalarUint(v uint64) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.FormatUint(v, 10)}
}

func scalarInt(v int64) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.FormatInt(v, 10)}
}
