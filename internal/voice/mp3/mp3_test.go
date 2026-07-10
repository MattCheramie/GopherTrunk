package mp3

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"os/exec"
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

// TestEncodeEncodesEveryFrame is the regression for root cause #1: Shine's
// Write drops every other frame of a mono stream, so a call encoded to ~half
// its true length (near-silence). Feeding a whole number of frames, the output
// must contain exactly that many audio frames plus the one Info header frame.
func TestEncodeEncodesEveryFrame(t *testing.T) {
	for _, tc := range []struct {
		rate int
		spf  int // Shine per-frame samples for this rate family
	}{
		{8000, 576}, {22050, 576}, {44100, 1152},
	} {
		const wholeFrames = 20
		n := wholeFrames * shineFrame // whole shineFrames → no padding rounding
		out, err := Encode(tone(n, tc.rate, 440), tc.rate)
		if err != nil {
			t.Fatalf("Encode(%d Hz): %v", tc.rate, err)
		}
		wantAudio := n / tc.spf // every frame encoded, not every other
		if got := countFrames(out); got != wantAudio+1 {
			t.Fatalf("Encode(%d Hz): counted %d frames, want %d audio + 1 Info = %d",
				tc.rate, got, wantAudio, wantAudio+1)
		}
	}
}

// supportedRates are the sample rates the Shine encoder accepts, spanning all
// three MPEG sample-rate families.
var supportedRates = []int{8000, 11025, 12000, 16000, 22050, 24000, 32000, 44100, 48000}

// TestEncodeFramesTileExactly is the regression for the core defect behind
// issue #874: at Shine's fixed 128 kbps, every single-granule (sub-32 kHz)
// frame is either header-corrupt (128 kbps is illegal below 16 kHz) or
// oversized (Shine's reservoir stuffing desyncs on large frames), so the byte
// stream cannot be walked frame-by-frame and ffmpeg rejects it. A correctly
// encoded CBR stream must tile exactly into whole frames from the first byte to
// the last with nothing left over.
func TestEncodeFramesTileExactly(t *testing.T) {
	for _, rate := range supportedRates {
		out, err := Encode(tone(30*576, rate, 600), rate)
		if err != nil {
			t.Fatalf("Encode(%d Hz): %v", rate, err)
		}
		pos, n := 0, 0
		for {
			h, ok := parseFrameHeader(out[pos:])
			if !ok {
				t.Fatalf("Encode(%d Hz): frame %d at byte %d has an invalid header: % x",
					rate, n, pos, out[pos:min(pos+4, len(out))])
			}
			n++
			next := pos + h.frameLen()
			if next >= len(out) {
				// Final frame: Shine leaves the last partial 32-bit word
				// unflushed, so its declared length can exceed the stream by a
				// few bytes. A gross overrun means the reservoir-stuffing bug.
				if next > len(out)+4 {
					t.Fatalf("Encode(%d Hz): frame %d overruns the stream by %d bytes",
						rate, n-1, next-len(out))
				}
				break
			}
			pos = next
		}
	}
}

// TestEncodeStartsWithXingInfoHeader is the regression for root cause #2: Shine
// emits headerless frames that ffmpeg cannot reliably parse on short clips. The
// stream must begin with a valid Xing/Info header frame carrying the stream's
// frame count, exactly as LAME does.
func TestEncodeStartsWithXingInfoHeader(t *testing.T) {
	out, err := Encode(tone(10*shineFrame, 8000, 440), 8000)
	if err != nil {
		t.Fatal(err)
	}
	h, ok := parseFrameHeader(out)
	if !ok {
		t.Fatalf("first frame is not a valid Layer-III header: % x", out[:4])
	}
	off := 4 + h.sideInfoLen()
	if tag := string(out[off : off+4]); tag != "Info" && tag != "Xing" {
		t.Fatalf("no Xing/Info tag at offset %d: %q", off, tag)
	}
	if flags := binary.BigEndian.Uint32(out[off+4:]); flags&0x0003 != 0x0003 {
		t.Fatalf("Info flags %#x missing frames+bytes bits", flags)
	}
	frameCount := binary.BigEndian.Uint32(out[off+8:])
	if int(frameCount) != countFrames(out) {
		t.Fatalf("Info frame count = %d, but stream has %d frames", frameCount, countFrames(out))
	}
	byteCount := binary.BigEndian.Uint32(out[off+12:])
	if int(byteCount) != len(out) {
		t.Fatalf("Info byte count = %d, but stream is %d bytes", byteCount, len(out))
	}
}

// TestEncodeFFmpegParses is the direct end-to-end reproduction from issue #874:
// pipe the encoded bytes to ffmpeg over stdin, exactly as Rdio Scanner does
// before transcoding an uploaded call. Every supported rate must parse, and a
// short clip (few frames) is covered too since that is the case ffmpeg's mp3
// demuxer handled worst. Skipped when ffmpeg is not installed.
func TestEncodeFFmpegParses(t *testing.T) {
	ff, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg not installed")
	}
	for _, rate := range supportedRates {
		for _, frames := range []int{30, 3} { // long and short clips
			out, err := Encode(tone(frames*576, rate, 600), rate)
			if err != nil {
				t.Fatalf("Encode(%d Hz): %v", rate, err)
			}
			cmd := exec.Command(ff, "-hide_banner", "-i", "-", "-f", "null", "-")
			cmd.Stdin = bytes.NewReader(out)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("ffmpeg rejected %d Hz / %d-frame MP3: %v\n%s",
					rate, frames, err, stderr.String())
			}
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
