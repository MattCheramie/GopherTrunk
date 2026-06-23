package log

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// waitForLines polls path until it has at least n non-empty lines or the
// deadline passes, returning whatever it last read.
func waitForLines(t *testing.T, path string, n int) []string {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		data, _ := os.ReadFile(path)
		lines := nonEmptyLines(string(data))
		if len(lines) >= n {
			return lines
		}
		time.Sleep(10 * time.Millisecond)
	}
	data, _ := os.ReadFile(path)
	return nonEmptyLines(string(data))
}

func nonEmptyLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}

func TestEventLogWritesOneJSONLinePerEvent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	bus := events.NewBus(16)
	defer bus.Close()

	el, err := NewEventLog(EventLogOptions{Bus: bus, Path: path})
	if err != nil {
		t.Fatalf("NewEventLog: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go el.Run(ctx)

	bus.Publish(events.Event{
		Kind:      events.KindGrant,
		Timestamp: time.Unix(1700000000, 0).UTC(),
		Payload: trunking.Grant{
			System: "Metro", Protocol: "p25", GroupID: 1234, SourceID: 99,
			FrequencyHz: 851_000_000,
		},
	})
	bus.Publish(events.Event{
		Kind:    events.KindAffiliation,
		Payload: trunking.Affiliation{System: "Metro", Protocol: "dmr", SourceID: 7, GroupID: 50},
	})

	lines := waitForLines(t, path, 2)
	cancel()
	el.Close()

	if len(lines) != 2 {
		t.Fatalf("got %d JSON lines, want 2:\n%s", len(lines), strings.Join(lines, "\n"))
	}
	// Every line must be a self-contained JSON object with a non-empty kind.
	var first struct {
		Kind      string          `json:"kind"`
		Timestamp time.Time       `json:"timestamp"`
		Payload   json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("line 0 is not valid JSON: %v\n%s", err, lines[0])
	}
	if first.Kind != string(events.KindGrant) {
		t.Errorf("line 0 kind = %q, want %q", first.Kind, events.KindGrant)
	}
	if len(first.Payload) == 0 {
		t.Errorf("line 0 has empty payload: %s", lines[0])
	}
	var second map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &second); err != nil {
		t.Fatalf("line 1 is not valid JSON: %v\n%s", err, lines[1])
	}
	if second["kind"] != string(events.KindAffiliation) {
		t.Errorf("line 1 kind = %v, want %q", second["kind"], events.KindAffiliation)
	}
}

func TestEventLogUsesInjectedEncoder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	bus := events.NewBus(16)
	defer bus.Close()

	el, err := NewEventLog(EventLogOptions{
		Bus:  bus,
		Path: path,
		Encode: func(ev events.Event) ([]byte, error) {
			return json.Marshal(map[string]string{"k": string(ev.Kind), "custom": "yes"})
		},
	})
	if err != nil {
		t.Fatalf("NewEventLog: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go el.Run(ctx)

	bus.Publish(events.Event{Kind: events.KindGrant, Payload: trunking.Grant{System: "X"}})

	lines := waitForLines(t, path, 1)
	cancel()
	el.Close()

	if len(lines) != 1 || !strings.Contains(lines[0], `"custom":"yes"`) {
		t.Fatalf("injected encoder not used:\n%v", lines)
	}
}

func TestEventLogRotatesPastSizeCap(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")
	bus := events.NewBus(64)
	defer bus.Close()

	// 1 MB cap via a tiny-payload encoder forces a rotation after enough events
	// without writing a huge file. Use a fixed ~1 KB line and a cap just above
	// it so the second batch trips the rotation.
	el, err := NewEventLog(EventLogOptions{
		Bus: bus, Path: path, MaxSizeMB: 1,
		Encode: func(ev events.Event) ([]byte, error) {
			return []byte(strings.Repeat("a", 200*1024)), nil // 200 KB per line
		},
	})
	if err != nil {
		t.Fatalf("NewEventLog: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go el.Run(ctx)

	// 6 × 200 KB = 1.2 MB > 1 MB cap ⇒ at least one rotation to <path>.1.
	for i := 0; i < 6; i++ {
		bus.Publish(events.Event{Kind: events.KindGrant, Payload: trunking.Grant{}})
	}

	rotated := filepath.Join(dir, "events.jsonl.1")
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(rotated); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	el.Close()

	if _, err := os.Stat(rotated); err != nil {
		t.Fatalf("expected rotated file %s to exist: %v", rotated, err)
	}
}
