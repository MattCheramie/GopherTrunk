package dstar

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// mmdvmTransmission builds a whole transmission the way MMDVM sends one
// (DStarTX::process): the 1010… preamble, the header burst, nFrames DV
// frames of the null AMBE frame with slow data, then the end pattern three
// times (DSTAR_END_SYNC_BYTES: 55 55 55 55 C8 7A 00 … 00).
func mmdvmTransmission(h Header, nFrames int) []uint8 {
	out := BitSyncBits(400)
	out = append(out, EncodeHeaderBurst(SealHeader(h))...)
	for n := 0; n < nFrames; n++ {
		out = append(out, EncodeDVFrame(NullAMBEFrame(), n, [3]byte{0x40 | byte(n%4), 'G', 'T'})...)
	}
	end := BytesToBitsLSB([]byte{0x55, 0x55, 0x55, 0x55, 0xC8, 0x7A, 0, 0, 0, 0, 0, 0})
	for k := 0; k < 3; k++ {
		out = append(out, end...)
	}
	return out
}

// A whole transmission in MMDVM's format decodes in either polarity: the
// header, and every DV frame's null AMBE frame without a Golay correction;
// the end pattern stops the voice cadence at once, so the zero fill after it
// produces nothing. The trailing noise after the end must not produce frames
// either.
func TestTransmissionDecodesInEitherPolarity(t *testing.T) {
	h := Header{RPT2: "KD0AAA G", RPT1: "KD0AAA B", UR: "CQCQCQ  ", MY1: "W1AW    ", MY2: "ID51"}
	const nFrames = 50
	for _, inv := range []bool{false, true} {
		stream := mmdvmTransmission(h, nFrames)
		stream = append(stream, make([]uint8, 3000)...) // carrier off: silence
		if inv {
			stream = invertBits(stream)
		}
		var headers []Header
		cc := New(Options{Bus: events.NewBus(8), Now: time.Now})
		cc.ingestHook = func(x Header) { headers = append(headers, x) }
		var frames []DVFrame
		vc := NewVoiceChannel(func(f DVFrame) { frames = append(frames, f) })
		for pos := 0; pos < len(stream); pos += 480 {
			end := min(pos+480, len(stream))
			cc.Process(stream[pos:end], pos)
			vc.Process(stream[pos:end], pos)
		}
		if len(headers) != 1 || headers[0].MY1 != h.MY1 || headers[0].UR != h.UR {
			t.Errorf("inverted=%v: headers %+v", inv, headers)
		}
		if len(frames) != nFrames {
			t.Errorf("inverted=%v: %d voice frames, want %d", inv, len(frames), nFrames)
		}
		for i, f := range frames {
			if f.Errors != 0 {
				t.Errorf("inverted=%v: frame %d needed %d corrections", inv, i, f.Errors)
				break
			}
		}
	}
}

// The slow data is scrambled with 0x70 0x4F 0x93 (MMDVMHost DSTAR_SCRAMBLER_
// BYTES; DSD XORs the same three bytes back), and the sync frame's field is
// sent as is.
func TestDVFrameSlowDataScrambling(t *testing.T) {
	f := EncodeDVFrame(NullAMBEFrame(), 1, [3]byte{0x00, 0x00, 0x00})
	want := BytesToBitsLSB([]byte{0x70, 0x4F, 0x93})
	for i, b := range f[DVVoiceBits:] {
		if b != want[i] {
			t.Fatalf("scrambled zero slow data differs at bit %d", i)
		}
	}
	// MMDVMHost's DSTAR_NULL_SLOW_DATA_BYTES (0x16 0x29 0xF5) unscramble
	// to 'f' 'f' 'f', its filler.
	f = EncodeDVFrame(NullAMBEFrame(), 2, [3]byte{'f', 'f', 'f'})
	want = BytesToBitsLSB([]byte{0x16, 0x29, 0xF5})
	for i, b := range f[DVVoiceBits:] {
		if b != want[i] {
			t.Fatalf("filler slow data differs from MMDVMHost's at bit %d", i)
		}
	}
	sync := EncodeDVFrame(NullAMBEFrame(), 21, [3]byte{1, 2, 3})
	want = BytesToBitsLSB([]byte{0x55, 0x2D, 0x16})
	for i, b := range sync[DVVoiceBits:] {
		if b != want[i] {
			t.Fatalf("sync frame slow data differs at bit %d", i)
		}
	}
}
