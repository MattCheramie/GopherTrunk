package api

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/hunt"
	"github.com/MattCheramie/GopherTrunk/internal/survey"
)

// TestEventToDTOHuntCandidateFinite confirms a normal hunt candidate marshals
// over the WS/SSE envelope. The hunt feature publishes hunt.DetectedSignal under
// events.KindHuntLiveCandidate, which eventToDTO passes straight through, so its
// floats must always be encodable.
func TestEventToDTOHuntCandidateFinite(t *testing.T) {
	dto := eventToDTO(events.Event{
		Kind: events.KindHuntLiveCandidate,
		Payload: hunt.DetectedSignal{
			FreqHz: 440_087_400, SNRDb: 28.0, Class: survey.ClassNBFM, Confidence: 0.8,
			Analog: &survey.AnalogReport{Active: true, PowerDbFS: -12.5, CTCSSHz: 67.0},
		},
	})
	if _, err := json.Marshal(dto); err != nil {
		t.Fatalf("marshal finite hunt candidate: %v", err)
	}
}

// TestEventToDTONonFiniteIsZeroed pins the issue #648 fix: eventToDTO strips a
// non-finite float (here +Inf, the value the field report's "json: unsupported
// value: +Inf" hit) from the payload, so json.Marshal now succeeds and the field
// crosses the wire as 0 instead of tearing the live stream down.
func TestEventToDTONonFiniteIsZeroed(t *testing.T) {
	dto := eventToDTO(events.Event{
		Kind: events.KindHuntLiveCandidate,
		Payload: hunt.DetectedSignal{
			FreqHz: 468_475_000, Class: survey.ClassPSK,
			Features: survey.ClassFeatures{SNRDb: math.Inf(1)},
		},
	})
	b, err := json.Marshal(dto)
	if err != nil {
		t.Fatalf("marshal sanitized payload: %v", err)
	}
	var got struct {
		Payload struct {
			Features struct {
				SNRDb float64 `json:"snr_db"`
			} `json:"features"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Payload.Features.SNRDb != 0 {
		t.Errorf("features.snr_db = %v, want 0 (sanitized)", got.Payload.Features.SNRDb)
	}
}
