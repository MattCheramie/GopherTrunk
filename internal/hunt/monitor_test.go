package hunt

import (
	"context"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestRunLiveHunt_Monitor drives the streaming long-dwell monitor path
// (MonitorSeconds > 0) over a synthesized P25 control-channel buffer served
// cyclically by a FileIQSource. It proves the live stream is decoded in real
// time (no whole-dwell buffer) and still maps the system the same way the
// buffered path does, while the dwell cap bounds the run.
func TestRunLiveHunt_Monitor(t *testing.T) {
	iq, meta, err := siglab.Synthesize(siglab.SynthOptions{
		Protocol: trunking.ProtocolP25,
		Format:   siglab.FormatF32,
	})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	rate := uint32(meta.SampleRateHz)
	src := NewFileIQSource(iq, rate)

	const candHz = 851_000_000
	sys, reports, err := RunLiveHunt(context.Background(), LiveHuntOptions{
		Source:     src,
		Candidates: []uint32{candHz},
		// Short identify prefix; the monitor then streams the cyclic buffer.
		DwellSeconds: float64(len(iq)) / float64(rate),
		// Cap the dwell so the test stops promptly even if identity never
		// corroborates (converge-and-stop may end it sooner).
		MonitorSeconds: 2,
		MinConfidence:  0.3,
		Name:           "Monitor Test",
	})
	if err != nil {
		t.Fatalf("RunLiveHunt (monitor): %v", err)
	}
	if len(reports) != 1 {
		t.Fatalf("len(reports) = %d, want 1", len(reports))
	}
	r := reports[0]
	if r.Error != "" {
		t.Fatalf("candidate errored: %s", r.Error)
	}
	if r.Skipped {
		t.Fatalf("candidate skipped: %s", r.SkipReason)
	}
	if r.Protocol != "p25" {
		t.Errorf("protocol = %q, want p25", r.Protocol)
	}
	if !r.Locked {
		t.Errorf("expected a lock on the synthesized P25 control channel")
	}
	if len(sys.Sites) != 1 {
		t.Fatalf("len(Sites) = %d, want 1", len(sys.Sites))
	}
	foundCC := false
	for _, ch := range sys.Sites[0].ControlChannels {
		if ch.FrequencyHz == candHz {
			foundCC = true
		}
	}
	if !foundCC {
		t.Errorf("control channel %d not recorded; site = %+v", candHz, sys.Sites[0])
	}
}

// TestConvergeAndStop checks the converge-and-stop policy: it holds until
// identity is present, waits out the minimum dwell, then stops once the
// topology stops changing for the stability window.
func TestConvergeAndStop(t *testing.T) {
	stop := convergeAndStop()
	withID := &siglab.TopologySnapshot{WACN: 0xBEE08, SystemID: 0x534}

	// No identity yet ⇒ never converges, however long the dwell.
	if stop(&siglab.TopologySnapshot{}, 1000) {
		t.Fatal("converged with no identity")
	}
	// Identity present but inside the minimum-dwell floor ⇒ keep going.
	if stop(withID, monitorMinDwell.Seconds()-1) {
		t.Fatal("converged before the minimum dwell")
	}
	// Identity present, past the floor, but not yet stable for the window.
	base := monitorMinDwell.Seconds() + 1
	if stop(withID, base) {
		t.Fatal("converged before the stability window elapsed")
	}
	// Stable for the whole window ⇒ converge.
	if !stop(withID, base+monitorStability.Seconds()) {
		t.Fatal("did not converge after a stable stability window")
	}
}
