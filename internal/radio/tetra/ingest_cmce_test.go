package tetra

import (
	"bytes"
	"log/slog"
	"strings"
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

// TestHandleCMCETxGrantedSuppressesSelfTalker: a D-TX-GRANTED whose party equals
// the call's own identity (an individual call GT tracks by the radio's ISSI) must
// NOT publish a KindCallTalker — a "tg==src" self-reference is impossible and was
// the "call source update tg==src" the operator reported. It's learned as an
// individual instead.
func TestHandleCMCETxGrantedSuppressesSelfTalker(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "Sys", FrequencyHz: 467_913_000})
	const radio = 1005499
	// No remembered call → resolveGroup returns the addressed SSI, so gssi == party.
	cc.handleCMCE(
		MACResource{Address: MACAddress{Type: addrSSI, SSI: radio}},
		CMCEMessage{Type: CMCETypeDTxGranted, CallIdentifier: 0x66, PartySSI: radio},
	)

	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCallTalker {
				tk := ev.Payload.(trunking.CallTalker)
				t.Fatalf("published a self-referential talker %+v (tg==src), want none", tk)
			}
		default:
			if !cc.isIndividual(radio) {
				t.Errorf("self-talker radio %d not learned as an individual", radio)
			}
			return
		}
	}
}

// TestHandleCMCETxGrantedDedupsTalkerLog: TETRA rebroadcasts D-TX-GRANTED for the
// same talker every few seconds (and keeps doing so after a voice-follow drops on
// hangtime while the group is still up), so the "tetra: call talker" debug line
// used to repeat for an unchanged talker — the "hanging call talker" a field
// report flagged. The dedup logs on a talker *change* only, while the
// KindCallTalker event still fires on every rebroadcast (the engine backfill is
// idempotent). Pre-fix this logs one line per rebroadcast; after, one per change.
func TestHandleCMCETxGrantedDedupsTalkerLog(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cc := New(Options{Bus: bus, SystemName: "Sys", FrequencyHz: 467_913_000, Log: log})

	const (
		gssi    = 0x0ABCDE
		talkerA = 0x00CAFE
		talkerB = 0x00BEEF
	)
	tx := func(party uint32) {
		cc.handleCMCE(
			MACResource{Address: MACAddress{Type: addrSSI, SSI: gssi}},
			CMCEMessage{Type: CMCETypeDTxGranted, CallIdentifier: 0x66, PartySSI: party},
		)
	}
	// Same talker rebroadcast four times, then a genuine talker change.
	tx(talkerA)
	tx(talkerA)
	tx(talkerA)
	tx(talkerA)
	tx(talkerB)

	// The event fires on every rebroadcast — the dedup is log-only.
	got := 0
drain:
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCallTalker {
				got++
			}
		default:
			break drain
		}
	}
	if got != 5 {
		t.Errorf("KindCallTalker events = %d, want 5 (the event must not be deduped)", got)
	}

	// The log collapses the four identical talkers to one line, plus one for the
	// change — two, not five.
	if n := strings.Count(buf.String(), "tetra: call talker"); n != 2 {
		t.Fatalf(`"tetra: call talker" log lines = %d, want 2 (4 identical rebroadcasts collapse to 1, +1 for the changed talker); pre-fix this is 5`, n)
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
