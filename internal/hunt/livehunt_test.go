package hunt

import (
	"context"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestRunLiveHunt_CandidateList drives the live hunt over a synthesized P25
// control-channel buffer served by a FileIQSource, skipping the sweep via an
// explicit candidate list. It proves the candidate→identify→decode→accumulate
// path produces the same kind of mapped system the offline Discover does.
func TestRunLiveHunt_CandidateList(t *testing.T) {
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
		Source:        src,
		Candidates:    []uint32{candHz},
		DwellSeconds:  float64(len(iq)) / float64(rate),
		MinConfidence: 0.3,
		Name:          "Live Test",
	})
	if err != nil {
		t.Fatalf("RunLiveHunt: %v", err)
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
	// The candidate frequency is recorded as the control channel on a site.
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
	if sys.DisplayName() != "Live Test" {
		t.Errorf("name = %q, want Live Test", sys.DisplayName())
	}
}

// TestRunLiveHunt_Progress confirms progress callbacks fire through the
// identifying/done phases on the candidate-list path.
func TestRunLiveHunt_Progress(t *testing.T) {
	iq, meta, err := siglab.Synthesize(siglab.SynthOptions{
		Protocol: trunking.ProtocolP25,
		Format:   siglab.FormatF32,
	})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	rate := uint32(meta.SampleRateHz)

	phases := map[LiveHuntPhase]int{}
	_, _, err = RunLiveHunt(context.Background(), LiveHuntOptions{
		Source:        NewFileIQSource(iq, rate),
		Candidates:    []uint32{851_000_000},
		DwellSeconds:  float64(len(iq)) / float64(rate),
		MinConfidence: 0.3,
		OnProgress:    func(p LiveHuntProgress) { phases[p.Phase]++ },
	})
	if err != nil {
		t.Fatalf("RunLiveHunt: %v", err)
	}
	if phases[PhaseIdentifying] == 0 {
		t.Error("expected at least one identifying-phase progress event")
	}
	if phases[PhaseDone] == 0 {
		t.Error("expected a done-phase progress event")
	}
}
