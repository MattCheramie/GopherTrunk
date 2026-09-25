package phase2

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// decodeOneSuperframe builds a 50-dibit lead-in + the encoded
// superframe and returns the single decoded Superframe.
func decodeOneSuperframe(t *testing.T, subs [SubframesPerSuperframe][]uint8) Superframe {
	t.Helper()
	stream := append(make([]uint8, 50), EncodeSuperframe(subs)...)
	got := NewSuperframeDecoder().Process(stream, 0)
	if len(got) != 1 {
		t.Fatalf("expected 1 superframe, got %d", len(got))
	}
	return got[0]
}

func countGrants(sub *events.Subscription) []trunking.Grant {
	var out []trunking.Grant
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				out = append(out, ev.Payload.(trunking.Grant))
			}
		default:
			return out
		}
	}
}

// TestIngestSuperframeRoutesMACSubframes confirms IngestSuperframe
// decodes the MAC-bearing sub-frames into grants and skips the voice
// sub-frames.
func TestIngestSuperframeRoutesMACSubframes(t *testing.T) {
	bus := events.NewBus(32)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "p2", FrequencyHz: 851_000_000})
	cc.SetTrellisMode(TrellisOn)

	grant := grantPDU(0x1234, 0x00ABCD, 0x1, 0x005)

	var subs [SubframesPerSuperframe][]uint8
	for i := range subs {
		if i == 0 {
			subs[i] = EncodeMACSubframe(SlotTypeMACSignaling, uint8(i), grant,
				TrellisOn, InterleaveOff)
		} else {
			subs[i] = EncodeVoiceSubframe(SlotTypeVoice4V, uint8(i),
				voicePayloads(Voice4VFrameCount))
		}
	}

	cc.IngestSuperframe(decodeOneSuperframe(t, subs))

	grants := countGrants(sub)
	if len(grants) != 1 {
		t.Fatalf("expected exactly 1 grant (voice sub-frames must be skipped), got %d", len(grants))
	}
	if grants[0].GroupID != 0x1234 {
		t.Errorf("grant GroupID = %#x, want 0x1234", grants[0].GroupID)
	}
}

// TestDecodeSuperframeMACPDUsReturnsAllMACSubframes confirms the
// pure-function MAC dispatch the voice composer uses returns every
// MAC-typed sub-frame's PDU in order and skips voice sub-frames.
func TestDecodeSuperframeMACPDUsReturnsAllMACSubframes(t *testing.T) {
	macSlots := []SlotType{SlotTypeMACSignaling, SlotTypeMACActive, SlotTypeMACPTT}
	pdus := []MACPDU{
		EncodeTalkerAliasFragment(TalkerAliasFragment{SourceID: 0xABC123, BlockIndex: 0, BlockCount: 2, Data: []byte("UNIT")}),
		EncodeTalkerAliasFragment(TalkerAliasFragment{SourceID: 0xABC123, BlockIndex: 1, BlockCount: 2, Data: []byte("-7")}),
		grantPDU(0x1234, 0x00ABCD, 0x1, 0x005),
	}
	var subs [SubframesPerSuperframe][]uint8
	macIdx := 0
	for i := range subs {
		if macIdx < len(pdus) {
			subs[i] = EncodeMACSubframe(macSlots[macIdx], uint8(i), pdus[macIdx],
				TrellisOn, InterleaveOff)
			macIdx++
		} else {
			subs[i] = EncodeVoiceSubframe(SlotTypeVoice4V, uint8(i),
				voicePayloads(Voice4VFrameCount))
		}
	}

	cfg := MACDecodeConfig{Trellis: TrellisOn}
	got := DecodeSuperframeMACPDUs(decodeOneSuperframe(t, subs), cfg)
	if len(got) != len(pdus) {
		t.Fatalf("DecodeSuperframeMACPDUs returned %d PDUs, want %d", len(got), len(pdus))
	}
	// The PTT sub-frame is the odd one out on purpose: a PTT PDU has no inner
	// opcode, its header byte *is* the opcode, so what comes back is 0x20 and
	// not whatever PDU the fixture handed the encoder.
	wantOps := []Opcode{pdus[0].Opcode, pdus[1].Opcode, 0x20}
	for i, want := range wantOps {
		if got[i].Opcode != want {
			t.Errorf("PDU[%d] opcode = %#x, want %#x", i, got[i].Opcode, want)
		}
	}
}

// TestDecodeSuperframeMACPDUsWithSlotTagsPTT confirms the slot-aware
// decode surfaces each PDU's originating SlotType — specifically that a
// MAC_PTT sub-frame is tagged SlotTypeMACPTT so the dispatcher can route
// the encryption sync it carries by slot type (issue #813).
func TestDecodeSuperframeMACPDUsWithSlotTagsPTT(t *testing.T) {
	es := EncryptionSync{AlgorithmID: 0x84, KeyID: 0x1234,
		MessageIndicator: [9]byte{9, 8, 7, 6, 5, 4, 3, 2, 1}}
	ptt := EncodePushToTalk(es)

	var subs [SubframesPerSuperframe][]uint8
	for i := range subs {
		if i == 0 {
			subs[i] = EncodeMACSubframe(SlotTypeMACPTT, uint8(i), ptt,
				TrellisOn, InterleaveOff)
		} else {
			subs[i] = EncodeVoiceSubframe(SlotTypeVoice4V, uint8(i),
				voicePayloads(Voice4VFrameCount))
		}
	}

	cfg := MACDecodeConfig{Trellis: TrellisOn}
	got := DecodeSuperframeMACPDUsWithSlot(decodeOneSuperframe(t, subs), cfg)
	if len(got) != 1 {
		t.Fatalf("DecodeSuperframeMACPDUsWithSlot returned %d PDUs, want 1", len(got))
	}
	if got[0].SlotType != SlotTypeMACPTT {
		t.Errorf("SlotType = %v, want SlotTypeMACPTT", got[0].SlotType)
	}
	es2, ok := got[0].PDU.AsPushToTalk()
	if !ok {
		t.Fatal("AsPushToTalk on PTT-slot PDU returned !ok")
	}
	if es2.AlgorithmID != 0x84 || es2.KeyID != 0x1234 {
		t.Errorf("PTT alg/key = %#x/%#x, want 0x84/0x1234", es2.AlgorithmID, es2.KeyID)
	}
}

// TestDecodeSuperframeMACPDUsRSValidFlag pins the per-PDU RS-integrity
// / TestDecodeSuperframeMACPDUsRSValid pins what the RSValid flag now reports:
// that the burst's outer RS(63,35,29) closed, alongside the CRC-12. That is a
// property of the burst rather than of the PDU inside it, so a well-formed
// sub-frame always carries it — and damage past the code's correction budget
// yields no PDU at all rather than an unvouched one.
//
// It replaces a test built on the older model, where RSValid meant an inner
// RS(24,16,9) parity carried in the PDU's own bytes. P25 Phase 2 has no such
// field; the outer code is the integrity the wire actually provides.
func TestDecodeSuperframeMACPDUsRSValid(t *testing.T) {
	const wantSrc = 315203
	base := GroupVoiceChannelUser{ServiceOptions: 0x40, GroupAddress: 0x4EEA, SourceID: wantSrc}
	cfg := MACDecodeConfig{}

	build := func(damage int) [SubframesPerSuperframe][]uint8 {
		var subs [SubframesPerSuperframe][]uint8
		for i := range subs {
			if i == 0 {
				subs[i] = EncodeMACSubframe(SlotTypeMACSignaling, uint8(i),
					EncodeGroupVoiceChannelUser(base, false), TrellisOn, InterleaveOff)
				for d := 0; d < damage; d++ {
					subs[i][ISCHRegionDibits+1+3*d] ^= 1
				}
			} else {
				subs[i] = EncodeVoiceSubframe(SlotTypeVoice4V, uint8(i), voicePayloads(Voice4VFrameCount))
			}
		}
		return subs
	}

	got := DecodeSuperframeMACPDUsWithSlot(decodeOneSuperframe(t, build(0)), cfg)
	if len(got) != 1 {
		t.Fatalf("clean superframe returned %d PDUs, want 1", len(got))
	}
	if !got[0].RSValid {
		t.Error("clean burst decoded with RSValid=false")
	}
	if u, ok := got[0].PDU.AsGroupVoiceChannelUser(); !ok || u.SourceID != wantSrc {
		t.Errorf("recovered %+v ok=%v, want SourceID=%d", u, ok, wantSrc)
	}

	// Within the correction budget the RS repairs the burst and the source
	// RID still comes back intact.
	got = DecodeSuperframeMACPDUsWithSlot(decodeOneSuperframe(t, build(6)), cfg)
	if len(got) != 1 || !got[0].RSValid {
		t.Fatalf("6 damaged symbols: got %d PDUs, want 1 RS-valid", len(got))
	}
	if u, ok := got[0].PDU.AsGroupVoiceChannelUser(); !ok || u.SourceID != wantSrc {
		t.Errorf("6 damaged symbols: recovered %+v ok=%v, want SourceID=%d", u, ok, wantSrc)
	}

	// Past it, nothing is emitted — an unvouched MAC PDU is a fabricated grant.
	if got := DecodeSuperframeMACPDUsWithSlot(decodeOneSuperframe(t, build(14)), cfg); len(got) != 0 {
		t.Errorf("14 damaged symbols returned %d PDUs, want 0: %+v", len(got), got)
	}
}
func TestIngestSuperframeAllVoicePublishesNothing(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "p2", FrequencyHz: 851_000_000})
	cc.SetTrellisMode(TrellisOn)

	var subs [SubframesPerSuperframe][]uint8
	for i := range subs {
		subs[i] = EncodeVoiceSubframe(SlotTypeVoice4V, uint8(i),
			voicePayloads(Voice4VFrameCount))
	}
	cc.IngestSuperframe(decodeOneSuperframe(t, subs))

	select {
	case ev := <-sub.C:
		t.Errorf("all-voice superframe published an event: %v", ev.Kind)
	default:
	}
}
