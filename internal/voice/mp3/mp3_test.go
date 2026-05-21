package mp3

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// tone builds n samples of a sine wave at freqHz for sample rate rate.
func tone(n, rate int, freqHz float64) []int16 {
	s := make([]int16, n)
	for i := range s {
		s[i] = int16(8000 * math.Sin(2*math.Pi*freqHz*float64(i)/float64(rate)))
	}
	return s
}

func TestEncodeProducesMP3FrameSync(t *testing.T) {
	for _, rate := range []int{8000, 16000, 22050, 44100} {
		out, err := Encode(tone(rate, rate, 440), rate)
		if err != nil {
			t.Fatalf("Encode(%d Hz): %v", rate, err)
		}
		if len(out) < 100 {
			t.Fatalf("Encode(%d Hz): output too short: %d bytes", rate, len(out))
		}
		// Every MP3 frame opens with an 11-bit sync word.
		if out[0] != 0xFF || out[1]&0xE0 != 0xE0 {
			t.Fatalf("Encode(%d Hz): no MP3 frame sync: %02x %02x", rate, out[0], out[1])
		}
	}
}

func TestEncodeRejectsBadInput(t *testing.T) {
	if _, err := Encode(nil, 8000); err == nil {
		t.Fatal("Encode(nil) should error")
	}
	if _, err := Encode([]int16{1, 2, 3}, 0); err == nil {
		t.Fatal("Encode(rate=0) should error")
	}
}

func TestReadWAVRoundTrip(t *testing.T) {
	samples := tone(8000, 8000, 440)
	path := filepath.Join(t.TempDir(), "call.wav")
	writeWAV(t, path, samples, 8000)

	got, rate, err := ReadWAV(path)
	if err != nil {
		t.Fatalf("ReadWAV: %v", err)
	}
	if rate != 8000 {
		t.Fatalf("rate = %d, want 8000", rate)
	}
	if len(got) != len(samples) {
		t.Fatalf("len = %d, want %d", len(got), len(samples))
	}
	for i := range got {
		if got[i] != samples[i] {
			t.Fatalf("sample %d = %d, want %d", i, got[i], samples[i])
		}
	}
}

func TestEncodeWAVFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "call.wav")
	writeWAV(t, path, tone(4000, 8000, 600), 8000)

	out, rate, err := EncodeWAVFile(path)
	if err != nil {
		t.Fatalf("EncodeWAVFile: %v", err)
	}
	if rate != 8000 {
		t.Fatalf("rate = %d, want 8000", rate)
	}
	if out[0] != 0xFF || out[1]&0xE0 != 0xE0 {
		t.Fatalf("no MP3 frame sync: %02x %02x", out[0], out[1])
	}
}

func TestReadWAVRejectsNonWAV(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.bin")
	if err := os.WriteFile(path, []byte("not a wav file at all"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadWAV(path); err == nil {
		t.Fatal("ReadWAV on non-WAV should error")
	}
}

// writeWAV emits a canonical 44-byte-header 16-bit PCM mono WAV.
func writeWAV(t *testing.T, path string, samples []int16, rate uint32) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	dataLen := uint32(len(samples) * 2)
	hdr := make([]byte, 44)
	copy(hdr[0:4], "RIFF")
	binary.LittleEndian.PutUint32(hdr[4:8], 36+dataLen)
	copy(hdr[8:12], "WAVE")
	copy(hdr[12:16], "fmt ")
	binary.LittleEndian.PutUint32(hdr[16:20], 16)
	binary.LittleEndian.PutUint16(hdr[20:22], 1)
	binary.LittleEndian.PutUint16(hdr[22:24], 1)
	binary.LittleEndian.PutUint32(hdr[24:28], rate)
	binary.LittleEndian.PutUint32(hdr[28:32], rate*2)
	binary.LittleEndian.PutUint16(hdr[32:34], 2)
	binary.LittleEndian.PutUint16(hdr[34:36], 16)
	copy(hdr[36:40], "data")
	binary.LittleEndian.PutUint32(hdr[40:44], dataLen)
	if _, err := f.Write(hdr); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, dataLen)
	for i, s := range samples {
		binary.LittleEndian.PutUint16(buf[2*i:], uint16(s))
	}
	if _, err := f.Write(buf); err != nil {
		t.Fatal(err)
	}
}
