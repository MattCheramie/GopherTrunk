package api

import (
	"encoding/json"
	"net/http"
	"os"
)

// ConfigActivator loads/hot-swaps the config file the daemon runs on.
// Supplied by the daemon; nil on the standalone `config serve` (which has
// no live daemon to reconfigure), so the activate endpoint returns 503.
type ConfigActivator interface {
	// ActivateReload re-points the daemon at path and reloads it in
	// place, returning the field keys it hot-applied vs. those that need
	// a restart to take effect.
	ActivateReload(path string) (applied, restartRequired []string, err error)
	// ActivateRestart re-execs the daemon into path so every field takes
	// effect. It returns once the restart is scheduled (the process tears
	// down and re-execs shortly after); an error means the restart could
	// not be initiated (e.g. the new config does not load).
	ActivateRestart(path string) error
}

// ConfigActivateRequest is the body of POST /api/v1/config/activate.
type ConfigActivateRequest struct {
	Path string `json:"path"`
	// Mode is "reload" (default — hot-apply what it can) or "restart"
	// (re-exec the daemon so every field takes effect).
	Mode string `json:"mode,omitempty"`
}

// ConfigActivateResponse reports the outcome. For reload it carries the
// applied / restart-required field keys; for restart it just echoes the
// target (the daemon is tearing down to re-exec).
type ConfigActivateResponse struct {
	Path            string   `json:"path"`
	Mode            string   `json:"mode"`
	Applied         []string `json:"applied,omitempty"`
	RestartRequired []string `json:"restart_required,omitempty"`
	Restarting      bool     `json:"restarting,omitempty"`
}

// handleConfigActivate answers POST /api/v1/config/activate (gated). It
// validates the requested path against the builder's allow-list and then
// either reloads the daemon in place or schedules a restart into the new
// config file.
func (s *Server) handleConfigActivate(w http.ResponseWriter, r *http.Request) {
	if s.configActiv == nil {
		s.writeError(w, http.StatusServiceUnavailable,
			"config: activation not supported (no daemon backs this server)")
		return
	}
	var req ConfigActivateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "config: "+err.Error())
		return
	}
	path, err := s.configBuilder.resolvePath(req.Path)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "config: "+err.Error())
		return
	}
	if _, err := os.Stat(path); err != nil {
		s.writeError(w, http.StatusNotFound, "config: "+err.Error())
		return
	}

	switch req.Mode {
	case "", "reload":
		applied, restartRequired, aerr := s.configActiv.ActivateReload(path)
		if aerr != nil {
			s.writeError(w, http.StatusBadRequest, "config: "+aerr.Error())
			return
		}
		writeJSON(w, http.StatusOK, ConfigActivateResponse{
			Path:            path,
			Mode:            "reload",
			Applied:         applied,
			RestartRequired: restartRequired,
		})
	case "restart":
		if aerr := s.configActiv.ActivateRestart(path); aerr != nil {
			s.writeError(w, http.StatusBadRequest, "config: "+aerr.Error())
			return
		}
		// 202: the daemon accepted the restart and is tearing down to
		// re-exec; the SPA should expect a disconnect + reconnect.
		writeJSON(w, http.StatusAccepted, ConfigActivateResponse{
			Path:       path,
			Mode:       "restart",
			Restarting: true,
		})
	default:
		s.writeError(w, http.StatusBadRequest, `config: mode must be "reload" or "restart"`)
	}
}
