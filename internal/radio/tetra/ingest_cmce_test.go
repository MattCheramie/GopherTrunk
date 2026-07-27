package tetra

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestHandleCMCEReleaseResolvesViaSetupMap: a D-SETUP teaches the call
// identifier → group mapping, so a later D-RELEASE (which carries only the call
// identifier) publishes a KindCallRelease keyed to that group — even when the
// release PDU's own MAC address is not the group.
func TestHandleCMCEReleaseResolvesViaSetupMap(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "Sys", FrequencyHz: 467_913_000})

	const (
		callID = 0x1234
		gssi   = 0x0F5670
	)
	// D-SETUP addressed to the GSSI (via the MAC address) learns the mapping.
	cc.handleCMCE(
		MACResource{Address: MACAddress{Type: addrSSI, SSI: gssi}},
		CMCEMessage{Type: CMCETypeDSetup, CallIdentifier: callID, GroupSSI: gssi},
	)
	// D-RELEASE whose MAC address is an individual, not the group: the map
	// resolves it back to the group.
	cc.handleCMCE(
		MACResource{Address: MACAddress{Type: addrSSI, SSI: 0x999999}},
		CMCEMessage{Type: CMCETypeDRelease, CallIdentifier: callID, DisconnectCause: 3},
	)

	rel := waitForRelease(t, sub)
	if rel.System != "Sys" || rel.GroupID != gssi {
		t.Errorf("CallRelease = %+v, want System=Sys GroupID=%#x", rel, gssi)
	}
	if rel.Reason != trunking.EndReasonReleased {
		t.Errorf("release reason = %v, want released", rel.Reason)
	}
}

// TestHandleCMCEReleaseFallsBackToMACAddress: with no prior D-SETUP, a D-RELEASE
// resolves its group from the MAC-RESOURCE address it was carried under (a group
// call's downlink signalling is addressed to the GSSI).
func TestHandleCMCEReleaseFallsBackToMACAddress(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "Sys", FrequencyHz: 467_913_000})
	const gssi = 0x0ABCDE
	cc.handleCMCE(
		MACResource{Address: MACAddress{Type: addrSSI, SSI: gssi}},
		CMCEMessage{Type: CMCETypeDRelease, CallIdentifier: 0x55},
	)
	if rel := waitForRelease(t, sub); rel.GroupID != gssi {
		t.Errorf("fallback release GroupID = %#x, want %#x", rel.GroupID, gssi)
	}
}

// TestHandleCMCETxGrantedPublishesTalker: a D-TX-GRANTED carrying the
// transmitting party SSI publishes a KindCallTalker for the call's group.
func TestHandleCMCETxGrantedPublishesTalker(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "Sys", FrequencyHz: 467_913_000})
	const (
		gssi   = 0x0ABCDE
		talker = 0x00CAFE
	)
	cc.handleCMCE(
		MACResource{Address: MACAddress{Type: addrSSI, SSI: gssi}},
		CMCEMessage{Type: CMCETypeDTxGranted, CallIdentifier: 0x66, PartySSI: talker},
	)

	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCallTalker {
				tk := ev.Payload.(trunking.CallTalker)
				if tk.GroupID != gssi || tk.SourceID != talker {
					t.Errorf("CallTalker = %+v, want group %#x src %#x", tk, gssi, talker)
				}
				return
			}
		default:
			t.Fatal("no KindCallTalker published for D-TX-GRANTED")
		}
	}
}

func waitForRelease(t *testing.T, sub *events.Subscription) trunking.CallRelease {
	t.Helper()
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCallRelease {
				return ev.Payload.(trunking.CallRelease)
			}
		default:
			t.Fatal("no KindCallRelease published")
			return trunking.CallRelease{}
		}
	}
}
