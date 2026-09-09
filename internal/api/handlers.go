package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gtdiag "github.com/MattCheramie/GopherTrunk/internal/diag"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// HealthDTO is the body shape returned by GET /api/v1/health. The
// extended fields (every key beyond status + now) let k8s / Nomad
// readiness probes and operator dashboards distinguish "the daemon
// process is up" from "the daemon process is actually doing work".
// All fields are best-effort — missing collaborators (no SDR pool,
// no engine, no history DB) just leave the corresponding field at
// its zero value rather than failing the request.
type HealthDTO struct {
	// Status is always "ok" for a serving daemon — present so old
	// callers that only check `.status == "ok"` keep working.
	Status string `json:"status"`
	// Now is the daemon-side timestamp, rendered in the configured display
	// timezone (display.timezone; local by default). It carries an explicit
	// offset, so a probe can still detect clock skew against its own clock.
	Now time.Time `json:"now"`
	// Version is the daemon build version, redundant with the
	// dedicated /api/v1/version endpoint but useful so probes can
	// confirm process identity in one round-trip.
	Version string `json:"version,omitempty"`
	// PoolAttachedCount is the number of currently-attached SDR
	// devices. Zero means no Devices provider is wired OR every
	// device has detached — both are operator-actionable signals.
	PoolAttachedCount int `json:"pool_attached_count"`
	// ActiveCalls is the count of in-flight voice calls.
	ActiveCalls int `json:"active_calls"`
	// DBConnected reports whether the call-history database is
	// wired. A daemon configured without `db_path` legitimately
	// runs with DBConnected = false.
	DBConnected bool `json:"db_connected"`
	// MetricsEnabled reports whether /metrics is mounted.
	MetricsEnabled bool `json:"metrics_enabled"`
	// AuthMode echoes the bearer-token auth policy
	// ("auto" / "required" / "disabled") so probes can flag a
	// misconfigured production deployment.
	AuthMode string `json:"auth_mode,omitempty"`
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	body := HealthDTO{
		Status:         "ok",
		Now:            time.Now().In(orLocal(s.displayLoc)),
		Version:        s.version,
		DBConnected:    s.history != nil,
		MetricsEnabled: s.metrics != nil,
	}
	if s.devices != nil {
		attached := 0
		for _, d := range s.devices.Snapshot() {
			if d.Attached {
				attached++
			}
		}
		body.PoolAttachedCount = attached
	}
	if s.engine != nil {
		body.ActiveCalls = len(s.engine.ActiveCalls())
	}
	if s.auth != nil {
		body.AuthMode = s.auth.mode.String()
	}
	writeJSON(w, http.StatusOK, body)
}

func (s *Server) handleVersion(w http.ResponseWriter, _ *http.Request) {
	v := s.version
	if v == "" {
		v = "dev"
	}
	writeJSON(w, http.StatusOK, map[string]string{"version": v})
}

// handleDiagBanner serves the diagnostics banner + system snapshot on
// demand so the web UI can show the same macro context (version, OS,
// system specs, detected dongles) the CLI error banner shows. The
// banner exposes host + dongle info, so it is gated: served to any
// caller when diagnostics.verbose_errors is on, otherwise only when
// the request opts in with ?verbose=1 AND is authorized (the same
// trust level the mutation gate grants). Returns 404 when no
// diagnostics collector is wired.
func (s *Server) handleDiagBanner(w http.ResponseWriter, r *http.Request) {
	if s.diagnostics == nil {
		s.writeError(w, http.StatusNotFound, "diagnostics not available")
		return
	}
	if !s.verboseErrors {
		if r.URL.Query().Get("verbose") != "1" {
			s.writeError(w, http.StatusForbidden, "diagnostics banner is restricted; pass ?verbose=1 or enable diagnostics.verbose_errors")
			return
		}
		if status, reason := s.auth.authorize(r); status != 0 {
			s.writeError(w, status, reason)
			return
		}
	}
	si := s.diagnostics.SysInfo()
	writeJSON(w, http.StatusOK, map[string]any{
		"banner":  gtdiag.FormatBannerPlain(si),
		"sysinfo": si,
	})
}

func (s *Server) handleListSystems(w http.ResponseWriter, _ *http.Request) {
	var live []trunking.SiteInfo
	if s.sites != nil {
		live = s.sites.Sites()
	}
	out := make([]SystemDTO, 0, len(s.systems))
	for _, sys := range s.systems {
		dto := systemToDTO(sys)
		overlayLiveIdentity(&dto, live)
		s.overlayTopology(&dto)
		dto.fillIdentityHex()
		out = append(out, dto)
	}
	writeJSON(w, http.StatusOK, map[string]any{"systems": out})
}

func (s *Server) handleGetSystem(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	for _, sys := range s.systems {
		if sys.Name == name {
			dto := systemToDTO(sys)
			if s.sites != nil {
				overlayLiveIdentity(&dto, s.sites.Sites())
			}
			s.overlayTopology(&dto)
			dto.fillIdentityHex()
			writeJSON(w, http.StatusOK, dto)
			return
		}
	}
	s.writeError(w, http.StatusNotFound, "system not found")
}

// handleGetSystemReport serves GET /api/v1/systems/{name}/report: the live
// human-readable P25 network-configuration report (WACN/SysID/NAC, the camped
// site's control channels and neighbours, and the band plan) rendered from the
// SiteTracker's latest topology snapshot. A configured-but-quiet system returns
// an empty report with available=false; an entirely unknown system 404s.
func (s *Server) handleGetSystemReport(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var report string
	available := false
	if reporter, ok := s.sites.(NetworkReporter); ok {
		report, available = reporter.Report(name)
	}
	if !available {
		known := false
		for _, sys := range s.systems {
			if sys.Name == name {
				known = true
				break
			}
		}
		if !known {
			s.writeError(w, http.StatusNotFound, "system not found")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"system": name, "report": report, "available": available})
}

// overlayLiveIdentity fills WACN/SystemID/RFSS/Site on dto from the live
// SiteTracker snapshot (the same data behind GET /api/v1/sites) for the
// matching system. Network identity (WACN/SYSID/RFSS/Site) is decoded from
// P25 status-broadcast TSBKs over the air, not configured, so the static
// config copy in s.systems leaves these zero — this is what populates the
// web "System Information" panel (issue #673). WACN and SystemID are
// system-wide; RFSS/Site reflect the most-recently-heard site (the currently
// camped one). Live non-zero values win; config values survive when nothing
// has been decoded yet.
func overlayLiveIdentity(dto *SystemDTO, sites []trunking.SiteInfo) {
	var rfssAt, wacnAt, sysAt time.Time
	for _, si := range sites {
		if si.System != dto.Name {
			continue
		}
		if si.LastSeen.After(rfssAt) {
			rfssAt = si.LastSeen
			dto.RFSS = si.RFSSID
			dto.Site = si.SiteID
		}
		if si.WACN != 0 && si.LastSeen.After(wacnAt) {
			wacnAt = si.LastSeen
			dto.WACN = si.WACN
		}
		if si.SystemID != 0 && si.LastSeen.After(sysAt) {
			sysAt = si.LastSeen
			dto.SystemID = si.SystemID
		}
	}
}

// overlayTopology fills dto.Neighbors and dto.FrequencyBands from the live
// topology snapshot for the system, when the wired SitesProvider exposes one
// (the SiteTracker does). Both the adjacent sites and the P25 IDEN_UP band plan
// are decoded from control-channel broadcasts over the air, so a
// configured-but-quiet system simply leaves them empty. One snapshot lookup
// feeds both.
func (s *Server) overlayTopology(dto *SystemDTO) {
	tp, ok := s.sites.(NetworkTopologyProvider)
	if !ok {
		return
	}
	if snap, ok := tp.Topology(dto.Name); ok {
		dto.Neighbors = neighborsFromTopology(snap)
		dto.FrequencyBands = bandPlanFromTopology(snap)
		dto.PrimaryControlChannel, dto.SecondaryControlChannels = siteChannelsFromTopology(snap)
		if snap.NAC != 0 {
			dto.NAC = snap.NAC
			dto.NACHex = trunking.IDHex(uint64(snap.NAC))
		}
		if snap.LRA != 0 {
			dto.LRA = snap.LRA
		}
		// TETRA has no WACN/RFSS/Site; its live identity (MCC/MNC/Location Area +
		// colour code) rides the same topology snapshot. Surface it so the Systems
		// panel can render a protocol-appropriate identity block instead of four
		// forever-empty P25 fields. MCC is always present once identity is decoded.
		if snap.MCC != 0 {
			dto.TETRAMCC = snap.MCC
			dto.TETRAMNC = snap.MNC
			dto.TETRALocationArea = snap.LocationArea
			dto.TETRADecodedColourCode = snap.ColorCode
			dto.HasTETRAIdentity = true
		}
	}
}

func (s *Server) handleListTalkgroups(w http.ResponseWriter, r *http.Request) {
	// ?discovered=false hides auto-discovered entries — the operator's "collapse
	// the phantoms" toggle. Any other value (or absent) includes them; the DTO's
	// `discovered` flag lets the UI badge/filter client-side too.
	includeDiscovered := true
	if v := r.URL.Query().Get("discovered"); v == "false" || v == "0" {
		includeDiscovered = false
	}
	all := s.talkgroups.All()
	out := make([]*TalkgroupDTO, 0, len(all))
	for _, tg := range all {
		if tg.Tag == trunking.DiscoveredTag {
			// A discovered entry whose id is really a subscriber radio is a leaked
			// phantom (the RadioID→TGID retraction race); never surface it, even
			// when discovered entries are included. Reactive retraction usually
			// removes it already; this closes the window where it hasn't yet.
			if s.engine != nil && s.engine.IsKnownRadio(tg.ID) {
				continue
			}
			if !includeDiscovered {
				continue
			}
		}
		out = append(out, talkgroupToDTO(tg))
	}
	writeJSON(w, http.StatusOK, map[string]any{"talkgroups": out})
}

func (s *Server) handleGetTalkgroup(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid talkgroup id")
		return
	}
	tg := s.talkgroups.Lookup(uint32(id))
	if tg == nil {
		s.writeError(w, http.StatusNotFound, "talkgroup not found")
		return
	}
	writeJSON(w, http.StatusOK, talkgroupToDTO(tg))
}

func (s *Server) handleActiveCalls(w http.ResponseWriter, _ *http.Request) {
	if s.engine == nil {
		writeJSON(w, http.StatusOK, map[string]any{"calls": []ActiveCallDTO{}})
		return
	}
	active := s.engine.ActiveCalls()
	observed := s.engine.ObservedCalls()
	out := make([]ActiveCallDTO, 0, len(active)+len(observed))
	for _, ac := range active {
		out = append(out, activeCallToDTO(ac, true))
	}
	// Control-channel-observed calls a tuner isn't following (every voice
	// device busy) so the operator sees every talkgroup up on the system.
	for _, ac := range observed {
		out = append(out, activeCallToDTO(ac, false))
	}
	writeJSON(w, http.StatusOK, map[string]any{"calls": out})
}

// handleCallHistory queries the persisted call_log table.
//
//	?system=<name>      filter by system
//	?group_id=<n>       filter by talkgroup
//	?since=<rfc3339>    only calls started at/after this time
//	?until=<rfc3339>    only calls started before this time
//	?limit=<n>          cap rows (default 100, max 1000)
//	?only_ended=true    skip calls that haven't ended
func (s *Server) handleCallHistory(w http.ResponseWriter, r *http.Request) {
	if s.history == nil {
		s.writeError(w, http.StatusServiceUnavailable, "call log persistence is not enabled")
		return
	}
	q := r.URL.Query()
	f := HistoryFilter{
		System: q.Get("system"),
		Limit:  100,
	}
	if v := q.Get("group_id"); v != "" {
		n, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid group_id")
			return
		}
		f.GroupID = uint32(n)
	}
	if v := q.Get("since"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid since (want RFC3339)")
			return
		}
		f.Since = t
	}
	if v := q.Get("until"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid until (want RFC3339)")
			return
		}
		f.Until = t
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			s.writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		if n > 1000 {
			n = 1000
		}
		f.Limit = n
	}
	if q.Get("only_ended") == "true" {
		f.OnlyEnded = true
	}
	rows, err := s.history.History(r.Context(), f)
	if err != nil {
		s.log.Warn("api: history query failed", "err", err)
		s.writeError(w, http.StatusInternalServerError, "history query failed")
		return
	}
	if rows == nil {
		rows = []CallRow{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"calls": rows})
}

// handleCallAudio streams the finished recording for one call so the web UI
// can play a past transmission (the call-history "play" button). The call id
// is resolved to a path through the RecordingProvider capability — the
// filesystem path is never exposed to the client — and the file is served with
// range support (http.ServeContent) so the browser can seek.
//
//	GET /api/v1/calls/{id}/audio
func (s *Server) handleCallAudio(w http.ResponseWriter, r *http.Request) {
	if s.history == nil {
		s.writeError(w, http.StatusServiceUnavailable, "call log persistence is not enabled")
		return
	}
	prov, ok := s.history.(RecordingProvider)
	if !ok {
		s.writeError(w, http.StatusNotImplemented, "recording playback is not available")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		s.writeError(w, http.StatusBadRequest, "invalid call id")
		return
	}
	path, err := prov.RecordingPathByID(r.Context(), id)
	if err != nil {
		s.log.Warn("api: recording path lookup failed", "err", err, "call", id)
		s.writeError(w, http.StatusInternalServerError, "recording lookup failed")
		return
	}
	if path == "" {
		s.writeError(w, http.StatusNotFound, "no recording for this call")
		return
	}
	// Rows written while recordings.dir resolved to a relative path (a
	// relative -config invocation, before config.Load absolutized the resolve
	// base) store cwd-relative recording paths. The recorder wrote them
	// relative to the daemon's working directory, so resolving against the
	// same cwd is exactly what makes the file reachable — without this every
	// such row 404'd as "unavailable" while the file was on disk.
	if !filepath.IsAbs(path) {
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
	}
	// The path is daemon-generated, but validate defensively before opening:
	// an absolute audio file, no traversal, actually a regular file. This keeps
	// the endpoint from ever serving a non-recording even if the row is bad.
	ext := strings.ToLower(filepath.Ext(path))
	if !filepath.IsAbs(path) || strings.Contains(path, "..") ||
		(ext != ".wav" && ext != ".mp3" && ext != ".flac") {
		s.writeError(w, http.StatusNotFound, "recording unavailable")
		return
	}
	f, err := os.Open(path)
	if err != nil {
		// Retention may have deleted it, or the volume moved.
		s.writeError(w, http.StatusNotFound, "recording file is gone")
		return
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil || fi.IsDir() {
		s.writeError(w, http.StatusNotFound, "recording unavailable")
		return
	}
	switch ext {
	case ".mp3":
		w.Header().Set("Content-Type", "audio/mpeg")
	case ".flac":
		w.Header().Set("Content-Type", "audio/flac")
	default:
		w.Header().Set("Content-Type", "audio/wav")
	}
	http.ServeContent(w, r, filepath.Base(path), fi.ModTime(), f)
}

// handleListDevices returns the SDR pool snapshot — every opened
// device with its role, configured gain/PPM/bias-tee, tuner identity,
// and the gain ladder. Returns 503 when the daemon was started without
// a pool (e.g. integration tests with no SDR hints in config).
func (s *Server) handleListDevices(w http.ResponseWriter, _ *http.Request) {
	if s.devices == nil {
		writeJSON(w, http.StatusOK, map[string]any{"devices": []any{}})
		return
	}
	devs := s.devices.Snapshot()
	if devs == nil {
		devs = []sdr.SDRStatus{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"devices": devs})
}
