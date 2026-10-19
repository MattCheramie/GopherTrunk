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
