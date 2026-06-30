package rfscope

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

func sceneWithEntropy(results []EntropyResult) *Scene {
	return &Scene{AnalyzerOutputs: map[string]any{"entropy": results}}
}

func TestEmitFramesFormat(t *testing.T) {
	t.Parallel()
	sc := sceneWithEntropy([]EntropyResult{
		{EmitterID: 1, FreqHz: 450_100_000, payload: []byte{0xde, 0xad, 0xbe, 0xef}},
		{EmitterID: 2, FreqHz: 450_500_000, payload: []byte{0x01, 0x02}, iv: []byte{0xaa, 0xbb}},
		{EmitterID: 3, FreqHz: 450_900_000, payload: nil}, // skipped (no payload)
	})

	var buf bytes.Buffer
	n, err := EmitFrames(sc, &buf)
	if err != nil {
		t.Fatalf("EmitFrames: %v", err)
	}
	if n != 2 {
		t.Fatalf("want 2 frames written, got %d", n)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 JSONL lines, got %d:\n%s", len(lines), buf.String())
	}

	// Each line must parse with the exact {label,iv,ct} fields the ks loader uses,
	// with iv/ct hex-decodable back to the originals.
	var first frameRec
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("frame 0 not valid JSON: %v", err)
	}
	ct, err := hex.DecodeString(first.CT)
	if err != nil || !bytes.Equal(ct, []byte{0xde, 0xad, 0xbe, 0xef}) {
		t.Fatalf("frame 0 ct round-trip failed: %q err=%v", first.CT, err)
	}
	if first.IV != "" {
		t.Fatalf("frame 0 should have empty iv, got %q", first.IV)
	}

	var second frameRec
	_ = json.Unmarshal([]byte(lines[1]), &second)
	if second.IV != "aabb" {
		t.Fatalf("frame 1 iv hex should be aabb, got %q", second.IV)
	}
}

func TestDetectKeystreamReuse(t *testing.T) {
	t.Parallel()
	// Two payloads sharing an IV ⇒ a reuse group; the third (distinct IV) does not.
	sc := sceneWithEntropy([]EntropyResult{
		{EmitterID: 1, payload: []byte("aaaaaaaa"), iv: []byte{0x01, 0x02, 0x03}},
		{EmitterID: 2, payload: []byte("bbbbbbbb"), iv: []byte{0x01, 0x02, 0x03}},
		{EmitterID: 3, payload: []byte("cccccccc"), iv: []byte{0x09, 0x09, 0x09}},
	})
	groups := DetectKeystreamReuse(sc)
	if len(groups) != 1 {
		t.Fatalf("want 1 reuse group, got %d", len(groups))
	}
	if len(groups[0].Frames) != 2 {
		t.Fatalf("reuse group should hold 2 frames, got %d", len(groups[0].Frames))
	}
	// No-IV payloads must never form a reuse group.
	noIV := sceneWithEntropy([]EntropyResult{
		{EmitterID: 1, payload: []byte("xxxxxxxx")},
		{EmitterID: 2, payload: []byte("yyyyyyyy")},
	})
	if g := DetectKeystreamReuse(noIV); len(g) != 0 {
		t.Fatalf("payloads without IV must not form reuse groups, got %d", len(g))
	}
}
