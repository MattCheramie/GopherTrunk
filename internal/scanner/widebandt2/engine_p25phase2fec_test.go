package widebandt2

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	p25phase1 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	p25phase2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
)

// TestBuildChannelP25Phase1WiresPhase2FEC is the regression test for
// issue #882: a wideband Phase 1 control channel on a hybrid system
// (Phase 1 CC issuing Phase 2 TDMA voice grants — e.g. Victorian MMR)
// must forward the per-system Phase 2 FEC config (trellis / RS /
// interleave / scrambler) into the Phase 1 CC decoder. Before the fix
// buildChannel constructed p25phase1.Options without those fields, so
// they defaulted to zero (Trellis=off, Scrambler=off) and every Phase 2
// grant the CC published decoded with the FEC chain disabled — the
// voice composer then warned and failed all MAC PDU decodes on the
// traffic channel, even with p25_phase2_trellis_mode=on configured.
//
// The equivalent ccdecoder pipeline already wires these (see
// newP25Phase1Pipeline in ccdecoder/pipelines.go); this pins the
// wideband path to the same behaviour.
func TestBuildChannelP25Phase1WiresPhase2FEC(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	log := slog.New(slog.NewTextHandler(discardWriter{}, nil))

	const ccHz = 420_012_500
	sys := p25Phase1System("mmr", ccHz)
	// Reporter's config: trellis explicitly on. Also pin a non-default
	// value (RS on) so the test proves the config is actually forwarded
	// rather than just landing on a coincidental default.
	sys.P25Phase2TrellisMode = "on"
	sys.P25Phase2RSMode = "on"

	ch := ChannelConfig{FrequencyHz: ccHz, SystemName: sys.Name}
	ec, err := buildChannel(sys, ch, narrowbandRateHz, bus, log, nil)
	if err != nil {
		t.Fatalf("buildChannel: %v", err)
	}

	cc, ok := ec.processor.(*p25phase1.ControlChannel)
	if !ok {
		t.Fatalf("processor type = %T, want *p25phase1.ControlChannel", ec.processor)
	}

	trellis, rs, _, scrambler := cc.P25Phase2FEC()
	if trellis != uint8(p25phase2.TrellisOn) {
		t.Errorf("Phase 2 trellis = %d, want %d (TrellisOn) — grant FEC not wired (issue #882)",
			trellis, p25phase2.TrellisOn)
	}
	if rs != uint8(p25phase2.RSOn) {
		t.Errorf("Phase 2 RS = %d, want %d (RSOn) — per-system FEC config not forwarded",
			rs, p25phase2.RSOn)
	}
	// Scrambler defaults to on (empty config), and must survive the wiring.
	if scrambler != uint8(p25phase2.ScramblerOn) {
		t.Errorf("Phase 2 scrambler = %d, want %d (ScramblerOn default)",
			scrambler, p25phase2.ScramblerOn)
	}
}
