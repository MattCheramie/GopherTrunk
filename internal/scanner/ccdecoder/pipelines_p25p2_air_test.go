package ccdecoder

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	p25phase2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestP25Phase2PipelineBootstrapsSeedAndGrants drives the production Phase 2
// control-channel pipeline with modulated IQ in the real TDMA air format and
// no configured site identity. Every superframe carries a Network Status
// Broadcast on a clear LCCH (decodable without any seed) followed by a group
// voice grant on a SACCH scrambled with the site's PN44 seed. The pipeline
// must lock, install the seed from the NSB, and then decode the scrambled
// grant. Before the air-format rebuild the pipeline framed nothing on this
// stream; with the old NSB parser the seed came out wrong (it read the
// Color Code from the channel bytes) and the grant never decoded.
func TestP25Phase2PipelineBootstrapsSeedAndGrants(t *testing.T) {
	const (
		wacn, sysID, nac = 0xABCDE, 0x123, 0x293
		group            = 0x4EEA
		src              = 315203
	)
	seed := framing.PN44SeedFromIdentity(wacn, sysID, nac)
	masks := p25phase2.NewAirMasks(seed)

	nsb := []byte{0xFB, 0x07, 0xAB, 0xCD, 0xE1, 0x23, 0x10, 0x05, 0x10, 0x06, 0x70, 0x02, 0x93}
	grant := []byte{0x40, 0x00, 0x10, 0x05, byte(group >> 8), byte(group & 0xFF), byte(src >> 16), byte(src >> 8 & 0xFF), byte(src & 0xFF)}
	lcch, err := p25phase2.AirMACPDUBytes(p25phase2.AirMACSignal, 0, 21, nsb)
	if err != nil {
		t.Fatal(err)
	}
	sacch, err := p25phase2.AirMACPDUBytes(p25phase2.AirMACSignal, 0, 21, grant)
	if err != nil {
		t.Fatal(err)
	}
	dibits := make([]uint8, 600)
	for i := range dibits {
		dibits[i] = uint8(i % 4)
	}
	for s := 0; s < 6; s++ {
		var sf [p25phase2.AirSlotsPerSuperframe]p25phase2.AirSlotContent
		for slot := range sf {
			sf[slot] = p25phase2.AirSlotContent{DUID: p25phase2.DUIDSACCHScrambled}
		}
		sf[0] = p25phase2.AirSlotContent{DUID: p25phase2.DUIDLCCHClear, MAC: lcch}
		sf[1] = p25phase2.AirSlotContent{DUID: p25phase2.DUIDSACCHScrambled, MAC: sacch}
		d, err := p25phase2.EncodeAirSuperframe(sf, masks, uint8(s&3))
		if err != nil {
			t.Fatal(err)
		}
		dibits = append(dibits, d...)
	}
	rate := DDCTargetForProtocol(trunking.ProtocolP25Phase2)
	iq := demod.ModulateHDQPSKSpec(dibits, int(rate/6000), 8, 0.20)

	bus := events.NewBus(256)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	pipe, ok, err := NewPipeline(trunking.ProtocolP25Phase2, PipelineOptions{
		Bus: bus, Log: discardLogger(), SystemName: "P2", FrequencyHz: 851_000_000,
		SampleRateHz: rate,
	})
	if !ok || err != nil {
		t.Fatalf("NewPipeline: ok=%v err=%v", ok, err)
	}
	defer pipe.Close()
	for off := 0; off < len(iq); off += 4096 {
		pipe.Process(iq[off:min(off+4096, len(iq))])
	}

	if got := pipe.(*p25Phase2Pipeline).cc.ScramblerSeed(); got != seed {
		t.Errorf("scrambler seed = %#x, want %#x (from the NSB's WACN / System ID / Color Code)", got, seed)
	}
	var locked, granted bool
	deadline := time.After(2 * time.Second)
	for !(locked && granted) {
		select {
		case ev := <-sub.C:
			switch ev.Kind {
			case events.KindCCLocked:
				locked = true
			case events.KindGrant:
				g, ok := ev.Payload.(trunking.Grant)
				if !ok {
					t.Fatalf("grant payload %T", ev.Payload)
				}
				if g.GroupID != group || g.SourceID != src {
					t.Errorf("grant group/src = %#x/%d, want %#x/%d", g.GroupID, g.SourceID, group, src)
				}
				granted = true
			}
		case <-deadline:
			t.Fatalf("locked=%v granted=%v: the scrambled grant never decoded", locked, granted)
		}
	}
}
