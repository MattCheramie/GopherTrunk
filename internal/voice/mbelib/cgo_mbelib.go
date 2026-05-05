//go:build mbelib && cgo

// Package mbelib's libmbe-backed wrapper. Compiled only when the
// caller supplies `-tags mbelib` and CGO is enabled. Linking is via
// `pkg-config --libs mbelib` if a pkg-config file is installed; the
// fallback explicit `-lmbe -lm` covers source-built installations.
//
// The wrapper targets the canonical szechyjs/mbelib API:
//
//	void mbe_initMbeParms(mbe_parms*, mbe_parms*, mbe_parms*);
//	void mbe_processImbe4400Dataf(float audio_out[], int *errs, int *errs2,
//	    char *err_str, char ambe_d[88], mbe_parms*, mbe_parms*,
//	    mbe_parms*, int uvquality);
//	void mbe_processAmbe2400Dataf(float audio_out[], int *errs, int *errs2,
//	    char *err_str, char ambe_d[49], mbe_parms*, mbe_parms*,
//	    mbe_parms*, int uvquality);
//
// Output is 160 floats per call (20 ms at 8 kHz), normalised roughly
// to the int16 PCM range. We clamp + cast to int16 so the wrapper
// satisfies the voice.Vocoder interface with the same PCM format
// the recorder expects.

package mbelib

/*
#cgo LDFLAGS: -lmbe -lm
#include <mbelib.h>
#include <string.h>
*/
import "C"

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/MattCheramie/GopherTrunk/internal/voice"
)

// Compile-time assertions that our wrappers satisfy voice.Vocoder.
var (
	_ voice.Vocoder = (*imbeDecoder)(nil)
	_ voice.Vocoder = (*ambe2Decoder)(nil)
)

const (
	// uvQuality is mbelib's per-frame voicing-decision quality knob.
	// Values in 1..7 are valid; 3 is the szechyjs/mbelib default and
	// matches what most out-of-the-box decoders use.
	uvQuality = 3

	// pcmSamplesPerFrame is the fixed mbelib output: 20 ms at 8 kHz.
	pcmSamplesPerFrame = 160

	// imbeFrameBytes / ambe2FrameBytes are the packed-bit frame
	// sizes we accept on input. mbelib expects each information bit
	// as a separate `char`; we unpack from the packed byte form.
	imbeFrameBytes  = 11 // 88 bits
	ambe2FrameBytes = 7  // 49 bits packed into 7 bytes (7 unused)
)

func init() {
	voice.DefaultRegistry.Register("imbe", func() (voice.Vocoder, error) {
		return newImbeDecoder(), nil
	})
	voice.DefaultRegistry.Register("ambe2", func() (voice.Vocoder, error) {
		return newAmbe2Decoder(), nil
	})
}

// commonState holds the three mbe_parms structs every decoder
// instance threads through libmbe across consecutive frames so
// across-frame harmonics + voicing tracking stays coherent.
type commonState struct {
	cur     C.mbe_parms
	prev    C.mbe_parms
	prevEnh C.mbe_parms
}

func newCommonState() commonState {
	var s commonState
	C.mbe_initMbeParms(&s.cur, &s.prev, &s.prevEnh)
	return s
}

func (s *commonState) reset() {
	C.mbe_initMbeParms(&s.cur, &s.prev, &s.prevEnh)
}

// imbeDecoder wraps mbe_processImbe4400Dataf. Used for P25 Phase 1
// LDU1 / LDU2 voice frames.
type imbeDecoder struct {
	state commonState
}

func newImbeDecoder() *imbeDecoder { return &imbeDecoder{state: newCommonState()} }

func (d *imbeDecoder) Name() string   { return "imbe" }
func (d *imbeDecoder) FrameSize() int { return imbeFrameBytes }

func (d *imbeDecoder) Decode(frame []byte) ([]int16, error) {
	if len(frame) != imbeFrameBytes {
		return nil, fmt.Errorf("mbelib: IMBE frame must be %d bytes (88 bits), got %d",
			imbeFrameBytes, len(frame))
	}
	var bits [88]C.char
	unpackBits(frame, bits[:])

	var audio [pcmSamplesPerFrame]C.float
	var errs, errs2 C.int
	var errStr [64]C.char

	C.mbe_processImbe4400Dataf(
		(*C.float)(unsafe.Pointer(&audio[0])),
		&errs, &errs2, &errStr[0],
		(*C.char)(unsafe.Pointer(&bits[0])),
		&d.state.cur, &d.state.prev, &d.state.prevEnh,
		C.int(uvQuality),
	)
	return floatsToPCM(audio[:]), nil
}

func (d *imbeDecoder) Reset()       { d.state.reset() }
func (d *imbeDecoder) Close() error { return nil }

// ambe2Decoder wraps mbe_processAmbe2400Dataf. Used for P25 Phase 2
// voice frames, DMR voice bursts, and NXDN voice frames.
type ambe2Decoder struct {
	state commonState
}

func newAmbe2Decoder() *ambe2Decoder { return &ambe2Decoder{state: newCommonState()} }

func (d *ambe2Decoder) Name() string   { return "ambe2" }
func (d *ambe2Decoder) FrameSize() int { return ambe2FrameBytes }

func (d *ambe2Decoder) Decode(frame []byte) ([]int16, error) {
	if len(frame) != ambe2FrameBytes {
		return nil, fmt.Errorf("mbelib: AMBE+2 frame must be %d bytes (49 bits), got %d",
			ambe2FrameBytes, len(frame))
	}
	// AMBE+2 frames are 49 bits — the upper 49 bits of the 7-byte
	// input. The remaining 7 bits are padding and ignored.
	var bits [49]C.char
	unpackBitsUpTo(frame, bits[:])

	var audio [pcmSamplesPerFrame]C.float
	var errs, errs2 C.int
	var errStr [64]C.char

	C.mbe_processAmbe2400Dataf(
		(*C.float)(unsafe.Pointer(&audio[0])),
		&errs, &errs2, &errStr[0],
		(*C.char)(unsafe.Pointer(&bits[0])),
		&d.state.cur, &d.state.prev, &d.state.prevEnh,
		C.int(uvQuality),
	)
	return floatsToPCM(audio[:]), nil
}

func (d *ambe2Decoder) Reset()       { d.state.reset() }
func (d *ambe2Decoder) Close() error { return nil }

// unpackBits unpacks `len(out)` bits from packed-MSB-first input.
func unpackBits(packed []byte, out []C.char) {
	for i := 0; i < len(out); i++ {
		if packed[i>>3]&(1<<uint(7-(i&7))) != 0 {
			out[i] = 1
		} else {
			out[i] = 0
		}
	}
}

// unpackBitsUpTo is the same as unpackBits but tolerates `len(packed)`
// being any value ≥ ⌈len(out)/8⌉.
func unpackBitsUpTo(packed []byte, out []C.char) {
	if (len(out)+7)/8 > len(packed) {
		// Caller-side error; pad zeros.
		for i := range out {
			out[i] = 0
		}
		return
	}
	unpackBits(packed, out)
}

// floatsToPCM converts mbelib's float output into clamped int16.
func floatsToPCM(audio []C.float) []int16 {
	out := make([]int16, len(audio))
	for i, f := range audio {
		v := float64(f)
		if v > 32767 {
			v = 32767
		} else if v < -32768 {
			v = -32768
		}
		out[i] = int16(v)
	}
	return out
}

// errLibMbe is reserved for future error reporting; mbelib's per-
// frame errStr buffer carries diagnostic strings we currently
// discard. Surfacing them is a follow-up.
var errLibMbe = errors.New("mbelib: decode error")
