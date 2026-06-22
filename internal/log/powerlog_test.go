package log

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// waitForFile polls path until it contains want or the deadline passes,
// returning the last-read contents.
func waitForFile(t *testing.T, path, want string) string {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		data, _ := os.ReadFile(path)
		if strings.Contains(string(data), want) {
			return string(data)
		}
		time.Sleep(10 * time.Millisecond)
	}
	data, _ := os.ReadFile(path)
	return string(data)
}

func powerEvent(low bool) events.Event {
	dbfs := -20.0
	if low {
		dbfs = -70.0
	}
	return events.Event{
		Kind: events.KindChannelPower,
		Payload: events.ChannelPower{
			System: "Metro", Protocol: "p25-phase2", FreqHz: 851_000_000,
			PowerDbFS: dbfs, Decoded: 4, LowPower: low,
		},
	}
}

// TestPowerLogLowPowerOnly: with AllWindows false (default), only low-power
// windows are written; healthy windows are skipped.
func TestPowerLogLowPowerOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "power.log")
	bus := events.NewBus(16)
	defer bus.Close()

	pl, err := NewPowerLog(PowerLogOptions{Bus: bus, Path: path})
	if err != nil {
		t.Fatalf("NewPowerLog: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go pl.Run(ctx)

	bus.Publish(powerEvent(false)) // healthy — must be skipped
	bus.Publish(powerEvent(true))  // low — must be written

	s := waitForFile(t, path, "low=true")
	cancel()
	pl.Close()

	if !strings.Contains(s, "POWER") || !strings.Contains(s, "freq=851000000") {
		t.Fatalf("low-power line missing from log:\n%s", s)
	}
	if strings.Contains(s, "low=false") {
		t.Fatalf("healthy window should have been skipped:\n%s", s)
	}
}

// TestPowerLogAllWindows: with AllWindows true, healthy windows are written
// too.
func TestPowerLogAllWindows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "power.log")
	bus := events.NewBus(16)
	defer bus.Close()

	pl, err := NewPowerLog(PowerLogOptions{Bus: bus, Path: path, AllWindows: true})
	if err != nil {
		t.Fatalf("NewPowerLog: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go pl.Run(ctx)

	bus.Publish(powerEvent(false))

	s := waitForFile(t, path, "low=false")
	cancel()
	pl.Close()

	if !strings.Contains(s, "low=false") {
		t.Fatalf("AllWindows should write a healthy window:\n%s", s)
	}
}

func TestPowerLogValidatesOptions(t *testing.T) {
	if _, err := NewPowerLog(PowerLogOptions{Path: "x"}); err == nil {
		t.Error("NewPowerLog without a bus should error")
	}
	if _, err := NewPowerLog(PowerLogOptions{Bus: events.NewBus(1)}); err == nil {
		t.Error("NewPowerLog without a path should error")
	}
}
