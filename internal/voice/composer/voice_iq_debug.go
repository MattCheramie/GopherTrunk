package composer

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// Per-call voice-channel IQ debug capture — the voice half of the
// "diagnostic container" workflow (the operator's feature request: a
// metadata + control-IQ + voice-IQ triplet per call, so a hopped voice
// channel's raw samples are on disk the moment a decode sounds wrong).
// When enabled, handleStart tees each call's channelised IQ — EXACTLY the
// stream the voice chain decodes, at the same rate — into
// `<dir>/<UTC>_<system>_tg<group>_<freq>_<rate>_voice.cs16` plus a
// self-describing `.metadata.json` sidecar (siglab schema, so the file
// drops straight into `gophertrunk replay` / siglab). The control-channel
// half is baseband.auto_record with on_voice_grant.
//
// The tee is lossless toward the decode chain (the chain sees the same
// chunks with the same backpressure); the DISK side is the best-effort
// one. A writer that cannot keep up stops the capture and marks the
// sidecar truncated rather than dropping chunks mid-file — a gap silently
// desynchronises the stream and every later DSP conclusion is wrong with
// nothing to show for it (the diversity-capture alignment lesson).

// VoiceIQDebugConfig mirrors config.VoiceIQDebugConfig for the composer.
type VoiceIQDebugConfig struct {
	Enabled bool
	Dir     string
	// Format is the on-disk container: cs16 (default), wav, or flac (and the
	// raw f32/u8 sample formats). The zero value (FormatU8) is remapped to cs16
	// by the writer so an unset format keeps the historical default.
	Format siglab.SampleFormat
	// MaxBytes caps one call's capture; 0 defaults to
	// voiceIQDebugDefaultMaxBytes.
	MaxBytes int64
}

// voiceIQDebugDefaultMaxBytes bounds one call's capture when the config
// doesn't: 512 MB (~45 min of 48 kHz cs16, ~55 s of 2.4 MS/s cs16).
const voiceIQDebugDefaultMaxBytes = 512 << 20

// voiceIQDebugQueue is the writer goroutine's chunk queue depth. At the
// channelised 48 kHz a chunk is a few ms of IQ, so this rides out normal
// filesystem latency; when the queue fills the capture ends (truncated),
// never stalls the decode chain.
const voiceIQDebugQueue = 256

// teeVoiceIQ wraps in with a per-call IQ capture. The returned channel
// delivers the same chunks with the same semantics; the capture goroutine
// owns the file and closes it when the input channel closes or ctx ends.
func (c *Composer) teeVoiceIQ(ctx context.Context, in <-chan []complex64, cs trunking.CallStart, rateHz float64) <-chan []complex64 {
	maxBytes := c.voiceIQDebug.MaxBytes
	if maxBytes <= 0 {
		maxBytes = voiceIQDebugDefaultMaxBytes
	}
	w, err := newVoiceIQDebugWriter(c.voiceIQDebug, cs, rateHz, maxBytes, c.log)
	if err != nil {
		c.log.Warn("composer: voice IQ debug capture failed to open; call decodes without capture",
			"system", cs.Grant.System, "group", cs.Grant.GroupID, "err", err)
		return in
	}
	out := make(chan []complex64)
	go func() {
		defer close(out)
		defer w.close()
		for {
			select {
			case <-ctx.Done():
				return
			case chunk, ok := <-in:
				if !ok {
					return
				}
				w.offer(chunk)
				select {
				case <-ctx.Done():
					return
				case out <- chunk:
				}
			}
		}
	}()
	return out
}

// voiceIQDebugWriter owns one call's capture file + sidecar.
type voiceIQDebugWriter struct {
	path     string
	metaPath string
	meta     *siglab.Metadata
	format   siglab.SampleFormat
	rateHz   float64
	log      *slog.Logger
	queue    chan []complex64
	done     chan struct{}

	// samples is written by the disk goroutine and read after done closes;
	// truncated is shared between the chain-side offer and the disk
	// goroutine, hence atomic.
	samples   int64
	truncated atomic.Bool
}

// newVoiceIQDebugWriter opens the capture files and starts the disk
// goroutine. The sidecar is written immediately (so a crashed daemon still
// leaves a replayable pair) and rewritten at close with the final counts.
func newVoiceIQDebugWriter(cfg VoiceIQDebugConfig, cs trunking.CallStart, rateHz float64, maxBytes int64, log *slog.Logger) (*voiceIQDebugWriter, error) {
	dir := cfg.Dir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	at := cs.StartedAt
	if at.IsZero() {
		at = time.Now()
	}
	base := fmt.Sprintf("%s_%s_tg%d_%d_%.0f_voice",
		at.UTC().Format("20060102T150405.000"),
		sanitizeFileToken(cs.Grant.System), cs.Grant.GroupID,
		cs.Grant.FrequencyHz, rateHz)
	base = strings.ReplaceAll(base, ".", "_") // keep one extension dot
	// The unset zero value (FormatU8) is not a valid voice-IQ container; treat
	// it as the historical cs16 default. wav/flac wrap the same 16-bit body.
	format := cfg.Format
	if format == siglab.FormatU8 {
		format = siglab.FormatS16
	}
	w := &voiceIQDebugWriter{
		path:     filepath.Join(dir, base+"."+format.String()),
		metaPath: filepath.Join(dir, base+".metadata.json"),
		format:   format,
		rateHz:   rateHz,
		log:      log,
		queue:    make(chan []complex64, voiceIQDebugQueue),
		done:     make(chan struct{}),
	}
	w.meta = &siglab.Metadata{
		Protocol:     cs.Grant.Protocol,
		Source:       "gophertrunk voice_iq_debug (per-call voice-channel IQ)",
		SampleRateHz: rateHz,
		CenterFreqHz: cs.Grant.FrequencyHz,
		Format:       format.String(),
		System: map[string]string{
			"system":     cs.Grant.System,
			"talkgroup":  strconv.FormatUint(uint64(cs.Grant.GroupID), 10),
			"source_id":  strconv.FormatUint(uint64(cs.Grant.SourceID), 10),
			"timeslot":   strconv.Itoa(int(cs.Grant.Timeslot)),
			"rfss":       strconv.Itoa(int(cs.Grant.RFSSID)),
			"site":       strconv.Itoa(int(cs.Grant.SiteID)),
			"nac":        strconv.Itoa(int(cs.Grant.NAC)),
			"device":     cs.DeviceSerial,
			"granted_at": at.UTC().Format(time.RFC3339Nano),
			"encrypted":  strconv.FormatBool(cs.Grant.Encrypted),
			"emergency":  strconv.FormatBool(cs.Grant.Emergency),
		},
	}
	f, err := os.Create(w.path)
	if err != nil {
		return nil, err
	}
	if err := siglab.WriteMetadata(w.metaPath, w.meta); err != nil {
		log.Warn("composer: voice IQ debug metadata write failed", "path", w.metaPath, "err", err)
	}
	go w.run(f, maxBytes)
	return w, nil
}

// offer hands a chunk to the disk goroutine without ever blocking the
// decode chain. The chunk is copied (the producer may reuse its buffer).
// A full queue trips truncation: the writer stops accepting and the
// sidecar records it, keeping the on-disk stream gap-free up to that
// point.
func (w *voiceIQDebugWriter) offer(chunk []complex64) {
	if w.truncatedNow() {
		return
	}
	cp := append([]complex64(nil), chunk...)
	select {
	case w.queue <- cp:
	default:
		w.markTruncated("writer backlog")
	}
}

func (w *voiceIQDebugWriter) truncatedNow() bool {
	select {
	case <-w.done:
		return true
	default:
		return w.truncated.Load()
	}
}

func (w *voiceIQDebugWriter) markTruncated(reason string) {
	if !w.truncated.Swap(true) {
		w.log.Warn("composer: voice IQ debug capture truncated", "path", w.path, "reason", reason)
	}
}

// close ends the capture: drains what is queued, closes the file, and
// rewrites the sidecar with the final sample count / truncation flag.
func (w *voiceIQDebugWriter) close() {
	close(w.queue)
	<-w.done
	w.meta.System["samples"] = strconv.FormatInt(w.samples, 10)
	if w.truncated.Load() {
		w.meta.System["truncated"] = "true"
	}
	if err := siglab.WriteMetadata(w.metaPath, w.meta); err != nil {
		w.log.Warn("composer: voice IQ debug metadata rewrite failed", "path", w.metaPath, "err", err)
	}
	w.log.Info("composer: voice IQ debug capture closed",
		"path", w.path, "samples", w.samples, "truncated", w.truncated.Load())
}

// run is the disk goroutine: encodes queued chunks into the chosen container
// (cs16/wav/flac) until the queue closes or the byte cap is hit. The byte cap
// counts the uncompressed 16-bit body (len(chunk)*4) so it bounds the capture
// the same way regardless of container compression.
func (w *voiceIQDebugWriter) run(f *os.File, maxBytes int64) {
	defer close(w.done)
	defer f.Close()
	enc, err := siglab.NewIQContainer(f, w.format, int(w.rateHz+0.5))
	if err != nil {
		w.markTruncated("open container: " + err.Error())
		// Drain the queue so offer() doesn't block the (already-closing) chain.
		for range w.queue {
		}
		return
	}
	var written int64
	for chunk := range w.queue {
		if w.truncated.Load() {
			continue // drain without writing once truncated
		}
		if written+int64(len(chunk)*4) > maxBytes {
			w.markTruncated("max_mb cap")
			continue
		}
		if err := enc.Write(chunk); err != nil {
			w.markTruncated("write error: " + err.Error())
			continue
		}
		written += int64(len(chunk) * 4)
		w.samples += int64(len(chunk))
	}
	if err := enc.Finalize(); err != nil {
		w.markTruncated("finalize error: " + err.Error())
	}
}

// sanitizeFileToken keeps a config-supplied name filesystem-safe.
func sanitizeFileToken(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '-'
		}
	}, s)
}
