package composer

import (
	"io"
	"log/slog"
	"reflect"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
)

// fakeDMOSink captures the raw ACELP speech frames the DMO voice decoder writes.
type fakeDMOSink struct{ frames [][]byte }

func (f *fakeDMOSink) WritePCM(string, []int16) error { return nil }
func (f *fakeDMOSink) WriteRawFrame(_ string, frame []byte) error {
	f.frames = append(f.frames, append([]byte(nil), frame...))
	return nil
}

func dmoVFiller(seed, n int) []uint8 {
	s := make([]uint8, n)
	for i := range s {
		s[i] = uint8((seed + i*3) & 3)
	}
	return s
}

// dmoVSlotDibits is one TETRA timeslot in dibits. The voice decoder now
// qualifies DNBs on the 255-dibit slot grid (tetra.DMSlotGrid) before scoring
// them for colour recovery, so a synthetic "real transmission" MUST lay its
// bursts one per slot — as a real radio transmits them — or nothing qualifies.
const dmoVSlotDibits = 255

// dmoVSlot places burst fields at a fixed offset inside one 255-dibit slot.
// A DNB (108 + 11 + 108 = 227 dibits) at offset 7 ends at 234, inside the slot.
func dmoVSlot(seed int, fields ...[]uint8) []uint8 {
	slot := dmoVFiller(seed, dmoVSlotDibits)
	at := 7
	for _, f := range fields {
		copy(slot[at:], f)
		at += len(f)
	}
	if at > dmoVSlotDibits {
		panic("dmoVSlot: fields overflow the slot")
	}
	return slot
}

func dmoVSeq(seed, n int) []byte {
	s := make([]byte, n)
	for i := range s {
		s[i] = byte((seed + i*5) % 2)
	}
	return s
}

func dmoVIdealDiffs(dibits []uint8) []complex64 {
	bits := tetra.TetraDibitsToBits(dibits)
	amp := func(b byte) float32 {
		if b&1 == 0 {
			return 1
		}
		return -1
	}
	diffs := make([]complex64, len(dibits))
	for i := range dibits {
		diffs[i] = complex(amp(bits[2*i+1]), amp(bits[2*i]))
	}
	return diffs
}

// buildDMODNBBursts builds n TCH/S DNBs scrambled with colour and returns them as real
// tetra.DMBursts (with soft info, via ExtractDMBurstsSoft), plus the encoded speech
// frames per DNB.
func buildDMODNBBursts(t *testing.T, colour uint32, n int) ([]tetra.DMBurst, [][2][]byte) {
	t.Helper()
	b2d := tetra.TetraBitsToDibits
	var dibits []uint8
	var want [][2][]byte
	dibits = append(dibits, dmoVFiller(0, dmoVSlotDibits)...)
	for i := 0; i < n; i++ {
		fa := dmoVSeq(7+i, 137)
		fb := dmoVSeq(9+i, 137)
		t4 := framing.UnpackBitsMSB(tetra.EncodeTCHS(fa, fb), 432)
		onair := framing.ScrambleTetra(t4, colour)
		dibits = append(dibits, dmoVSlot(11+i,
			b2d(onair[:216]),
			tetra.NormalSyncDibits(),
			b2d(onair[216:]),
		)...)
		want = append(want, [2][]byte{framing.PackBitsMSB(fa), framing.PackBitsMSB(fb)})
	}
	dibits = append(dibits, dmoVFiller(99, dmoVSlotDibits)...)

	all := tetra.ExtractDMBurstsSoft(dibits, dmoVIdealDiffs(dibits), 0)
	var dnbs []tetra.DMBurst
	for _, b := range all {
		if b.Kind == tetra.DMBurstNormal {
			dnbs = append(dnbs, b)
		}
	}
	if len(dnbs) != n {
		t.Fatalf("built %d DNB bursts, want %d", len(dnbs), n)
	}
	return dnbs, want
}

// TestDMOVoiceDecoderRecoversColourAndEmits pins the voice chain's core logic: when the
// grant carries no colour (hint 0), the decoder buffers DNBs, brute-force-recovers the
// DM colour code, decodes the buffered DNBs retroactively (so no leading speech is
// lost), and emits every call's two 137-bit speech frames to the recorder sink — the
// same frames that were encoded. This is the composer half of the #1003 DMO voice path
// (the receiver→extractor half is covered by the pipeline test); on-air A/B still gates
// closing #1003.
func TestDMOVoiceDecoderRecoversColourAndEmits(t *testing.T) {
	const colour = 3
	bursts, want := buildDMODNBBursts(t, colour, 28)

	sink := &fakeDMOSink{}
	c := &Composer{log: slog.New(slog.NewTextHandler(io.Discard, nil)), hangtime: time.Second}
	bt := c.newBoundaryTracker("s", 0, nil)
	dec := &dmoVoiceDecoder{c: c, serial: "s", bt: bt, rs: sink, grid: tetra.NewDMSlotGrid()}

	for _, b := range bursts {
		dec.onBurst(b)
	}
	dec.flush()

	if !dec.colourKnown || dec.colour != colour {
		t.Fatalf("recovered colour=%d known=%v, want %d", dec.colour, dec.colourKnown, colour)
	}
	if len(sink.frames) != 2*len(want) {
		t.Fatalf("emitted %d speech frames, want %d", len(sink.frames), 2*len(want))
	}
	for i, w := range want {
		if !reflect.DeepEqual(sink.frames[2*i], w[0]) || !reflect.DeepEqual(sink.frames[2*i+1], w[1]) {
			t.Errorf("DNB %d: emitted speech frame mismatch", i)
		}
	}
	if dec.dnb.Load() != uint64(len(bursts)) {
		t.Errorf("counted %d DNBs, want %d", dec.dnb.Load(), len(bursts))
	}
}

// TestDMOVoiceDecoderUsesGrantColour checks the fast path: when the grant already
// carries the recovered colour, the decoder uses it directly with no buffering — every
// DNB decodes immediately.
func TestDMOVoiceDecoderUsesGrantColour(t *testing.T) {
	const colour = 3
	bursts, want := buildDMODNBBursts(t, colour, 4)
	sink := &fakeDMOSink{}
	c := &Composer{log: slog.New(slog.NewTextHandler(io.Discard, nil)), hangtime: time.Second}
	dec := &dmoVoiceDecoder{
		c: c, serial: "s", bt: c.newBoundaryTracker("s", 0, nil), rs: sink,
		grid:   tetra.NewDMSlotGrid(),
		colour: colour, colourKnown: true, colourRecovered: true,
	}
	for _, b := range bursts {
		dec.onBurst(b)
	}
	if len(dec.buffer) != 0 {
		t.Errorf("decoder buffered %d bursts despite a known colour", len(dec.buffer))
	}
	if len(sink.frames) != 2*len(want) {
		t.Fatalf("emitted %d speech frames, want %d", len(sink.frames), 2*len(want))
	}
}

// buildDMOUndecodableDNBs builds n well-formed DNBs (correct geometry and training
// sequence, so the extractor finds them) carrying payload that is not a valid TCH/S
// codeword at any colour — an encrypted call, or a transmission too weak to decode.
// Every colour then sits at the ~1/256 chance floor, so RecoverDMColourCode's
// confidence gate correctly never clears: the path that used to burn ~450k Viterbi
// decodes per call.
func buildDMOUndecodableDNBs(t *testing.T, n int) []tetra.DMBurst {
	t.Helper()
	var dibits []uint8
	dibits = append(dibits, dmoVFiller(0, dmoVSlotDibits)...)
	for i := 0; i < n; i++ {
		dibits = append(dibits, dmoVSlot(11+i,
			dmoVFiller(200+i*7, 108), // BKN1: not a TCH/S codeword
			tetra.NormalSyncDibits(),
			dmoVFiller(500+i*13, 108), // BKN2
		)...)
	}
	dibits = append(dibits, dmoVFiller(99, dmoVSlotDibits)...)

	var dnbs []tetra.DMBurst
	for _, b := range tetra.ExtractDMBurstsSoft(dibits, dmoVIdealDiffs(dibits), 0) {
		if b.Kind == tetra.DMBurstNormal {
			dnbs = append(dnbs, b)
		}
	}
	if len(dnbs) < n {
		t.Fatalf("built %d DNB bursts, want at least %d", len(dnbs), n)
	}
	return dnbs
}

// buildDMOGoodWithNoise builds ONE dibit stream carrying nGood slot-aligned
// colour-scrambled TCH/S DNBs interleaved with ~3 correlator-noise DNBs each
// (well-formed training sequences at varying NON-slot-aligned offsets inside
// whole filler slots, so the good bursts keep one residue and the noise does
// not). This is what a live tap actually feeds the voice chain: the DNB
// correlator false-alarms ~18/s against ~17/s of real traffic. Returns all
// extracted DNBs in stream order plus the encoded speech frames per good DNB.
func buildDMOGoodWithNoise(t *testing.T, colour uint32, nGood int) ([]tetra.DMBurst, [][2][]byte) {
	t.Helper()
	b2d := tetra.TetraBitsToDibits
	var dibits []uint8
	var want [][2][]byte
	dibits = append(dibits, dmoVFiller(0, dmoVSlotDibits)...)
	for i := 0; i < nGood; i++ {
		fa := dmoVSeq(7+i, 137)
		fb := dmoVSeq(9+i, 137)
		t4 := framing.UnpackBitsMSB(tetra.EncodeTCHS(fa, fb), 432)
		onair := framing.ScrambleTetra(t4, colour)
		dibits = append(dibits, dmoVSlot(11+i,
			b2d(onair[:216]),
			tetra.NormalSyncDibits(),
			b2d(onair[216:]),
		)...)
		want = append(want, [2][]byte{framing.PackBitsMSB(fa), framing.PackBitsMSB(fb)})
		// Four whole slots of filler carrying three bogus DNB training
		// sequences at drifting offsets. The good DNB's training lead sits at
		// slot offset 7+108=115; the noise offsets (see below) never land on
		// 115±1 mod 255, so they cannot vote for the traffic residue.
		noise := dmoVFiller(300+i*17, 4*dmoVSlotDibits)
		for j := 0; j < 3; j++ {
			off := 130 + ((i*3+j)*37)%520 // lead at off+108; varies, ≢ 115 (mod 255)
			if lead := (off + 108) % dmoVSlotDibits; lead >= 114 && lead <= 116 {
				off += 5
			}
			copy(noise[off+108:], tetra.NormalSyncDibits())
		}
		dibits = append(dibits, noise...)
	}
	dibits = append(dibits, dmoVFiller(99, dmoVSlotDibits)...)

	var dnbs []tetra.DMBurst
	for _, b := range tetra.ExtractDMBurstsSoft(dibits, dmoVIdealDiffs(dibits), 0) {
		if b.Kind == tetra.DMBurstNormal {
			dnbs = append(dnbs, b)
		}
	}
	if len(dnbs) < 3*nGood {
		t.Fatalf("built %d DNB bursts, want ≥ %d (good + noise)", len(dnbs), 3*nGood)
	}
	return dnbs, want
}

// TestDMOVoiceDecoderAdoptsLiveColourHint pins the pipeline→voice-chain colour
// hand-off: a DMO grant structurally fires before the control pipeline finishes
// its own colour recovery, so the chain polls liveColour and must adopt a
// verified answer with far fewer bursts than the local brute force needs. Only
// 12 bursts are fed — below dmoVoiceColourBatch — so with no hint plumbing the
// brute force could not have fired mid-stream at all; adoption must land with
// zero local attempts and every burst's speech emitted. Fails against the old
// code (no liveColour parameter existed).
func TestDMOVoiceDecoderAdoptsLiveColourHint(t *testing.T) {
	const colour = 44
	bursts, want := buildDMODNBBursts(t, colour, 12)

	sink := &fakeDMOSink{}
	c := &Composer{log: slog.New(slog.NewTextHandler(io.Discard, nil)), hangtime: time.Second}
	calls := 0
	dec := &dmoVoiceDecoder{
		c: c, serial: "s", bt: c.newBoundaryTracker("s", 0, nil), rs: sink,
		grid: tetra.NewDMSlotGrid(),
		liveColour: func() (uint32, bool) {
			// The pipeline's recovery lands a few bursts into the call.
			calls++
			if calls > 3 {
				return colour, true
			}
			return 0, false
		},
	}
	for _, b := range bursts {
		dec.onBurst(b)
	}
	dec.flush()

	if !dec.colourRecovered || dec.colour != colour {
		t.Fatalf("adopted colour=%d recovered=%v, want %d via the live hint",
			dec.colour, dec.colourRecovered, colour)
	}
	if dec.colourTries != 0 {
		t.Errorf("ran %d local brute-force attempts despite a valid live hint, want 0", dec.colourTries)
	}
	if len(sink.frames) != 2*len(want) {
		t.Fatalf("emitted %d speech frames, want %d (buffered speech lost?)",
			len(sink.frames), 2*len(want))
	}
}

// TestDMOVoiceDecoderRejectsBadLiveHint is the companion: a wrong/stale hint
// (different DMO net, cleared-too-late atomic) must NOT blindly latch — it
// fails CRC verification against the buffered bursts and local recovery still
// lands the true colour.
func TestDMOVoiceDecoderRejectsBadLiveHint(t *testing.T) {
	const colour = 3
	bursts, want := buildDMODNBBursts(t, colour, 30)

	sink := &fakeDMOSink{}
	c := &Composer{log: slog.New(slog.NewTextHandler(io.Discard, nil)), hangtime: time.Second}
	dec := &dmoVoiceDecoder{
		c: c, serial: "s", bt: c.newBoundaryTracker("s", 0, nil), rs: sink,
		grid:       tetra.NewDMSlotGrid(),
		liveColour: func() (uint32, bool) { return 17, true }, // wrong colour
	}
	for _, b := range bursts {
		dec.onBurst(b)
	}
	dec.flush()

	if dec.colour != colour || !dec.colourRecovered {
		t.Fatalf("colour=%d recovered=%v, want %d recovered locally despite the bad hint",
			dec.colour, dec.colourRecovered, colour)
	}
	if dec.colourTries == 0 {
		t.Errorf("local recovery never ran — the bad hint was adopted?")
	}
	if len(sink.frames) != 2*len(want) {
		t.Fatalf("emitted %d speech frames, want %d", len(sink.frames), 2*len(want))
	}
}

// TestDMOVoiceDecoderRecoversColourThroughNoise reproduces the on-air failure
// that left every real DMO call at colour_attempts=6, colour_recovered=false,
// all-BFI: on a live tap the DNB correlator's ~18/s false alarms outnumber real
// traffic, and the old un-gated colour scoring windows were mostly noise, so
// RecoverDMColourCode's dominance gate could never clear and the attempt budget
// burned out before enough real bursts arrived. With slot-grid gating, only the
// grid-qualified (real) bursts are scored, recovery lands, and every real
// burst's speech is emitted. Fails against the old un-gated code.
func TestDMOVoiceDecoderRecoversColourThroughNoise(t *testing.T) {
	const colour = 3
	bursts, want := buildDMOGoodWithNoise(t, colour, 30)

	sink := &fakeDMOSink{}
	c := &Composer{log: slog.New(slog.NewTextHandler(io.Discard, nil)), hangtime: time.Second}
	dec := &dmoVoiceDecoder{c: c, serial: "s", bt: c.newBoundaryTracker("s", 0, nil), rs: sink, grid: tetra.NewDMSlotGrid()}

	for _, b := range bursts {
		dec.onBurst(b)
	}
	dec.flush()

	if !dec.colourRecovered || dec.colour != colour {
		t.Fatalf("colour=%d recovered=%v (attempts=%d), want %d recovered through the noise",
			dec.colour, dec.colourRecovered, dec.colourTries, colour)
	}
	// Every GOOD burst's two speech frames must be present, in order; the noise
	// bursts BFI at the recovered colour and contribute nothing.
	if len(sink.frames) != 2*len(want) {
		t.Fatalf("emitted %d speech frames, want %d", len(sink.frames), 2*len(want))
	}
	for i, w := range want {
		if !reflect.DeepEqual(sink.frames[2*i], w[0]) || !reflect.DeepEqual(sink.frames[2*i+1], w[1]) {
			t.Errorf("good DNB %d: emitted speech frame mismatch", i)
		}
	}
}

// TestDMOVoiceDecoderBoundsColourRecovery pins the cost bound on the 64-colour brute
// force. RecoverDMColourCode is a full soft-Viterbi TCH/S decode per colour per burst
// (plus a hard decode on the failing fallback), and it used to be re-run on EVERY
// arriving burst from buffer size 20 to 120 — 64·Σ(20..120) ≈ 450 000 decodes per
// call, on exactly the calls that cannot be recovered anyway. That is what starved the
// same-carrier IQ tap feeding this chain. It must now run at most
// dmoVoiceColourMaxAttempts times, scoring a bounded window.
func TestDMOVoiceDecoderBoundsColourRecovery(t *testing.T) {
	bursts := buildDMOUndecodableDNBs(t, 200)
	sink := &fakeDMOSink{}
	c := &Composer{log: slog.New(slog.NewTextHandler(io.Discard, nil)), hangtime: time.Second}
	dec := &dmoVoiceDecoder{c: c, serial: "s", bt: c.newBoundaryTracker("s", 0, nil), rs: sink, grid: tetra.NewDMSlotGrid()}

	for _, b := range bursts {
		dec.onBurst(b)
	}
	dec.flush()

	if dec.colourTries > dmoVoiceColourMaxAttempts {
		t.Errorf("ran %d colour-recovery passes, want at most %d",
			dec.colourTries, dmoVoiceColourMaxAttempts)
	}
	if dec.colourTries == 0 {
		t.Errorf("never attempted colour recovery — the test is not exercising the path")
	}
	if len(sink.frames) != 0 {
		t.Errorf("emitted %d speech frames from undecodable bursts, want 0", len(sink.frames))
	}
}

// TestDMOVoiceDecoderDoesNotClaimUnrecoveredColour is the honesty regression for the
// end-of-call log. flush() used to set colourKnown = true OUTSIDE the confidence-gate
// branch, so a call where nothing was ever recovered reported `colour=0
// colour_known=true` — which reads as "recovered colour 0" and sent the investigation
// after the wrong thing. The fallback still has to pick a colour to decode at, so
// colourKnown is separate from colourRecovered.
func TestDMOVoiceDecoderDoesNotClaimUnrecoveredColour(t *testing.T) {
	bursts := buildDMOUndecodableDNBs(t, 60)
	sink := &fakeDMOSink{}
	c := &Composer{log: slog.New(slog.NewTextHandler(io.Discard, nil)), hangtime: time.Second}
	dec := &dmoVoiceDecoder{c: c, serial: "s", bt: c.newBoundaryTracker("s", 0, nil), rs: sink, grid: tetra.NewDMSlotGrid()}

	for _, b := range bursts {
		dec.onBurst(b)
	}
	dec.flush()

	if dec.colourRecovered {
		t.Errorf("reported colour %d as recovered when the confidence gate never cleared",
			dec.colour)
	}
	if !dec.colourKnown {
		t.Errorf("colourKnown = false after flush; the buffer would never be decoded")
	}
}

// TestDMOVoiceDecoderKeepsBufferedSpeechAcrossAttempts guards the interaction between
// the new attempt cap and the retroactive decode: the control pipeline decimates its
// candidate buffer between passes because it only wants the colour, but this chain
// must still emit every buffered burst once the colour lands, or the start of the
// transmission is missing from the recording. Recovery is deliberately delayed past
// the first attempt here by prefixing undecodable bursts.
func TestDMOVoiceDecoderKeepsBufferedSpeechAcrossAttempts(t *testing.T) {
	const colour = 3
	good, want := buildDMODNBBursts(t, colour, 24)
	bad := buildDMOUndecodableDNBs(t, dmoVoiceColourBatch)

	sink := &fakeDMOSink{}
	c := &Composer{log: slog.New(slog.NewTextHandler(io.Discard, nil)), hangtime: time.Second}
	dec := &dmoVoiceDecoder{c: c, serial: "s", bt: c.newBoundaryTracker("s", 0, nil), rs: sink, grid: tetra.NewDMSlotGrid()}

	for _, b := range append(append([]tetra.DMBurst{}, bad...), good...) {
		dec.onBurst(b)
	}
	dec.flush()

	if !dec.colourRecovered || dec.colour != colour {
		t.Fatalf("recovered colour=%d recovered=%v, want %d", dec.colour, dec.colourRecovered, colour)
	}
	// Every good burst's two speech frames must still be emitted, in order, even
	// though they were buffered across more than one recovery attempt.
	if len(sink.frames) != 2*len(want) {
		t.Fatalf("emitted %d speech frames, want %d (buffered speech was dropped)",
			len(sink.frames), 2*len(want))
	}
	for i, w := range want {
		if !reflect.DeepEqual(sink.frames[2*i], w[0]) || !reflect.DeepEqual(sink.frames[2*i+1], w[1]) {
			t.Errorf("DNB %d: emitted speech frame mismatch", i)
		}
	}
}
