package api

import (
	"net/http"
	"strconv"
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
	// Now is the daemon-side timestamp in UTC. Useful for detecting
	// clock skew between probe and daemon.
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
		Now:            time.Now().UTC(),
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
		s.overlayNeighbors(&dto)
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
			s.overlayNeighbors(&dto)
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

// overlayNeighbors fills dto.Neighbors from the live topology snapshot for the
// system, when the wired SitesProvider exposes one (the SiteTracker does). The
// adjacent sites are decoded from P25 status broadcasts over the air, so a
// configured-but-quiet system simply leaves the list empty.
func (s *Server) overlayNeighbors(dto *SystemDTO) {
	tp, ok := s.sites.(NetworkTopologyProvider)
	if !ok {
		return
	}
	if snap, ok := tp.Topology(dto.Name); ok {
		dto.Neighbors = neighborsFromTopology(snap)
	}
}

func (s *Server) handleListTalkgroups(w http.ResponseWriter, _ *http.Request) {
	all := s.talkgroups.All()
	out := make([]*TalkgroupDTO, 0, len(all))
	for _, tg := range all {
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
	out := make([]ActiveCallDTO, 0, len(active))
	for _, ac := range active {
		out = append(out, activeCallToDTO(ac))
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
