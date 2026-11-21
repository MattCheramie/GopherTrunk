package sigfollow

import (
	"io"
	"log/slog"
	"testing"

	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
)

func quietLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// gvcuMsg is a GROUP_VOICE_CHANNEL_USER (abbreviated) message: opcode 0x01
// and the 7-byte length OP25's mac_msg_len gives it.
func gvcuMsg(src uint32, group uint16) []byte {
	return []byte{0x01, 0x40, byte(group >> 8), byte(group), byte(src >> 16), byte(src >> 8), byte(src)}
}

// TestDispatcherMACPTTDrivesEncryptionAndSource builds a FACCH-S MAC_PTT —
// the PDU a radio keys up with, carrying the call's MI / ALGID / KID and its
// own source address — and asserts both hooks fire with its fields. On real
// air the encryption sync rides MAC_PTT, not a separate message (#813).
func TestDispatcherMACPTTDrivesEncryptionAndSource(t *testing.T) {
	masks := p25p2.NewAirMasks(testSeed)
	ptt := p25p2.MACPTT{
		Encryption: p25p2.EncryptionSync{AlgorithmID: 0x84, KeyID: 0x1234,
			MessageIndicator: [9]byte{1, 2, 3, 4, 5, 6, 7, 8, 9}},
		Source: 315203, Group: 0x4EEA,
	}
	sf := voiceSlots()
	sf[2] = p25p2.AirSlotContent{DUID: p25p2.DUIDFACCHScrambled, MAC: p25p2.AirMACPTTBytes(ptt)}

	var gotES p25p2.EncryptionSync
	var gotUser p25p2.GroupVoiceChannelUser
	var encCalls, srcCalls int
	d := NewMACDispatcher(MACDispatcherOptions{
		Log: quietLog(), System: "TestSys", Serial: "tap-0",
		OnCallEncryption: func(e p25p2.EncryptionSync) { gotES = e; encCalls++ },
		OnCallSource:     func(u p25p2.GroupVoiceChannelUser) { gotUser = u; srcCalls++ },
	})
	if sig, dec := d.DispatchAir(airBursts(t, masks, sf), masks); sig != 1 || dec != 1 {
		t.Fatalf("DispatchAir = %d signalling / %d decoded, want 1/1", sig, dec)
	}
	if encCalls != 1 || gotES != ptt.Encryption {
		t.Errorf("OnCallEncryption = %d call(s), %+v; want 1, %+v", encCalls, gotES, ptt.Encryption)
	}
	if srcCalls != 1 || gotUser.SourceID != ptt.Source || gotUser.GroupAddress != ptt.Group {
		t.Errorf("OnCallSource = %d call(s), %+v; want 1, src %d group %#x", srcCalls, gotUser, ptt.Source, ptt.Group)
	}
	if gotUser.ServiceOptions&0x40 == 0 {
		t.Error("an AES MAC_PTT must mark the source update encrypted")
	}
}

// TestDispatcherSourceCallback carries an in-call GROUP_VOICE_CHANNEL_USER
// in a SACCH-S MAC_ACTIVE PDU behind a Null-padded second message slot and
// asserts it reaches OnCallSource: the message walk splits the PDU by
// OP25's message lengths, and the RS(63,35) + CRC-12 decode is what makes
// the RID trustworthy for the completed-call webhook (#915).
func TestDispatcherSourceCallback(t *testing.T) {
	const wantSrc = 315203
	masks := p25p2.NewAirMasks(testSeed)
	sf := voiceSlots()
	sf[4] = p25p2.AirSlotContent{DUID: p25p2.DUIDSACCHScrambled,
		MAC: macPDU(t, p25p2.AirMACActive, p25p2.DUIDSACCHScrambled, gvcuMsg(wantSrc, 0x4EEA))}

	var got []uint32
	d := NewMACDispatcher(MACDispatcherOptions{
		Log: quietLog(), System: "TestSys", Serial: "tap-0",
		OnCallSource: func(u p25p2.GroupVoiceChannelUser) { got = append(got, u.SourceID) },
	})
	d.DispatchAir(airBursts(t, masks, sf), masks)
	if len(got) != 1 || got[0] != wantSrc {
		t.Fatalf("OnCallSource got %v, want [%d]", got, wantSrc)
	}
}

// TestDispatcherDropsSignallingWithoutTheRightSeed is the #915 integrity
// guard on the air path: a scrambled SACCH descrambled with the wrong site
// identity (or none) must decode nothing — the CRC rejects it — so a bogus
// RID can never reach the webhook. The clear SACCH beside it still decodes
// without any seed.
func TestDispatcherDropsSignallingWithoutTheRightSeed(t *testing.T) {
	masks := p25p2.NewAirMasks(testSeed)
	sf := voiceSlots()
	sf[4] = p25p2.AirSlotContent{DUID: p25p2.DUIDSACCHScrambled,
		MAC: macPDU(t, p25p2.AirMACActive, p25p2.DUIDSACCHScrambled, gvcuMsg(111, 1))}
	sf[5] = p25p2.AirSlotContent{DUID: p25p2.DUIDSACCHClear,
		MAC: macPDU(t, p25p2.AirMACActive, p25p2.DUIDSACCHClear, gvcuMsg(222, 2))}
	bursts := airBursts(t, masks, sf)

	for _, tc := range []struct {
		name  string
		masks *p25p2.AirMasks
		want  []uint32
	}{
		{"right seed", masks, []uint32{111, 222}},
		{"wrong seed", p25p2.NewAirMasks(testSeed ^ 1), []uint32{222}},
		{"no seed", nil, []uint32{222}},
	} {
		var got []uint32
		d := NewMACDispatcher(MACDispatcherOptions{
			Log: quietLog(), System: "TestSys", Serial: "tap-0",
			OnCallSource: func(u p25p2.GroupVoiceChannelUser) { got = append(got, u.SourceID) },
		})
		sig, dec := d.DispatchAir(bursts, tc.masks)
		if sig != 2 || dec != len(tc.want) {
			t.Errorf("%s: DispatchAir = %d/%d, want 2/%d", tc.name, sig, dec, len(tc.want))
		}
		if len(got) != len(tc.want) {
			t.Errorf("%s: sources %v, want %v", tc.name, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("%s: sources %v, want %v", tc.name, got, tc.want)
			}
		}
	}
}

// TestDispatchAirSkipsVoice confirms voice bursts never count as
// signalling: the census's mac_bursts must measure MAC framing alone.
func TestDispatchAirSkipsVoice(t *testing.T) {
	d := NewMACDispatcher(MACDispatcherOptions{Log: quietLog(), System: "TestSys", Serial: "tap-0"})
	masks := p25p2.NewAirMasks(testSeed)
	if sig, dec := d.DispatchAir(airBursts(t, masks, voiceSlots()), masks); sig != 0 || dec != 0 {
		t.Fatalf("DispatchAir on voice = %d/%d, want 0/0", sig, dec)
	}
}

// TestDispatcherReadsPrivateAndRegroupCallTalkers: a unit-to-unit call and a
// Motorola regroup (patch) call announce their talker in their own in-call
// messages (opcode 0x02; MFID 0x90 opcode 0x80), not in
// GROUP_VOICE_CHANNEL_USER. Neither used to reach OnCallSource, so such a
// call's recording had no source radio unless its MAC_PTT happened to be
// caught (issue #1364). Byte layouts are SDRTrunk's
// UnitToUnitVoiceChannelUserAbbreviated (service options, target, source)
// and MotorolaGroupRegroupVoiceChannelUserAbbreviated (MFID, one octet,
// supergroup, source).
func TestDispatcherReadsPrivateAndRegroupCallTalkers(t *testing.T) {
	masks := p25p2.NewAirMasks(testSeed)
	uu := []byte{0x02, 0x40, 0x00, 0xAB, 0xCD, 0x04, 0xCF, 0x43}      // target 0x00ABCD, source 315203
	regroup := []byte{0x80, 0x90, 0x08, 0x0C, 0x82, 0x04, 0xCF, 0x44} // supergroup 3202, source 315204
	sf := voiceSlots()
	sf[2] = p25p2.AirSlotContent{DUID: p25p2.DUIDFACCHScrambled,
		MAC: macPDU(t, p25p2.AirMACActive, p25p2.DUIDFACCHScrambled, uu)}
	sf[4] = p25p2.AirSlotContent{DUID: p25p2.DUIDFACCHScrambled,
		MAC: macPDU(t, p25p2.AirMACActive, p25p2.DUIDFACCHScrambled, regroup)}

	var got []p25p2.GroupVoiceChannelUser
	d := NewMACDispatcher(MACDispatcherOptions{
		Log: quietLog(), System: "TestSys", Serial: "tap-0",
		OnCallSource: func(u p25p2.GroupVoiceChannelUser) { got = append(got, u) },
	})
	if sig, dec := d.DispatchAir(airBursts(t, masks, sf), masks); sig != 2 || dec != 2 {
		t.Fatalf("DispatchAir = %d signalling / %d decoded, want 2/2", sig, dec)
	}
	if len(got) != 2 {
		t.Fatalf("OnCallSource called %d times, want 2: %+v", len(got), got)
	}
	if got[0].SourceID != 315203 || got[0].ServiceOptions != 0x40 {
		t.Errorf("unit-to-unit user: %+v, want source 315203, encrypted", got[0])
	}
	if got[1].SourceID != 315204 || got[1].GroupAddress != 3202 {
		t.Errorf("regroup user: %+v, want source 315204 on supergroup 3202", got[1])
	}
}

// TestDispatcherSynchronizationBroadcastIsNotEncryption: opcode 0x70 is the
// site's clock. The dispatcher used to parse it as an encryption sync
// (algorithm from its first payload octet, key from the next two), so a
// clock whose octet happened to be a registered algorithm stamped that
// algorithm and a random key on a clear call. Issue #1364.
func TestDispatcherSynchronizationBroadcastIsNotEncryption(t *testing.T) {
	masks := p25p2.NewAirMasks(testSeed)
	clock := []byte{0x70, 0x84, 0x01, 0x23, 0x35, 0x74, 0x9A, 0xBC, 0xDE}
	sf := voiceSlots()
	sf[4] = p25p2.AirSlotContent{DUID: p25p2.DUIDSACCHScrambled,
		MAC: macPDU(t, p25p2.AirMACActive, p25p2.DUIDSACCHScrambled, clock, gvcuMsg(315203, 0x4EEA))}

	enc, src := 0, 0
	d := NewMACDispatcher(MACDispatcherOptions{
		Log: quietLog(), System: "TestSys", Serial: "tap-0",
		OnCallEncryption: func(p25p2.EncryptionSync) { enc++ },
		OnCallSource:     func(p25p2.GroupVoiceChannelUser) { src++ },
	})
	if _, dec := d.DispatchAir(airBursts(t, masks, sf), masks); dec != 1 {
		t.Fatalf("decoded %d PDUs, want 1", dec)
	}
	if enc != 0 {
		t.Errorf("a Synchronization Broadcast reached OnCallEncryption %d time(s)", enc)
	}
	// The message after it in the same PDU is still walked to.
	if src != 1 {
		t.Errorf("OnCallSource called %d times, want 1", src)
	}
}

// TestDispatcherEndPTTEndsTheTransmission: MAC_END_PTT is the radio
// releasing its PTT. The dispatcher reports each copy.
func TestDispatcherEndPTTEndsTheTransmission(t *testing.T) {
	masks := p25p2.NewAirMasks(testSeed)
	end := make([]byte, 18)
	end[0] = byte(p25p2.AirMACEndPTT) << 5
	end[1], end[2] = 0x01, 0x64 // colour code 0x164
	end[13], end[14], end[15] = 0x04, 0xCF, 0x43
	end[16], end[17] = 0x4E, 0xEA
	sf := voiceSlots()
	sf[2] = p25p2.AirSlotContent{DUID: p25p2.DUIDFACCHScrambled, MAC: end}
	sf[4] = p25p2.AirSlotContent{DUID: p25p2.DUIDFACCHScrambled, MAC: end}

	ends, src := 0, 0
	d := NewMACDispatcher(MACDispatcherOptions{
		Log: quietLog(), System: "TestSys", Serial: "tap-0",
		OnTransmissionEnd: func() { ends++ },
		OnCallSource:      func(p25p2.GroupVoiceChannelUser) { src++ },
	})
	if _, dec := d.DispatchAir(airBursts(t, masks, sf), masks); dec != 2 {
		t.Fatalf("decoded %d PDUs, want 2", dec)
	}
	if ends != 2 || src != 0 {
		t.Errorf("OnTransmissionEnd %d, OnCallSource %d; want 2 and 0", ends, src)
	}
}
