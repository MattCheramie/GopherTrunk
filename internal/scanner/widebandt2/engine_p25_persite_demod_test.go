package widebandt2

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	p25phase1 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
)

// TestBuildChannelP25Phase1PerSiteDemodOverride is the regression test for
// issue #935: within a single P25 system, individual sites can transmit
// different modulation — a site that genuinely transmits linear/CQPSK
// (Linear Simulcast Modulation, LSM) alongside sites on C4FM. A wideband
// dongle decodes each site's control channel in parallel, so the per-tap
// demod path must be selectable per control-channel frequency rather than
// only once at the system level. (The override's trigger is real CQPSK
// deployment, not a site being simulcast — most simulcast systems are C4FM;
// see #935.)
//
// The override also has to reach the voice grants a tap publishes: without
// it a CQPSK site would lock its CC yet decode no LDU on a granted call
// (issue #356 follow-up), because the composer's voice chain reads the
// grant's demod mode. The wideband path previously never stamped a demod
// mode onto grants at all, so this pins both halves.
func TestBuildChannelP25Phase1PerSiteDemodOverride(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	log := slog.New(slog.NewTextHandler(discardWriter{}, nil))

	const (
		cqpskHz = 460_100_000 // a site confirmed to transmit linear/CQPSK
		c4fmHz  = 460_350_000 // a C4FM site in the same system
	)

	// System default is c4fm (a real, non-empty value) so the inheriting
	// tap proves inheritance rather than landing on a coincidental empty
	// default, and the override tap proves the per-channel value wins.
	sys := p25Phase1System("mixed-p25", cqpskHz)
	sys.ControlChannels = []uint32{cqpskHz, c4fmHz}
	sys.P25Phase1DemodMode = "c4fm"

	// Override this one tap onto the linear/CQPSK path; leave the other
	// inheriting the C4FM system default.
	cqpskCh := ChannelConfig{FrequencyHz: cqpskHz, SystemName: sys.Name, P25Phase1DemodMode: "cqpsk"}
	c4fmCh := ChannelConfig{FrequencyHz: c4fmHz, SystemName: sys.Name}

	demodOf := func(t *testing.T, ch ChannelConfig) string {
		t.Helper()
		ec, err := buildChannel(sys, ch, narrowbandRateHz, bus, log, nil)
		if err != nil {
			t.Fatalf("buildChannel(%d): %v", ch.FrequencyHz, err)
		}
		cc, ok := ec.processor.(*p25phase1.ControlChannel)
		if !ok {
			t.Fatalf("processor type = %T, want *p25phase1.ControlChannel", ec.processor)
		}
		return cc.P25Phase1DemodMode()
	}

	if got := demodOf(t, cqpskCh); got != "cqpsk" {
		t.Errorf("CQPSK tap demod mode = %q, want %q — per-channel override not applied to grants (issue #935)", got, "cqpsk")
	}
	if got := demodOf(t, c4fmCh); got != "c4fm" {
		t.Errorf("inheriting tap demod mode = %q, want %q — system default not forwarded to grants", got, "c4fm")
	}
}
