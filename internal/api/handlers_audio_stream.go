package api

import (
	"encoding/binary"
	"net/http"
	"strconv"
	"strings"
	"time"
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
func (s *Server) handleAudioStream(w http.ResponseWriter, r *http.Request) {
	if s.audioPub == nil {
		writeError(w, http.StatusServiceUnavailable, "audio stream not wired")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})

	filter := parseAudioStreamFilter(r)
	sub := s.audioPub.Subscribe(filter)
	defer s.audioPub.Unsubscribe(sub)

	// Sample rate is fixed at 8000 Hz today (matches the composer's
	// recorder rate; see audio_publisher.go buildPCMFrame). When
	// the composer grows variable rates, the AudioController's
	// SampleRate is the source of truth.
	rate := uint32(8000)
	if s.audio != nil && s.audio.SampleRate() > 0 {
		rate = s.audio.SampleRate()
	}

	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	// No Content-Length: this is an open-ended stream. Chrome,
	// Firefox, and Safari play streaming WAV when the data chunk
	// claims a near-2 GiB length and the response uses chunked
	// transfer encoding.
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(streamingWAVHeader(rate)); err != nil {
		return
	}
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
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

	const maxDataSize uint32 = 0x7FFFFFFF
	chunkSize := maxDataSize - 36 // best-effort: data chunk + header overhead

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
	binary.LittleEndian.PutUint32(buf[40:44], maxDataSize)
	return buf
}
