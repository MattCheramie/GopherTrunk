package cryptocap

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestFileWriterEmitsKsParseableLines(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "cap.jsonl")
	w, err := NewFileWriter(path)
	if err != nil {
		t.Fatal(err)
	}
	w.WriteCryptoFrame(Frame{
		System: "MMR", Protocol: "p25-phase1", Serial: "VOICE-1", CallID: 7, TG: 4321,
		AlgID: 0x84, KeyID: 0x1234,
		MI: []byte{1, 2, 3}, CT: []byte{0xde, 0xad, 0xbe, 0xef},
		At: time.Unix(1700000000, 0),
	})
	// Empty IV or CT must be dropped.
	w.WriteCryptoFrame(Frame{Serial: "VOICE-1", MI: nil, CT: []byte{1}})
	w.WriteCryptoFrame(Frame{Serial: "VOICE-1", MI: []byte{1}, CT: nil})
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var lines []map[string]any
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var m map[string]any
		if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
			t.Fatalf("line is not valid JSON: %v", err)
		}
		lines = append(lines, m)
	}
	if len(lines) != 1 {
		t.Fatalf("want 1 record (2 empties dropped), got %d", len(lines))
	}
	rec := lines[0]
	// The three fields cryptolab `ks` parses must be present and correct.
	if rec["iv"] != "010203" {
		t.Errorf("iv = %v, want 010203", rec["iv"])
	}
	if rec["ct"] != "deadbeef" {
		t.Errorf("ct = %v, want deadbeef", rec["ct"])
	}
	if _, ok := rec["label"].(string); !ok || rec["label"] == "" {
		t.Errorf("missing label: %v", rec["label"])
	}
}

func TestFileWriterConcurrent(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "cap.jsonl")
	w, err := NewFileWriter(path)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				w.WriteCryptoFrame(Frame{Serial: "S", MI: []byte{byte(g)}, CT: []byte{byte(i)}})
			}
		}(g)
	}
	wg.Wait()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	n := 0
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var m map[string]any
		if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
			t.Fatalf("corrupt concurrent write (interleaved line): %v", err)
		}
		n++
	}
	if n != 8*50 {
		t.Fatalf("want %d records, got %d", 8*50, n)
	}
}
