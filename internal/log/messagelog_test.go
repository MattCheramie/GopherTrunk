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

func TestMessageLogValidatesOptions(t *testing.T) {
	if _, err := NewMessageLog(MessageLogOptions{Path: "x"}); err == nil {
		t.Error("NewMessageLog without a bus should error")
	}
	if _, err := NewMessageLog(MessageLogOptions{Bus: events.NewBus(1)}); err == nil {
		t.Error("NewMessageLog without a path should error")
	}
}
