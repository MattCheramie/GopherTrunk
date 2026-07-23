package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

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
			case "f32", "u8", "cs16":
				spec.Format = f
			default:
				return iqCaptureSpec{}, fmt.Errorf("iq-capture: format must be u8, f32, or cs16, got %q", v)
			}
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
func runIQCapture(ctx context.Context, broker *iqtap.Broker, spec iqCaptureSpec, log *slog.Logger) error {
	if broker == nil {
		return fmt.Errorf("iq-capture: no broker for serial %q", spec.Serial)
	}

	f, err := os.Create(spec.Path)
	if err != nil {
		return fmt.Errorf("iq-capture: create %s: %w", spec.Path, err)
	}
	defer f.Close()

	sub := broker.Subscribe()
	defer sub.Close()

	// Stream each chunk straight to disk via siglab.CaptureWriter (one reused
	// scratch buffer, no whole-capture buffering). spec.Format is already
	// validated to u8/f32/cs16 by parseIQCaptureSpec.
	format, err := siglab.ParseSampleFormat(spec.Format)
	if err != nil {
		return fmt.Errorf("iq-capture: %w", err)
	}
	_, bytesPerSample := format.Decoder()
	enc := siglab.NewCaptureWriter(f, format)

	// Safety timer as an explicit select arm: the broker pauses fan-out when no
	// primary StreamIQ session is running, so a receive-then-check deadline
	// would block on sub.C forever if the primary stalls. The timer fires the
	// normal end-of-capture path after the requested duration regardless.
	timer := time.NewTimer(time.Duration(spec.Seconds) * time.Second)
	defer timer.Stop()
	deadline := time.Now().Add(time.Duration(spec.Seconds) * time.Second)
	log.Info("iq-capture: started",
		"serial", spec.Serial, "path", spec.Path,
		"seconds", spec.Seconds, "format", spec.Format)

	var samplesWritten int64
	for {
		select {
		case <-ctx.Done():
			return finishIQCapture(log, spec, f, samplesWritten, bytesPerSample, sub.Dropped(), ctx.Err())
		case <-timer.C:
			return finishIQCapture(log, spec, f, samplesWritten, bytesPerSample, sub.Dropped(), nil)
		case chunk, ok := <-sub.C:
			if !ok {
				return finishIQCapture(log, spec, f, samplesWritten, bytesPerSample, sub.Dropped(), errors.New("iq-capture: broker closed before capture finished"))
			}
			if err := enc.Write(chunk); err != nil {
				return finishIQCapture(log, spec, f, samplesWritten, bytesPerSample, sub.Dropped(), fmt.Errorf("write: %w", err))
			}
			samplesWritten += int64(len(chunk))
			if time.Now().After(deadline) {
				return finishIQCapture(log, spec, f, samplesWritten, bytesPerSample, sub.Dropped(), nil)
			}
		}
	}
}

func finishIQCapture(log *slog.Logger, spec iqCaptureSpec, f io.Closer, samples int64, bytesPerSample int, drops uint64, runErr error) error {
	// Best-effort close; the deferred Close in runIQCapture also fires
	// but we want flush/error visibility right here.
	closeErr := f.Close()
	args := []any{
		"serial", spec.Serial, "path", spec.Path,
		"samples", samples, "bytes", samples * int64(bytesPerSample),
		"drops", drops,
	}
	if runErr != nil {
		args = append(args, "err", runErr)
		log.Warn("iq-capture: stopped", args...)
		// Closing twice is harmless on *os.File; ignore the second
		// "file already closed" error.
		if closeErr != nil && !errors.Is(closeErr, os.ErrClosed) {
			return fmt.Errorf("%w (and close: %v)", runErr, closeErr)
		}
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
	if closeErr != nil && !errors.Is(closeErr, os.ErrClosed) {
		return closeErr
	}
	return nil
}
