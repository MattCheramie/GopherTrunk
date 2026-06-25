package log

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func TestMessageLogWritesEventLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "messages.log")
	bus := events.NewBus(16)
	defer bus.Close()

	ml, err := NewMessageLog(MessageLogOptions{Bus: bus, Path: path})
	if err != nil {
		t.Fatalf("NewMessageLog: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go ml.Run(ctx)

	bus.Publish(events.Event{
		Kind: events.KindGrant,
		Payload: trunking.Grant{
			System: "Metro", Protocol: "p25", GroupID: 1234, SourceID: 99,
			FrequencyHz: 851_000_000,
		},
	})
	bus.Publish(events.Event{
		Kind: events.KindAffiliation,
		Payload: trunking.Affiliation{
			System: "Metro", Protocol: "dmr", SourceID: 7, GroupID: 50,
		},
	})

	var data []byte
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		data, _ = os.ReadFile(path)
		if strings.Contains(string(data), "GRANT") && strings.Contains(string(data), "AFFILIATION") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	ml.Close()

	s := string(data)
	if !strings.Contains(s, "GRANT") || !strings.Contains(s, "tg=1234") {
		t.Fatalf("grant line missing from log:\n%s", s)
	}
	if !strings.Contains(s, "AFFILIATION") || !strings.Contains(s, "src=7") {
		t.Fatalf("affiliation line missing from log:\n%s", s)
	}
}

func TestMessageLogRotates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "messages.log")
	// Pre-fill the file past a 1 MiB cap so the next write rotates.
	big := make([]byte, 1024*1024+10)
	for i := range big {
		big[i] = 'x'
	}
	if err := os.WriteFile(path, big, 0o644); err != nil {
		t.Fatal(err)
	}
	bus := events.NewBus(8)
	defer bus.Close()
	ml, err := NewMessageLog(MessageLogOptions{Bus: bus, Path: path, MaxSizeMB: 1})
	if err != nil {
		t.Fatalf("NewMessageLog: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go ml.Run(ctx)

	bus.Publish(events.Event{
		Kind:    events.KindGrant,
		Payload: trunking.Grant{System: "Metro", GroupID: 1},
	})

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path + ".1"); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	ml.Close()

	if _, err := os.Stat(path + ".1"); err != nil {
		t.Fatal("expected rotated file messages.log.1")
	}
}

// TestMessageLogFormatsInConfiguredLocation pins the timezone fix: a displayed
// timestamp is rendered in the configured location with an explicit numeric
// offset, not forced to UTC ("…Z"). Fails against the old ts.UTC() formatting.
func TestMessageLogFormatsInConfiguredLocation(t *testing.T) {
	loc := time.FixedZone("TST", 5*3600+30*60) // +05:30
	m := &MessageLog{loc: loc, lastNeighbors: map[string]string{}}
	ts := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC) // 12:00 UTC → 17:30 +05:30
	line := m.formatEvent(events.Event{
		Kind:      events.KindGrant,
		Timestamp: ts,
		Payload:   trunking.Grant{System: "Metro", Protocol: "p25", GroupID: 1, FrequencyHz: 851_000_000},
	})
	const wantPrefix = "2026-06-25T17:30:00.000+05:30 "
	if !strings.HasPrefix(line, wantPrefix) {
		t.Fatalf("want localized stamp prefix %q, got line: %q", wantPrefix, line)
	}
}

// TestMessageLogNeighborBlock checks the SDRtrunk-style neighbor block: it is
// emitted on a SiteUpdate carrying neighbors, deduplicated when the set is
// unchanged, and re-emitted when the set changes.
func TestMessageLogNeighborBlock(t *testing.T) {
	m := &MessageLog{loc: time.UTC, lastNeighbors: map[string]string{}}
	topo := &trunking.TopologySnapshot{
		Neighbors: []trunking.TopoNeighborRef{
			{RFSS: 1, Site: 7, ChannelID: 2, ChannelNumber: 1754, FrequencyHz: 450_962_500},
			{RFSS: 1, Site: 9, ChannelID: 2, ChannelNumber: 1654, FrequencyHz: 450_337_500},
		},
	}
	ev := events.Event{
		Kind:      events.KindSiteUpdate,
		Timestamp: time.Unix(0, 0),
		Payload:   trunking.SiteUpdate{System: "Metro", Topology: topo},
	}

	first := m.formatEvent(ev)
	if !strings.Contains(first, "NEIGHBORS") || !strings.Contains(first, "count=2") {
		t.Fatalf("first block missing header: %q", first)
	}
	if got := strings.Count(first, " NEIGHBOR     "); got != 2 {
		t.Fatalf("want 2 neighbor lines, got %d: %q", got, first)
	}
	if !strings.Contains(first, "RFSS:1[1] SITE:7[7] CHANNEL:2-1754 DOWNLINK:450.962500 MHz") {
		t.Errorf("neighbor line format unexpected: %q", first)
	}

	// Same set again → deduplicated to nothing.
	if dup := m.formatEvent(ev); dup != "" {
		t.Errorf("unchanged neighbor set should dedup, got: %q", dup)
	}

	// Changed set → emitted again.
	ev2 := events.Event{
		Kind:      events.KindSiteUpdate,
		Timestamp: time.Unix(0, 0),
		Payload: trunking.SiteUpdate{System: "Metro", Topology: &trunking.TopologySnapshot{
			Neighbors: []trunking.TopoNeighborRef{{RFSS: 2, Site: 12, ChannelID: 2, ChannelNumber: 1660, FrequencyHz: 450_375_000}},
		}},
	}
	if changed := m.formatEvent(ev2); !strings.Contains(changed, "SITE:C[12]") {
		t.Errorf("changed neighbor set should re-emit, got: %q", changed)
	}
}

func TestMessageLogValidatesOptions(t *testing.T) {
	if _, err := NewMessageLog(MessageLogOptions{Path: "x"}); err == nil {
		t.Error("NewMessageLog without a bus should error")
	}
	if _, err := NewMessageLog(MessageLogOptions{Bus: events.NewBus(1)}); err == nil {
		t.Error("NewMessageLog without a path should error")
	}
}
