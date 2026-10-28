package voice

import (
	"context"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// mkSkipEncKeyedRecorder builds a SkipEncrypted recorder whose KeyConfigured
// predicate holds keys 1 and 3 for every system — the shape of the #1261
// reporter's scan-list channel (NXDN scrambler keys on key ids 1 and 3).
func mkSkipEncKeyedRecorder(t *testing.T) (*Recorder, *events.Bus, string) {
	t.Helper()
	bus := events.NewBus(8)
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus:           bus,
		OutDir:        dir,
		SampleRate:    8000,
		SkipEncrypted: true,
		KeyConfigured: func(_ string, _ uint8, keyID uint16) bool {
			return keyID == 0 || keyID == 1 || keyID == 3
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return r, bus, dir
}

// TestRecorderSkipEncryptedKeepsDecryptableMidCallEncryption pins #1261: with
// recordings.skip_encrypted on, an NXDN scrambled call whose key IS configured
// was aborted the moment its VCALL named the cipher, although the composer was
// descrambling every frame (`descrambled_frames=168 ciphertext_frames=0` in the
// reporter's log) — so no decrypted call was ever recorded. skip_encrypted
// means "don't record what can't be decoded"; a call an operator key covers
// is clear audio and must record.
func TestRecorderSkipEncryptedKeepsDecryptableMidCallEncryption(t *testing.T) {
	r, bus, dir := mkSkipEncKeyedRecorder(t)
	defer r.Close()
	defer bus.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant:        trunking.Grant{System: "scanner", Protocol: "nxdn", GroupID: 44710000},
		Talkgroup:    &trunking.TalkGroup{ID: 44710000, Record: true},
		DeviceSerial: "R1",
		StartedAt:    time.Now(),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	waitSession(t, r, "R1", true)
	if err := r.WritePCM("R1", make([]int16, 1600)); err != nil {
		t.Fatal(err)
	}

	// VCALL: cipher type 1 (scrambler), key id 1 — configured.
	bus.Publish(events.Event{Kind: events.KindCallEncryption, Payload: trunking.CallEncryption{
		DeviceSerial: "R1", Protocol: "nxdn", AlgorithmID: 1, KeyID: 1, At: time.Now(),
	}})
	time.Sleep(50 * time.Millisecond)
	if !r.HasSession("R1") {
		t.Fatal("skip_encrypted aborted a call whose key is configured (it decrypts)")
	}
	if got := wavFiles(t, dir); len(got) != 1 {
		t.Fatalf("expected the decryptable call's wav to survive, got %d files", len(got))
	}
}

// TestRecorderSkipEncryptedStillAbortsUndecryptable is the guard on the other
// side: a mid-call encryption naming a key id no entry covers still aborts.
func TestRecorderSkipEncryptedStillAbortsUndecryptable(t *testing.T) {
	r, bus, dir := mkSkipEncKeyedRecorder(t)
	defer r.Close()
	defer bus.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant:        trunking.Grant{System: "S", Protocol: "p25", GroupID: 705},
		Talkgroup:    &trunking.TalkGroup{ID: 705, Record: true},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Now(),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	waitSession(t, r, "VOICE-1", true)
	if err := r.WritePCM("VOICE-1", make([]int16, 800)); err != nil {
		t.Fatal(err)
	}

	bus.Publish(events.Event{Kind: events.KindCallEncryption, Payload: trunking.CallEncryption{
		DeviceSerial: "VOICE-1", AlgorithmID: 0x84, KeyID: 7, At: time.Now(),
	}})
	waitSession(t, r, "VOICE-1", false)
	assertNoWav(t, dir)
}

// TestRecorderSkipEncryptedRecordsDecryptableGrant: a grant that already
// flags encryption at CallStart opens a session when a configured key covers
// it, and is still dropped when none does.
func TestRecorderSkipEncryptedRecordsDecryptableGrant(t *testing.T) {
	r, bus, _ := mkSkipEncKeyedRecorder(t)
	defer r.Close()
	defer bus.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	start := func(serial string, keyID uint16) {
		bus.Publish(events.Event{Kind: events.KindCallStart, Payload: trunking.CallStart{
			Grant: trunking.Grant{
				System: "S", Protocol: "p25", GroupID: 706,
				Encrypted: true, AlgorithmID: 0xAA, KeyID: keyID,
			},
			Talkgroup:    &trunking.TalkGroup{ID: 706, Record: true},
			DeviceSerial: serial,
			StartedAt:    time.Now(),
		}})
	}
	start("KEYED", 3)
	waitSession(t, r, "KEYED", true)

	start("UNKEYED", 9)
	time.Sleep(100 * time.Millisecond)
	if r.HasSession("UNKEYED") {
		t.Fatal("skip_encrypted opened a session for an undecryptable call")
	}
}
