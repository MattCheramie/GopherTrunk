package ccdecoder

import (
	"encoding/binary"
	"log/slog"
	"os"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	tetrarx "github.com/MattCheramie/GopherTrunk/internal/radio/tetra/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// marginalCCFixture is a ~2 s / 144 kHz cs16 slice of a real on-air TETRA
// control channel captured by gophertrunk's on_cc_sync_loss auto-recorder
// during a re-acquisition (system 250_013, CC 467.9125 MHz, Airspy). In-channel
// SNR is ~10 dB (a healthy period on the same site measures ~18 dB) and the
// π/4-DQPSK constellation is ISI-smeared. Replayed through the current CC path
// it LOCKS but decodes only ~22 % of its BSCH synchronisation bursts — the
// marginal-yield regime that produced the field's ~210 CC transitions/hour.
const marginalCCFixture = "testdata/tetra_cc_sync_loss_2s_144k.cs16"

// loadCS16 reads a headerless interleaved little-endian int16 IQ file (I then Q,
// ÷32768) — the cs16 format the auto-recorder writes and `gophertrunk replay
// -format cs16` consumes.
func loadCS16(t *testing.T, path string) []complex64 {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	iq := make([]complex64, len(raw)/4)
	for i := range iq {
		re := int16(binary.LittleEndian.Uint16(raw[i*4:]))
		im := int16(binary.LittleEndian.Uint16(raw[i*4+2:]))
		iq[i] = complex(float32(re)/32768, float32(im)/32768)
	}
	return iq
}

// decodeCCBSCHYield hand-wires a TETRA control-channel receiver + decoder at the
// 144 kHz channel rate with the equalizer toggled, feeds the IQ in chunks, and
// returns the BSCH decode counts. The wiring mirrors newTETRAPipeline exactly
// (same clock/gain/AFC/channel-filter, soft-decision sink, colour-0 auto-acquire)
// EXCEPT that EnableEqualizer is a parameter — so a single fixture isolates the
// equalizer as the only variable.
func decodeCCBSCHYield(t *testing.T, iq []complex64, enableEqualizer bool) (ok, fail int64) {
	t.Helper()
	bus := events.NewBus(8)
	t.Cleanup(bus.Close)
	cc := tetra.New(tetra.Options{SystemName: "eqtest", Bus: bus})
	cc.SetChannelCoding(tetra.ChannelCodingOn)
	ch, _ := tetra.ParseChannelType("") // default SCH/HD, as newTETRAPipeline
	cc.SetExpectedChannel(ch)
	cc.SetColourCode(0) // auto-acquire from the BSCH sync burst
	clk, _ := tetrarx.ParseClockMode("")

	rx := tetrarx.New(tetrarx.Options{
		SampleRateHz:        144_000,
		DibitSink:           func(dibits []uint8, baseIdx int) { cc.Process(dibits, baseIdx) },
		SoftSink:            func(diffs []complex64, baseIdx int) { cc.StashSoft(diffs, baseIdx) },
		ClockMode:           clk,
		GardnerGain:         0.005,
		EnableAFC:           true,
		EnableChannelFilter: true,
		EnableEqualizer:     enableEqualizer,
	})

	const chunk = 65536
	for pos := 0; pos < len(iq); pos += chunk {
		end := pos + chunk
		if end > len(iq) {
			end = len(iq)
		}
		rx.Process(iq[pos:end])
	}
	return cc.BSCHCounts()
}

// TestTETRACCEqualizerABIsolatesCause is the causal proof: on the SAME marginal
// fixture, enabling the SnapshotCMA equalizer materially lifts CRC-clean BSCH
// yield. Because the two chains differ only in EnableEqualizer, the improvement
// cannot be chunking, noise, or timing — it is the equalizer inverting the ISI
// that smears the constellation on this ~10 dB capture.
func TestTETRACCEqualizerABIsolatesCause(t *testing.T) {
	iq := loadCS16(t, marginalCCFixture)

	okOff, failOff := decodeCCBSCHYield(t, iq, false)
	okOn, failOn := decodeCCBSCHYield(t, iq, true)

	yOff := float64(okOff) / float64(okOff+failOff+1)
	yOn := float64(okOn) / float64(okOn+failOn+1)
	t.Logf("BSCH: equalizer OFF = %d ok / %d fail (%.0f%%), ON = %d ok / %d fail (%.0f%%)",
		okOff, failOff, yOff*100, okOn, failOn, yOn*100)

	// On this real ~10 dB capture the measured effect is decisive: ~12 % BSCH
	// yield without the equalizer, ~100 % with it (the SnapshotCMA inverts the
	// ISI that smears the constellation). Guard both ends so the equalizer's
	// causal role — not chunking, noise, or timing — is what the test pins.
	if okOn < 25 {
		t.Errorf("equalizer-on decoded only %d clean BSCH bursts from the fixture (want >=25)", okOn)
	}
	if yOn < 0.90 {
		t.Errorf("equalizer-on BSCH yield %.0f%% did not reach the healthy regime (want >=90%%)", yOn*100)
	}
	if yOff > 0.40 {
		t.Errorf("equalizer-off yield %.0f%% unexpectedly high — fixture is not the intended marginal capture", yOff*100)
	}
	if okOn < 4*okOff {
		t.Errorf("equalizer did not multiply CRC-clean yield: ON %d not >= 4× OFF %d", okOn, okOff)
	}
}

// TestTETRACCEqualizerLiftsMarginalBSCHYield is the ship-gate for the wiring
// actually used in production: it drives the LIVE newTETRAPipeline (whatever it
// is configured to do) on the marginal fixture and requires it to reach the
// healthy BSCH regime. RED before newTETRAPipeline enables the equalizer
// (~22 %), GREEN after. If this does not pass, the equalizer must not ship.
func TestTETRACCEqualizerLiftsMarginalBSCHYield(t *testing.T) {
	iq := loadCS16(t, marginalCCFixture)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	bus := events.NewBus(8)
	t.Cleanup(bus.Close)
	pp, err := newTETRAPipeline(PipelineOptions{
		Bus: bus, Log: logger, SystemName: "eqtest", FrequencyHz: 467_912_500,
		SampleRateHz: 144_000,
		System: trunking.System{
			Name: "eqtest", Protocol: trunking.ProtocolTETRA,
			ControlChannels: []uint32{467_912_500},
		},
	})
	if err != nil {
		t.Fatalf("newTETRAPipeline: %v", err)
	}
	p := pp.(*tetraPipeline)

	const chunk = 65536
	for pos := 0; pos < len(iq); pos += chunk {
		end := pos + chunk
		if end > len(iq) {
			end = len(iq)
		}
		p.Process(iq[pos:end])
	}
	ok, fail := p.BSCHCounts()
	yield := float64(ok) / float64(ok+fail+1)
	t.Logf("live newTETRAPipeline BSCH on marginal fixture = %d ok / %d fail (%.0f%%)", ok, fail, yield*100)
	// Without the equalizer the live pipeline recovers only a handful of BSCH
	// bursts from this fixture (~1 ok / ~12 %); with SnapshotCMA wired in it
	// reaches the healthy regime (~36 ok / ~100 %). Requiring both a healthy
	// yield and a real count of clean bursts makes this a true ship-gate:
	// RED until newTETRAPipeline sets EnableEqualizer, GREEN after.
	if ok < 25 || yield < 0.80 {
		t.Errorf("live CC pipeline decoded %d clean BSCH (%.0f%%) — below the healthy regime; is the equalizer wired into newTETRAPipeline?", ok, yield*100)
	}
}
