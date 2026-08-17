package api

import (
	"encoding/binary"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// WAV streaming envelope sizes. The header bakes in a near-2 GiB data
// chunk so streaming consumers keep reading until the connection
// closes; the Range responses below advertise a matching total so
// Safari (which only plays media from a byte-serving server) accepts
// the stream.
const (
	wavHeaderSize  = 44
	wavMaxDataSize = uint32(0x7FFFFFFF) // declared data chunk size
	wavTotalSize   = int64(wavHeaderSize) + int64(wavMaxDataSize)
)

// handleAudioStream emits a continuous WAV body assembled from PCM
// frames published by the live-audio composer. Designed for browser
// playback: any `<audio src="/api/v1/audio/stream">` element can
// stream this URL directly because the response is a single,
// open-ended WAV file with a fixed RIFF/WAVE header followed by
// concatenated 16-bit little-endian mono samples.
//
// Query parameters mirror the gRPC StreamAudio filter:
//
//	?device=<serial>     restrict to one SDR (repeatable)
//	?talkgroup=<id>      restrict to one talkgroup (repeatable)
//
// Empty filters match everything. When the publisher hasn't been
// wired (audio off, no composer), the endpoint returns 503.
//
// The handler disables the server-level WriteTimeout per-request via
// http.ResponseController so the long-lived connection doesn't get
// torn down mid-call.
//
// Range requests are honored so Safari (macOS/iOS) plays the stream:
// its media element refuses to play unless the server answers a
// `Range:` request with 206 + Accept-Ranges + Content-Range. Safari
// first issues a small probe (e.g. `bytes=0-1`) to learn the size,
// then an open-ended `bytes=0-` for the bulk body. Clients that don't
// send a Range header (Chrome, Firefox, curl) keep the plain 200
// streaming path.
func (s *Server) handleAudioStream(w http.ResponseWriter, r *http.Request) {
	if s.audioPub == nil {
		s.writeError(w, http.StatusServiceUnavailable, "audio stream not wired")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// Sample rate is fixed at 8000 Hz today (matches the composer's
	// recorder rate; see audio_publisher.go buildPCMFrame). When
	// the composer grows variable rates, the AudioController's
	// SampleRate is the source of truth.
	rate := uint32(8000)
	if s.audio != nil && s.audio.SampleRate() > 0 {
		rate = s.audio.SampleRate()
	}

	start, end, hasRange := parseRange(r.Header.Get("Range"))

	// Safari's size probe: a bounded range fully inside the
	// deterministic WAV header. Serve the exact header bytes and
	// close — no live subscription required.
	if hasRange && end >= 0 && end < wavHeaderSize {
		header := streamingWAVHeader(rate)
		body := header[start : end+1]
		w.Header().Set("Content-Type", "audio/wav")
		w.Header().Set("Cache-Control", "no-cache, no-store")
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, wavTotalSize))
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(body)
		return
	}

	// Long-lived response from here on: clear the write deadline so
	// the connection isn't torn down mid-call.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})

	filter := parseAudioStreamFilter(r)
	sub := s.audioPub.Subscribe(filter)
	defer s.audioPub.Unsubscribe(sub)

	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.Header().Set("X-Accel-Buffering", "no")

	if hasRange {
		// Open-ended `bytes=N-` (or a range extending past the
		// header). A live stream has no real seek target, so we
		// advertise a matching Content-Range and stream live; the
		// only meaningful offset is whether the caller still needs
		// the leading header bytes.
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, wavTotalSize-1, wavTotalSize))
		w.WriteHeader(http.StatusPartialContent)
		if start < wavHeaderSize {
			if _, err := w.Write(streamingWAVHeader(rate)[start:]); err != nil {
				return
			}
		}
	} else {
		// No Range: plain open-ended stream. Chrome, Firefox, and
		// curl play streaming WAV when the data chunk claims a
		// near-2 GiB length and the response uses chunked transfer
		// encoding.
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(streamingWAVHeader(rate)); err != nil {
			return
		}
	}
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.closing():
			// See handleSSE: Shutdown waits on this request without
			// cancelling it, so the server's stop signal is what ends it.
			return
		case frame, ok := <-sub.ch:
			if !ok {
				return
			}
			pcm := frame.GetPcm()
			if pcm == nil || len(pcm.GetSamples()) == 0 {
				continue
			}
			// Frames are already 16-bit little-endian mono per
			// buildPCMFrame; write them straight through.
			if _, err := w.Write(pcm.GetSamples()); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// parseRange parses a single "bytes=start-end" Range header. It
// returns start, end (end == -1 for an open-ended "bytes=N-"), and ok.
// ok is false for an empty header, a non-"bytes=" unit, suffix ranges
// ("bytes=-N"), multi-range requests, or any malformed input — callers
// then fall back to the plain 200 stream, which is safe for non-Safari
// clients.
func parseRange(h string) (start int64, end int64, ok bool) {
	h = strings.TrimSpace(h)
	const prefix = "bytes="
	if !strings.HasPrefix(h, prefix) {
		return 0, 0, false
	}
	spec := strings.TrimPrefix(h, prefix)
	if strings.Contains(spec, ",") { // multi-range unsupported
		return 0, 0, false
	}
	dash := strings.IndexByte(spec, '-')
	if dash <= 0 { // missing start (suffix range) or no dash
		return 0, 0, false
	}
	startStr := strings.TrimSpace(spec[:dash])
	endStr := strings.TrimSpace(spec[dash+1:])
	start, err := strconv.ParseInt(startStr, 10, 64)
	if err != nil || start < 0 {
		return 0, 0, false
	}
	if endStr == "" {
		return start, -1, true
	}
	end, err = strconv.ParseInt(endStr, 10, 64)
	if err != nil || end < start {
		return 0, 0, false
	}
	return start, end, true
}

// parseAudioStreamFilter pulls the device + talkgroup filters off
// the URL query, mirroring the gRPC StreamAudioRequest fields.
func parseAudioStreamFilter(r *http.Request) AudioSubFilter {
	q := r.URL.Query()
	devices := q["device"]
	tgs := q["talkgroup"]
	out := AudioSubFilter{
		DeviceSerials: nil,
		TalkgroupIDs:  nil,
	}
	for _, d := range devices {
		d = strings.TrimSpace(d)
		if d != "" {
			out.DeviceSerials = append(out.DeviceSerials, d)
		}
	}
	for _, t := range tgs {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if n, err := strconv.ParseUint(t, 10, 32); err == nil {
			out.TalkgroupIDs = append(out.TalkgroupIDs, uint32(n))
		}
	}
	return out
}

// streamingWAVHeader builds a 44-byte canonical RIFF/WAVE header for
// 16-bit little-endian mono PCM at the given sample rate. The two
// size fields are set to (2^31 - 36) and (2^31), the largest values
// a WAV reader will accept, so a streaming consumer keeps reading
// until the underlying connection closes. Browsers tolerate the
// oversized header for continuous playback.
func streamingWAVHeader(sampleRate uint32) []byte {
	const (
		numChannels   uint16 = 1
		bitsPerSample uint16 = 16
		audioFormat   uint16 = 1 // PCM
	)
	byteRate := sampleRate * uint32(numChannels) * uint32(bitsPerSample) / 8
	blockAlign := numChannels * bitsPerSample / 8

	chunkSize := wavMaxDataSize - 36 // best-effort: data chunk + header overhead

	buf := make([]byte, 44)
	copy(buf[0:4], "RIFF")
	binary.LittleEndian.PutUint32(buf[4:8], chunkSize)
	copy(buf[8:12], "WAVE")
	copy(buf[12:16], "fmt ")
	binary.LittleEndian.PutUint32(buf[16:20], 16) // PCM fmt chunk size
	binary.LittleEndian.PutUint16(buf[20:22], audioFormat)
	binary.LittleEndian.PutUint16(buf[22:24], numChannels)
	binary.LittleEndian.PutUint32(buf[24:28], sampleRate)
	binary.LittleEndian.PutUint32(buf[28:32], byteRate)
	binary.LittleEndian.PutUint16(buf[32:34], blockAlign)
	binary.LittleEndian.PutUint16(buf[34:36], bitsPerSample)
	copy(buf[36:40], "data")
	binary.LittleEndian.PutUint32(buf[40:44], wavMaxDataSize)
	return buf
}
