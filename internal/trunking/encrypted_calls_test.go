package trunking

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// mkEncEngine builds an engine wired for encrypted-call-policy tests: a
// one-device pool, an injectable clock, a long call timeout (so the
// inactivity watchdog never interferes with what we're testing), and the
// supplied mode / follow window / configured keys. The mode + follow are
// applied per-system to the system names these tests drive grants for
// (the policy is per-system since issue #711's per-system refactor).
func mkEncEngine(t *testing.T, mode EncryptedMode, follow time.Duration, keys map[string]map[uint16]bool) (*Engine, *VoicePool, *events.Bus, *fakeClock) {
	t.Helper()
	bus := events.NewBus(16)
	pool, _ := mkPool(1)
	clock := &fakeClock{t: time.Unix(1000, 0)}
	modes := map[string]EncryptedMode{}
	follows := map[string]time.Duration{}
	for _, sys := range []string{"X", "DMR-EP"} {
		modes[sys] = mode
		if follow > 0 {
			follows[sys] = follow
		}
	}
	e, err := NewEngine(EngineOptions{
		Bus:              bus,
		VoicePool:        pool,
		Talkgroups:       NewTalkgroupDB(),
		CallTimeout:      time.Hour,
		Now:              clock.Now,
		EncryptedModes:   modes,
		EncryptedFollows: follows,
		ConfiguredKeys:   keys,
	})
	if err != nil {
		t.Fatal(err)
	}
	return e, pool, bus, clock
}

func TestEngineIgnoreDropsEncryptedGrant(t *testing.T) {
	e, _, bus, _ := mkEncEngine(t, EncryptedIgnore, 0, nil)
	defer bus.Close()

	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 100, FrequencyHz: 851_000_000, Encrypted: true})

	if got := e.ActiveCalls(); len(got) != 0 {
		t.Fatalf("ignore mode must not allocate a voice SDR to an encrypted grant; active = %+v", got)
	}
}

func TestEngineIgnoreStillFollowsClearGrant(t *testing.T) {
	e, _, bus, _ := mkEncEngine(t, EncryptedIgnore, 0, nil)
	defer bus.Close()

	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 100, FrequencyHz: 851_000_000})

	if got := e.ActiveCalls(); len(got) != 1 {
		t.Fatalf("ignore mode must still follow clear calls; active = %+v", got)
	}
}

// TestEngineFollowHoldsEncryptedGrant is the backward-compatibility guard:
// the default mode keeps tying up a voice SDR for an encrypted call, with
// no metadata release armed.
func TestEngineFollowHoldsEncryptedGrant(t *testing.T) {
	e, _, bus, clock := mkEncEngine(t, EncryptedFollow, time.Second, nil)
	defer bus.Close()

	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 100, FrequencyHz: 851_000_000, Encrypted: true})
	if len(e.ActiveCalls()) != 1 {
		t.Fatal("follow mode should bind the encrypted call")
	}
	clock.t = clock.t.Add(10 * time.Second)
	e.runWatchdog()
	if got := e.ActiveCalls(); len(got) != 1 {
		t.Fatalf("follow mode must hold the encrypted call; active = %+v", got)
	}
}

// TestEngineMetadataReleasesEncryptedGrantAfterWindow covers the grant-time
// path: a P25 Phase 2 grant that already signals encryption is followed,
// then released by the watchdog once the metadata window elapses.
func TestEngineMetadataReleasesEncryptedGrantAfterWindow(t *testing.T) {
	e, _, bus, clock := mkEncEngine(t, EncryptedMetadata, time.Second, nil)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	e.HandleGrant(Grant{System: "X", Protocol: "p25-phase2", GroupID: 100, FrequencyHz: 851_000_000, Encrypted: true})
	if len(e.ActiveCalls()) != 1 {
		t.Fatal("metadata mode should follow the encrypted call briefly")
	}

	// Before the window: the watchdog must not release it yet.
	clock.t = clock.t.Add(500 * time.Millisecond)
	e.runWatchdog()
	if len(e.ActiveCalls()) != 1 {
		t.Fatal("metadata mode released the call before the follow window elapsed")
	}

	// After the window: released with EndReasonEncrypted.
	clock.t = clock.t.Add(time.Second)
	e.runWatchdog()
	if got := e.ActiveCalls(); len(got) != 0 {
		t.Fatalf("metadata mode should release after the window; active = %+v", got)
	}
	end := drainCallEnd(t, sub, 500*time.Millisecond)
	if end.Reason != EndReasonEncrypted {
		t.Errorf("CallEnd.Reason = %v, want EndReasonEncrypted", end.Reason)
	}
}

// TestEngineMetadataKeepsClearCall guards against arming a release on a
// clear call: a clear call under metadata mode must outlive the follow
// window (only the inactivity watchdog, set to an hour here, can end it).
func TestEngineMetadataKeepsClearCall(t *testing.T) {
	e, _, bus, clock := mkEncEngine(t, EncryptedMetadata, time.Second, nil)
	defer bus.Close()

	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 100, FrequencyHz: 851_000_000})
	clock.t = clock.t.Add(10 * time.Second)
	e.runWatchdog()
	if got := e.ActiveCalls(); len(got) != 1 {
		t.Fatalf("metadata mode must not release a clear call; active = %+v", got)
	}
}

// TestEngineIgnoreEndsCallOnMidCallEncryption covers the common P25 case:
// the grant looks clear, a voice SDR binds, then the traffic channel
// reveals encryption — ignore mode must free the tuner immediately.
func TestEngineIgnoreEndsCallOnMidCallEncryption(t *testing.T) {
	e, _, bus, _ := mkEncEngine(t, EncryptedIgnore, 0, nil)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 100, FrequencyHz: 851_000_000})
	actives := e.ActiveCalls()
	if len(actives) != 1 {
		t.Fatal("clear-looking grant should bind")
	}
	dev := actives[0].Device.Serial

	e.handleCallEncryption(CallEncryption{DeviceSerial: dev, AlgorithmID: 0x84, KeyID: 0x10})

	if got := e.ActiveCalls(); len(got) != 0 {
		t.Fatalf("ignore mode should free the tuner on mid-call encryption; active = %+v", got)
	}
	end := drainCallEnd(t, sub, 500*time.Millisecond)
	if end.Reason != EndReasonEncrypted {
		t.Errorf("CallEnd.Reason = %v, want EndReasonEncrypted", end.Reason)
	}
}

// TestEngineMetadataArmsOnMidCallSourceUpdate covers a Phase 2 compressed
// grant whose encryption only surfaces via a GROUP_VOICE_CHANNEL_USER PDU:
// metadata mode arms the release, and the watchdog reaps it after the
// window.
func TestEngineMetadataArmsOnMidCallSourceUpdate(t *testing.T) {
	e, _, bus, clock := mkEncEngine(t, EncryptedMetadata, time.Second, nil)
	defer bus.Close()

	e.HandleGrant(Grant{System: "X", Protocol: "p25-phase2", GroupID: 100, FrequencyHz: 851_000_000})
	dev := e.ActiveCalls()[0].Device.Serial

	e.handleCallSourceUpdate(CallSourceUpdate{DeviceSerial: dev, SourceID: 555, Encrypted: true})

	clock.t = clock.t.Add(2 * time.Second)
	e.runWatchdog()
	if got := e.ActiveCalls(); len(got) != 0 {
		t.Fatalf("metadata mode should release after the window once encryption surfaces; active = %+v", got)
	}
}

// TestEngineExemptsDecryptableGrant covers the configured-key exemption: a
// call whose KeyID matches a configured encryption_keys entry is always
// followed, even under ignore mode.
func TestEngineExemptsDecryptableGrant(t *testing.T) {
	keys := map[string]map[uint16]bool{"DMR-EP": {0x2B: true}}
	e, _, bus, _ := mkEncEngine(t, EncryptedIgnore, 0, keys)
	defer bus.Close()

	e.HandleGrant(Grant{System: "DMR-EP", Protocol: "dmr", GroupID: 7, FrequencyHz: 460_000_000, Encrypted: true, KeyID: 0x2B})

	if got := e.ActiveCalls(); len(got) != 1 {
		t.Fatalf("a call with a configured key must be followed even under ignore; active = %+v", got)
	}
}

// TestEngineMetadataDisarmsWhenKeyDiscoveredMidCall covers the case where a
// call is armed for release (encryption known, KeyID not yet matched) and a
// later in-call encryption sync reveals a KeyID the operator holds a key
// for: the pending release is cancelled and the call is followed.
func TestEngineMetadataDisarmsWhenKeyDiscoveredMidCall(t *testing.T) {
	keys := map[string]map[uint16]bool{"DMR-EP": {0x2B: true}}
	e, pool, bus, clock := mkEncEngine(t, EncryptedMetadata, time.Second, keys)
	defer bus.Close()

	// Bind a clear-looking call, then surface encryption with an unknown
	// KeyID — this arms the metadata release.
	e.HandleGrant(Grant{System: "DMR-EP", Protocol: "dmr", GroupID: 7, FrequencyHz: 460_000_000})
	dev := e.ActiveCalls()[0].Device.Serial
	e.handleCallEncryption(CallEncryption{DeviceSerial: dev, AlgorithmID: 0xAA, KeyID: 0x99})
	if pool.EncryptedReleasesDue(clock.t.Add(2*time.Second)) == nil {
		t.Fatal("expected a release armed for the unknown-key encrypted call")
	}

	// A later sync reveals the operator's key ID — the release is cancelled.
	e.handleCallEncryption(CallEncryption{DeviceSerial: dev, AlgorithmID: 0xAA, KeyID: 0x2B})
	clock.t = clock.t.Add(2 * time.Second)
	e.runWatchdog()
	if got := e.ActiveCalls(); len(got) != 1 {
		t.Fatalf("discovering a configured key must cancel the release; active = %+v", got)
	}
}

// TestEngineMetadataReleasesOnTalkerAlias covers the alias short-circuit
// (issue #711 follow-up): a metadata-mode call armed for release is torn
// down immediately when the system's talker alias completes, well before
// the metadata_follow_ms window would have elapsed.
func TestEngineMetadataReleasesOnTalkerAlias(t *testing.T) {
	e, _, bus, clock := mkEncEngine(t, EncryptedMetadata, time.Hour, nil)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	// A Phase 2 grant whose source surfaces mid-call, flagged encrypted —
	// this arms the metadata release with a long (1h) window.
	e.HandleGrant(Grant{System: "X", Protocol: "p25-phase2", GroupID: 100, FrequencyHz: 851_000_000})
	dev := e.ActiveCalls()[0].Device.Serial
	e.handleCallSourceUpdate(CallSourceUpdate{DeviceSerial: dev, SourceID: 4242, Encrypted: true})

	// The alias completes for this system + source: release immediately,
	// without advancing the clock anywhere near the follow window.
	e.handleTalkerAlias(TalkerAlias{System: "X", Protocol: "p25-phase2", SourceID: 4242, Alias: "MEDIC 7"})

	if got := e.ActiveCalls(); len(got) != 0 {
		t.Fatalf("a completed talker alias should release the armed metadata call; active = %+v", got)
	}
	end := drainCallEnd(t, sub, 500*time.Millisecond)
	if end.Reason != EndReasonEncrypted {
		t.Errorf("CallEnd.Reason = %v, want EndReasonEncrypted", end.Reason)
	}
	_ = clock
}

// TestEngineFollowIgnoresTalkerAlias guards the short-circuit's scope: a
// follow-mode call is never armed for release, so a completed alias must
// not tear it down.
func TestEngineFollowIgnoresTalkerAlias(t *testing.T) {
	e, _, bus, _ := mkEncEngine(t, EncryptedFollow, time.Hour, nil)
	defer bus.Close()

	e.HandleGrant(Grant{System: "X", Protocol: "p25-phase2", GroupID: 100, FrequencyHz: 851_000_000})
	dev := e.ActiveCalls()[0].Device.Serial
	e.handleCallSourceUpdate(CallSourceUpdate{DeviceSerial: dev, SourceID: 4242, Encrypted: true})

	e.handleTalkerAlias(TalkerAlias{System: "X", Protocol: "p25-phase2", SourceID: 4242, Alias: "MEDIC 7"})

	if got := e.ActiveCalls(); len(got) != 1 {
		t.Fatalf("follow mode must keep the call despite a completed alias; active = %+v", got)
	}
}
