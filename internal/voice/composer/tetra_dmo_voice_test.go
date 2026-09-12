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
		// Keep only the bursts on the transmitter's slot grid (lead at slot
		// offset 7+108): a scrambled payload can contain chance 11-dibit
		// training-sequence matches (the correlator's ~18/s false-alarm class),
		// which the live chain drops via DMSlotGrid and the noise test covers.
		if b.Kind == tetra.DMBurstNormal && b.Lead%dmoVSlotDibits == 115 {
			dnbs = append(dnbs, b)
		}
	}
	if len(dnbs) != n {
		t.Fatalf("built %d DNB bursts, want %d", len(dnbs), n)
	}
	return dnbs, want
}

// newTestDMODecoder builds a voice decoder with the production wiring minus the
// receiver: an empty seed tracker, a slot grid, a fake raw-frame sink.
func newTestDMODecoder(sink *fakeDMOSink, live func() (uint32, bool)) *dmoVoiceDecoder {
	c := &Composer{log: slog.New(slog.NewTextHandler(io.Discard, nil)), hangtime: time.Second}
	return &dmoVoiceDecoder{
		c: c, serial: "s", bt: c.newBoundaryTracker("s", 0, nil), rs: sink,
		seeds: tetra.NewDMSeedTracker(), grid: tetra.NewDMSlotGrid(), liveColour: live,
	}
}

// TestDMOVoiceDecoderRecoversSeedAndEmits pins the voice chain's core logic: when
// the grant carries no seed, the decoder buffers DNBs, learns the scramble seed
// from the first error-free burst (an exact GF(2) solve — no colour space, no
// configuration), decodes the buffered DNBs retroactively (so no leading speech
// is lost) and emits every burst's two 137-bit speech frames to the recorder —
// the same frames that were encoded. The seed is a real on-air value from the
// 12 Sep #1003 capture (0x012915c0), outside the 0..63 the old brute force
// searched: this test fails against the old code, which never recovered it.
func TestDMOVoiceDecoderRecoversSeedAndEmits(t *testing.T) {
	const seed = 0x012915c0
	bursts, want := buildDMODNBBursts(t, seed, 28)

	sink := &fakeDMOSink{}
	dec := newTestDMODecoder(sink, nil)
	for _, b := range bursts {
		dec.onBurst(b)
	}
	dec.flush()

	if got, known := dec.seeds.Seed(); !known || got != seed || !dec.seeds.Verified() {
		t.Fatalf("recovered seed=%#x known=%v verified=%v, want %#x", got, known, dec.seeds.Verified(), seed)
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

// TestDMOVoiceDecoderUsesGrantSeed checks the fast path: when the grant already
// carries the pipeline's seed, the decoder uses it directly with no buffering —
// every DNB decodes immediately.
func TestDMOVoiceDecoderUsesGrantSeed(t *testing.T) {
	const seed = 3
	bursts, want := buildDMODNBBursts(t, seed, 4)
	sink := &fakeDMOSink{}
	dec := newTestDMODecoder(sink, nil)
	dec.seeds.Adopt(seed)
	for _, b := range bursts {
		dec.onBurst(b)
	}
	if len(dec.buffer) != 0 {
		t.Errorf("decoder buffered %d bursts despite a known seed", len(dec.buffer))
	}
	if len(sink.frames) != 2*len(want) {
		t.Fatalf("emitted %d speech frames, want %d", len(sink.frames), 2*len(want))
	}
}

// TestDMOVoiceDecoderFollowsSeedChangeMidStream is the on-air behaviour of the
// 12 Sep capture: three back-to-back transmissions, each with its own seed, on
// one chain (the grant re-arm did not separate them). Every burst must decode at
// its own transmission's seed — the tracker switches on the first clean burst of
// the next PTT — with no garbage from a stale seed.
func TestDMOVoiceDecoderFollowsSeedChangeMidStream(t *testing.T) {
	var bursts []tetra.DMBurst
	var want [][2][]byte
	for k, seed := range []uint32{0x012915c0, 0x015d9c07, 0x01671384} {
		b, w := buildDMODNBBursts(t, seed, 10)
		for i := range b {
			b[i].Lead += k * 40 * dmoVSlotDibits // later transmissions sit later in the stream
		}
		bursts = append(bursts, b...)
		want = append(want, w...)
	}
	sink := &fakeDMOSink{}
	dec := newTestDMODecoder(sink, nil)
	for _, b := range bursts {
		dec.onBurst(b)
	}
	dec.flush()
	if len(sink.frames) != 2*len(want) {
		t.Fatalf("emitted %d speech frames, want %d", len(sink.frames), 2*len(want))
	}
	for i, w := range want {
		if !reflect.DeepEqual(sink.frames[2*i], w[0]) || !reflect.DeepEqual(sink.frames[2*i+1], w[1]) {
			t.Errorf("DNB %d: emitted speech frame mismatch", i)
		}
	}
	if dec.seeds.ExactAdopts != 3 {
		t.Errorf("exact seed adoptions = %d, want 3 (one per transmission)", dec.seeds.ExactAdopts)
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

// TestDMOVoiceDecoderConfirmsLiveHint pins the pipeline→voice-chain hand-off: a
// DMO grant structurally fires before the control pipeline has solved the seed, so
// the chain polls liveColour. On bursts that carry bit errors (no exact solve) the
// hint is what decodes them, once two of them CRC-confirm it; every burst's
// speech is then emitted from the buffer.
func TestDMOVoiceDecoderConfirmsLiveHint(t *testing.T) {
	const seed = 0x015d9c07
	bursts, want := buildDMODNBBurstsWithErrors(t, seed, 12, 3)

	sink := &fakeDMOSink{}
	calls := 0
	dec := newTestDMODecoder(sink, func() (uint32, bool) {
		// The pipeline's answer lands a few bursts into the call.
		calls++
		if calls > 3 {
			return seed, true
		}
		return 0, false
	})
	for _, b := range bursts {
		dec.onBurst(b)
	}
	dec.flush()

	if got, _ := dec.seeds.Seed(); got != seed || !dec.seeds.Verified() || dec.seeds.HintAdopts != 1 {
		t.Fatalf("seed=%#x verified=%v hint_adopts=%d, want %#x via the confirmed hint",
			got, dec.seeds.Verified(), dec.seeds.HintAdopts, seed)
	}
	if len(sink.frames) != 2*len(want) {
		t.Fatalf("emitted %d speech frames, want %d (buffered speech lost?)",
			len(sink.frames), 2*len(want))
	}
}

// TestDMOVoiceDecoderRejectsBadLiveHint is the companion: a wrong/stale hint
// (another radio's DSB, a cleared-too-late atomic) must NOT latch — it never CRC-
// confirms against the buffered bursts, and the exact solve lands the true seed.
func TestDMOVoiceDecoderRejectsBadLiveHint(t *testing.T) {
	const seed = 3
	bursts, want := buildDMODNBBursts(t, seed, 30)

	sink := &fakeDMOSink{}
	dec := newTestDMODecoder(sink, func() (uint32, bool) { return 17, true }) // wrong seed
	for _, b := range bursts {
		dec.onBurst(b)
	}
	dec.flush()

	if got, _ := dec.seeds.Seed(); got != seed || !dec.seeds.Verified() || dec.seeds.HintAdopts != 0 {
		t.Fatalf("seed=%#x verified=%v hint_adopts=%d, want %#x solved locally despite the bad hint",
			got, dec.seeds.Verified(), dec.seeds.HintAdopts, seed)
	}
	if len(sink.frames) != 2*len(want) {
		t.Fatalf("emitted %d speech frames, want %d", len(sink.frames), 2*len(want))
	}
}

// TestDMOVoiceDecoderRecoversSeedThroughNoise reproduces the live-tap condition:
// the DNB correlator's ~18/s false alarms outnumber real traffic. Only slot-grid-
// qualified bursts may teach the tracker, so the noise cannot steer it; the
// first clean real burst solves the seed and every real burst's speech is
// emitted in order, the noise bursts BFI.
func TestDMOVoiceDecoderRecoversSeedThroughNoise(t *testing.T) {
	const seed = 0x01671384
	bursts, want := buildDMOGoodWithNoise(t, seed, 30)

	sink := &fakeDMOSink{}
	dec := newTestDMODecoder(sink, nil)
	for _, b := range bursts {
		dec.onBurst(b)
	}
	dec.flush()

	if got, _ := dec.seeds.Seed(); got != seed || !dec.seeds.Verified() {
		t.Fatalf("seed=%#x verified=%v, want %#x recovered through the noise", got, dec.seeds.Verified(), seed)
	}
	if len(sink.frames) != 2*len(want) {
		t.Fatalf("emitted %d speech frames, want %d", len(sink.frames), 2*len(want))
	}
	for i, w := range want {
		if !reflect.DeepEqual(sink.frames[2*i], w[0]) || !reflect.DeepEqual(sink.frames[2*i+1], w[1]) {
			t.Errorf("good DNB %d: emitted speech frame mismatch", i)
		}
	}
}

// TestDMOVoiceDecoderDoesNotClaimUnrecoveredSeed is the honesty regression for the
// end-of-call log: undecodable bursts (an encrypted call) must never produce a
// "verified" seed — the give-up fallback decodes at a guess, emits nothing, and
// reports seed_verified=false so a fallback is not mistaken for a recovery.
func TestDMOVoiceDecoderDoesNotClaimUnrecoveredSeed(t *testing.T) {
	bursts := buildDMOUndecodableDNBs(t, 200)
	sink := &fakeDMOSink{}
	dec := newTestDMODecoder(sink, nil)
	for _, b := range bursts {
		dec.onBurst(b)
	}
	dec.flush()

	if dec.seeds.Verified() {
		s, _ := dec.seeds.Seed()
		t.Errorf("reported seed %#x as verified on undecodable bursts", s)
	}
	if _, known := dec.seeds.Seed(); !known {
		t.Errorf("no fallback seed after flush; the buffer would never be decoded")
	}
	if len(dec.buffer) != 0 {
		t.Errorf("%d bursts still buffered past the cap", len(dec.buffer))
	}
	if len(sink.frames) != 0 {
		t.Errorf("emitted %d speech frames from undecodable bursts, want 0", len(sink.frames))
	}
}

// buildDMODNBBurstsWithErrors is buildDMODNBBursts with flips random coded bits
// corrupted in every burst, so no burst solves exactly and only a hinted seed
// (confirmed by the soft decode's CRC) can decode them.
func buildDMODNBBurstsWithErrors(t *testing.T, seed uint32, n, flips int) ([]tetra.DMBurst, [][2][]byte) {
	t.Helper()
	b2d := tetra.TetraBitsToDibits
	var dibits []uint8
	var want [][2][]byte
	dibits = append(dibits, dmoVFiller(0, dmoVSlotDibits)...)
	for i := 0; i < n; i++ {
		fa := dmoVSeq(7+i, 137)
		fb := dmoVSeq(9+i, 137)
		t4 := framing.UnpackBitsMSB(tetra.EncodeTCHS(fa, fb), 432)
		onair := framing.ScrambleTetra(t4, seed)
		for k := 0; k < flips; k++ {
			// Coded region only (type-3 bits 102..431 through the 24x18 interleave).
			t3 := 102 + (i*53+k*97)%330
			onair[(t3%18)*24+t3/18] ^= 1
		}
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
		// Keep only the bursts on the transmitter's slot grid (lead at slot
		// offset 7+108): a scrambled payload can contain chance 11-dibit
		// training-sequence matches (the correlator's ~18/s false-alarm class),
		// which the live chain drops via DMSlotGrid and the noise test covers.
		if b.Kind == tetra.DMBurstNormal && b.Lead%dmoVSlotDibits == 115 {
			dnbs = append(dnbs, b)
		}
	}
	if len(dnbs) != n {
		t.Fatalf("built %d DNB bursts, want %d", len(dnbs), n)
	}
	return dnbs, want
}
