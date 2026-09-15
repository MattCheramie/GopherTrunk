package baseband

import (
	"bytes"
	"context"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// writeThirdPartyIQWav writes a two-channel WAV the way SDR# / sox do, in
// the given encoding: fmt tag 3 (IEEE float, 32-bit), tag 1 at 8 bits, or
// tag 1 at 16 bits, optionally as WAVE_FORMAT_EXTENSIBLE (tag 0xFFFE with the
// real tag in the SubFormat GUID) and with a LIST chunk before data.
func writeThirdPartyIQWav(t *testing.T, path string, enc IQWavEncoding, extensible, listChunk bool, rate uint32, iq []complex64) {
	t.Helper()
	var body bytes.Buffer
	for _, s := range iq {
		switch enc {
		case IQWavFloat32:
			binary.Write(&body, binary.LittleEndian, math.Float32bits(real(s)))
			binary.Write(&body, binary.LittleEndian, math.Float32bits(imag(s)))
		case IQWavPCM8:
			body.WriteByte(byte(real(s)*127.5 + 127.5))
			body.WriteByte(byte(imag(s)*127.5 + 127.5))
		default:
			binary.Write(&body, binary.LittleEndian, int16(real(s)*32767))
			binary.Write(&body, binary.LittleEndian, int16(imag(s)*32767))
		}
	}
	var tag uint16 = wavFormatPCM
	var bits uint16 = 16
	switch enc {
	case IQWavFloat32:
		tag, bits = wavFormatIEEEFloat, 32
	case IQWavPCM8:
		bits = 8
	}
	blockAlign := uint16(2 * bits / 8)
	var fmtChunk bytes.Buffer
	if extensible {
		binary.Write(&fmtChunk, binary.LittleEndian, uint16(wavFormatExtensible))
	} else {
		binary.Write(&fmtChunk, binary.LittleEndian, tag)
	}
	binary.Write(&fmtChunk, binary.LittleEndian, uint16(2))
	binary.Write(&fmtChunk, binary.LittleEndian, rate)
	binary.Write(&fmtChunk, binary.LittleEndian, rate*uint32(blockAlign))
	binary.Write(&fmtChunk, binary.LittleEndian, blockAlign)
	binary.Write(&fmtChunk, binary.LittleEndian, bits)
	if extensible {
		binary.Write(&fmtChunk, binary.LittleEndian, uint16(22)) // cbSize
		binary.Write(&fmtChunk, binary.LittleEndian, bits)       // valid bits
		binary.Write(&fmtChunk, binary.LittleEndian, uint32(3))  // channel mask
		guid := []byte{0, 0, 0, 0, 0x00, 0x00, 0x10, 0x00, 0x80, 0x00, 0x00, 0xAA, 0x00, 0x38, 0x9B, 0x71}
		binary.LittleEndian.PutUint16(guid[0:2], tag)
		fmtChunk.Write(guid)
	}
	var f bytes.Buffer
	f.WriteString("RIFF")
	binary.Write(&f, binary.LittleEndian, uint32(0)) // patched below
	f.WriteString("WAVE")
	f.WriteString("fmt ")
	binary.Write(&f, binary.LittleEndian, uint32(fmtChunk.Len()))
	f.Write(fmtChunk.Bytes())
	if listChunk {
		// An odd-length chunk is padded to an even boundary (RIFF rule) —
		// the pad byte is not counted in the chunk size.
		list := []byte("INFOISFT\x05\x00\x00\x00SDR#\x00")
		f.WriteString("LIST")
		binary.Write(&f, binary.LittleEndian, uint32(len(list)))
		f.Write(list)
		if len(list)%2 == 1 {
			f.WriteByte(0)
		}
	}
	f.WriteString("data")
	binary.Write(&f, binary.LittleEndian, uint32(body.Len()))
	f.Write(body.Bytes())
	out := f.Bytes()
	binary.LittleEndian.PutUint32(out[4:8], uint32(len(out)-8))
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
}

func testIQ(n int) []complex64 {
	iq := make([]complex64, n)
	for i := range iq {
		ph := 2 * math.Pi * float64(i) / 17
		iq[i] = complex(float32(0.6*math.Cos(ph)), float32(0.6*math.Sin(ph)))
	}
	return iq
}

// TestIQWavReadersHonourFmtChunk is the #1184 regression: a two-channel WAV
// is decoded in the encoding its fmt chunk declares. Before this the header
// parsers rejected anything but 16-bit PCM ("IQ recordings need 16-bit") and
// the siglab container reader silently read a 32-bit float SDR# capture as
// int16 pairs.
func TestIQWavReadersHonourFmtChunk(t *testing.T) {
	iq := testIQ(2000)
	for _, tc := range []struct {
		name       string
		enc        IQWavEncoding
		extensible bool
		list       bool
		tol        float32
	}{
		{"sdrsharp float32", IQWavFloat32, false, false, 0},
		{"float32 extensible + LIST chunk", IQWavFloat32, true, true, 0},
		{"pcm16", IQWavPCM16, false, false, 2.0 / 32767},
		{"pcm8", IQWavPCM8, false, true, 2.0 / 127},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "cap.wav")
			writeThirdPartyIQWav(t, path, tc.enc, tc.extensible, tc.list, 2_048_000, iq)

			info, err := ReadIQWavInfo(path)
			if err != nil {
				t.Fatalf("ReadIQWavInfo: %v", err)
			}
			if info.Encoding != tc.enc || info.SampleRate != 2_048_000 || info.Samples != len(iq) || info.BlockAlign != tc.enc.BlockAlign() {
				t.Fatalf("info = %+v, want encoding %v, 2048000 Hz, %d samples", info, tc.enc, len(iq))
			}
			f, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			sinfo, err := ReadIQWavStreamHeader(f)
			if err != nil {
				t.Fatalf("ReadIQWavStreamHeader: %v", err)
			}
			if sinfo.Encoding != tc.enc || sinfo.SampleRate != 2_048_000 {
				t.Fatalf("stream info = %+v", sinfo)
			}

			// The replay driver must stream the same samples back.
			drv := NewFileDriver([]ReplaySpec{{Path: path}})
			dev, err := drv.Open(0)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			defer dev.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			ch, err := dev.StreamIQ(ctx)
			if err != nil {
				t.Fatalf("StreamIQ: %v", err)
			}
			var got []complex64
			for chunk := range ch {
				got = append(got, chunk...)
			}
			if len(got) != len(iq) {
				t.Fatalf("replayed %d samples, want %d", len(got), len(iq))
			}
			for i := range iq {
				if d := got[i] - iq[i]; math.Abs(float64(real(d))) > float64(tc.tol) || math.Abs(float64(imag(d))) > float64(tc.tol) {
					t.Fatalf("sample %d = %v, want %v (tol %g)", i, got[i], iq[i], tc.tol)
				}
			}
		})
	}
}

func TestIQWavReadersRejectUnsupportedEncodings(t *testing.T) {
	iq := testIQ(10)
	path := filepath.Join(t.TempDir(), "cap.wav")
	writeThirdPartyIQWav(t, path, IQWavPCM16, false, false, 48000, iq)
	b, _ := os.ReadFile(path)
	binary.LittleEndian.PutUint16(b[34:36], 24) // 24-bit PCM: not supported
	os.WriteFile(path, b, 0o644)
	if _, err := ReadIQWavInfo(path); err == nil {
		t.Fatal("24-bit PCM accepted")
	}
	b, _ = os.ReadFile(path)
	binary.LittleEndian.PutUint16(b[34:36], 16)
	binary.LittleEndian.PutUint16(b[22:24], 1) // mono
	os.WriteFile(path, b, 0o644)
	if _, err := ReadIQWavInfo(path); err == nil {
		t.Fatal("mono WAV accepted as IQ")
	}
}
