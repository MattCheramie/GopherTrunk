package main

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/api"
	"github.com/MattCheramie/GopherTrunk/internal/hunt"
	"github.com/MattCheramie/GopherTrunk/internal/huntrr"
	"github.com/MattCheramie/GopherTrunk/internal/radioreference"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// defaultHuntMonitorSeconds is the streaming long-dwell cap applied to a
// trunked-system hunt when the caller doesn't specify one. Converge-and-stop
// (internal/hunt/monitor.go) normally ends the dwell well before this once the
// site's identity, neighbors and band plan have settled; the cap only bounds a
// site whose status broadcasts never fully arrive.
const defaultHuntMonitorSeconds = 120

// huntCockpit adapts the daemon's hunt.Manager to the api.HuntCockpit surface
// (GET /api/v1/hunt + start/stop/export/commit). It mirrors scannerCockpit.
type huntCockpit struct {
	mgr     *hunt.Manager
	cfgPath string
	rrAuth  radioreference.Auth // RadioReference credentials for the cross-reference endpoint
}

// RadioReference cross-references a run's discovered system against
// RadioReference (duplicate hints + frequency/talkgroup diff), using the
// daemon's configured credentials and the caller-supplied comparison targets.
func (c huntCockpit) RadioReference(id, countyID int, checkSIDs []int) (api.HuntRRReport, error) {
	sys, ok, err := c.runSystem(id)
	if err != nil {
		return api.HuntRRReport{}, err
	}
	if !ok {
		return api.HuntRRReport{}, fmt.Errorf("hunt: no discovered system to cross-reference yet")
	}
	client, err := radioreference.NewClient(c.rrAuth)
	if err != nil {
		return api.HuntRRReport{}, fmt.Errorf("hunt: RadioReference credentials not configured (set radioreference.* or GOPHERTRUNK_RR_KEY/USER/PASS): %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	res := huntrr.Gather(ctx, client, sys, countyID, checkSIDs, nil)
	return api.HuntRRReport{Hints: res.Hints, Diff: res.Diff, Compared: res.Compared}, nil
}

// Status builds the REST snapshot from the manager's run state + latest map.
func (c huntCockpit) Status() api.HuntStatus {
	st := c.mgr.Status()
	out := api.HuntStatus{
		RunID:      st.RunID,
		State:      string(st.State),
		Running:    st.Running,
		Mode:       st.Mode,
		Phase:      string(st.Progress.Phase),
		Detail:     st.Progress.Detail,
		Sites:      st.Sites,
		Talkgroups: st.Talkgroups,
		SystemName: st.SystemName,
		Signals:    st.Signals,
		Error:      st.Error,

		GainRecommendations: st.GainRecommendations,
		GainNote:            st.GainNote,
	}
	if sys, reports, ok := c.mgr.Current(); ok {
		out.System = sys
		out.Reports = reports
	}
	return out
}

// Start translates the REST request into hunt.LiveHuntOptions and launches a run.
func (c huntCockpit) Start(req api.HuntStartRequest) (int, error) {
	bands, err := parseBandSpecs(req.Bands)
	if err != nil {
		return 0, err
	}
	var candidates []uint32
	for _, mhz := range req.Candidates {
		candidates = append(candidates, uint32(mhz*1e6+0.5))
	}
	if len(bands) == 0 && len(candidates) == 0 {
		return 0, fmt.Errorf("hunt: provide bands to sweep or candidates to probe")
	}
	if req.NoSweep {
		bands = nil
	}
	gainSet, err := parseGainSetDB(req.AutoGainSet)
	if err != nil {
		return 0, err
	}

	var proto trunking.Protocol
	if req.Protocol != "" {
		proto, err = siglab.ParseProtocolCLI(req.Protocol)
		if err != nil {
			return 0, err
		}
	}

	opts := hunt.LiveHuntOptions{
		Bands:           bands,
		Candidates:      candidates,
		Protocol:        proto,
		Survey:          req.Survey,
		ClassifyOnly:    req.ClassifyOnly,
		PersistSurvey:   req.PersistSurvey,
		Resume:          req.Resume,
		AutoGain:        req.AutoGain,
		AutoGainSet:     gainSet,
		MaxDwellSeconds: req.MaxDwellSeconds,
		MonitorSeconds:  req.MonitorSeconds,
		Serial:          req.Serial,
		ConfirmBorrow:   req.ConfirmBorrow,
		FFTSize:         req.FFTSize,
		DwellSeconds:    req.DwellSeconds,
		MinConfidence:   req.MinConfidence,
		Name:            req.Name,
		State:           req.State,
		County:          req.County,
		Location:        req.Location,
		PeakOpts: hunt.PeakOptions{
			ThresholdDb:  float32(req.PeakThresholdDb),
			MinSpacingHz: req.MinSpacingHz,
		},
	}
	if req.SweepDwellMs > 0 {
		opts.SweepDwell = msToDuration(req.SweepDwellMs, 0)
	}
	// A trunked-system hunt needs the streaming long-dwell monitor to see the
	// site's slow-cycling status broadcasts (Network Status / RFSS Status /
	// adjacent sites) — they rarely land inside the short buffered identify
	// dwell, so a default hunt would surface NAC + talkgroups but leave
	// WACN/SystemID/RFSS/Site and neighbors "awaiting status broadcasts". Opt
	// into the monitor by default; converge-and-stop ends it early (typically
	// well under the cap) once identity + topology settle. Survey runs and
	// explicit caller overrides are left untouched.
	if !req.Survey && opts.MonitorSeconds <= 0 {
		opts.MonitorSeconds = defaultHuntMonitorSeconds
	}
	return c.mgr.Start(opts)
}

func (c huntCockpit) Stop() bool { return c.mgr.Stop() }

// huntSurveyDir derives the directory for daemon survey NDJSON persistence from
// the configured storage path (survey files sit alongside the DB). An empty
// storage path disables persistence (returns "").
func huntSurveyDir(storagePath string) string {
	if storagePath == "" {
		return ""
	}
	return filepath.Dir(storagePath)
}

// ExportSurvey serializes a run's classified signal inventory in the requested
// format (json|csv).
func (c huntCockpit) ExportSurvey(id int, format string) ([]byte, string, error) {
	sf, err := hunt.ParseSurveyFormat(format)
	if err != nil {
		return nil, "", err
	}
	var buf bytes.Buffer
	if err := c.mgr.ExportSurvey(id, &buf, sf); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), "survey." + sf.FileExtension(), nil
}

// Export serializes the latest discovered system in the requested format.
func (c huntCockpit) Export(id int, format string) ([]byte, string, error) {
	hf, err := hunt.ParseFormat(format)
	if err != nil {
		return nil, "", err
	}
	sys, ok, err := c.runSystem(id)
	if err != nil {
		return nil, "", err
	}
	if !ok {
		return nil, "", fmt.Errorf("hunt: no discovered system to export yet")
	}
	var buf bytes.Buffer
	if err := hunt.Write(&buf, sys, hf, nil); err != nil {
		return nil, "", err
	}
	base := slugName(sys.DisplayName())
	name := base + "." + hf.FileExtension()
	if hf == hunt.FormatRR {
		name = base + "-radioreference." + hf.FileExtension()
	}
	return buf.Bytes(), name, nil
}

// Commit merges the latest discovered system into config.yaml via the shared
// importer writer.
// runSystem resolves a run id (0 = latest) to its discovered system. It returns
// hunt.ErrNoSuchRun for an unknown/evicted id, and ok=false when the run is
// known but produced no system yet.
func (c huntCockpit) runSystem(id int) (*hunt.DiscoveredSystem, bool, error) {
	sys, _, ok := c.mgr.Run(id)
	if ok {
		return sys, true, nil
	}
	if id != 0 && !c.mgr.KnownRun(id) {
		return nil, false, hunt.ErrNoSuchRun
	}
	return nil, false, nil
}

func (c huntCockpit) Commit(id int, force, dryRun bool) ([]string, error) {
	sys, ok, err := c.runSystem(id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("hunt: no discovered system to commit yet")
	}
	if c.cfgPath == "" {
		return nil, fmt.Errorf("hunt: daemon started without -config; nothing to merge into")
	}
	res, err := mergeIntoConfig([]parsedSystem{discoveredToParsed(sys)}, mergeOptions{
		ConfigPath: c.cfgPath,
		Force:      force,
		DryRun:     dryRun,
	})
	if err != nil {
		return nil, err
	}
	return res.Changes, nil
}

// parseBandSpecs parses "low:high" MHz band strings into hunt.Band (Hz).
func parseBandSpecs(specs []string) ([]hunt.Band, error) {
	var out []hunt.Band
	for _, sp := range specs {
		lo, hi, ok := strings.Cut(sp, ":")
		if !ok {
			return nil, fmt.Errorf("hunt: band %q: want low:high in MHz", sp)
		}
		loMHz, e1 := strconv.ParseFloat(strings.TrimSpace(lo), 64)
		hiMHz, e2 := strconv.ParseFloat(strings.TrimSpace(hi), 64)
		if e1 != nil || e2 != nil || hiMHz <= loMHz {
			return nil, fmt.Errorf("hunt: band %q: invalid range", sp)
		}
		out = append(out, hunt.Band{LowHz: uint32(loMHz*1e6 + 0.5), HighHz: uint32(hiMHz*1e6 + 0.5)})
	}
	return out, nil
}
