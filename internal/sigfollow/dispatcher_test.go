package sigfollow

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func quietLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// aliasSuperframes builds n superframes that interleave Voice4V
// sub-frames with MAC sub-frames carrying the two talker-alias fragments
// for "UNIT-7" (source 0xABC123), decodes the dibit stream back into
// Superframes, and returns them — the input to MACDispatcher.Dispatch.
func aliasSuperframes(t *testing.T, n int) (sfs []p25p2.Superframe, src uint32, alias string) {
	t.Helper()
	src, alias = 0xABC123, "UNIT-7"
	frag0 := p25p2.EncodeTalkerAliasFragment(p25p2.TalkerAliasFragment{
		SourceID: src, BlockIndex: 0, BlockCount: 2, Data: []byte("UNIT"),
	})
	frag1 := p25p2.EncodeTalkerAliasFragment(p25p2.TalkerAliasFragment{
		SourceID: src, BlockIndex: 1, BlockCount: 2, Data: []byte("-7"),
	})
	dibits := make([]uint8, 50) // sync settling lead-in
	for s := 0; s < n; s++ {
		var subs [p25p2.SubframesPerSuperframe][]uint8
		for i := range subs {
			switch i {
			case 0:
				subs[i] = p25p2.EncodeMACSubframe(p25p2.SlotTypeMACSignaling, uint8(i),
					frag0, p25p2.TrellisOn, p25p2.InterleaveOff)
			case 6:
				subs[i] = p25p2.EncodeMACSubframe(p25p2.SlotTypeMACSignaling, uint8(i),
					frag1, p25p2.TrellisOn, p25p2.InterleaveOff)
			default:
				payloads := make([][]byte, p25p2.Voice4VFrameCount)
				for j := range payloads {
					payloads[j] = make([]byte, p25p2.VoiceFrameBytes)
				}
				subs[i] = p25p2.EncodeVoiceSubframe(p25p2.SlotTypeVoice4V, uint8(i), payloads)
			}
		}
		dibits = append(dibits, p25p2.EncodeSuperframe(subs)...)
	}
	sfs = p25p2.NewSuperframeDecoder().Process(dibits, 0)
	if len(sfs) == 0 {
		t.Fatal("no superframes decoded from synthesized alias stream")
	}
	return sfs, src, alias
}

// TestDispatcherPublishesTalkerAlias feeds alias-bearing superframes
// through the shared dispatcher and asserts a KindTalkerAlias event with
// the reassembled name surfaces on the bus — the core decode the
// signalling follower harvests off the traffic channel (#376).
func TestDispatcherPublishesTalkerAlias(t *testing.T) {
	bus := events.NewBus(64)
	sub := bus.Subscribe()
	defer sub.Close()

	d := NewMACDispatcher(MACDispatcherOptions{
		Bus: bus, Log: quietLog(), LogPrefix: "sigfollow",
		System: "TestSys", Serial: "tap-0",
	})
	macCfg := p25p2.MACDecodeConfig{Trellis: p25p2.TrellisOn}

	sfs, wantSrc, wantAlias := aliasSuperframes(t, 2)
	for _, sf := range sfs {
		d.Dispatch(sf, macCfg)
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
			if ta.SourceID != wantSrc {
				t.Errorf("SourceID = %#x, want %#x", ta.SourceID, wantSrc)
			}
			if ta.Alias != wantAlias {
				t.Errorf("Alias = %q, want %q", ta.Alias, wantAlias)
			}
			if ta.System != "TestSys" {
				t.Errorf("System = %q, want TestSys", ta.System)
			}
			if ta.Protocol != "p25-phase2" {
				t.Errorf("Protocol = %q, want p25-phase2", ta.Protocol)
			}
			return
		case <-deadline:
			t.Fatal("no KindTalkerAlias published")
		}
	}
}

// TestDispatcherPTTSlotDrivesEncryptionHook builds a superframe whose
// MAC_PTT sub-frame carries an Encryption Sync and asserts the dispatcher
// routes its ALGID/KID to the OnCallEncryption hook (the voice composer
// wires this to its engine-backfill publisher). Routing is by slot type,
// not opcode, since the PTT message has no normal MAC opcode (#813).
func TestDispatcherPTTSlotDrivesEncryptionHook(t *testing.T) {
	es := p25p2.EncryptionSync{
		AlgorithmID:      0x84,
		KeyID:            0x1234,
		MessageIndicator: [9]byte{1, 2, 3, 4, 5, 6, 7, 8, 9},
	}
	ptt := p25p2.EncodePushToTalk(es)

	dibits := make([]uint8, 50)
	var subs [p25p2.SubframesPerSuperframe][]uint8
	for i := range subs {
		if i == 0 {
			subs[i] = p25p2.EncodeMACSubframe(p25p2.SlotTypeMACPTT, uint8(i),
				ptt, p25p2.TrellisOn, p25p2.InterleaveOff)
			continue
		}
		payloads := make([][]byte, p25p2.Voice4VFrameCount)
		for j := range payloads {
			payloads[j] = make([]byte, p25p2.VoiceFrameBytes)
		}
		subs[i] = p25p2.EncodeVoiceSubframe(p25p2.SlotTypeVoice4V, uint8(i), payloads)
	}
	dibits = append(dibits, p25p2.EncodeSuperframe(subs)...)
	sfs := p25p2.NewSuperframeDecoder().Process(dibits, 0)
	if len(sfs) == 0 {
		t.Fatal("no superframes decoded")
	}

	var got p25p2.EncryptionSync
	var called int
	d := NewMACDispatcher(MACDispatcherOptions{
		Log: quietLog(), System: "TestSys", Serial: "tap-0",
		OnCallEncryption: func(e p25p2.EncryptionSync) { got = e; called++ },
	})
	macCfg := p25p2.MACDecodeConfig{Trellis: p25p2.TrellisOn}
	for _, sf := range sfs {
		d.Dispatch(sf, macCfg)
	}
	if called == 0 {
		t.Fatal("OnCallEncryption never invoked for PTT-slot encryption sync")
	}
	if got.AlgorithmID != 0x84 || got.KeyID != 0x1234 {
		t.Errorf("OnCallEncryption alg/key = %#x/%#x, want 0x84/0x1234",
			got.AlgorithmID, got.KeyID)
	}
}

// TestDispatcherSourceCallback verifies the in-call GROUP_VOICE_CHANNEL_USER
// PDU routes to the OnCallSource hook (the voice composer wires this to its
// engine-backfill publisher; the follower leaves it nil).
func TestDispatcherSourceCallback(t *testing.T) {
	const wantSrc = 315203
	user := p25p2.EncodeGroupVoiceChannelUser(p25p2.GroupVoiceChannelUser{
		ServiceOptions: 0x40, GroupAddress: 0x4EEA, SourceID: wantSrc,
	}, false)

	dibits := make([]uint8, 50)
	var subs [p25p2.SubframesPerSuperframe][]uint8
	for i := range subs {
		if i == 0 {
			subs[i] = p25p2.EncodeMACSubframe(p25p2.SlotTypeMACSignaling, uint8(i),
				user, p25p2.TrellisOn, p25p2.InterleaveOff)
			continue
		}
		payloads := make([][]byte, p25p2.Voice4VFrameCount)
		for j := range payloads {
			payloads[j] = make([]byte, p25p2.VoiceFrameBytes)
		}
		subs[i] = p25p2.EncodeVoiceSubframe(p25p2.SlotTypeVoice4V, uint8(i), payloads)
	}
	dibits = append(dibits, p25p2.EncodeSuperframe(subs)...)
	sfs := p25p2.NewSuperframeDecoder().Process(dibits, 0)
	if len(sfs) == 0 {
		t.Fatal("no superframes decoded")
	}

	var gotSrc uint32
	var called int
	d := NewMACDispatcher(MACDispatcherOptions{
		Log: quietLog(), System: "TestSys", Serial: "tap-0",
		OnCallSource: func(u p25p2.GroupVoiceChannelUser) { gotSrc = u.SourceID; called++ },
	})
	macCfg := p25p2.MACDecodeConfig{Trellis: p25p2.TrellisOn}
	for _, sf := range sfs {
		d.Dispatch(sf, macCfg)
	}
	if called == 0 {
		t.Fatal("OnCallSource never invoked")
	}
	if gotSrc != wantSrc {
		t.Errorf("OnCallSource SourceID = %d, want %d", gotSrc, wantSrc)
	}
}
