package composer

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// Issue #1364: what the Phase 2 voice chain reads from the voice bursts and
// the MAC PDUs around them, beyond the talker's ID.

// p25p2ESSB / p25p2ESSA are one Encryption Synchronization Signal word from
// OP25's own Reed-Solomon code (see the phase2 package's essReference):
// ALGID 0x84, key 0xBEEF.
const (
	p25p2ESSB = "210b3b2f042311161e092a3c372f000f"
	p25p2ESSA = "1e362910062f1021123912233a251d1f27142a1b0502022005233b04"
)

// p25p2TalkStream builds n superframes of a call on logical channel 0 as a
// real radio sends it: runs of four VOICE-4 bursts and one VOICE-2, the ESS
// laid in at OP25's positions (xored_burst[84..], body dibit = index - 10,
// the DUID dibit at 132 skipped). mac, when it returns a slot content for a
// (superframe, timeslot), replaces that burst. Logical channel 1 is idle.
func p25p2TalkStream(t *testing.T, n int, mac func(sf, slot int) *p25p2.AirSlotContent) []uint8 {
	t.Helper()
	masks := p25p2.NewAirMasks(p25p2TestSeed)
	essB, _ := hex.DecodeString(p25p2ESSB)
	essA, _ := hex.DecodeString(p25p2ESSA)
	place := func(pkt []uint8, hexbits []byte) {
		body := pkt[p25p2.AirISCHDibits:]
		at := 84
		for _, h := range hexbits {
			for k := 0; k < 3; k++ {
				if at == 132 {
					at++
				}
				body[at-10] ^= h >> uint(4-2*k) & 3
				at++
			}
		}
	}
	dibits := make([]uint8, 600)
	for i := range dibits {
		dibits[i] = uint8(i % 4)
	}
	k := 0
	for s := 0; s <= n; s++ { // one trailing superframe flushes the last bursts
		for slot := 0; slot < p25p2.AirSlotsPerSuperframe; slot++ {
			var pkt []uint8
			var err error
			idx := slot / 2
			switch {
			case s < n && mac != nil && mac(s, slot) != nil:
				m := mac(s, slot)
				pkt, err = p25p2.EncodeAirMACPacket(slot, m.DUID, m.MAC, masks, uint8(s&3))
			case p25p2.AirSlotLogicalChannel(slot) != 0 || slot == 11:
				pkt, err = p25p2.EncodeAirPacket(slot, p25p2.DUIDSACCHScrambled, nil, masks, uint8(s&3))
			default:
				duid, frames := p25p2.DUIDVoice4, 4
				if idx == 4 {
					duid, frames = p25p2.DUIDVoice2, 2
				}
				var voice [][]byte
				for j := 0; j < frames; j++ {
					voice = append(voice, p25p2VoicePayload(k))
					k++
				}
				pkt, err = p25p2.EncodeAirPacket(slot, duid, voice, masks, uint8(s&3))
				if err == nil && idx < 4 {
					place(pkt, essB[4*idx:4*idx+4])
				} else if err == nil {
					place(pkt, essA)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			dibits = append(dibits, pkt...)
		}
	}
	return dibits
}

// runP25P2Chain modulates dibits and runs them through the Phase 2 voice
// chain, returning every event it published.
func runP25P2Chain(t *testing.T, dibits []uint8, split bool) []events.Event {
	t.Helper()
	const sampleRate = 48_000.0
	iq := demod.ModulateHDQPSKSpec(dibits, 8, 8, 0.20)
	bus := events.NewBus(1024)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	c, err := New(Options{
		Bus:                  bus,
		Devices:              &fakeDevices{src: map[string]IQSource{}},
		Sink:                 &recordingSink{},
		Engine:               &fakeEngine{},
		IQSampleRate:         uint32(sampleRate),
		PCMSampleRate:        8000,
		TouchInterval:        30 * time.Millisecond,
		SplitPerTransmission: split,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Chunks the size a wideband tap delivers: several bursts apiece, so
	// the order bursts are handled in within a chunk matters.
	iqCh := make(chan []complex64, len(iq)/4096+2)
	for i := 0; i < len(iq); i += 4096 {
		iqCh <- iq[i:min(i+4096, len(iq))]
	}
	close(iqCh)
	done := make(chan struct{})
	c.runP25Phase2VoiceChain(context.Background(), "VOICE-1", "MMR",
		p25p2.MACDecodeConfig{Seed: p25p2TestSeed}, 1, iqCh, sampleRate, done)
	<-done
	var out []events.Event
	for {
		select {
		case ev := <-sub.C:
			out = append(out, ev)
			continue
		default:
		}
		return out
	}
}

// TestComposerP25Phase2ESSNamesTheEncryption: a call the chain joins after
// its key-up has no MAC_PTT to read. The algorithm and key still ride every
// 360 ms of its voice, in the ESS. Before this the chain published no
// call.encryption for such a call at all; now it publishes one, and only
// one for as long as the algorithm and key stay the same.
func TestComposerP25Phase2ESSNamesTheEncryption(t *testing.T) {
	var got []trunking.CallEncryption
	for _, ev := range runP25P2Chain(t, p25p2TalkStream(t, 6, nil), false) {
		if ev.Kind == events.KindCallEncryption {
			got = append(got, ev.Payload.(trunking.CallEncryption))
		}
	}
	if len(got) != 1 {
		t.Fatalf("%d call.encryption events from six superframes of one call, want 1", len(got))
	}
	if got[0].AlgorithmID != 0x84 || got[0].KeyID != 0xBEEF || got[0].DeviceSerial != "VOICE-1" {
		t.Errorf("call.encryption = %+v, want AES-256 (0x84), key 0xBEEF, serial VOICE-1", got[0])
	}
}

// TestComposerP25Phase2EndPTTRollsTheRecording: in per-transmission mode a
// MAC_END_PTT (the radio releasing its PTT) ends the over, as the Phase 1
// terminator does, so the next radio's voice goes to a file of its own. The
// chain used to ignore MAC_END_PTT, and a Phase 2 call was one file however
// many radios keyed. Two overs here, each closed by two MAC_END_PTT copies:
// two segment boundaries, not four, and none without the split mode.
func TestComposerP25Phase2EndPTTRollsTheRecording(t *testing.T) {
	end := make([]byte, 18)
	end[0] = byte(p25p2.AirMACEndPTT) << 5
	end[1], end[2] = 0x01, 0x61
	end[13], end[14], end[15] = 0x04, 0xCF, 0x43
	mac := func(sf, slot int) *p25p2.AirSlotContent {
		// The last two voice timeslots of superframes 1 and 3.
		if (sf == 1 || sf == 3) && (slot == 6 || slot == 8) {
			return &p25p2.AirSlotContent{DUID: p25p2.DUIDFACCHScrambled, MAC: end}
		}
		return nil
	}
	count := func(split bool) int {
		n := 0
		for _, ev := range runP25P2Chain(t, p25p2TalkStream(t, 5, mac), split) {
			if ev.Kind == events.KindCallSegment {
				n++
			}
		}
		return n
	}
	if got := count(true); got != 2 {
		t.Errorf("%d call segments for two overs in per-transmission mode, want 2", got)
	}
	if got := count(false); got != 0 {
		t.Errorf("%d call segments with per-transmission mode off, want 0", got)
	}
}

// TestCallEncryptionFilter: the same algorithm and key again is not news; a
// different key, or the algorithm changing, is.
func TestCallEncryptionFilter(t *testing.T) {
	var f callEncryptionFilter
	aes := p25p2.EncryptionSync{AlgorithmID: 0x84, KeyID: 1, MessageIndicator: [9]byte{1}}
	if !f.admit(aes) {
		t.Error("the first sync must pass")
	}
	aes.MessageIndicator = [9]byte{2} // the MI changes every superframe
	if f.admit(aes) {
		t.Error("the same algorithm and key passed again")
	}
	if !f.admit(p25p2.EncryptionSync{AlgorithmID: 0x84, KeyID: 2}) {
		t.Error("a new key must pass")
	}
	if !f.admit(p25p2.EncryptionSync{AlgorithmID: 0x80}) {
		t.Error("the call going clear must pass")
	}
}
