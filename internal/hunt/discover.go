package hunt

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// CaptureInput is one IQ capture to fold into a discovery — typically a
// recording centered on a suspected control channel. The hunt orchestrator
// identifies the protocol (unless Protocol is set), decodes it, and
// accumulates the result into the running system map.
type CaptureInput struct {
	Path         string
	Format       siglab.SampleFormat
	SampleRateHz float64
	// FrequencyHz is the capture's nominal center frequency (informational;
	// the decoded lock frequency is what gets recorded).
	FrequencyHz uint32
	AutoTune    bool
	Conjugate   bool
	IQCorrect   bool
	// Protocol forces a decoder; trunking.ProtocolUnknown (the zero value)
	// triggers auto-identification across every protocol.
	Protocol trunking.Protocol
	// IdentifyMaxSamples caps the prefix the identifier scans (0 ⇒ a built-in
	// default prefix). Only consulted when Protocol is unknown.
	IdentifyMaxSamples int64
}

// DiscoverConfig carries operator metadata and thresholds for a discovery run.
type DiscoverConfig struct {
	Name          string
	State         string
	County        string
	Location      string
	MinConfidence float64 // skip auto-identified captures below this (0 ⇒ 0.40)
	Log           *slog.Logger
}

// CaptureReport records what happened to one capture so the CLI/cockpit can
// explain the outcome (decoded, skipped as not-trunked, errored).
type CaptureReport struct {
	Path       string  `json:"path"`
	Protocol   string  `json:"protocol"`
	Confidence float64 `json:"confidence"`
	Locked     bool    `json:"locked"`
	ControlHz  uint32  `json:"control_hz,omitempty"`
	Talkgroups int     `json:"talkgroups"`
	// ErrorRate is the decode-error events per 1000 symbols from the decode
	// (siglab Signal.DecodeErrorRate), a protocol-neutral demod-quality proxy.
	// Zero when no signal analysis was produced. Used by the auto-gain sweep.
	ErrorRate  float64 `json:"error_rate,omitempty"`
	Skipped    bool    `json:"skipped"`
	SkipReason string  `json:"skip_reason,omitempty"`
	Error      string  `json:"error,omitempty"`
	// Verdict is the wideband survey's system-level summary line (set only by
	// DiscoverWideband). Empty for single-carrier captures.
	Verdict string `json:"verdict,omitempty"`
}

// Discover folds every capture into a single DiscoveredSystem. Captures whose
// protocol can't be identified with sufficient confidence (and weren't given
// an explicit protocol) are skipped, not errored, so a wideband sweep that
// surfaced non-trunked carriers degrades gracefully. The per-capture reports
// are always returned, even on a nil error, so the caller can show progress.
func Discover(captures []CaptureInput, cfg DiscoverConfig) (*DiscoveredSystem, []CaptureReport, error) {
	log := cfg.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	minConf := cfg.MinConfidence
	if minConf <= 0 {
		minConf = 0.40
	}

	sys := &DiscoveredSystem{
		Name:     cfg.Name,
		State:    cfg.State,
		County:   cfg.County,
		Location: cfg.Location,
	}
	reports := make([]CaptureReport, 0, len(captures))

	for _, cap := range captures {
		f, err := os.Open(cap.Path)
		if err != nil {
			reports = append(reports, CaptureReport{Path: cap.Path, Error: fmt.Sprintf("open: %v", err)})
			continue
		}
		rep := decodeAndAccumulate(sys, f, cap.Path, decodeParams{
			Protocol:           cap.Protocol,
			Format:             cap.Format,
			SampleRateHz:       cap.SampleRateHz,
			FrequencyHz:        cap.FrequencyHz,
			AutoTune:           cap.AutoTune,
			Conjugate:          cap.Conjugate,
			IQCorrect:          cap.IQCorrect,
			IdentifyMaxSamples: cap.IdentifyMaxSamples,
			MinConfidence:      minConf,
			Log:                log,
		})
		f.Close()
		reports = append(reports, rep)
	}

	sys.sortAll()
	return sys, reports, nil
}
