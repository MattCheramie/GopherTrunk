package hunt

import (
	"fmt"
	"log/slog"
	"time"

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
	Skipped    bool    `json:"skipped"`
	SkipReason string  `json:"skip_reason,omitempty"`
	Error      string  `json:"error,omitempty"`
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
		rep := CaptureReport{Path: cap.Path}

		proto := cap.Protocol
		conf := 1.0 // operator-supplied protocol ⇒ full confidence
		if proto == trunking.ProtocolUnknown {
			idr, err := siglab.Identify(cap.Path, siglab.IdentifyConfig{
				SampleRateHz: cap.SampleRateHz,
				Format:       cap.Format,
				AutoTune:     cap.AutoTune,
				Conjugate:    cap.Conjugate,
				IQCorrect:    cap.IQCorrect,
				MaxSamples:   cap.IdentifyMaxSamples,
				Log:          log,
			})
			if err != nil {
				rep.Error = fmt.Sprintf("identify: %v", err)
				reports = append(reports, rep)
				continue
			}
			conf = idr.Confidence
			if idr.Inconclusive || idr.Confidence < minConf {
				rep.Protocol = idr.Winner
				rep.Confidence = idr.Confidence
				rep.Skipped = true
				rep.SkipReason = fmt.Sprintf("no trunked control channel above confidence %.2f (best: %s @ %.2f)",
					minConf, idr.Winner, idr.Confidence)
				reports = append(reports, rep)
				continue
			}
			p, err := idr.WinnerProtocol()
			if err != nil {
				rep.Error = fmt.Sprintf("winner protocol: %v", err)
				reports = append(reports, rep)
				continue
			}
			proto = p
		}
		rep.Protocol = proto.String()
		rep.Confidence = conf

		res, err := siglab.Run(cap.Path, siglab.Config{
			Protocol:     proto,
			FrequencyHz:  cap.FrequencyHz,
			SampleRateHz: cap.SampleRateHz,
			Format:       cap.Format,
			AutoTune:     cap.AutoTune,
			Conjugate:    cap.Conjugate,
			IQCorrect:    cap.IQCorrect,
			// CollectIQDiag engages P25 Phase 1's deep decode path, which is
			// what accumulates and snapshots the system topology
			// (WACN/SYSID/RFSS/Site + neighbors + band plan) onto Result.Topology.
			// Without it P25 runs the generic factory pipeline and the map would
			// be NAC-only.
			CollectIQDiag: true,
			Log:           log,
		})
		if err != nil {
			rep.Error = fmt.Sprintf("decode: %v", err)
			reports = append(reports, rep)
			continue
		}

		before := len(sys.Talkgroups)
		Accumulate(sys, Observation{
			Protocol:       proto.String(),
			Confidence:     conf,
			Result:         res,
			FallbackFreqHz: cap.FrequencyHz,
			At:             time.Now(),
		})
		rep.Locked = res.Locked
		if res.Lock != nil {
			rep.ControlHz = res.Lock.FrequencyHz
		}
		rep.Talkgroups = len(sys.Talkgroups) - before
		reports = append(reports, rep)
	}

	sys.sortAll()
	return sys, reports, nil
}
