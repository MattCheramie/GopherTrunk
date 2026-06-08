package config

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

// MarshalMerge renders cfg to YAML while preserving the comments and key
// ordering of an existing file. It parses `original` into a yaml.Node tree
// (which carries comments), parses a fresh Marshal(cfg) into an overlay
// tree, then deep-merges the overlay onto the original:
//
//   - keys present in both: scalar/sequence values are replaced (the old
//     value node's trailing comment is carried over); nested mappings are
//     merged recursively, so section and key comments survive;
//   - keys new to cfg are appended;
//   - keys removed from cfg are dropped.
//
// Per-list-item comments inside a replaced sequence are not preserved
// (sequences are swapped wholesale) — config comments are overwhelmingly
// section/key level, so this keeps the annotated file readable after an
// edit from the web Config Builder. If `original` isn't a YAML mapping
// (empty/new file), it falls back to plain Marshal.
func MarshalMerge(original []byte, cfg Config) ([]byte, error) {
	overlayBytes, err := Marshal(cfg)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(original)) == 0 {
		return overlayBytes, nil
	}

	var baseDoc, overlayDoc yaml.Node
	if err := yaml.Unmarshal(original, &baseDoc); err != nil {
		// Original isn't parseable; don't try to preserve it.
		return overlayBytes, nil
	}
	if err := yaml.Unmarshal(overlayBytes, &overlayDoc); err != nil {
		return nil, fmt.Errorf("config: merge: parse overlay: %w", err)
	}
	baseRoot := docRoot(&baseDoc)
	overlayRoot := docRoot(&overlayDoc)
	if baseRoot == nil || overlayRoot == nil {
		return overlayBytes, nil
	}
	mergeMappingNodes(baseRoot, overlayRoot)

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&baseDoc); err != nil {
		return nil, fmt.Errorf("config: merge: encode: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("config: merge: encode: %w", err)
	}
	return buf.Bytes(), nil
}

// mergeMappingNodes merges overlay into base in place, preserving base's
// key order + comments for retained keys, appending overlay-only keys, and
// dropping base keys absent from overlay.
func mergeMappingNodes(base, overlay *yaml.Node) {
	type pair struct{ key, val *yaml.Node }
	overlayPairs := make([]pair, 0, len(overlay.Content)/2)
	overlayByKey := make(map[string]*yaml.Node, len(overlay.Content)/2)
	for i := 0; i+1 < len(overlay.Content); i += 2 {
		overlayPairs = append(overlayPairs, pair{overlay.Content[i], overlay.Content[i+1]})
		overlayByKey[overlay.Content[i].Value] = overlay.Content[i+1]
	}

	merged := make([]*yaml.Node, 0, len(base.Content))
	kept := make(map[string]bool, len(overlayByKey))
	for i := 0; i+1 < len(base.Content); i += 2 {
		keyNode := base.Content[i]
		valNode := base.Content[i+1]
		ov, ok := overlayByKey[keyNode.Value]
		if !ok {
			continue // removed from cfg
		}
		kept[keyNode.Value] = true
		if valNode.Kind == yaml.MappingNode && ov.Kind == yaml.MappingNode {
			mergeMappingNodes(valNode, ov)
			merged = append(merged, keyNode, valNode)
			continue
		}
		carryComments(valNode, ov)
		merged = append(merged, keyNode, ov)
	}
	// Append keys new to cfg, in overlay order.
	for _, p := range overlayPairs {
		if kept[p.key.Value] {
			continue
		}
		merged = append(merged, p.key, p.val)
	}
	base.Content = merged
}

// carryComments copies the old value node's comments onto the replacement
// when the replacement has none, so a trailing `# note` survives a scalar
// edit.
func carryComments(old, new *yaml.Node) {
	if new.HeadComment == "" {
		new.HeadComment = old.HeadComment
	}
	if new.LineComment == "" {
		new.LineComment = old.LineComment
	}
	if new.FootComment == "" {
		new.FootComment = old.FootComment
	}
}
