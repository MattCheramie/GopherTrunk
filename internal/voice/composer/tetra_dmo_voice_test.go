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
	dibits = append(dibits, dmoVFiller(0, 50)...)
	for i := 0; i < n; i++ {
		fa := dmoVSeq(7+i, 137)
		fb := dmoVSeq(9+i, 137)
		t4 := framing.UnpackBitsMSB(tetra.EncodeTCHS(fa, fb), 432)
		onair := framing.ScrambleTetra(t4, colour)
		dibits = append(dibits, dmoVFiller(11+i, 30)...)
		dibits = append(dibits, b2d(onair[:216])...)
		dibits = append(dibits, tetra.NormalSyncDibits()...)
		dibits = append(dibits, b2d(onair[216:])...)
		want = append(want, [2][]byte{framing.PackBitsMSB(fa), framing.PackBitsMSB(fb)})
	}
	dibits = append(dibits, dmoVFiller(99, 200)...)

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
	bursts, want := buildDMODNBBursts(t, colour, 24)

	sink := &fakeDMOSink{}
	c := &Composer{log: slog.New(slog.NewTextHandler(io.Discard, nil)), hangtime: time.Second}
	bt := c.newBoundaryTracker("s", 0, nil)
	dec := &dmoVoiceDecoder{c: c, serial: "s", bt: bt, rs: sink}

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
		colour: colour, colourKnown: true,
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
	dibits = append(dibits, dmoVFiller(0, 50)...)
	for i := 0; i < n; i++ {
		dibits = append(dibits, dmoVFiller(11+i, 30)...)
		dibits = append(dibits, dmoVFiller(200+i*7, 108)...) // BKN1: not a TCH/S codeword
		dibits = append(dibits, tetra.NormalSyncDibits()...)
		dibits = append(dibits, dmoVFiller(500+i*13, 108)...) // BKN2
	}
	dibits = append(dibits, dmoVFiller(99, 200)...)

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
	dec := &dmoVoiceDecoder{c: c, serial: "s", bt: c.newBoundaryTracker("s", 0, nil), rs: sink}

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
	dec := &dmoVoiceDecoder{c: c, serial: "s", bt: c.newBoundaryTracker("s", 0, nil), rs: sink}

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
	dec := &dmoVoiceDecoder{c: c, serial: "s", bt: c.newBoundaryTracker("s", 0, nil), rs: sink}

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
