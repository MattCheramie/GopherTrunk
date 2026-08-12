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
