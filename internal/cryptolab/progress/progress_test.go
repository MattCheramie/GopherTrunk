package progress

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadCheckpointRoundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "checkpoint.json")

	type state struct {
		Pinned int            `json:"pinned"`
		Cands  map[string]int `json:"cands"`
	}
	want := state{Pinned: 7, Cands: map[string]int{"a": 1, "b": 2}}
	if err := SaveCheckpoint(path, want); err != nil {
		t.Fatalf("save: %v", err)
	}

	var got state
	found, err := LoadCheckpoint(path, &got)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !found {
		t.Fatal("found = false for a written checkpoint")
	}
	if got.Pinned != want.Pinned || len(got.Cands) != 2 || got.Cands["b"] != 2 {
		t.Fatalf("round-trip mismatch: got %+v want %+v", got, want)
	}
}

func TestLoadCheckpointMissing(t *testing.T) {
	t.Parallel()
	var v map[string]int
	found, err := LoadCheckpoint(filepath.Join(t.TempDir(), "nope.json"), &v)
	if err != nil {
		t.Fatalf("missing checkpoint should not error: %v", err)
	}
	if found {
		t.Fatal("found = true for a missing checkpoint")
	}
}

func TestSaveCheckpointAtomicNoTemp(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "ck.json")
	if err := SaveCheckpoint(path, map[string]int{"x": 1}); err != nil {
		t.Fatal(err)
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 1 || ents[0].Name() != "ck.json" {
		t.Fatalf("expected only ck.json, got %v", ents)
	}
}

func TestJSONLNilSafe(t *testing.T) {
	t.Parallel()
	var j *JSONL
	if err := j.Append(map[string]int{"a": 1}); err != nil {
		t.Fatalf("nil Append: %v", err)
	}
	if err := j.Close(); err != nil {
		t.Fatalf("nil Close: %v", err)
	}

	j2, err := OpenJSONL(t.TempDir(), "out.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if err := j2.Append(map[string]int{"a": 1}); err != nil {
		t.Fatal(err)
	}
	if err := j2.Close(); err != nil {
		t.Fatal(err)
	}
}
