package sigfollow

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// The three real Motorola FACCH-S alias PDUs from issue #376 (Victorian MMR,
// TG 20208, RADIO:ISSI 781824.356.200062): HEADER 0x91 + two DATA 0x95
// blocks, post-FEC info bytes with the FACCH CRCs dropped — the same
// literal dump the phase2 assembler tests pin. SDRTrunk decodes this radio's
// alias as "CRIO 0062".
var (
	realAliasHeaderMSG = []byte{0x91, 0x90, 0x11, 0x4E, 0xF0, 0x02, 0x01, 0x00, 0x06,
		0xBE, 0xE0, 0x01, 0x64, 0x03, 0x0D, 0x7E, 0x24}
	realAliasData1MSG = []byte{0x95, 0x90, 0x11, 0x01, 0x04,
		0x4F, 0x6F, 0xF2, 0xFA, 0x9A, 0xC3, 0xEC, 0x34, 0x43, 0x2F, 0xA6, 0x3C}
	realAliasData2MSG = []byte{0x95, 0x90, 0x11, 0x02, 0x0C,
		0x81, 0xC3, 0xC5, 0xD9, 0x6A, 0x96, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
)

// realAliasBursts carries the three real alias messages the way a live
// Motorola site sends them: one per FACCH-S burst in MAC_HANGTIME PDUs,
// scrambled, framed back by the air framer. Each message is 17 bytes (its
// own length byte, 0x11), so it fills the 18-byte FACCH PDU after the
// header — the live 144-bit PDU shape the fixed-length fragment parsers
// were fixed for (the 15-byte SDRTrunk dumps could not show it).
func realAliasBursts(t *testing.T) ([]p25p2.AirBurst, *p25p2.AirMasks) {
	t.Helper()
	masks := p25p2.NewAirMasks(testSeed)
	sf1, sf2 := voiceSlots(), voiceSlots()
	facch := func(msg []byte) p25p2.AirSlotContent {
		return p25p2.AirSlotContent{DUID: p25p2.DUIDFACCHScrambled,
			MAC: macPDU(t, p25p2.AirMACHangtime, p25p2.DUIDFACCHScrambled, msg)}
	}
	sf1[0], sf1[6] = facch(realAliasHeaderMSG), facch(realAliasData1MSG)
	sf2[0] = facch(realAliasData2MSG)
	return airBursts(t, masks, sf1, sf2), masks
}

// TestDispatcherPublishesRealMotorolaAliasAsReliable is the end-to-end pin
// for #773 on the shared live path: the real #376 FACCH-S PDUs, carried on
// FACCH-S bursts through the air framer and the MACDispatcher both
// the voice composer and the signalling follower run, surface on the bus as
// the radio's actual display name — "CRIO 0062" for RID 200062 — flagged
// reliable (not "⚠ decode unreliable"). It fails if the cipher is ever
// re-gated (CipherVerified=false ⇒ Unreliable=true) or the framing drifts.
func TestDispatcherPublishesRealMotorolaAliasAsReliable(t *testing.T) {
	bus := events.NewBus(64)
	sub := bus.Subscribe()
	defer sub.Close()

	d := NewMACDispatcher(MACDispatcherOptions{
		Bus: bus, Log: quietLog(), LogPrefix: "sigfollow",
		System: "MMR", Serial: "tap-0",
	})
	bursts, masks := realAliasBursts(t)
	if sig, dec := d.DispatchAir(bursts, masks); sig != 3 || dec != 3 {
		t.Fatalf("DispatchAir = %d signalling / %d decoded, want 3/3", sig, dec)
	}

	deadline := time.After(2 * time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindTalkerAlias {
				continue
			}
			ta, ok := ev.Payload.(trunking.TalkerAlias)
			if !ok {
				t.Fatalf("KindTalkerAlias payload type = %T", ev.Payload)
			}
			if ta.SourceID != 200062 {
				t.Errorf("SourceID = %d, want 200062", ta.SourceID)
			}
			if ta.Alias != "CRIO 0062" {
				t.Errorf("Alias = %q, want %q", ta.Alias, "CRIO 0062")
			}
			if ta.Unreliable {
				t.Error("Unreliable = true; the verified cipher's clean decode must publish as reliable")
			}
			if ta.System != "MMR" || ta.Protocol != "p25-phase2" {
				t.Errorf("System/Protocol = %q/%q, want MMR/p25-phase2", ta.System, ta.Protocol)
			}
			return
		case <-deadline:
			t.Fatal("no KindTalkerAlias published for the real #376 alias PDUs")
		}
	}
}
