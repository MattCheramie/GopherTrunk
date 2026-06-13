package hunt

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"
)

// NDJSONSink appends one DetectedSignal per line to a newline-delimited JSON
// file, fsyncing after each record so an interrupted survey leaves a complete,
// resumable trail on disk (the per-record durability the end-of-run survey.json
// can't offer). It is the streaming peer of WriteSurvey: WriteSurvey snapshots
// the whole inventory at the end, NDJSONSink records each carrier as it is
// classified, via the survey's OnSignal hook.
type NDJSONSink struct {
	mu sync.Mutex
	f  *os.File
}

// OpenNDJSONSink opens (creating if needed) path for append. The caller must
// Close it when the run ends.
func OpenNDJSONSink(path string) (*NDJSONSink, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &NDJSONSink{f: f}, nil
}

// Write marshals ds to one line and fsyncs. Safe for concurrent callers (the
// daemon invokes OnSignal from a single goroutine, but the lock keeps it sound
// regardless).
func (s *NDJSONSink) Write(ds DetectedSignal) error {
	b, err := json.Marshal(ds)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.f.Write(append(b, '\n')); err != nil {
		return err
	}
	return s.f.Sync()
}

// Close closes the underlying file.
func (s *NDJSONSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.f.Close()
}

// LoadSurveyedFreqs reads the frequencies already recorded in an NDJSON survey
// file so a resumed run can skip them. A missing file yields an empty set and no
// error (first run). Unparseable lines (e.g. a record half-written before a
// crash) are skipped rather than failing the load.
func LoadSurveyedFreqs(path string) (map[uint32]struct{}, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[uint32]struct{}{}, nil
		}
		return nil, err
	}
	defer f.Close()

	out := map[uint32]struct{}{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var rec struct {
			FreqHz uint32 `json:"freq_hz"`
		}
		if err := json.Unmarshal(line, &rec); err != nil || rec.FreqHz == 0 {
			continue
		}
		out[rec.FreqHz] = struct{}{}
	}
	if err := sc.Err(); err != nil {
		return out, err
	}
	return out, nil
}
