package phase1

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// onAirFrameDibits builds a non-LDU data unit at its on-air length: frame
// sync, a valid NID for duid, zero fill, and a status symbol after every 70
// bits — the same 72-bit stride as an LDU (TIA-102.BAAA; OP25 p25_framer's
// frame lengths: HDU 792 bits, TDU 144, TDULC 432).
func onAirFrameDibits(t *testing.T, nac uint16, duid DUID, totalBits int) []uint8 {
	t.Helper()
	payload := make([]byte, 0, totalBits)
	payload = append(payload, FrameSyncBits()...)
	payload = append(payload, EncodeNIDBits(nac, duid)...)
	stride := LDUStatusInterval + 2
	if totalBits%stride != 0 {
		t.Fatalf("frame of %d bits is not a whole number of %d-bit status blocks", totalBits, stride)
	}
	want := totalBits / stride * LDUStatusInterval
	for len(payload) < want {
		payload = append(payload, 0)
	}
	bits := make([]byte, 0, totalBits)
	for i := 0; i < totalBits/stride; i++ {
		bits = append(bits, payload[i*LDUStatusInterval:(i+1)*LDUStatusInterval]...)
		bits = append(bits, 1, 0) // status symbol
	}
	return framing.BitsToDibits(bits)
}

func onAirLDUDibits(t *testing.T, nac uint16, duid DUID) []uint8 {
	t.Helper()
	var voice [LDUVoiceSubframeCount][]byte
	for i := range voice {
		voice[i] = make([]byte, LDUVoiceSubframeBits)
	}
	var lces [LDULCESBlockCount][]byte
	var lsd [LDULSDBlockCount][]byte
	bits, err := AssembleLDU(nac, duid, voice, lces, lsd)
	if err != nil {
		t.Fatal(err)
	}
	return framing.BitsToDibits(bits)
}

// TestLDUAssemblerFramesEachDataUnitAtItsOwnLength pins #1242's head loss:
// the assembler collected 864 dibits (an LDU's length) after EVERY frame-sync
// match. A transmission opens with a 396-dibit HDU, so the HDU's "LDU" window
// swallowed the first LDU1's frame sync — LDU1 was lost on every over, and the
// HDU bits went to the voice extractor as if they were IMBE frames. A 72-dibit
// TDU likewise swallowed the next over's HDU + LDU1 when a reply keyed up
// within 180 ms. Each data unit must consume only its own length.
func TestLDUAssemblerFramesEachDataUnitAtItsOwnLength(t *testing.T) {
	const nac = 0x293
	var stream []uint8
	stream = append(stream, dibitsFor(50, 1, 3, 0, 2)...) // pre-key noise
	stream = append(stream, onAirFrameDibits(t, nac, DUIDHeader, 792)...)
	stream = append(stream, onAirLDUDibits(t, nac, DUIDLogicalLink1)...)
	stream = append(stream, onAirLDUDibits(t, nac, DUIDLogicalLink2)...)
	stream = append(stream, onAirFrameDibits(t, nac, DUIDTerminator, 144)...)
	// The reply keys up straight after the terminator.
	stream = append(stream, onAirFrameDibits(t, nac, DUIDHeader, 792)...)
	stream = append(stream, onAirLDUDibits(t, nac, DUIDLogicalLink1)...)
	stream = append(stream, dibitsFor(LDUDibitCount, 1, 3, 0, 2)...) // tail so the last window completes

	var got []DUID
	a := NewLDUAssembler(func(ldu []byte) {
		d, err := LDUDuid(ldu)
		if err != nil {
			t.Fatalf("emitted window with undecodable NID: %v", err)
		}
		got = append(got, d)
	}, 0)
	a.Process(stream)

	want := []DUID{DUIDLogicalLink1, DUIDLogicalLink2, DUIDTerminator, DUIDLogicalLink1}
	if len(got) != len(want) {
		t.Fatalf("emitted %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("emitted %v, want %v", got, want)
		}
	}
}

// TestLDUAssemblerChunkInvariantAcrossDataUnits: the same stream fed one
// dibit at a time emits the same frames (the receiver hands the assembler
// arbitrary chunk sizes).
func TestLDUAssemblerChunkInvariantAcrossDataUnits(t *testing.T) {
	const nac = 0x293
	var stream []uint8
	stream = append(stream, onAirFrameDibits(t, nac, DUIDHeader, 792)...)
	stream = append(stream, onAirLDUDibits(t, nac, DUIDLogicalLink1)...)
	stream = append(stream, onAirFrameDibits(t, nac, DUIDTerminatorWithLC, 432)...)
	stream = append(stream, onAirLDUDibits(t, nac, DUIDLogicalLink2)...)
	stream = append(stream, dibitsFor(LDUDibitCount, 2, 0)...)

	collect := func(chunk int) []DUID {
		var got []DUID
		a := NewLDUAssembler(func(ldu []byte) {
			d, _ := LDUDuid(ldu)
			got = append(got, d)
		}, 0)
		for i := 0; i < len(stream); i += chunk {
			a.Process(stream[i:min(i+chunk, len(stream))])
		}
		return got
	}
	whole, single := collect(len(stream)), collect(1)
	want := []DUID{DUIDLogicalLink1, DUIDTerminatorWithLC, DUIDLogicalLink2}
	for _, got := range [][]DUID{whole, single} {
		if len(got) != len(want) {
			t.Fatalf("emitted %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("emitted %v, want %v", got, want)
			}
		}
	}
}
