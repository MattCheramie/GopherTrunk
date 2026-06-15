package api

import (
	"encoding/json"
	"errors"
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

// TestEventToDTONonFiniteIsUnsupported documents the failure mode behind issue
// #648: a payload carrying a non-finite float (here +Inf, as the field report's
// "json: unsupported value: +Inf" showed) is rejected by encoding/json. The WS
// handler must therefore skip such a frame rather than tear the connection down,
// and the survey now sanitizes its signals so this can't reach the wire in
// practice. This test pins both halves of that contract.
func TestEventToDTONonFiniteIsUnsupported(t *testing.T) {
	dto := eventToDTO(events.Event{
		Kind: events.KindHuntLiveCandidate,
		Payload: hunt.DetectedSignal{
			FreqHz: 468_475_000, Class: survey.ClassPSK,
			Features: survey.ClassFeatures{SNRDb: math.Inf(1)},
		},
	})
	_, err := json.Marshal(dto)
	if err == nil {
		t.Fatal("expected json.Marshal to reject a non-finite payload")
	}
	var ue *json.UnsupportedValueError
	if !errors.As(err, &ue) {
		t.Fatalf("error = %v, want *json.UnsupportedValueError", err)
	}
}
