package widebandt2

import (
	"encoding/binary"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	p25phase2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestP25Phase2ChannelRealAirEveryStartPhase is the wideband engine's copy of
// ccdecoder's TestP25Phase2PipelineRealAirEveryStartPhase (issue #1364): a
// Phase 2 channel built by buildChannel must decode the real MMR carrier's
// signalling from every start phase. Without the feed-forward timing seed the
// channel decodes nothing in 1.09 s from one of the eight.
func TestP25Phase2ChannelRealAirEveryStartPhase(t *testing.T) {
	raw, err := os.ReadFile("../../radio/p25/phase2/receiver/testdata/mmr-s9-420650k-p2-48k.cs16")
	if err != nil {
		t.Fatal(err)
	}
	iq := make([]complex64, len(raw)/4)
	for i := range iq {
		re := int16(binary.LittleEndian.Uint16(raw[4*i:]))
		im := int16(binary.LittleEndian.Uint16(raw[4*i+2:]))
		iq[i] = complex(float32(re)/32768, float32(im)/32768)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	const freq = 420_650_000
	sys := trunking.System{
		Name: "mmr", Protocol: trunking.ProtocolP25Phase2,
		ControlChannels: []uint32{freq}, WACN: 0xBEE00, SystemID: 0x164,
	}
	const want = 8 // MAC messages in 1.09 s; a prompt start decodes 10 to 14
	for k := 0; k < 8; k++ {
		bus := events.NewBus(64)
		ec, err := buildChannel(sys, ChannelConfig{FrequencyHz: freq, SystemName: "mmr"}, 48_000, bus, log, time.Now)
		if err != nil {
			t.Fatal(err)
		}
		// The 12-bit colour code does not fit the config's 8-bit site field;
		// a real TDMA control channel installs the seed from its own NSB.
		ec.processor.(*p25phase2.ControlChannel).SetScramblerSeed(
			framing.PN44SeedFromIdentity(0xBEE00, 0x164, 0x164))
		in := iq[k:]
		for i := 0; i < len(in); i += 655 {
			ec.receiver.Process(in[i:min(i+655, len(in))])
		}
		got := int(ec.decoded())
		t.Logf("start phase %d: %d MAC messages decoded", k, got)
		if got < want {
			t.Errorf("start phase %d: %d MAC messages decoded in 1.09 s, want at least %d", k, got, want)
		}
		bus.Close()
	}
}

// A wideband Phase 2 control channel installs no descramble seed of its own
// (issue #1364): the one it used to build from the system's WACN / System ID /
// Site named a sequence no site sends (the configuration sets none of them,
// and Site is not the 12-bit Color Code), and grants made before the
// channel's Network Status Broadcast carried it to the voice chain.
func TestP25Phase2ChannelInstallsNoPlaceholderSeed(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	const freq = 420_650_000
	sys := trunking.System{
		Name: "mmr", Protocol: trunking.ProtocolP25Phase2,
		ControlChannels: []uint32{freq}, WACN: 0xBEE00, SystemID: 0x164, Site: 3,
	}
	bus := events.NewBus(8)
	defer bus.Close()
	ec, err := buildChannel(sys, ChannelConfig{FrequencyHz: freq, SystemName: "mmr"}, 48_000, bus, log, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if got := ec.processor.(*p25phase2.ControlChannel).ScramblerSeed(); got != 0 {
		t.Fatalf("seed %#x before any Network Status Broadcast, want none", got)
	}
}
