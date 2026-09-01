package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/iqtap"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// iqCaptureSpec is the parsed form of the `--iq-capture` flag.
// Empty Serial means the flag wasn't set; runIQCapture is a no-op.
//
// File format default is "f32" (GNU Radio cfile, little-endian
// interleaved float32) — the IQ broker delivers complex64 chunks
// directly, so f32 round-trips losslessly through replay.go's
// decodeF32Replay. "u8" emits the rtl_sdr-native unsigned-8-bit shape
// for operators who want to feed the capture into other tooling.
type iqCaptureSpec struct {
	Serial  string
	Path    string
	Seconds int
	Format  string // "f32", "u8", or "cs16"
	// Decimate is the integer software-decimation factor applied to the
	// recorded stream: the capture is anti-alias filtered and written at
	// SDR-rate / Decimate, cutting the file size (and effective bandwidth)
	// by the same factor. 0 or 1 records the full SDR rate (the historical
	// behaviour). This is the remedy for a source like the USRP B210 whose
	// hardware sample-rate floor (~1 MS/s) is far above the bandwidth a
	// single narrowband channel needs — record at the floor, decimate in
	// software to a manageable long capture.
	Decimate int
}

// parseIQCaptureSpec parses "serial=<s>,path=<file>,seconds=<n>[,format=u8|f32]".
// Returns the zero value when input is empty (flag not set).
func parseIQCaptureSpec(s string) (iqCaptureSpec, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return iqCaptureSpec{}, nil
	}
	spec := iqCaptureSpec{Format: "f32"}
	for _, kv := range strings.Split(s, ",") {
		kv = strings.TrimSpace(kv)
		if kv == "" {
			continue
		}
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			return iqCaptureSpec{}, fmt.Errorf("iq-capture: malformed key=value %q", kv)
		}
		k = strings.ToLower(strings.TrimSpace(k))
		v = strings.TrimSpace(v)
		switch k {
		case "serial":
			spec.Serial = v
		case "path":
			spec.Path = v
		case "seconds":
			n, err := strconv.Atoi(v)
			if err != nil || n <= 0 {
				return iqCaptureSpec{}, fmt.Errorf("iq-capture: seconds must be a positive integer, got %q", v)
			}
			spec.Seconds = n
		case "format":
			f := strings.ToLower(v)
			switch f {
			case "f32", "u8", "cs16", "wav", "flac":
				spec.Format = f
			default:
				return iqCaptureSpec{}, fmt.Errorf("iq-capture: format must be u8, f32, cs16, wav, or flac, got %q", v)
			}
		case "decimate":
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 {
				return iqCaptureSpec{}, fmt.Errorf("iq-capture: decimate must be an integer >= 1, got %q", v)
			}
			spec.Decimate = n
		default:
			return iqCaptureSpec{}, fmt.Errorf("iq-capture: unknown key %q", k)
		}
	}
	if spec.Serial == "" {
		return iqCaptureSpec{}, errors.New("iq-capture: serial=<s> is required")
	}
	if spec.Path == "" {
		return iqCaptureSpec{}, errors.New("iq-capture: path=<file> is required")
	}
	if spec.Seconds == 0 {
		return iqCaptureSpec{}, errors.New("iq-capture: seconds=<n> is required")
	}
	return spec, nil
}

// runIQCapture subscribes to broker, writes the configured number of
// seconds of raw IQ to spec.Path, then returns. ctx cancels the
// capture early. Subscriber drops are counted in the broker's drop
// counter and surfaced once at the end so the operator knows the
// capture is incomplete — drops are NOT retried (the primary IQ
// stream is what the daemon needs unimpeded). Issue #402 diagnostic.
//
// When spec.Decimate > 1 the recorded stream is anti-alias decimated to
// SDR-rate / Decimate before it hits disk, and a self-describing
// `.metadata.json` sidecar is written next to the capture (carrying the
// reduced rate + centre) so the smaller file replays correctly without the
// operator having to remember the decimation factor.
func runIQCapture(ctx context.Context, broker *iqtap.Broker, spec iqCaptureSpec, log *slog.Logger) error {
	if broker == nil {
		return fmt.Errorf("iq-capture: no broker for serial %q", spec.Serial)
	}
	// spec.Format is already validated to u8/f32/cs16 by parseIQCaptureSpec.
	format, err := siglab.ParseSampleFormat(spec.Format)
	if err != nil {
		return fmt.Errorf("iq-capture: %w", err)
	}

	// Build the software decimator from the SDR rate the broker is streaming.
	// A factor of 0/1 is a no-op (nil ddc = byte-identical to the old path).
	baseRate := broker.SampleRateHz()
	ddc := newCaptureDecimator(baseRate, spec.Decimate)
	recRate := baseRate
	if ddc != nil {
		recRate = uint32(ddc.OutRateHz() + 0.5)
	}

	log.Info("iq-capture: started",
		"serial", spec.Serial, "path", spec.Path,
		"seconds", spec.Seconds, "format", spec.Format,
		"decimate", max(spec.Decimate, 1), "record_rate_hz", recRate)
	samples, bytesPerSample, drops, capErr := captureIQToFile(ctx, broker, spec.Path, format, spec.Seconds, ddc, recRate)
	if capErr == nil && ddc != nil {
		// Only decimated captures get a sidecar: their on-disk rate differs
		// from the configured sdr.sample_rate, so the file is unusable for
		// replay without it. A full-rate capture keeps the historical
		// no-sidecar behaviour (the operator supplies metadata explicitly).
		meta := &siglab.Metadata{
			Source:       fmt.Sprintf("gophertrunk iq-capture (decimate %d)", spec.Decimate),
			SampleRateHz: float64(recRate),
			CenterFreqHz: broker.CenterHz(),
			Format:       format.String(),
		}
		metaPath := strings.TrimSuffix(spec.Path, filepath.Ext(spec.Path)) + ".metadata.json"
		if werr := siglab.WriteMetadata(metaPath, meta); werr != nil {
			log.Warn("iq-capture: metadata sidecar write failed", "path", metaPath, "err", werr)
		}
	}
	return finishIQCapture(log, spec, samples, bytesPerSample, drops, capErr)
}

// newCaptureDecimator returns a down-converter that anti-alias decimates a
// baseRateHz stream by an integer factor, or nil when no decimation is wanted
// (factor <= 1, or an unknown base rate). It reuses ccdecoder.Downconverter —
// the same rational-resample path the `gophertrunk capture -bandwidth` slice
// and the live receiver DDC use — so a decimated capture has the identical
// >60 dB anti-alias shaping and no third DDC implementation is introduced
// (CLAUDE.md #764/#771: keep the DDC paths from diverging).
func newCaptureDecimator(baseRateHz uint32, factor int) *ccdecoder.Downconverter {
	if factor <= 1 || baseRateHz == 0 {
		return nil
	}
	return ccdecoder.NewDownconverter(float64(baseRateHz), float64(baseRateHz)/float64(factor))
}

// captureIQToFile owns the full lifecycle of one raw-IQ capture: create the
// file, subscribe to the broker, stream `seconds` of IQ through
// siglab.CaptureWriter (one reused scratch buffer, no whole-capture
// buffering), then close the file. It returns the samples written, the
// bytes-per-sample of the chosen format, and the subscriber drop count so the
// caller can log/report the result. ctx cancels the capture early.
//
// Shared by the `--iq-capture` CLI flag (runIQCapture) and the event-driven
// IQ auto-recorder, so both produce byte-identical captures.
//
// A non-nil ddc anti-alias decimates every chunk before it is encoded, so the
// file lands at the down-converter's output rate (software decimation). A nil
// ddc is a byte-identical pass-through of the historical full-rate path.
func captureIQToFile(ctx context.Context, broker *iqtap.Broker, path string, format siglab.SampleFormat, seconds int, ddc *ccdecoder.Downconverter, rateHz uint32) (samples int64, bytesPerSample int, drops uint64, err error) {
	_, bytesPerSample = format.Decoder()
	f, err := os.Create(path)
	if err != nil {
		return 0, bytesPerSample, 0, fmt.Errorf("iq-capture: create %s: %w", path, err)
	}

	sub := broker.Subscribe()
	defer sub.Close()
	enc, err := newCaptureEncoder(f, format, ddc, rateHz)
	if err != nil {
		f.Close()
		return 0, bytesPerSample, 0, fmt.Errorf("iq-capture: container %s: %w", path, err)
	}

	// Safety timer as an explicit select arm: the broker pauses fan-out when no
	// primary StreamIQ session is running, so a receive-then-check deadline
	// would block on sub.C forever if the primary stalls. The timer fires the
	// normal end-of-capture path after the requested duration regardless.
	timer := time.NewTimer(time.Duration(seconds) * time.Second)
	defer timer.Stop()
	deadline := time.Now().Add(time.Duration(seconds) * time.Second)

	streamErr := func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			case chunk, ok := <-sub.C:
				if !ok {
					return errors.New("iq-capture: broker closed before capture finished")
				}
				n, werr := enc.write(chunk)
				if werr != nil {
					return fmt.Errorf("write: %w", werr)
				}
				samples += int64(n)
				if time.Now().After(deadline) {
					return nil
				}
			}
		}
	}()

	// Finalize the container (patch wav lengths / close the flac stream), then
	// close the file. Report a finalize/close error only when the stream itself
	// succeeded (so the real cause isn't masked).
	if finErr := enc.finalize(); streamErr == nil && finErr != nil {
		streamErr = finErr
	}
	closeErr := f.Close()
	if streamErr == nil && closeErr != nil && !errors.Is(closeErr, os.ErrClosed) {
		streamErr = closeErr
	}
	return samples, bytesPerSample, sub.Dropped(), streamErr
}

// captureEncoder streams complex64 chunks to disk in a chosen sample format,
// optionally anti-alias decimating each chunk first so a capture can be
// recorded at a fraction of the SDR rate. A nil ddc makes write a
// pass-through: byte-identical to encoding through siglab.CaptureWriter
// directly, so an undecimated capture is unchanged.
type captureEncoder struct {
	enc     *siglab.IQContainer
	ddc     *ccdecoder.Downconverter // nil ⇒ no decimation
	scratch []complex64              // reused decimation output buffer
}

// newCaptureEncoder wraps f with the chosen container format and optional
// decimator. rateHz is the on-disk (post-decimation) sample rate, needed for
// the wav/flac headers.
func newCaptureEncoder(f *os.File, format siglab.SampleFormat, ddc *ccdecoder.Downconverter, rateHz uint32) (*captureEncoder, error) {
	enc, err := siglab.NewIQContainer(f, format, int(rateHz))
	if err != nil {
		return nil, err
	}
	return &captureEncoder{enc: enc, ddc: ddc}, nil
}

// finalize completes the container (patch/finalize) without closing the file.
func (c *captureEncoder) finalize() error { return c.enc.Finalize() }

// write encodes one chunk, decimating it first when a down-converter is set,
// and returns the number of samples actually written (post-decimation, so the
// caller's running total reflects the on-disk length rather than the input).
func (c *captureEncoder) write(chunk []complex64) (int, error) {
	out := chunk
	if c.ddc != nil {
		out = c.ddc.Process(c.scratch, chunk)
		c.scratch = out
	}
	if len(out) == 0 {
		// A short input chunk can leave the polyphase decimator with no
		// output this round; the samples are carried in its filter history.
		return 0, nil
	}
	if err := c.enc.Write(out); err != nil {
		return 0, err
	}
	return len(out), nil
}

func finishIQCapture(log *slog.Logger, spec iqCaptureSpec, samples int64, bytesPerSample int, drops uint64, runErr error) error {
	args := []any{
		"serial", spec.Serial, "path", spec.Path,
		"samples", samples, "bytes", samples * int64(bytesPerSample),
		"drops", drops,
	}
	if runErr != nil {
		args = append(args, "err", runErr)
		log.Warn("iq-capture: stopped", args...)
		return runErr
	}
	log.Info("iq-capture: finished", args...)
	if drops > 0 {
		// drops is the count of IQ chunks the capture subscriber dropped
		// because its buffer was full — i.e. the writer fell behind the
		// stream, NOT an SDR overflow. Each dropped chunk is a time gap
		// of missing samples in the .cfile, which breaks symbol timing
		// for any downstream decode (the DMR/P25 sync + clock loops
		// assume a continuous stream). Surface it loudly with the
		// remedy so a non-zero drop count is actionable rather than a
		// silent corrupt capture. The per-device SDR-overflow counter is
		// separate (Prometheus sdr_iq_underruns_total) — if that is also
		// climbing the bottleneck is upstream of the writer.
		log.Warn("iq-capture: dropped IQ chunks — the capture has time gaps that corrupt downstream decode",
			"drops", drops, "path", spec.Path,
			"cause", "capture writer fell behind the IQ stream (subscriber buffer full), not an SDR overflow",
			"remedy", "write to faster storage, lower sdr.sample_rate, or reduce concurrent IQ sinks; check Prometheus sdr_iq_underruns_total to rule out an SDR overrun")
	}
	return nil
}
