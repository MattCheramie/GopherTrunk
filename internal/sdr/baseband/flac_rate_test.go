package baseband

import (
	"math"
	"path/filepath"
	"strings"
	"testing"
)

// TestFLACIQWriterEncodesFrameHeaderUnrepresentableRates is the 15 Sep siglab
// regression: a narrowband slice carved from a 6.25 MS/s X310 stream at
// 880 kHz landed at 880029 Hz, and `format: flac` aborted the capture with
// "unable to encode sample rate 880029" after 3908 samples. FLAC's frame
// header can only carry rates ≤ 65535 Hz directly, ≤ 655350 Hz in tens of
// Hz, and ≤ 255 kHz in whole kHz — an odd rate above 65535 (880029), and
// ANY rate above 655350 (a clean 880000 included), must use the "get from
// STREAMINFO" code instead, whose 20-bit field runs to 1048575 Hz. The
// upstream encoder never picks that code on its own; the shared IQ encode
// core has to ask for it. Fails against the old encoder at every rate here.
func TestFLACIQWriterEncodesFrameHeaderUnrepresentableRates(t *testing.T) {
	iq := rampIQ(10_000) // > 2 blocks of 4096 so several frame headers are written
	for _, rate := range []uint32{880_029, 880_000, 1_000_000, FLACMaxSampleRateHz} {
		path := filepath.Join(t.TempDir(), "rec.flac")
		w, err := NewFLACIQWriter(path, rate)
		if err != nil {
			t.Fatalf("rate %d: NewFLACIQWriter: %v", rate, err)
		}
		if err := w.Write(iq); err != nil {
			t.Fatalf("rate %d: Write: %v", rate, err)
		}
		if err := w.Close(); err != nil {
			t.Fatalf("rate %d: Close: %v", rate, err)
		}
		got, gotRate, err := ReadIQFLACSamples(path)
		if err != nil {
			t.Fatalf("rate %d: ReadIQFLACSamples: %v", rate, err)
		}
		if gotRate != rate {
			t.Errorf("rate %d: decoded STREAMINFO rate = %d", rate, gotRate)
		}
		if len(got) != len(iq) {
			t.Fatalf("rate %d: decoded %d samples, want %d", rate, len(got), len(iq))
		}
		// ReadIQFLACSamples normalises the decoded int16 by 32768 (the replay
		// driver's convention), so undo exactly that to compare against the
		// writer's ×32767 clamp/scale.
		for i := range iq {
			wantI, wantQ := floatToI16(real(iq[i])), floatToI16(imag(iq[i]))
			gotI := int16(math.Round(float64(real(got[i])) * 32768))
			gotQ := int16(math.Round(float64(imag(got[i])) * 32768))
			if gotI != wantI || gotQ != wantQ {
				t.Fatalf("rate %d: sample %d = (%d,%d), want (%d,%d)", rate, i, gotI, gotQ, wantI, wantQ)
			}
		}
	}
}

// TestFLACFrameSampleRate pins the frame-header rate policy: rates the frame
// header can carry verbatim stay verbatim (a stream a third-party decoder
// can seek without STREAMINFO), everything else defers to STREAMINFO.
func TestFLACFrameSampleRate(t *testing.T) {
	for _, tc := range []struct{ rate, want uint32 }{
		{8_000, 8_000}, {48_000, 48_000}, {144_000, 144_000}, {250_000, 250_000},
		{65_535, 65_535}, {655_350, 655_350},
		{65_537, 0}, {880_029, 0}, {880_000, 0}, {1_000_000, 0},
	} {
		if got := FLACFrameSampleRate(tc.rate); got != tc.want {
			t.Errorf("FLACFrameSampleRate(%d) = %d, want %d", tc.rate, got, tc.want)
		}
	}
}

// TestNewFLACIQEncoderRejectsOverStreamInfoCeiling: STREAMINFO's sample-rate
// field is 20 bits, and the upstream encoder writes the low 20 bits of
// whatever it is given — a 2.4 MS/s stream would be silently labelled
// 351424 Hz. Refuse it up front instead.
func TestNewFLACIQEncoderRejectsOverStreamInfoCeiling(t *testing.T) {
	_, err := NewFLACIQWriter(filepath.Join(t.TempDir(), "x.flac"), FLACMaxSampleRateHz+1)
	if err == nil || !strings.Contains(err.Error(), "1048575") {
		t.Fatalf("rate %d: err = %v, want a STREAMINFO-ceiling error naming 1048575", FLACMaxSampleRateHz+1, err)
	}
}
