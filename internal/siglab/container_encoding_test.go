package siglab

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// writeSDRSharpFloatWav writes a two-channel 32-bit IEEE-float WAV (fmt tag
// 3) — the SDR# "baseband" recording format the #1184 FleetSync captures
// arrived in.
func writeSDRSharpFloatWav(t *testing.T, path string, rate uint32, iq []complex64) {
	t.Helper()
	var body bytes.Buffer
	for _, s := range iq {
		binary.Write(&body, binary.LittleEndian, math.Float32bits(real(s)))
		binary.Write(&body, binary.LittleEndian, math.Float32bits(imag(s)))
	}
	var f bytes.Buffer
	f.WriteString("RIFF")
	binary.Write(&f, binary.LittleEndian, uint32(36+body.Len()))
	f.WriteString("WAVEfmt ")
	binary.Write(&f, binary.LittleEndian, uint32(16))
	binary.Write(&f, binary.LittleEndian, uint16(3)) // IEEE float
	binary.Write(&f, binary.LittleEndian, uint16(2))
	binary.Write(&f, binary.LittleEndian, rate)
	binary.Write(&f, binary.LittleEndian, rate*8)
	binary.Write(&f, binary.LittleEndian, uint16(8))
	binary.Write(&f, binary.LittleEndian, uint16(32))
	f.WriteString("data")
	binary.Write(&f, binary.LittleEndian, uint32(body.Len()))
	f.Write(body.Bytes())
	if err := os.WriteFile(path, f.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestDecodeContainerFileReadsFloat32WAV is the #1184 regression: a 32-bit
// float SDR# recording decodes to its own samples. The old reader stripped
// 44 bytes and read the body as int16 pairs — twice the sample count of
// garbage at -2.5 dBFS, which no decoder could lock to.
func TestDecodeContainerFileReadsFloat32WAV(t *testing.T) {
	iq := make([]complex64, 1500)
	for i := range iq {
		ph := 2 * math.Pi * float64(i) / 23
		iq[i] = complex(float32(0.3*math.Cos(ph)), float32(0.3*math.Sin(ph)))
	}
	path := filepath.Join(t.TempDir(), "sdrsharp.wav")
	writeSDRSharpFloatWav(t, path, 2_048_000, iq)

	got, rate, err := DecodeContainerFile(path)
	if err != nil {
		t.Fatalf("DecodeContainerFile: %v", err)
	}
	if rate != 2_048_000 || len(got) != len(iq) {
		t.Fatalf("rate=%d samples=%d, want 2048000 / %d", rate, len(got), len(iq))
	}
	for i := range iq {
		if got[i] != iq[i] {
			t.Fatalf("sample %d = %v, want %v", i, got[i], iq[i])
		}
	}

	// The streaming unwrap hands the same body to the float32 decoder.
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	body, format, hdrRate, err := UnwrapContainer(f, FormatWAV)
	if err != nil {
		t.Fatalf("UnwrapContainer: %v", err)
	}
	if format != FormatF32 || hdrRate != 2_048_000 {
		t.Fatalf("UnwrapContainer format=%v rate=%d, want f32 / 2048000", format, hdrRate)
	}
	rest := make([]byte, 8*len(iq))
	if _, err := body.Read(rest[:8]); err != nil {
		t.Fatal(err)
	}
	decode, per := format.Decoder()
	first := make([]complex64, 1)
	decode(rest[:per], first)
	if first[0] != iq[0] {
		t.Fatalf("first unwrapped sample = %v, want %v", first[0], iq[0])
	}
}
