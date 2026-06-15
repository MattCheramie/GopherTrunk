package hunt

import (
	"math"
	"sort"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/survey"
)

// SignalSurvey is the inventory a live survey produces: every carrier the sweep
// detected, classified by modulation family and (where the class warranted it)
// decoded. It is the peer artifact of DiscoveredSystem — where DiscoveredSystem
// maps one trunked system, SignalSurvey catalogues everything on the air across
// the swept band(s). When the survey finds a trunking control channel it still
// folds it into System, so a survey is a strict superset of a hunt: it yields
// the same system map plus the surrounding signal landscape.
type SignalSurvey struct {
	StartedAt  time.Time        `json:"started_at"`
	FinishedAt time.Time        `json:"finished_at"`
	Signals    []DetectedSignal `json:"signals"`
	// System is the trunked system accumulated from the trunk-control carriers,
	// or nil when none was found. It is exported exactly like a hunt result.
	System *DiscoveredSystem `json:"system,omitempty"`
}

// DetectedSignal is one classified (and possibly decoded) carrier in a survey.
// Exactly one of Trunking / Analog / Pages is populated, per the Class.
type DetectedSignal struct {
	FreqHz       uint32             `json:"freq_hz"`
	SNRDb        float32            `json:"snr_db"`
	OccupiedBwHz uint32             `json:"occupied_bw_hz"`
	Class        survey.SignalClass `json:"class"`
	Confidence   float64            `json:"confidence"`
	BaudHz       float64            `json:"baud_hz,omitempty"`

	// Decode summary — set by the router for the carriers it could decode.
	Trunking *TrunkingRef         `json:"trunking,omitempty"`
	Analog   *survey.AnalogReport `json:"analog,omitempty"`
	Pages    []survey.PageRef     `json:"pages,omitempty"`

	// Features carries the raw classifier measurements for diagnostics.
	Features survey.ClassFeatures `json:"features"`
	// Error notes a per-carrier failure (tune/capture), leaving Class as-is.
	Error string `json:"error,omitempty"`
}

// TrunkingRef summarises the trunking decode of a carrier the router handed to
// the siglab identify path. It mirrors the fields of a CaptureReport that are
// meaningful at the inventory level; the full system map lives in Survey.System.
type TrunkingRef struct {
	Protocol   string  `json:"protocol"`
	Confidence float64 `json:"confidence"`
	Locked     bool    `json:"locked"`
	ControlHz  uint32  `json:"control_hz,omitempty"`
}

// finiteOr0 returns f unless it is NaN or ±Inf, in which case it returns 0.
// JSON has no encoding for non-finite floats, so a stray Inf/NaN from a
// divide-by-zero or log-of-zero on a dead/marginal carrier would otherwise
// break the whole event stream the moment it was marshaled (issue #648).
func finiteOr0(f float64) float64 {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0
	}
	return f
}

// sanitize clamps every non-finite float (NaN / ±Inf) on the signal — including
// the nested Features, Trunking, and Analog measurements — to 0 so the signal is
// always JSON-encodable. A marginal or dead carrier can produce an infinite SNR
// (noise-free residual), an infinite confidence (siglab scoring), or a -Inf
// analog power (log of zero); any one of those would fail json.Marshal and, on
// the WebSocket event stream, tear the live survey's connection down (issue #648).
func (ds *DetectedSignal) sanitize() {
	ds.SNRDb = float32(finiteOr0(float64(ds.SNRDb)))
	ds.Confidence = finiteOr0(ds.Confidence)
	ds.BaudHz = finiteOr0(ds.BaudHz)
	ds.Features.SNRDb = finiteOr0(ds.Features.SNRDb)
	ds.Features.EnvelopeCV = finiteOr0(ds.Features.EnvelopeCV)
	ds.Features.IFStd = finiteOr0(ds.Features.IFStd)
	ds.Features.IFKurtosis = finiteOr0(ds.Features.IFKurtosis)
	ds.Features.BaudHz = finiteOr0(ds.Features.BaudHz)
	ds.Features.BaudProminence = finiteOr0(ds.Features.BaudProminence)
	if ds.Trunking != nil {
		ds.Trunking.Confidence = finiteOr0(ds.Trunking.Confidence)
	}
	if ds.Analog != nil {
		ds.Analog.PowerDbFS = finiteOr0(ds.Analog.PowerDbFS)
		ds.Analog.CTCSSHz = finiteOr0(ds.Analog.CTCSSHz)
	}
}

// sortSignals orders the inventory by frequency so the output is deterministic
// (golden tests, stable diffs), mirroring DiscoveredSystem.sortAll.
func (s *SignalSurvey) sortSignals() {
	sort.Slice(s.Signals, func(i, j int) bool { return s.Signals[i].FreqHz < s.Signals[j].FreqHz })
}

// Counts tallies the inventory by broad category for status summaries.
func (s *SignalSurvey) Counts() (trunking, analog, paging, other int) {
	for _, sig := range s.Signals {
		switch {
		case sig.Class == survey.ClassTrunkControl || sig.Class == survey.ClassTrunkVoice:
			trunking++
		case len(sig.Pages) > 0 || sig.Class == survey.ClassPaging:
			paging++
		case sig.Class == survey.ClassNBFM || sig.Class == survey.ClassWideFM || sig.Class == survey.ClassAM:
			analog++
		default:
			other++
		}
	}
	return
}
