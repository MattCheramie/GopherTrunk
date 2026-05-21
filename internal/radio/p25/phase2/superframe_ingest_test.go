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
	grant.Payload = append(grant.Payload, make([]byte, 17-len(grant.Payload))...)

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

// TestIngestSuperframeAllVoicePublishesNothing confirms an all-voice
// superframe drives no control-channel events — the composer owns voice.
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
