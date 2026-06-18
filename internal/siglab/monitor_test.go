package siglab

import (
	"bytes"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestRunReaderMonitor_TickAndStop checks the streaming monitor entry point:
// the tick callback fires on the stream-time cadence with the live topology,
// and returning true ends the run early.
func TestRunReaderMonitor_TickAndStop(t *testing.T) {
	iq, meta, err := Synthesize(SynthOptions{Protocol: trunking.ProtocolP25, Format: FormatF32})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	data := EncodeCapture(iq, FormatF32)
	cfg := Config{
		Protocol:     trunking.ProtocolP25,
		SystemName:   "monitor-test",
		SampleRateHz: meta.SampleRateHz,
		Format:       FormatF32,
		// Deep path required for topology() to be wired (the tick reads it).
		CollectIQDiag: true,
	}

	// Tick fires; returning false lets the run read to EOF.
	ticks := 0
	res, err := RunReaderMonitor(bytes.NewReader(data), "monitor", cfg, 50*time.Millisecond,
		func(topo *TopologySnapshot, sec float64) bool {
			ticks++
			return false
		})
	if err != nil {
		t.Fatalf("RunReaderMonitor: %v", err)
	}
	if res == nil {
		t.Fatal("nil result")
	}
	if ticks == 0 {
		t.Error("expected at least one monitor tick over the capture")
	}

	// Returning true on the first tick must stop the run early without error.
	stopped := 0
	res2, err := RunReaderMonitor(bytes.NewReader(data), "monitor", cfg, 50*time.Millisecond,
		func(topo *TopologySnapshot, sec float64) bool {
			stopped++
			return true
		})
	if err != nil {
		t.Fatalf("RunReaderMonitor (early stop): %v", err)
	}
	if res2 == nil {
		t.Fatal("nil result on early stop")
	}
	if stopped != 1 {
		t.Errorf("early-stop callback fired %d times, want 1", stopped)
	}
}

// TestRunReaderMonitor_RejectsAutoTune documents that the streaming monitor
// requires a pre-resolved carrier (cfg.TuneHz), since it cannot seek to
// re-estimate one mid-stream.
func TestRunReaderMonitor_RejectsAutoTune(t *testing.T) {
	cfg := Config{
		Protocol:     trunking.ProtocolP25,
		SampleRateHz: 48_000,
		Format:       FormatF32,
		AutoTune:     true,
	}
	if _, err := RunReaderMonitor(bytes.NewReader(nil), "x", cfg, time.Second, func(*TopologySnapshot, float64) bool { return false }); err == nil {
		t.Fatal("expected an error when AutoTune is set")
	}
}
