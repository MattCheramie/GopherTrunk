package hunt

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// DiscoverWideband folds wideband multi-carrier captures into a single
// DiscoveredSystem. Each capture is surveyed (siglab.SurveyWideband splits it
// into per-carrier baseband streams, identifies each, and recognizes a DMR
// Tier III trunked system), and every DMR carrier of the capture is accumulated
// into the same system — so the control channels register their identity +
// frequencies and the voice carriers' grants become talkgroups, all under one
// discovered system. It is the offline-hunt counterpart of Discover for the
// `hunt -wideband -in <capture>` path.
//
// Each capture's CaptureInput.FrequencyHz is the capture CENTER frequency (the
// dongle's tuned center), not a single carrier; the per-carrier absolute
// frequencies fall out of the survey's spectral offsets.
func DiscoverWideband(captures []CaptureInput, cfg DiscoverConfig) (*DiscoveredSystem, []CaptureReport, error) {
	log := cfg.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
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
		wr, err := siglab.SurveyWideband(f, cap.Path, siglab.WidebandConfig{
			SampleRateHz: cap.SampleRateHz,
			CenterHz:     cap.FrequencyHz,
			Format:       cap.Format,
			Conjugate:    cap.Conjugate,
			IQCorrect:    cap.IQCorrect,
			Log:          log,
		})
		f.Close()
		if err != nil {
			reports = append(reports, CaptureReport{Path: cap.Path, Error: fmt.Sprintf("survey: %v", err)})
			continue
		}
		reports = append(reports, accumulateSurvey(sys, cap.Path, wr))
	}

	sys.sortAll()
	return sys, reports, nil
}

// accumulateSurvey folds every DMR carrier of one wideband survey into sys and
// returns a one-line report for the capture.
func accumulateSurvey(sys *DiscoveredSystem, path string, wr *siglab.WidebandResult) CaptureReport {
	before := len(sys.Talkgroups)
	rep := CaptureReport{Path: path}

	var controls, voices int
	var bestConf float64
	for _, c := range wr.Carriers {
		if c.Role != siglab.RoleControl && c.Role != siglab.RoleVoice {
			continue // not a DMR member of a trunked system
		}
		res := c.Result()
		if res == nil {
			continue
		}
		Accumulate(sys, Observation{
			Protocol:       c.Protocol,
			Confidence:     c.Confidence,
			Result:         res,
			FallbackFreqHz: c.FreqHz,
			At:             time.Now(),
		})
		switch c.Role {
		case siglab.RoleControl:
			controls++
			if c.Locked {
				rep.Locked = true
			}
			if rep.ControlHz == 0 {
				rep.ControlHz = c.FreqHz
			}
		case siglab.RoleVoice:
			voices++
		}
		if c.Confidence > bestConf {
			bestConf = c.Confidence
		}
	}

	rep.Protocol = sys.Protocol
	rep.Confidence = bestConf
	rep.Talkgroups = len(sys.Talkgroups) - before
	if len(wr.Systems) > 0 {
		rep.Verdict = wr.Systems[0].Verdict
	} else if controls == 0 && voices == 0 {
		rep.Skipped = true
		rep.SkipReason = fmt.Sprintf("no DMR carriers recognized among %d detected", wr.DetectedCount)
	}
	return rep
}
