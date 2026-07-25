package composer

import (
	"context"
	"math"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// buildTETRATrafficDibits lays out nSlots Normal Continuous Downlink Bursts
// (255-dibit TDMA slots) into a dibit stream, each with the normal training
// sequence at its §9.4.4.3.2 intra-slot position (bit 244 → dibit 122). The
// data blocks are deterministic filler — the chain test asserts that bursts
// are recovered and drive the call lifecycle, not their payload. (Payload
// round-trips are covered by TrafficExtractor and TCHSpeechFrames unit tests;
// the CRC gate legitimately drops these filler bursts, so no decoded speech is
// asserted here.)
func buildTETRATrafficDibits(nSlots int) []uint8 {
	const (
		slot   = 255
		leadIn = 512 // receiver clock/AFC warm-up
		ntsAt  = 122 // dibit offset of the NTS within the slot
	)
	nts := tetra.NormalSyncDibits()
	stream := make([]uint8, leadIn+nSlots*slot+slot)
	for i := range stream {
		stream[i] = uint8((i*3 + 1) % 4) // non-sync filler
	}
	for s := 0; s < nSlots; s++ {
		L := leadIn + s*slot + ntsAt
		copy(stream[L:], nts)
	}
	return stream
}

// TestComposerTETRAVoiceChainLifecycle drives the composer's TETRA
// traffic-follow path end to end: modulated π/4-DQPSK IQ → TETRA receiver →
// TrafficExtractor → TCH/S decode, and confirms the chain starts and — because
// the filler bursts carry no valid TCH/S speech (the CRC gate rejects them) —
// the call is NOT kept alive by raw carrier activity and tears down via the
// no-voice startup timeout instead of hanging on the carrier forever. This is
// the regression guard for the "active call never ends" bug: before the fix,
// every recovered burst refreshed the hangtime timer, so a continuous carrier
// held the single voice device indefinitely.
//
// The speech-keeps-the-call-alive / ends-on-hangtime path is covered
// deterministically by TestTETRATrafficBurstLivenessGatedByCRC (a clean
// EncodeTCHS→TCHSpeechFrames round trip, no DSP), which avoids depending on the
// class-2 CRC surviving the full modulate→demod round trip under a loaded
// -race CI.
func TestComposerTETRAVoiceChainLifecycle(t *testing.T) {
	const (
		sps   = 8 // 18000 baud × 8 = 144 kHz intermediate rate
		span  = 8
		alpha = 0.35
		slots = 24
	)
	dibits := buildTETRATrafficDibits(slots)
	iq := demod.ModulatePiOver4DQPSK(dibits, sps, span, alpha, math.Pi/4)

	src := newFakeSource()
	bus := events.NewBus(8)
	sink := &recordingSink{}
	eng := &fakeEngine{}
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{"VOICE-1": src}},
		Sink:          sink,
		Engine:        eng,
		IQSampleRate:  uint32(sps * 18000), // 144 kHz tap
		PCMSampleRate: 8000,
		TouchInterval: 30 * time.Millisecond,
		// Generous hangtime: the no-voice teardown fires at 2×hangtime, and the
		// heavy TETRA receiver processing the whole IQ chunk must emit its first
		// burst before then — otherwise, under a saturated CI running the whole
		// suite under -race, the call would end as a no-voice Timeout instead of
		// decoding traffic. 2 s → 4 s no-voice window is ample.
		VoiceHangtime: 2 * time.Second,
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
				System: "TETRASite", Protocol: "tetra",
				GroupID: 1234, FrequencyHz: 390_000_000, Timeslot: 1,
			},
			DeviceSerial: "VOICE-1",
			StartedAt:    time.Now().UTC(),
		},
	})

	// Generous timeouts throughout: this test runs in the full `-race` suite on
	// shared CI, where the TETRA receiver DSP is CPU-heavy and slow to schedule.
	waitFor(t, 10*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)

	// The filler bursts carry no valid TCH/S speech, so no matching voice frame
	// ever decodes. The boundary tracker must therefore end the call via the
	// no-voice startup timeout (2×VoiceHangtime) — NOT keep it alive off raw
	// carrier activity. Reason timeout matches the "silent-from-start" teardown.
	waitFor(t, 15*time.Second, func() bool { return eng.endReason("VOICE-1") == trunking.EndReasonTimeout })
}

// TestTETRATrafficBurstLivenessGatedByCRC is the direct regression for the
// "active call never ends" bug. It exercises onTETRATrafficBurst without any DSP
// and asserts that call liveness is driven ONLY by CRC-valid TCH/S speech:
//
//   - a non-TCH/S (junk) burst must NOT touch the boundary tracker — no onVoice,
//     no sawVoice, no lastVoiceNano advance, no speech frame written; and
//   - a CRC-valid TCH/S burst (EncodeTCHS) must mark voice activity and emit the
//     two 137-bit speech frames.
//
// Before the fix, onVoice fired on every raw burst, so the junk case would set
// sawVoice / advance lastVoiceNano — and a continuous carrier of such bursts
// held the voice device forever. This test fails on that code and passes once
// liveness is gated on TCHSpeechFrames.
func TestTETRATrafficBurstLivenessGatedByCRC(t *testing.T) {
	c, _, _ := mkBoundaryComposer(t, false, 50*time.Millisecond)
	bt := c.newBoundaryTracker("VOICE-1", 0, nil)
	rs := &recordingSink{}
	var bursts, speech, offSlot atomic.Uint64

	// Case A: a deterministic non-TCH/S 54-byte burst. Guard that the CRC gate
	// rejects it (it does for this fixed vector), then confirm it leaves the
	// call-liveness state completely untouched. slot 0 = unanchored (the
	// single-call fallback), grantSlot 0 = no per-slot filter.
	junk := make([]byte, tetra.TrafficFrameBytes)
	rng := rand.New(rand.NewSource(42))
	rng.Read(junk)
	if tetra.TCHSpeechFrames(junk) != nil {
		t.Fatalf("test vector precondition: junk frame unexpectedly passed the TCH/S CRC gate")
	}
	c.onTETRATrafficBurst(bt, rs, "VOICE-1", junk, 0, 0, &bursts, &speech, &offSlot)
	if bt.sawVoice.Load() {
		t.Error("non-speech burst set sawVoice — liveness not gated on CRC-valid speech")
	}
	if got := bt.lastVoiceNano.Load(); got != 0 {
		t.Errorf("non-speech burst advanced lastVoiceNano to %d, want 0", got)
	}
	if n := len(rs.rawFrames("VOICE-1")); n != 0 {
		t.Errorf("non-speech burst wrote %d raw frames, want 0", n)
	}
	if got := speech.Load(); got != 0 {
		t.Errorf("non-speech burst counted %d speech frames, want 0", got)
	}
	if got := bursts.Load(); got != 1 {
		t.Errorf("bursts = %d after one call, want 1", got)
	}

	// Case B: a CRC-valid TCH/S burst carrying two 137-bit speech frames. It must
	// mark voice activity and emit both speech frames to the recorder.
	frame := tetra.EncodeTCHS(make([]byte, 137), make([]byte, 137))
	c.onTETRATrafficBurst(bt, rs, "VOICE-1", frame, 0, 0, &bursts, &speech, &offSlot)
	if !bt.sawVoice.Load() {
		t.Error("CRC-valid TCH/S burst did not set sawVoice")
	}
	if bt.lastVoiceNano.Load() == 0 {
		t.Error("CRC-valid TCH/S burst did not advance lastVoiceNano")
	}
	if n := len(rs.rawFrames("VOICE-1")); n != 2 {
		t.Errorf("CRC-valid burst wrote %d raw frames, want 2", n)
	}
	if got := speech.Load(); got != 2 {
		t.Errorf("speech frames = %d, want 2", got)
	}
	if got := bursts.Load(); got != 2 {
		t.Errorf("bursts = %d after two calls, want 2", got)
	}
}

// TestTETRATrafficBurstUsageMarkerFilter is the regression for concurrent
// same-carrier calls: a burst whose AACH downlink usage marker differs from the
// chain's granted usage marker belongs to another call sharing the carrier and
// must be dropped (counted off-call) and never decoded — even if it is a
// CRC-valid TCH/S frame for the OTHER call. A burst carrying the granted marker
// must be decoded; a burst whose AACH did not decode (marker 0) must fall through
// to the CRC gate rather than being dropped, so an occasional AACH miss never
// drops the granted call's own speech.
func TestTETRATrafficBurstUsageMarkerFilter(t *testing.T) {
	c, _, _ := mkBoundaryComposer(t, false, 50*time.Millisecond)
	bt := c.newBoundaryTracker("VOICE-2", 0, nil)
	rs := &recordingSink{}
	var bursts, speech, offSlot atomic.Uint64

	// A CRC-valid TCH/S frame (belongs to whichever call it arrives on).
	frame := tetra.EncodeTCHS(make([]byte, 137), make([]byte, 137))
	const grantUsage uint8 = 20 // this call's downlink usage marker

	// Another call's slot (marker 19 != granted 20): dropped off-call, no decode.
	c.onTETRATrafficBurst(bt, rs, "VOICE-2", frame, 19, grantUsage, &bursts, &speech, &offSlot)
	if got := offSlot.Load(); got != 1 {
		t.Errorf("foreign-call burst offSlot = %d, want 1", got)
	}
	if n := len(rs.rawFrames("VOICE-2")); n != 0 {
		t.Errorf("foreign-call burst wrote %d frames, want 0 (must not decode another call's slot)", n)
	}
	if bt.sawVoice.Load() {
		t.Error("foreign-call burst set sawVoice — a concurrent call would keep this call alive")
	}

	// Granted marker (20 == granted 20): decoded and written.
	c.onTETRATrafficBurst(bt, rs, "VOICE-2", frame, grantUsage, grantUsage, &bursts, &speech, &offSlot)
	if n := len(rs.rawFrames("VOICE-2")); n != 2 {
		t.Errorf("granted-marker burst wrote %d frames, want 2", n)
	}
	if !bt.sawVoice.Load() {
		t.Error("granted-marker burst did not mark voice activity")
	}

	// Undecoded AACH (marker 0): CRC-gated fallback still processes it — an AACH
	// miss must not drop the granted call's own speech.
	c.onTETRATrafficBurst(bt, rs, "VOICE-2", frame, 0, grantUsage, &bursts, &speech, &offSlot)
	if n := len(rs.rawFrames("VOICE-2")); n != 4 {
		t.Errorf("undecoded-AACH burst wrote total %d frames, want 4", n)
	}
	if got := offSlot.Load(); got != 1 {
		t.Errorf("offSlot = %d after one foreign + one granted + one undecoded-AACH, want 1", got)
	}
}

// TestTETRATrafficBurstNoGrantMarkerFallback pins the safety net: when the grant
// carries no usage marker (0 — addressed by plain SSI), every CRC-valid burst is
// accepted regardless of the burst's own marker, so a call's speech is never
// silently discarded on a guess (the pre-demux single-call behaviour).
func TestTETRATrafficBurstNoGrantMarkerFallback(t *testing.T) {
	c, _, _ := mkBoundaryComposer(t, false, 50*time.Millisecond)
	bt := c.newBoundaryTracker("VOICE-3", 0, nil)
	rs := &recordingSink{}
	var bursts, speech, offSlot atomic.Uint64
	frame := tetra.EncodeTCHS(make([]byte, 137), make([]byte, 137))

	// Grant marker 0 (unknown): a burst with any marker is decoded, none dropped.
	c.onTETRATrafficBurst(bt, rs, "VOICE-3", frame, 19, 0, &bursts, &speech, &offSlot)
	c.onTETRATrafficBurst(bt, rs, "VOICE-3", frame, 0, 0, &bursts, &speech, &offSlot)
	if n := len(rs.rawFrames("VOICE-3")); n != 4 {
		t.Errorf("no-grant-marker fallback wrote %d frames, want 4 (accept all CRC-valid speech)", n)
	}
	if got := offSlot.Load(); got != 0 {
		t.Errorf("offSlot = %d, want 0 (no dropping without a grant marker)", got)
	}
}
