// Package calibrate compares an in-tree Vocoder's PCM output against
// a reference WAV (typically produced by DSD-FME or OP25) from the
// same raw vocoder-frame source. It quantifies loudness offset
// (RMSRatioDb) and waveform similarity (PeakXcorr).
//
// The harness exists so the AGC + level-related constants in
// internal/voice/mbe/agc.go can be tuned against an external
// reference decoder. Fixture .raw + WAV files live under
// internal/voice/{imbe,ambe2}/testdata/; the matching tests skip
// when those aren't checked in, so CI stays green until the
// reference data lands. The same testdata layout is documented in
// the README "Status & known gaps" → vocoder level calibration
// bullet.
package calibrate

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/MattCheramie/GopherTrunk/internal/voice"
)

// MaxLagSamples is the half-window in samples (±) over which Compare
// searches for the best cross-correlation peak. At 8 kHz PCM, 200
// samples ≈ ±25 ms — large enough to absorb each decoder's pipeline
// group delay, small enough to keep the O(N · maxLag) inner loop
// cheap on a multi-second sample.
const MaxLagSamples = 200

// expectedSampleRate is the PCM rate both the in-tree decoder and
// the reference WAV must emit. The MBE-family decoders (IMBE,
// AMBE+2) all produce 8 kHz / 160-samples-per-frame / 20 ms output;
// DSD-FME and OP25 default to the same.
const expectedSampleRate uint32 = 8000

// Result is what Compare returns.
//
// RMSRatioDb is the loudness offset the in-tree decoder needs to
// move to match the reference, in dB: positive means the in-tree
// decoder is louder than the reference. The MBE AGC's TargetPeak in
// internal/voice/mbe/agc.go is the knob to tune from here.
//
// PeakXcorr is the normalised cross-correlation magnitude at the
// best alignment, in [0, 1]. Values above 0.85 indicate the two
// waveforms are recognisably the same speech; below 0.5 the two
// decoders are very likely not on the same source.
//
// LagSamples is the alignment offset that produced PeakXcorr. Small
// lags (|lag| < 50) mean the two decoders' group delays are
// reasonably aligned; values near ±MaxLagSamples mean we picked an
// edge of the search window and the peak might be outside it —
// re-run with a larger window if the in-tree decoder has more
// pipeline latency than expected.
//
// InTreeSampleCount and RefSampleCount help the operator diagnose
// length mismatches (e.g. the in-tree decoder repeated frames the
// reference dropped).
type Result struct {
	RMSRatioDb        float64
	PeakXcorr         float64
	LagSamples        int
	InTreeSampleCount int
	RefSampleCount    int
}

// Compare reads vocoder frames from rawPath, decodes them through
// the named vocoder from voice.DefaultRegistry, reads the reference
// WAV from refWavPath, and computes a Result.
//
// Both decoders are expected to emit 8 kHz / 16-bit / mono PCM.
// Sample-rate or channel-count mismatches are reported as an error.
//
// The Vocoder is constructed fresh per call; its Close is called on
// the way out. The raw file must contain a whole number of vocoder
// frames (FrameSize-aligned).
func Compare(rawPath, refWavPath, vocoderName string) (Result, error) {
	rawBytes, err := os.ReadFile(rawPath)
	if err != nil {
		return Result{}, fmt.Errorf("calibrate: read raw %q: %w", rawPath, err)
	}

	v, err := voice.DefaultRegistry.New(vocoderName)
	if err != nil {
		return Result{}, fmt.Errorf("calibrate: vocoder %q: %w", vocoderName, err)
	}
	defer v.Close()

	frameSize := v.FrameSize()
	if frameSize <= 0 {
		return Result{}, fmt.Errorf("calibrate: vocoder %q reports invalid FrameSize=%d",
			vocoderName, frameSize)
	}
	if len(rawBytes)%frameSize != 0 {
		return Result{}, fmt.Errorf("calibrate: raw file size %d is not a multiple of %s frame size %d",
			len(rawBytes), vocoderName, frameSize)
	}

	var inTree []int16
	for i := 0; i < len(rawBytes); i += frameSize {
		samples, err := v.Decode(rawBytes[i : i+frameSize])
		if err != nil {
			return Result{}, fmt.Errorf("calibrate: decode frame %d: %w", i/frameSize, err)
		}
		inTree = append(inTree, samples...)
	}

	refSamples, refRate, err := readWav(refWavPath)
	if err != nil {
		return Result{}, fmt.Errorf("calibrate: read ref WAV %q: %w", refWavPath, err)
	}
	if refRate != expectedSampleRate {
		return Result{}, fmt.Errorf("calibrate: reference WAV rate %d Hz != expected %d Hz",
			refRate, expectedSampleRate)
	}

	return CompareSamples(inTree, refSamples), nil
}

// CompareSamples runs the RMS + cross-correlation math directly on
// two PCM streams. Compare is the operator-facing entry point that
// decodes the in-tree vocoder + reads the reference WAV before
// delegating here; CompareSamples is exposed separately so the
// loudness / similarity math is testable with synthetic inputs (no
// vocoder frame fixtures, no on-disk WAV files).
//
// Both streams are expected to be 8 kHz / 16-bit / mono PCM. RMS
// silence in either stream produces RMSRatioDb = 0 (the ratio isn't
// defined); PeakXcorr will still surface the mismatch.
func CompareSamples(inTree, refSamples []int16) Result {
	rmsIn := rms(inTree)
	rmsRef := rms(refSamples)
	var rmsDb float64
	switch {
	case rmsRef <= 0 || rmsIn <= 0:
		// One stream is identically silent; the ratio isn't defined.
		// Leave RMSRatioDb at 0 — the test's |Δ| < 3 dB threshold
		// will still flag the mismatch through PeakXcorr.
	default:
		rmsDb = 20 * math.Log10(rmsIn/rmsRef)
	}

	peakXc, lag := bestXcorr(inTree, refSamples, MaxLagSamples)

	return Result{
		RMSRatioDb:        rmsDb,
		PeakXcorr:         peakXc,
		LagSamples:        lag,
		InTreeSampleCount: len(inTree),
		RefSampleCount:    len(refSamples),
	}
}

// rms returns the root-mean-square amplitude of an int16 sample
// stream. Empty input returns 0.
func rms(samples []int16) float64 {
	if len(samples) == 0 {
		return 0
	}
	var sumSq float64
	for _, s := range samples {
		f := float64(s)
		sumSq += f * f
	}
	return math.Sqrt(sumSq / float64(len(samples)))
}

// bestXcorr returns the peak normalised cross-correlation magnitude
// and the lag (in samples) at which it occurred, searching
// lag ∈ [−maxLag, +maxLag]. Both inputs are truncated to the
// shorter length so the comparison covers the same span regardless
// of decoder-length mismatch. Returns (0, 0) when either input is
// too short or has zero energy.
//
// The result is the absolute value of the correlation — a
// 180°-phase-inverted decoder still produces |xcorr| ≈ 1 and is
// caught instead by the RMSRatioDb / sign-check.
func bestXcorr(x, y []int16, maxLag int) (float64, int) {
	n := len(x)
	if len(y) < n {
		n = len(y)
	}
	if n <= 2*maxLag {
		return 0, 0
	}

	var energyX, energyY float64
	for i := 0; i < n; i++ {
		energyX += float64(x[i]) * float64(x[i])
		energyY += float64(y[i]) * float64(y[i])
	}
	denom := math.Sqrt(energyX * energyY)
	if denom <= 0 {
		return 0, 0
	}

	bestVal := -1.0
	bestLag := 0
	for lag := -maxLag; lag <= maxLag; lag++ {
		var dot float64
		for i := 0; i < n; i++ {
			j := i + lag
			if j < 0 || j >= n {
				continue
			}
			dot += float64(x[i]) * float64(y[j])
		}
		v := math.Abs(dot / denom)
		if v > bestVal {
			bestVal = v
			bestLag = lag
		}
	}
	return bestVal, bestLag
}

// readWav reads a 16-bit / mono / PCM WAV file and returns the
// samples plus the file's sample rate. Mirrors voice.WavWriter's
// output format: RIFF / WAVE / fmt(16) / data. Non-PCM, non-mono,
// or non-16-bit files are rejected. Auxiliary chunks (LIST, INFO,
// JUNK) between fmt and data are skipped.
func readWav(path string) ([]int16, uint32, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	// RIFF + WAVE header.
	riff := make([]byte, 12)
	if _, err := io.ReadFull(f, riff); err != nil {
		return nil, 0, fmt.Errorf("read RIFF header: %w", err)
	}
	if string(riff[0:4]) != "RIFF" || string(riff[8:12]) != "WAVE" {
		return nil, 0, fmt.Errorf("not a RIFF/WAVE file (got %q / %q)",
			string(riff[0:4]), string(riff[8:12]))
	}

	var sampleRate uint32
	gotFmt := false
	for {
		chunkHeader := make([]byte, 8)
		_, err := io.ReadFull(f, chunkHeader)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, fmt.Errorf("read chunk header: %w", err)
		}
		chunkID := string(chunkHeader[0:4])
		chunkSize := binary.LittleEndian.Uint32(chunkHeader[4:8])

		switch chunkID {
		case "fmt ":
			body := make([]byte, chunkSize)
			if _, err := io.ReadFull(f, body); err != nil {
				return nil, 0, fmt.Errorf("read fmt chunk: %w", err)
			}
			if chunkSize < 16 {
				return nil, 0, fmt.Errorf("fmt chunk too short (%d bytes)", chunkSize)
			}
			fmtTag := binary.LittleEndian.Uint16(body[0:2])
			if fmtTag != 1 {
				return nil, 0, fmt.Errorf("unsupported format tag %d (want 1 = PCM)", fmtTag)
			}
			channels := binary.LittleEndian.Uint16(body[2:4])
			if channels != 1 {
				return nil, 0, fmt.Errorf("unsupported channel count %d (want 1 = mono)", channels)
			}
			sampleRate = binary.LittleEndian.Uint32(body[4:8])
			bitsPerSample := binary.LittleEndian.Uint16(body[14:16])
			if bitsPerSample != 16 {
				return nil, 0, fmt.Errorf("unsupported bits per sample %d (want 16)", bitsPerSample)
			}
			gotFmt = true
		case "data":
			if !gotFmt {
				return nil, 0, fmt.Errorf("data chunk before fmt chunk")
			}
			samples := make([]int16, chunkSize/2)
			if err := binary.Read(f, binary.LittleEndian, &samples); err != nil {
				return nil, 0, fmt.Errorf("read data chunk: %w", err)
			}
			return samples, sampleRate, nil
		default:
			// Skip unknown chunks (LIST, JUNK, etc.). Chunks are
			// word-aligned; pad to even size when skipping.
			skip := int64(chunkSize)
			if skip%2 == 1 {
				skip++
			}
			if _, err := f.Seek(skip, io.SeekCurrent); err != nil {
				return nil, 0, fmt.Errorf("skip %s chunk: %w", chunkID, err)
			}
		}
	}
	return nil, 0, fmt.Errorf("no data chunk found in WAV")
}
