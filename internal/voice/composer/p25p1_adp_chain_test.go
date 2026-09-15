package composer

import (
	"context"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"github.com/MattCheramie/GopherTrunk/internal/voice/imbe"
)

// p25InfoBitsFromFrame is the inverse of imbe.PackInfoBitsToFrame.
func p25InfoBitsFromFrame(frame []byte) []byte {
	info := make([]byte, imbe.InfoBits)
	for i := range info {
		info[i] = (frame[i/8] >> (7 - uint(i%8))) & 1
	}
	return info
}

// buildP25P1ADPStream emits a clock-settling lead-in followed by
// `superframes` LDU1+LDU2 pairs whose IMBE frames are scrambled exactly as
// a radio transmitting ADP does — phase1's own keystream layout under an
// LFSR-chained Message Indicator schedule, each LDU2's Encryption Sync
// announcing the NEXT superframe's MI (the capture-pinned convention). want
// holds the CLEAR packed frames in transmission order.
func buildP25P1ADPStream(t *testing.T, key []byte, keyID uint16, mi0 [9]byte, superframes int) (dibits []uint8, want [][]byte) {
	t.Helper()
	// A longer lead-in than the clear-voice builders use, so the receiver
	// has locked before the FIRST LDU1 — that LDU1 is the one the chain
	// must hold until its superframe's ES arrives, and losing it to
	// acquisition would skip the path under test.
	dibits = make([]uint8, 2400)
	for i := range dibits {
		dibits[i] = uint8(i % 4)
	}
	seed := 0
	mi := mi0
	for k := 0; k < superframes; k++ {
		ks, err := phase1.ADPSuperframeKeystream(key, mi)
		if err != nil {
			t.Fatal(err)
		}
		next := phase1.AdvanceMI(mi)
		for _, duid := range []phase1.DUID{phase1.DUIDLogicalLink1, phase1.DUIDLogicalLink2} {
			var clear [phase1.LDUVoiceSubframeCount][]byte
			for s := range clear {
				f, err := imbe.PackInfoBitsToFrame(p25p1VoiceInfo(seed))
				if err != nil {
					t.Fatal(err)
				}
				seed++
				clear[s] = f
				want = append(want, append([]byte(nil), f...))
			}
			scrambled := clear
			for s := range scrambled {
				scrambled[s] = append([]byte(nil), clear[s]...)
			}
			if _, err := phase1.ADPDescrambleVoiceFrames(ks, duid, &scrambled); err != nil {
				t.Fatal(err)
			}
			var voice [phase1.LDUVoiceSubframeCount][]byte
			for s := range voice {
				onAir, err := imbe.EncodeFrameToChannel(p25InfoBitsFromFrame(scrambled[s]))
				if err != nil {
					t.Fatalf("EncodeFrameToChannel: %v", err)
				}
				voice[s] = onAir
			}
			var lces [phase1.LDULCESBlockCount][]byte
			if duid == phase1.DUIDLogicalLink2 {
				lces = phase1.AssembleEncryptionSync(phase1.EncryptionSync{
					MessageIndicator: next, AlgorithmID: p25.AlgorithmADP, KeyID: keyID,
				})
			}
			var lsd [phase1.LDULSDBlockCount][]byte
			ldu, err := phase1.AssembleLDU(0x123, duid, voice, lces, lsd)
			if err != nil {
				t.Fatalf("AssembleLDU: %v", err)
			}
			dibits = append(dibits, framing.BitsToDibits(ldu)...)
		}
		mi = next
	}
	return dibits, want
}

// runP25ADPChain drives one Phase 1 voice chain over iq with the given key
// resolver and returns the recorded raw frames once at least minFrames
// have landed (or the deadline passes).
func runP25ADPChain(t *testing.T, iq []complex64, resolver KeyResolver, minFrames int) [][]byte {
	t.Helper()
	const sampleRate = 48_000.0
	src := newFakeSource()
	bus := events.NewBus(8)
	sink := &recordingSink{}
	eng := &fakeEngine{}
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{"VOICE-ADP": src}},
		Sink:          sink,
		Engine:        eng,
		IQSampleRate:  uint32(sampleRate),
		PCMSampleRate: 8000,
		TouchInterval: 30 * time.Millisecond,
		KeyResolver:   resolver,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)
	defer c.Close()
	defer bus.Close()

	bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: trunking.CallStart{
			Grant: trunking.Grant{
				System: "P25ADPSite", Protocol: "p25",
				GroupID: 42, FrequencyHz: 851_000_000, Encrypted: true,
			},
			DeviceSerial: "VOICE-ADP",
			StartedAt:    time.Now().UTC(),
		},
	})
	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) && len(sink.rawFrames("VOICE-ADP")) < minFrames {
		time.Sleep(20 * time.Millisecond)
	}
	return sink.rawFrames("VOICE-ADP")
}

// TestComposerP25Phase1ADPDecryptsWithConfiguredKey is the failing-first
// regression for in-process P25 ADP (issue #1187): with a key configured for
// the ES's key ID, the frames the recorder receives are the CLEAR ones —
// including the first superframe, whose MI is only learnable by rewinding
// the first LDU2's ES (the chain holds those frames until then). Without a
// resolver the recorder gets the ciphertext, byte-for-byte as before.
func TestComposerP25Phase1ADPDecryptsWithConfiguredKey(t *testing.T) {
	const (
		sampleRate  = 48_000.0
		deviation   = 1800.0
		superframes = 5
		keyID       = 3
	)
	key := []byte{0x12, 0x34, 0x56, 0x78, 0x90}
	mi0 := [9]byte{0xC5, 0x25, 0xBC, 0xC7, 0xF7, 0x2B, 0x67, 0xE9, 0x00}
	dibits, clear := buildP25P1ADPStream(t, key, keyID, mi0, superframes)
	iq := demod.ModulateP25C4FM(dibits, sampleRate, deviation)
	framesPerSF := 2 * phase1.LDUVoiceSubframeCount

	resolver := func(system, algorithm string, kid uint16) ([]byte, bool) {
		if system == "P25ADPSite" && algorithm == "rc4" && kid == keyID {
			return key, true
		}
		return nil, false
	}
	// The assembler needs the sync of the FOLLOWING LDU to close one, so
	// the stream's last LDU is never delivered; ask for all but one
	// superframe's worth.
	minFrames := (superframes-2)*framesPerSF + phase1.LDUVoiceSubframeCount
	got := runP25ADPChain(t, iq, resolver, minFrames)
	if len(got) < minFrames {
		t.Fatalf("recorded %d frames, want ≥ %d", len(got), minFrames)
	}
	matches := bestAlignmentMatches(got, clear)
	// A decrypted stream matches the clear frames wholesale; ciphertext
	// matches none. The receiver's first LDU right after lock carries a
	// few FEC-partial frames (the extractor still returns them), so allow
	// a handful of misses.
	if matches < len(got)-5 {
		t.Fatalf("only %d of %d recorded frames match the clear frames — ADP descramble not applied to every superframe", matches, len(got))
	}
	// The chain must have started on an LDU1 — the superframe whose MI is
	// only learnable by rewinding the ES of the LDU2 that follows it, and
	// whose frames are therefore held — and that superframe must be clear
	// in the recording. Locate the first recorded frame among the clear
	// ones: it has to be the first frame of some LDU1.
	firstIdx := -1
	for i, w := range clear {
		if string(w) == string(got[0]) {
			firstIdx = i
			break
		}
	}
	if firstIdx < 0 || firstIdx%framesPerSF != 0 {
		t.Fatalf("first recorded frame is clear frame %d (want the u0 of an LDU1, a multiple of %d) — the held/rewound first superframe was not decrypted", firstIdx, framesPerSF)
	}

	// No resolver: ciphertext is recorded exactly as before.
	raw := runP25ADPChain(t, iq, nil, minFrames)
	if m := bestAlignmentMatches(raw, clear); m >= phase1.LDUVoiceSubframeCount {
		t.Fatalf("without a key %d frames match the clear frames — ciphertext should be recorded as-is", m)
	}
	// A resolver that has no key for this key ID leaves ciphertext too.
	none := runP25ADPChain(t, iq, func(string, string, uint16) ([]byte, bool) { return nil, false }, minFrames)
	if m := bestAlignmentMatches(none, clear); m >= phase1.LDUVoiceSubframeCount {
		t.Fatalf("with no matching key %d frames match the clear frames", m)
	}
}

// TestComposerP25Phase1ADPLeavesClearCallsAlone: a key resolver configured
// on a system must not change a clear call's recording — the first
// superframe is held only until its ES says the call is clear.
func TestComposerP25Phase1ADPLeavesClearCallsAlone(t *testing.T) {
	const (
		sampleRate = 48_000.0
		deviation  = 1800.0
		ldus       = 8
	)
	dibits, want := buildP25P1VoiceStream(t, ldus)
	iq := demod.ModulateP25C4FM(dibits, sampleRate, deviation)
	resolver := func(string, string, uint16) ([]byte, bool) { return []byte{1, 2, 3, 4, 5}, true }
	got := runP25ADPChain(t, iq, resolver, (ldus-2)*phase1.LDUVoiceSubframeCount)
	if m := bestAlignmentMatches(got, want); m < 3*phase1.LDUVoiceSubframeCount {
		t.Fatalf("clear call with a key resolver: only %d of %d frames match the modulated frames", m, len(got))
	}
}
