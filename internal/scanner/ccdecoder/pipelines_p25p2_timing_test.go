package ccdecoder

import (
	"encoding/binary"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// mmrPhase2Fixture is the real P25 Phase 2 carrier the Phase 2 receiver's own
// tests use: 1.09 s of the MMR idle voice channel at 420.650 MHz, 48 kHz
// int16 I/Q (see internal/radio/p25/phase2/receiver/timing_acq_test.go). The
// site is WACN BEE00 / system 164 / colour code 164, which is the identity the
// descramble seed is built from.
const mmrPhase2Fixture = "../../radio/p25/phase2/receiver/testdata/mmr-s9-420650k-p2-48k.cs16"

func loadMMRPhase2(t *testing.T) []complex64 {
	t.Helper()
	raw, err := os.ReadFile(mmrPhase2Fixture)
	if err != nil {
		t.Fatal(err)
	}
	iq := make([]complex64, len(raw)/4)
	for i := range iq {
		re := int16(binary.LittleEndian.Uint16(raw[4*i:]))
		im := int16(binary.LittleEndian.Uint16(raw[4*i+2:]))
		iq[i] = complex(float32(re)/32768, float32(im)/32768)
	}
	return iq
}

// TestP25Phase2PipelineRealAirEveryStartPhase is the issue #1364 regression
// for the control-channel pipeline. A hunt dwell, a retune and a decode-drought
// resync all start the receiver at an arbitrary point in the carrier's symbol
// timing. On this real carrier the timing loop alone finds the frame grid
// 32 ms in from six of the eight start phases, 0.27 s in from one and 0.99 s
// in from another (the receiver's own test measures that), and a start that
// slow leaves most of a dwell undecoded. The carrier sends one signalling
// burst per logical channel per 360 ms superframe; the pipeline must decode
// MAC messages from the first superframes at every start phase.
func TestP25Phase2PipelineRealAirEveryStartPhase(t *testing.T) {
	iq := loadMMRPhase2(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	const want = 8 // MAC messages in 1.09 s; a prompt start decodes 10 to 14
	for k := 0; k < 8; k++ {
		bus := events.NewBus(64)
		p, err := newP25Phase2Pipeline(PipelineOptions{
			Bus: bus, Log: log, SystemName: "mmr",
			FrequencyHz: 420_650_000, SampleRateHz: 48_000,
			System: trunking.System{
				Name: "mmr", Protocol: trunking.ProtocolP25Phase2,
				WACN: 0xBEE00, SystemID: 0x164,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		pp := p.(*p25Phase2Pipeline)
		// The colour code is 12 bits and the system config's site field
		// (which the factory uses for it) is 8, so the seed is set here. A
		// real TDMA control channel installs it from its own clear Network
		// Status Broadcast.
		pp.cc.SetScramblerSeed(framing.PN44SeedFromIdentity(0xBEE00, 0x164, 0x164))
		in := iq[k:]
		for i := 0; i < len(in); i += 655 {
			pp.Process(in[i:min(i+655, len(in))])
		}
		got := int(pp.cc.DecodedFrames())
		t.Logf("start phase %d: %d MAC messages decoded", k, got)
		if got < want {
			t.Errorf("start phase %d: %d MAC messages decoded in 1.09 s, want at least %d", k, got, want)
		}
		bus.Close()
	}
}
