package acelp

import (
	"encoding/binary"
	"math"
	"os"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
)

// TestReplayCapture is an offline validation harness (skipped unless
// GT_TETRA_RAW points at a `.raw` sidecar of descrambled 54-byte type-4 TETRA
// traffic frames, e.g. an old-build recorder sidecar). It TCH/S-decodes each
// frame, renders the CRC-valid speech frames through the ACELP vocoder, and
// writes an 8 kHz mono WAV (GT_TETRA_WAV, default /tmp/tetra_replay.wav). It
// reports the CRC-pass rate and coarse PCM stats so the decoded audio can be
// listened to / inspected. Not a CI test — a manual real-air acceptance check.
func TestReplayCapture(t *testing.T) {
	rawPath := os.Getenv("GT_TETRA_RAW")
	if rawPath == "" {
		t.Skip("set GT_TETRA_RAW to a .raw capture to run the offline ACELP replay")
	}
	data, err := os.ReadFile(rawPath)
	if err != nil {
		t.Fatal(err)
	}
	const frameBytes = tetra.TrafficFrameBytes // 54
	if len(data)%frameBytes != 0 {
		t.Logf("warning: %d bytes is not a multiple of %d; trailing bytes ignored", len(data), frameBytes)
	}
	nFrames := len(data) / frameBytes

	voc := NewVocoder()
	var pcm []int16
	crcPass, speechFrames := 0, 0
	for i := 0; i < nFrames; i++ {
		frame := data[i*frameBytes : (i+1)*frameBytes]
		sfs := tetra.TCHSpeechFrames(frame)
		if len(sfs) == 0 {
			continue
		}
		crcPass++
		for _, sf := range sfs {
			speechFrames++
			out, _ := voc.Decode(sf)
			pcm = append(pcm, out...)
		}
	}

	// Coarse PCM stats.
	var sumSq float64
	var peak int16
	for _, s := range pcm {
		sumSq += float64(s) * float64(s)
		if a := abs16(s); a > peak {
			peak = a
		}
	}
	rms := 0.0
	if len(pcm) > 0 {
		rms = math.Sqrt(sumSq / float64(len(pcm)))
	}
	durSec := float64(len(pcm)) / 8000.0
	t.Logf("frames=%d crc_pass_slots=%d (%.1f%%) speech_frames=%d pcm_samples=%d dur=%.2fs rms=%.0f peak=%d",
		nFrames, crcPass, 100*float64(crcPass)/float64(max(nFrames, 1)), speechFrames, len(pcm), durSec, rms, peak)

	wavPath := os.Getenv("GT_TETRA_WAV")
	if wavPath == "" {
		wavPath = "/tmp/tetra_replay.wav"
	}
	if err := writeWAV8k(wavPath, pcm); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s", wavPath)
}

func abs16(x int16) int16 {
	if x < 0 {
		return -x
	}
	return x
}

func writeWAV8k(path string, pcm []int16) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	const rate = 8000
	dataLen := len(pcm) * 2
	h := make([]byte, 44)
	copy(h[0:], "RIFF")
	binary.LittleEndian.PutUint32(h[4:], uint32(36+dataLen))
	copy(h[8:], "WAVE")
	copy(h[12:], "fmt ")
	binary.LittleEndian.PutUint32(h[16:], 16)
	binary.LittleEndian.PutUint16(h[20:], 1) // PCM
	binary.LittleEndian.PutUint16(h[22:], 1) // mono
	binary.LittleEndian.PutUint32(h[24:], rate)
	binary.LittleEndian.PutUint32(h[28:], rate*2)
	binary.LittleEndian.PutUint16(h[32:], 2)
	binary.LittleEndian.PutUint16(h[34:], 16)
	copy(h[36:], "data")
	binary.LittleEndian.PutUint32(h[40:], uint32(dataLen))
	if _, err := f.Write(h); err != nil {
		return err
	}
	buf := make([]byte, dataLen)
	for i, s := range pcm {
		binary.LittleEndian.PutUint16(buf[i*2:], uint16(s))
	}
	_, err = f.Write(buf)
	return err
}
