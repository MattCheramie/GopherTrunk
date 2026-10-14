package ysf

import (
	"encoding/hex"
	"testing"

	dmrvoice "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/voice"
)

// Every literal below came out of MMDVMHost's own encoder (CYSFFICH::encode,
// CGolay24128::encode24128) or its YSF2DMR constants, compiled outside the
// repo; none was produced by this package. That is the point: the FICH in
// fich.go was pinned only by its own encoder and decodes no real frame.

// hexDibits unpacks on-air bytes into dibits, MSB first.
func hexDibits(t *testing.T, h string) []uint8 {
	t.Helper()
	b, err := hex.DecodeString(h)
	if err != nil {
		t.Fatal(err)
	}
	var out []uint8
	for _, x := range b {
		for s := 6; s >= 0; s -= 2 {
			out = append(out, (x>>uint(s))&3)
		}
	}
	return out
}

// TestGolay24MatchesMMDVMHost pins the extended Golay(24,12) layout against
// CGolay24128::encode24128 outputs.
func TestGolay24MatchesMMDVMHost(t *testing.T) {
	for _, tc := range []struct {
		data uint16
		cw   uint32
	}{
		{0x000, 0x000000}, {0x001, 0x0018EB}, {0x002, 0x00293E}, {0x124, 0x124BEE},
		{0x800, 0x800C75}, {0xABC, 0xABC23C}, {0xFFF, 0xFFFFFF},
	} {
		if got := golay24Encode(tc.data); got != tc.cw {
			t.Errorf("encode(%03x) = %06x, want %06x", tc.data, got, tc.cw)
		}
		for _, flip := range []uint32{0, 1 << 23, 1<<23 | 1<<5, 1<<20 | 1<<9 | 1} {
			d, ok := golay24Decode(tc.cw ^ flip)
			if !ok || d != tc.data {
				t.Errorf("decode(%06x ^ %06x) = %03x ok=%v, want %03x", tc.cw, flip, d, ok, tc.data)
			}
		}
	}
	if _, ok := golay24Decode(0x0018EB ^ 0xF); ok {
		t.Error("a 4-bit error decoded as within the correction radius")
	}
}

// airFICHVectors are the 25 on-air FICH bytes MMDVMHost's encoder wrote
// for each field set.
var airFICHVectors = []struct {
	onair string
	want  AirFICH
}{
	{"0037b4ad9d002d9f9071003fe1e3ac0025d16449000b81cc5f", AirFICH{FI: 0, DT: 2, FN: 0, FT: 7, MR: 1}},
	{"3127cca4bacfbe90326f4b9d70efea4fd5207d0684c823065b", AirFICH{FI: 1, DT: 2, FN: 3, FT: 7, MR: 1}},
	{"c397b4716241cd9ca0ea4c0fe395858a65d1e502c3cb8161af", AirFICH{FI: 2, DT: 2, FN: 0, FT: 7, MR: 1}},
	{"3d0ff81103c5dc8c9b524857becc50418657352d83189b987b", AirFICH{FI: 1, DT: 0, FN: 5, FT: 6, MR: 2, DGID: 11}},
	{"30d604e125c3ac2fd73441e33208254cd2a384df8a0d823064", AirFICH{FI: 1, DT: 3, FN: 1, FT: 7, MR: 0}},
}

func TestDecodeAirFICHReferenceVectors(t *testing.T) {
	for _, v := range airFICHVectors {
		got, ok := DecodeAirFICH(hexDibits(t, v.onair))
		if !ok {
			t.Errorf("%s: did not decode", v.onair)
			continue
		}
		if got != v.want {
			t.Errorf("%s: got %+v, want %+v", v.onair, got, v.want)
		}
	}
}

// The FICH's convolutional code and Golay words must carry it through bit
// errors, and a frame too damaged to trust must fail the CRC, not decode
// as garbage.
func TestDecodeAirFICHCorrectsAndRejects(t *testing.T) {
	v := airFICHVectors[1]
	dib := hexDibits(t, v.onair)
	for _, i := range []int{3, 37, 71} {
		dib[i] ^= 1
	}
	if got, ok := DecodeAirFICH(dib); !ok || got != v.want {
		t.Errorf("3 symbol errors: got %+v ok=%v, want %+v", got, ok, v.want)
	}
	noise := make([]uint8, 100)
	for i := range noise {
		noise[i] = uint8((i*7 + i/3) & 3)
	}
	if got, ok := DecodeAirFICH(noise); ok {
		t.Errorf("pseudo-random dibits decoded as a FICH: %+v", got)
	}
}

// TestDecodeVD2VoiceMatchesDMRSilence pins the V/D mode 2 voice layout end
// to end. YSF2DMR carries the same AMBE+2 silence frame on both sides: as
// YSF_SILENCE, the 13 on-air bytes of one VCH, and as DMR_SILENCE, the 9
// on-air bytes of one DMR voice frame. Decoding the first with this package
// and the second with the DMR decoder must give the same 49 bits, which
// proves deinterleave, whitening, triplet order and bit order together.
func TestDecodeVD2VoiceMatchesDMRSilence(t *testing.T) {
	ysfSilence, _ := hex.DecodeString("7bb28e4336e4a23978493368" + "33")
	dmrSilence, _ := hex.DecodeString("b9e88152617300" + "2a6b")

	payload := make([]uint8, PayloadDibits)
	for j := 0; j < vd2VCHs; j++ { // the same VCH in all five slots
		for i := 0; i < vd2VCHBits; i++ {
			b := (ysfSilence[i/8] >> uint(7-i%8)) & 1
			n := vd2FirstVCH + j*vd2Section + i
			payload[n/2] |= b << uint(1-n%2)
		}
	}
	frames := DecodeVD2Voice(payload)
	if len(frames) != vd2VCHs {
		t.Fatalf("got %d voice frames, want %d", len(frames), vd2VCHs)
	}

	dmrBits := make([]byte, 72)
	for i := range dmrBits {
		dmrBits[i] = (dmrSilence[i/8] >> uint(7-i%8)) & 1
	}
	want, errs, err := dmrvoice.DecodeAMBEFrame(dmrBits)
	if err != nil || errs != 0 {
		t.Fatalf("DMR silence decode: errs=%d err=%v", errs, err)
	}
	for j, fr := range frames {
		if fr.Corrected != 0 {
			t.Errorf("VCH %d: %d triplets corrected on a clean frame", j, fr.Corrected)
		}
		if string(fr.Info) != string(want) {
			t.Fatalf("VCH %d: info %v\nwant (DMR silence) %v", j, fr.Info, want)
		}
	}

	// One flipped copy of a triplet is voted back.
	payload[(vd2FirstVCH+vd2Interleave(4))/2] ^= 1 << uint(1-(vd2FirstVCH+vd2Interleave(4))%2)
	if fr := DecodeVD2Voice(payload)[0]; fr.Corrected != 1 || string(fr.Info) != string(want) {
		t.Errorf("single-copy error: corrected=%d info match=%v", fr.Corrected, string(fr.Info) == string(want))
	}
}

// buildAirFrame lays out one on-air frame: sync, a reference FICH, payload.
func buildAirFrame(t *testing.T, fichHex string, payload []uint8) []uint8 {
	t.Helper()
	fr := append([]uint8(nil), FSWPattern...)
	fr = append(fr, hexDibits(t, fichHex)...)
	return append(fr, payload...)
}

// TestAirFramerCutsFramesAcrossChunks: the framer finds every frame at its
// sync word whatever the chunking, and hands back the FICH and payload.
func TestAirFramerCutsFramesAcrossChunks(t *testing.T) {
	var infos [vd2VCHs][]byte
	for j := range infos {
		infos[j] = make([]byte, VD2InfoBits)
		for k := range infos[j] {
			infos[j][k] = uint8((j*7 + k*3 + k/5) & 1)
		}
	}
	payload := EncodeVD2Voice(infos)
	stream := make([]uint8, 37) // leading junk
	stream = append(stream, buildAirFrame(t, airFICHVectors[0].onair, make([]uint8, PayloadDibits))...)
	for i := 0; i < 3; i++ {
		stream = append(stream, buildAirFrame(t, airFICHVectors[1].onair, payload)...)
	}
	stream = append(stream, buildAirFrame(t, airFICHVectors[2].onair, make([]uint8, PayloadDibits))...)

	for _, chunk := range []int{1, 13, 480, 4096} {
		f := NewAirFramer(2)
		var frames []AirFrame
		for i := 0; i < len(stream); i += chunk {
			end := min(i+chunk, len(stream))
			frames = append(frames, f.Process(stream[i:end], i)...)
		}
		if len(frames) != 5 {
			t.Fatalf("chunk %d: %d frames, want 5", chunk, len(frames))
		}
		wantFI := []uint8{AirFIHeader, AirFIComms, AirFIComms, AirFIComms, AirFITerminator}
		for i, fr := range frames {
			if !fr.FICHOK || fr.FICH.FI != wantFI[i] {
				t.Fatalf("chunk %d frame %d: FICH ok=%v FI=%d, want FI %d", chunk, i, fr.FICHOK, fr.FICH.FI, wantFI[i])
			}
		}
		for j, vf := range DecodeVD2Voice(frames[2].Payload) {
			if string(vf.Info) != string(infos[j]) {
				t.Fatalf("chunk %d: VCH %d round trip mismatch", chunk, j)
			}
		}
	}
}
