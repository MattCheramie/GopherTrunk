package phase2

import (
	"encoding/hex"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// essReference holds ESS words produced by OP25's own code, compiled outside
// the repository: ezpwd::RS<63,35> (the rs28 object p25p2_tdma.h declares)
// encoding an ESS-B packed as handle_4V2V_ess unpacks it (ALGID 8 bits, KID
// 16, MI 72, six bits to a hexbit). essB is the 16 data hexbits, essA the 28
// parity hexbits, one byte per hexbit.
var essReference = []struct {
	alg  uint8
	kid  uint16
	mi   string
	essB string
	essA string
}{
	{0x84, 0xBEEF, "123456789abcdef00f",
		"210b3b2f042311161e092a3c372f000f",
		"1e362910062f1021123912233a251d1f27142a1b0502022005233b04"},
	{0x80, 0x0000, "000000000000000000",
		"20000000000000000000000000000000",
		"0d0d083c041d07352a100d0c38051d0f0b0f1f3c15202c2b3b090c03"},
	{0xAA, 0x0001, "ffeeddccbbaa998877",
		"2a2000013f3e3b1d330b2e2a26182137",
		"1132043b2509132f343f39002513082e1e373b173f36201c1d14201b"},
}

// essTalkSpurt builds one run of five voice bursts (four VOICE-4, one
// VOICE-2) on logical channel 0, starting at voice timeslot index first, with
// the ESS hexbits laid at OP25's positions: its xored_burst[84..95] for the
// four ESS-B hexbits of a VOICE-4 burst; xored_burst[84..131] and [133..168]
// for ESS-A in the VOICE-2 burst (xored_burst[132] is the DUID's third dibit,
// "skip dibit containing DUID#3"). xored_burst is the packet from its 11th
// dibit, so body dibit = index - 10.
func essTalkSpurt(t *testing.T, masks *AirMasks, first int, essB, essA []byte, startDibit int) []AirBurst {
	t.Helper()
	silence := make([]byte, VoiceFrameBytes)
	place := func(pkt []uint8, hexbits []byte, xoredIdx int) {
		body := pkt[AirISCHDibits:]
		for _, h := range hexbits {
			for k := 0; k < 3; k++ {
				if xoredIdx == 132 {
					xoredIdx++
				}
				body[xoredIdx-10] ^= h >> uint(4-2*k) & 3
				xoredIdx++
			}
		}
	}
	var out []AirBurst
	for k := 0; k < 5; k++ {
		idx := (first + k) % 5
		slot := 2 * idx // logical channel 0's timeslots 0, 2, 4, 6, 8
		duid, frames := DUIDVoice4, [][]byte{silence, silence, silence, silence}
		if k == 4 {
			duid, frames = DUIDVoice2, [][]byte{silence, silence}
		}
		pkt, err := EncodeAirPacket(slot, duid, frames, masks, 0)
		if err != nil {
			t.Fatal(err)
		}
		if k < 4 {
			place(pkt, essB[4*k:4*k+4], 84)
		} else {
			place(pkt, essA, 84)
		}
		// Packets on one logical channel are two timeslots apart, and the
		// SACCH pair sits between index 4 and index 0.
		at := startDibit + 2*AirPacketDibits*k
		if first+k >= 5 {
			at += 2 * AirPacketDibits
		}
		out = append(out, AirBurst{StartDibit: at, Slot: slot, LogicalChannel: 0, DUID: duid, Packet: pkt})
	}
	return out
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestESSTrackerDecodesReferenceWords: every reference word, at each of the
// five timeslot indexes a talk spurt can start on, decodes to its ALGID, KID
// and MI with nothing corrected.
func TestESSTrackerDecodesReferenceWords(t *testing.T) {
	masks := NewAirMasks(framing.PN44SeedFromIdentity(0xBEE00, 0x164, 0x164))
	for _, ref := range essReference {
		essB, essA := mustHex(t, ref.essB), mustHex(t, ref.essA)
		for first := 0; first < 5; first++ {
			var tr ESSTracker
			var got EncryptionSync
			var corrected int
			done := 0
			for _, b := range essTalkSpurt(t, masks, first, essB, essA, 10_000) {
				if es, c, ok := tr.Add(b, masks); ok {
					got, corrected = es, c
					done++
				}
			}
			if done != 1 {
				t.Fatalf("alg %02X, first index %d: %d syncs from one talk spurt", ref.alg, first, done)
			}
			if got.AlgorithmID != ref.alg || got.KeyID != ref.kid ||
				hex.EncodeToString(got.MessageIndicator[:]) != ref.mi || corrected != 0 {
				t.Errorf("alg %02X, first index %d: got alg %02X key %04X mi %x corrected %d",
					ref.alg, first, got.AlgorithmID, got.KeyID, got.MessageIndicator, corrected)
			}
		}
	}
}

// TestESSTrackerCorrectsAndFillsIn: 14 wrong hexbits are corrected (OP25's
// limit), a VOICE-4 burst that never arrived is four erasures, and a burst
// left over from the run before is not taken for this run's.
func TestESSTrackerCorrectsAndFillsIn(t *testing.T) {
	masks := NewAirMasks(framing.PN44SeedFromIdentity(0xBEE00, 0x164, 0x164))
	ref := essReference[0]
	essB, essA := mustHex(t, ref.essB), mustHex(t, ref.essA)
	want := EncryptionSync{AlgorithmID: ref.alg, KeyID: ref.kid}
	copy(want.MessageIndicator[:], mustHex(t, ref.mi))

	run := func(bursts []AirBurst, tr *ESSTracker) (EncryptionSync, int, bool) {
		var es EncryptionSync
		var c int
		ok := false
		for _, b := range bursts {
			if e, n, good := tr.Add(b, masks); good {
				es, c, ok = e, n, true
			}
		}
		return es, c, ok
	}

	// 14 hexbit errors: 7 in ESS-B, 7 in ESS-A.
	badB, badA := append([]byte(nil), essB...), append([]byte(nil), essA...)
	for i := 0; i < 7; i++ {
		badB[2*i] ^= 0x15
		badA[3*i+1] ^= 0x2A
	}
	if es, c, ok := run(essTalkSpurt(t, masks, 2, badB, badA, 10_000), &ESSTracker{}); !ok || es != want || c != 14 {
		t.Errorf("14 errors: %+v corrected %d ok=%v", es, c, ok)
	}
	// 15 are past the code.
	badB[1] ^= 0x01
	if _, _, ok := run(essTalkSpurt(t, masks, 2, badB, badA, 10_000), &ESSTracker{}); ok {
		t.Error("15 hexbit errors decoded")
	}

	// The second VOICE-4 burst is lost.
	spurt := essTalkSpurt(t, masks, 3, essB, essA, 10_000)
	lost := append(append([]AirBurst(nil), spurt[:1]...), spurt[2:]...)
	if es, c, ok := run(lost, &ESSTracker{}); !ok || es != want || c != 4 {
		t.Errorf("one burst lost: %+v filled %d ok=%v", es, c, ok)
	}

	// The run before carried another sync, and its VOICE-2 burst was lost,
	// so its four parts were never used. Its second part must not stand in
	// for this run's lost one: a part a whole superframe old is left out.
	other := essReference[2]
	var tr ESSTracker
	run(essTalkSpurt(t, masks, 3, mustHex(t, other.essB), mustHex(t, other.essA), 10_000)[:4], &tr)
	next := essTalkSpurt(t, masks, 3, essB, essA, 10_000+AirSlotsPerSuperframe*AirPacketDibits)
	lost = append(append([]AirBurst(nil), next[:1]...), next[2:]...)
	if es, c, ok := run(lost, &tr); !ok || es != want || c != 4 {
		t.Errorf("stale part from the run before: %+v filled %d ok=%v", es, c, ok)
	}

	// Only the VOICE-2 burst: the parity alone names the word.
	if es, c, ok := run(spurt[4:], &ESSTracker{}); !ok || es != want || c != 16 {
		t.Errorf("VOICE-2 alone: %+v filled %d ok=%v", es, c, ok)
	}
}

// TestESSTrackerIgnoresWhatIsNotVoice: signalling bursts, a voice DUID in a
// SACCH timeslot (a misread burst) and a missing seed give nothing.
func TestESSTrackerIgnoresWhatIsNotVoice(t *testing.T) {
	masks := NewAirMasks(framing.PN44SeedFromIdentity(0xBEE00, 0x164, 0x164))
	ref := essReference[0]
	spurt := essTalkSpurt(t, masks, 0, mustHex(t, ref.essB), mustHex(t, ref.essA), 10_000)
	var tr ESSTracker
	for _, b := range spurt {
		if _, _, ok := tr.Add(b, nil); ok {
			t.Fatal("decoded without descramble masks")
		}
	}
	tr.Reset()
	for _, b := range spurt {
		b.Slot = 11
		if _, _, ok := tr.Add(b, masks); ok {
			t.Fatal("decoded a voice DUID in a SACCH timeslot")
		}
	}
	tr.Reset()
	for _, b := range spurt {
		b.DUID = DUIDSACCHScrambled
		if _, _, ok := tr.Add(b, masks); ok {
			t.Fatal("decoded a signalling burst")
		}
	}
}
