package voice

import (
	"fmt"
	"math"
	"os"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/loudness"
)

// NormalizeConfig configures per-call EBU R128 / BS.1770 loudness
// normalization of a finished recording. It mirrors ffmpeg's two-pass
// linear "loudnorm": measure the whole call's integrated loudness, apply
// a single linear gain toward TargetLUFS, then back the gain off if the
// true peak would exceed TruePeakDBTP. Pure gain — no compression — so
// the within-call dynamics the vocoder AGC produced are preserved.
type NormalizeConfig struct {
	// Enabled turns normalization on. When false the recorder never
	// rewrites WAVs (the default).
	Enabled bool
	// TargetLUFS is the integrated-loudness target (e.g. -16).
	TargetLUFS float64
	// TruePeakDBTP is the true-peak ceiling in dBTP (e.g. -1.5).
	TruePeakDBTP float64
	// MaxBoostDB caps how much gain may be applied in either direction,
	// so a near-silent call doesn't amplify hiss up to the target.
	MaxBoostDB float64
}

// params returns the loudness.NormalizeParams for this config, with
// defaults filled in for any zero field.
func (c NormalizeConfig) params() loudness.NormalizeParams {
	return loudness.NormalizeParams{
		TargetLUFS:   c.TargetLUFS,
		TruePeakDBTP: c.TruePeakDBTP,
		MaxBoostDB:   c.MaxBoostDB,
	}.WithDefaults()
}

// withDefaults returns a copy of cfg with zero numeric fields replaced by
// the package defaults. Only meaningful when Enabled.
func (c NormalizeConfig) withDefaults() NormalizeConfig {
	p := c.params()
	c.TargetLUFS, c.TruePeakDBTP, c.MaxBoostDB = p.TargetLUFS, p.TruePeakDBTP, p.MaxBoostDB
	return c
}

// NormalizeWAVFile applies loudness normalization to the WAV at path,
// filling in default parameters for any zero fields in cfg. It is the
// exported entry point for offline tools (e.g. `gophertrunk decode
// -normalize`); the recorder uses the internal path with defaults already
// applied at construction.
func NormalizeWAVFile(path string, cfg NormalizeConfig) error {
	return normalizeWAVFile(path, cfg.withDefaults())
}

// normalizeWAVFile measures the integrated loudness of the WAV at path,
// computes a single linear gain toward cfg.TargetLUFS (clamped to
// ±MaxBoostDB and limited so the true peak stays under TruePeakDBTP),
// applies it, and atomically rewrites the file in place.
//
// It returns nil without rewriting when the audio is too short or too
// quiet to measure (IntegratedLUFS reports !ok) — there is nothing
// meaningful to normalize. cfg is assumed to already carry defaults.
func normalizeWAVFile(path string, cfg NormalizeConfig) error {
	// Content-sniffed read + same-container rewrite, so a flac recording is
	// normalized in place as flac.
	isFLAC := isFLACAudioFile(path)
	samples, sampleRate, err := ReadAudioSamples(path)
	if err != nil {
		return fmt.Errorf("voice/normalize: read %s: %w", path, err)
	}
	if len(samples) == 0 || sampleRate == 0 {
		return nil
	}

	// int16 -> float64 in [-1, 1).
	fs := make([]float64, len(samples))
	for i, s := range samples {
		fs[i] = float64(s) / 32768.0
	}

	gain, ok := loudness.NormalizeGain(fs, int(sampleRate), cfg.params())
	if !ok {
		return nil // silent / too short — leave the recording untouched
	}

	// Apply gain, round to int16 with a hard clamp as a rounding-safety
	// net (the true-peak limit should already keep us in range).
	for i, v := range fs {
		samples[i] = clampInt16(v * gain * 32768.0)
	}

	if isFLAC {
		return rewriteAudio(path, samples, sampleRate, "flac")
	}
	return rewriteWAV(path, samples, sampleRate)
}

// clampInt16 rounds and saturates a float sample value to int16.
func clampInt16(v float64) int16 {
	r := math.Round(v)
	if r > 32767 {
		return 32767
	}
	if r < -32768 {
		return -32768
	}
	return int16(r)
}

// rewriteWAV writes samples to a temp file next to path and atomically
// renames it over the original, so a crash mid-write can't truncate the
// recording.
func rewriteWAV(path string, samples []int16, sampleRate uint32) error {
	return rewriteAudio(path, samples, sampleRate, "wav")
}

// rewriteAudio is rewriteWAV generalized over the recording container.
func rewriteAudio(path string, samples []int16, sampleRate uint32, format string) error {
	tmp := path + ".tmp"
	w, err := NewAudioFileWriter(tmp, sampleRate, format)
	if err != nil {
		return fmt.Errorf("voice/normalize: create temp: %w", err)
	}
	if err := w.WriteSamples(samples); err != nil {
		w.Close()
		os.Remove(tmp)
		return fmt.Errorf("voice/normalize: write temp: %w", err)
	}
	if err := w.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("voice/normalize: close temp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("voice/normalize: rename: %w", err)
	}
	return nil
}
