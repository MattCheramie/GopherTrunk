package api

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// AudioStatusDTO is the JSON shape returned by GET /api/v1/audio. Mirrors
// the AudioController interface so the TUI doesn't need to know how
// the daemon plumbed the player + recorder together.
type AudioStatusDTO struct {
	// BackendEnabled is true when a real audio sink is attached.
	// False = audio.enabled was off in config or the backend
	// failed to init; PATCH still works but takes effect only on
	// the recorder gate.
	BackendEnabled bool `json:"backend_enabled"`
	// SampleRate is the host playback rate in Hz.
	SampleRate uint32 `json:"sample_rate"`
	// Volume is the software gain (0..1).
	Volume float32 `json:"volume"`
	// Muted reports the mute state.
	Muted bool `json:"muted"`
	// RecordingEnabled is the recorder's "create new sessions"
	// gate. In-flight sessions are unaffected.
	RecordingEnabled bool `json:"recording_enabled"`
	// DropsTotal is a monotonically increasing counter of PCM
	// samples lost because the playback queue was full.
	DropsTotal uint64 `json:"drops_total"`
}

// audioPatchRequest is the body of PATCH /api/v1/audio. Every field
// is a pointer so JSON-omitted fields are not zeroed: only supplied
// fields are applied. Matches the talkgroup-patch convention.
type audioPatchRequest struct {
	Volume    *float32 `json:"volume"`
	Muted     *bool    `json:"muted"`
	Recording *bool    `json:"recording_enabled"`
}

// handleAudioStatus reports the live-audio cockpit's current state.
//
//	GET /api/v1/audio
//
// Responses:
//
//	200 with an AudioStatusDTO
//	503 if the daemon doesn't have an AudioController wired
func (s *Server) handleAudioStatus(w http.ResponseWriter, _ *http.Request) {
	if s.audio == nil {
		writeError(w, http.StatusServiceUnavailable, "audio not wired")
		return
	}
	writeJSON(w, http.StatusOK, audioStatusDTO(s.audio))
}

// handleAudioPatch applies one or more cockpit mutations and returns
// the updated state. Each field is optional; omit a field to leave
// that knob alone.
//
//	PATCH /api/v1/audio
//	Content-Type: application/json
//	{"volume":0.5,"muted":false,"recording_enabled":true}
//
// Responses:
//
//	200 with the updated AudioStatusDTO
//	400 if the body is malformed or out of range
//	503 if the daemon doesn't have an AudioController wired
func (s *Server) handleAudioPatch(w http.ResponseWriter, r *http.Request) {
	if s.audio == nil {
		writeError(w, http.StatusServiceUnavailable, "audio not wired")
		return
	}
	var req audioPatchRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			writeError(w, http.StatusBadRequest, "invalid json body")
			return
		}
	}
	if req.Volume == nil && req.Muted == nil && req.Recording == nil {
		writeError(w, http.StatusBadRequest, "supply volume, muted, and/or recording_enabled")
		return
	}
	if req.Volume != nil {
		if *req.Volume < 0 || *req.Volume > 1 {
			writeError(w, http.StatusBadRequest, "volume must be between 0.0 and 1.0")
			return
		}
		s.audio.SetVolume(*req.Volume)
	}
	if req.Muted != nil {
		s.audio.SetMuted(*req.Muted)
	}
	if req.Recording != nil {
		s.audio.SetRecordingEnabled(*req.Recording)
	}
	state := audioStatusDTO(s.audio)
	// Push the new state to SSE subscribers so other TUIs / web
	// clients converge instantly instead of waiting for the next
	// 3 s poll tick. The events.Bus passes the DTO through to the
	// SSE pump's default payload case unchanged.
	if s.bus != nil {
		s.bus.Publish(events.Event{
			Kind:      events.KindAudioState,
			Timestamp: time.Now().UTC(),
			Payload:   state,
		})
	}
	writeJSON(w, http.StatusOK, state)
}

func audioStatusDTO(a AudioController) AudioStatusDTO {
	return AudioStatusDTO{
		BackendEnabled:   a.BackendEnabled(),
		SampleRate:       a.SampleRate(),
		Volume:           a.Volume(),
		Muted:            a.Muted(),
		RecordingEnabled: a.RecordingEnabled(),
		DropsTotal:       a.DropsTotal(),
	}
}
