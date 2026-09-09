package baseband

import (
	"context"
	"math"
	"path/filepath"
	"testing"

	"github.com/mewkiz/flac"
)

// rampIQ builds a deterministic low-amplitude complex ramp long enough to
// span multiple FLAC blocks.
func rampIQ(n int) []complex64 {
	out := make([]complex64, n)
	for i := range out {
		out[i] = complex(
			float32(0.5*math.Sin(2*math.Pi*float64(i)/97)),
			float32(0.5*math.Cos(2*math.Pi*float64(i)/89)),
		)
	}
	return out
}

// TestFLACIQWriterRoundTrip pins the FLAC baseband recording end to end:
// write a ramp through FLACIQWriter, then decode the file with the upstream
// FLAC parser and require bit-exact int16 samples against the writer's own
// clamp/scale — the same contract IQWriter's WAV body carries.
func TestFLACIQWriterRoundTrip(t *testing.T) {
	const rate = 48_000
	iq := rampIQ(10_000) // > 2 blocks of 4096

	path := filepath.Join(t.TempDir(), "rec.flac")
	w, err := NewFLACIQWriter(path, rate)
	if err != nil {
		t.Fatalf("NewFLACIQWriter: %v", err)
	}
	if err := w.Write(iq); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got, want := w.BytesWritten(), uint32(len(iq)*iqWavBlockAlign); got != want {
		t.Errorf("BytesWritten = %d, want %d", got, want)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	stream, err := flac.ParseFile(path)
	if err != nil {
		t.Fatalf("flac.ParseFile: %v", err)
	}
	if stream.Info.SampleRate != rate {
		t.Errorf("sample rate = %d, want %d", stream.Info.SampleRate, rate)
	}
	if stream.Info.NChannels != 2 || stream.Info.BitsPerSample != 16 {
		t.Errorf("stream is %d-ch %d-bit, want 2-ch 16-bit",
			stream.Info.NChannels, stream.Info.BitsPerSample)
	}
	var got int
	for {
		fr, err := stream.ParseNext()
		if err != nil {
			break
		}
		l, q := fr.Subframes[0].Samples, fr.Subframes[1].Samples
		for i := range l {
			wantI := floatToI16(real(iq[got]))
			wantQ := floatToI16(imag(iq[got]))
			if int16(l[i]) != wantI || int16(q[i]) != wantQ {
				t.Fatalf("sample %d = (%d,%d), want (%d,%d)", got, l[i], q[i], wantI, wantQ)
			}
			got++
		}
	}
	if got != len(iq) {
		t.Fatalf("decoded %d samples, want %d", got, len(iq))
	}
}

// TestReadIQRecordingInfoSniffsContainer proves the replay driver's
// content-based sniffing describes both containers identically.
func TestReadIQRecordingInfoSniffsContainer(t *testing.T) {
	dir := t.TempDir()
	iq := rampIQ(5000)

	wavPath := filepath.Join(dir, "rec.wav")
	ww, err := NewIQWriter(wavPath, 25_000)
	if err != nil {
		t.Fatalf("NewIQWriter: %v", err)
	}
	if err := ww.Write(iq); err != nil {
		t.Fatalf("wav Write: %v", err)
	}
	if err := ww.Close(); err != nil {
		t.Fatalf("wav Close: %v", err)
	}

	flacPath := filepath.Join(dir, "rec.flac")
	fw, err := NewFLACIQWriter(flacPath, 25_000)
	if err != nil {
		t.Fatalf("NewFLACIQWriter: %v", err)
	}
	if err := fw.Write(iq); err != nil {
		t.Fatalf("flac Write: %v", err)
	}
	if err := fw.Close(); err != nil {
		t.Fatalf("flac Close: %v", err)
	}

	for _, path := range []string{wavPath, flacPath} {
		info, err := ReadIQRecordingInfo(path)
		if err != nil {
			t.Fatalf("ReadIQRecordingInfo(%s): %v", path, err)
		}
		if info.SampleRate != 25_000 || info.Channels != 2 || info.Samples != len(iq) {
			t.Errorf("%s: info = %+v, want rate 25000 / 2 ch / %d samples", path, info, len(iq))
		}
	}
}

// TestFileDriverReplaysFLACRecording mounts a FLAC baseband recording through
// the replay FileDriver and requires the streamed IQ to match the recorded
// ramp through the writer's quantization — the same virtual-tuner path a WAV
// recording takes.
func TestFileDriverReplaysFLACRecording(t *testing.T) {
	iq := rampIQ(6000)
	path := filepath.Join(t.TempDir(), "rec.flac")
	w, err := NewFLACIQWriter(path, 200_000) // fast replay metering
	if err != nil {
		t.Fatalf("NewFLACIQWriter: %v", err)
	}
	if err := w.Write(iq); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	drv := NewFileDriver([]ReplaySpec{{Path: path, Loop: false}})
	infos, err := drv.Enumerate()
	if err != nil || len(infos) != 1 {
		t.Fatalf("Enumerate = %v, %v; want 1 recording", infos, err)
	}
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
		t.Fatalf("streamed %d samples, want %d", len(got), len(iq))
	}
	for i, s := range got {
		want := complex(
			float32(floatToI16(real(iq[i])))/32768,
			float32(floatToI16(imag(iq[i])))/32768,
		)
		if s != want {
			t.Fatalf("sample %d = %v, want %v", i, s, want)
		}
	}
}

// TestNewIQRecorderWriterDispatch pins the format dispatch and the loud
// failure on an unknown format.
func TestNewIQRecorderWriterDispatch(t *testing.T) {
	dir := t.TempDir()
	if w, err := NewIQRecorderWriter(filepath.Join(dir, "a.wav"), 1000, ""); err != nil {
		t.Fatalf("default format: %v", err)
	} else {
		if _, ok := w.(*IQWriter); !ok {
			t.Errorf("default format writer = %T, want *IQWriter", w)
		}
		w.Close()
	}
	if w, err := NewIQRecorderWriter(filepath.Join(dir, "b.flac"), 1000, "flac"); err != nil {
		t.Fatalf("flac format: %v", err)
	} else {
		if _, ok := w.(*FLACIQWriter); !ok {
			t.Errorf("flac format writer = %T, want *FLACIQWriter", w)
		}
		w.Close()
	}
	if _, err := NewIQRecorderWriter(filepath.Join(dir, "c.x"), 1000, "mp3"); err == nil {
		t.Fatal("unknown format should error")
	}
}
