package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/configbuilder"
	"github.com/MattCheramie/GopherTrunk/internal/radioreference"
)

// handleConfigList answers GET /api/v1/config/files — the config files
// discovered in the allowed directories, each tagged with whether it
// currently validates.
func (s *Server) handleConfigList(w http.ResponseWriter, r *http.Request) {
	resp := ConfigListResponse{Dirs: s.configBuilder.dirs}
	for _, dir := range s.configBuilder.dirs {
		for _, path := range config.DirConfigFiles(dir) {
			info, err := os.Stat(path)
			if err != nil {
				continue
			}
			fi := ConfigFileInfo{
				Path:     path,
				Dir:      dir,
				Name:     filepath.Base(path),
				Size:     info.Size(),
				Modified: info.ModTime().UTC().Format(time.RFC3339),
				Valid:    true,
			}
			if _, lerr := config.Load(path); lerr != nil {
				fi.Valid = false
				fi.Error = lerr.Error()
			}
			resp.Files = append(resp.Files, fi)
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleConfigLoad answers GET /api/v1/config/file?path=... — parse one
// config file and return it as JSON with its validation result + mtime.
func (s *Server) handleConfigLoad(w http.ResponseWriter, r *http.Request) {
	path, err := s.configBuilder.resolvePath(r.URL.Query().Get("path"))
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "config: "+err.Error())
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "config: "+err.Error())
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "config: "+err.Error())
		return
	}
	var cfg config.Config
	if err := yamlUnmarshalConfig(data, &cfg); err != nil {
		s.writeError(w, http.StatusBadRequest, "config: parse: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ConfigLoadResponse{
		Path:       path,
		Config:     cfg,
		Validation: validationErrorsFrom(cfg.ValidateAll()),
		Mtime:      info.ModTime().UnixNano(),
	})
}

// handleConfigDefaults answers GET /api/v1/config/defaults — the
// zero-config seed (config.Default()) the builder starts a blank file from.
func (s *Server) handleConfigDefaults(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, config.Default())
}

// handleConfigValidate answers POST /api/v1/config/validate. With an empty
// Section it validates the whole config (one error per section); otherwise
// it validates just that section.
func (s *Server) handleConfigValidate(w http.ResponseWriter, r *http.Request) {
	var req ConfigValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "config: "+err.Error())
		return
	}
	var errs []error
	if req.Section == "" {
		errs = req.Config.ValidateAll()
	} else {
		errs = req.Config.ValidateSection(req.Section)
	}
	writeJSON(w, http.StatusOK, validationErrorsFrom(errs))
}

// handleConfigSave answers POST /api/v1/config/file (gated). It validates,
// writes any talkgroup CSV sidecars, then atomically writes the config.
func (s *Server) handleConfigSave(w http.ResponseWriter, r *http.Request) {
	var req ConfigSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "config: "+err.Error())
		return
	}
	path, err := s.configBuilder.resolvePath(req.Path)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "config: "+err.Error())
		return
	}

	// Write talkgroup CSV sidecars first (relative to the config file's
	// dir, constrained to the same allowed roots). If the config write
	// then fails the sidecars are harmless leftovers the operator can
	// see in the preview.
	var written []string
	for rel, rows := range req.Talkgroups {
		csvPath, perr := s.configBuilder.resolveSidecarPath(filepath.Join(filepath.Dir(path), rel))
		if perr != nil {
			s.writeError(w, http.StatusBadRequest, "config: talkgroup csv "+rel+": "+perr.Error())
			return
		}
		if werr := writeTalkgroupCSV(csvPath, rows); werr != nil {
			s.writeError(w, http.StatusInternalServerError, "config: talkgroup csv "+rel+": "+werr.Error())
			return
		}
		written = append(written, csvPath)
	}

	mtime, err := config.WriteConfigFile(path, req.Config, req.Mtime, req.Overwrite)
	if err != nil {
		switch {
		case errors.Is(err, config.ErrFileExists), errors.Is(err, config.ErrModifiedExternally):
			s.writeError(w, http.StatusConflict, "config: "+err.Error())
		default:
			s.writeError(w, http.StatusBadRequest, "config: "+err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, ConfigSaveResponse{Path: path, Mtime: mtime, TalkgroupCSVs: written})
}

// handleConfigParse answers POST /api/v1/config/parse — a stateless,
// parse-only PDF/CSV upload (multipart `files`) that returns the parsed
// systems with control channels + talkgroups so the builder can fold them
// into its draft, without committing anything to disk.
func (s *Server) handleConfigParse(w http.ResponseWriter, r *http.Request) {
	if s.importer == nil {
		s.writeError(w, http.StatusServiceUnavailable, "config: import parser not wired")
		return
	}
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		s.writeError(w, http.StatusBadRequest, "config: "+err.Error())
		return
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		s.writeError(w, http.StatusBadRequest, "config: at least one `files` part is required")
		return
	}
	parsed := make([]ParsedSystemDTO, 0, len(files))
	for _, fh := range files {
		kind := classifyImportFile(fh.Filename)
		if kind == "" {
			s.writeError(w, http.StatusBadRequest,
				fmt.Sprintf("config: %q: unsupported extension (need .pdf or .csv)", fh.Filename))
			return
		}
		f, err := fh.Open()
		if err != nil {
			s.writeError(w, http.StatusBadRequest, fmt.Sprintf("config: %q: %v", fh.Filename, err))
			return
		}
		body, err := io.ReadAll(f)
		_ = f.Close()
		if err != nil {
			s.writeError(w, http.StatusBadRequest, fmt.Sprintf("config: %q: %v", fh.Filename, err))
			return
		}
		dto, err := s.importer.Parse(ImportSource{Filename: fh.Filename, Kind: kind, Data: body})
		if err != nil {
			s.writeError(w, http.StatusBadRequest, fmt.Sprintf("config: %q: %v", fh.Filename, err))
			return
		}
		parsed = append(parsed, dto)
	}
	writeJSON(w, http.StatusOK, map[string]any{"systems": parsed})
}

// RRSystemResponse couples the raw RadioReference detail with a
// config-ready SystemConfig + talkgroup rows the builder folds into its
// draft.
type RRSystemResponse struct {
	System     radioreference.FullSystem `json:"system"`
	Config     config.SystemConfig       `json:"config"`
	Talkgroups []TalkgroupCSVRow         `json:"talkgroups"`
}

// handleConfigRRSearch answers GET /api/v1/config/rr/search?zip=|county=|state=.
func (s *Server) handleConfigRRSearch(w http.ResponseWriter, r *http.Request) {
	client, err := s.configBuilder.rrClient()
	if err != nil {
		s.writeError(w, http.StatusServiceUnavailable,
			"config: RadioReference credentials not configured (set radioreference.* or GOPHERTRUNK_RR_KEY/USER/PASS)")
		return
	}
	q := r.URL.Query()
	var hits []radioreference.SearchHit
	switch {
	case q.Get("zip") != "":
		hits, err = client.SearchByZip(r.Context(), q.Get("zip"))
	case q.Get("county") != "":
		ctid, cerr := strconv.Atoi(q.Get("county"))
		if cerr != nil {
			s.writeError(w, http.StatusBadRequest, "config: county must be a numeric RadioReference ctid")
			return
		}
		hits, err = client.SearchByCounty(r.Context(), ctid)
	case q.Get("state") != "":
		stid, serr := strconv.Atoi(q.Get("state"))
		if serr != nil {
			s.writeError(w, http.StatusBadRequest, "config: state must be a numeric RadioReference stid")
			return
		}
		hits, err = client.SearchByState(r.Context(), stid)
	default:
		s.writeError(w, http.StatusBadRequest, "config: provide one of zip, county, or state")
		return
	}
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "config: RadioReference: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": hits})
}

// handleConfigRRStates answers GET /api/v1/config/rr/states — the state
// list (id+name) backing the name-based browse picker.
func (s *Server) handleConfigRRStates(w http.ResponseWriter, r *http.Request) {
	client, err := s.configBuilder.rrClient()
	if err != nil {
		s.writeError(w, http.StatusServiceUnavailable,
			"config: RadioReference credentials not configured (set radioreference.* or GOPHERTRUNK_RR_KEY/USER/PASS)")
		return
	}
	states, err := client.GetStateList(r.Context())
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "config: RadioReference: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": states})
}

// handleConfigRRCounties answers GET /api/v1/config/rr/counties?state=<stid>
// — the counties in a state (id+name) for the browse picker.
func (s *Server) handleConfigRRCounties(w http.ResponseWriter, r *http.Request) {
	client, err := s.configBuilder.rrClient()
	if err != nil {
		s.writeError(w, http.StatusServiceUnavailable,
			"config: RadioReference credentials not configured (set radioreference.* or GOPHERTRUNK_RR_KEY/USER/PASS)")
		return
	}
	stid, serr := strconv.Atoi(r.URL.Query().Get("state"))
	if serr != nil {
		s.writeError(w, http.StatusBadRequest, "config: state must be a numeric RadioReference stid")
		return
	}
	counties, err := client.GetCountyList(r.Context(), stid)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "config: RadioReference: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": counties})
}

// handleConfigRRSystem answers GET /api/v1/config/rr/system/{sid} —
// full system detail plus a config-ready projection.
func (s *Server) handleConfigRRSystem(w http.ResponseWriter, r *http.Request) {
	client, err := s.configBuilder.rrClient()
	if err != nil {
		s.writeError(w, http.StatusServiceUnavailable,
			"config: RadioReference credentials not configured (set radioreference.* or GOPHERTRUNK_RR_KEY/USER/PASS)")
		return
	}
	sid, err := strconv.Atoi(r.PathValue("sid"))
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "config: sid must be numeric")
		return
	}
	full, err := client.GetFullSystem(r.Context(), sid)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "config: RadioReference: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rrToResponse(full))
}

// rrToResponse projects a RadioReference FullSystem into a config-ready
// SystemConfig (control channels collapsed across sites) + talkgroup rows,
// wrapping the shared projection in the HTTP response shape.
func rrToResponse(full radioreference.FullSystem) RRSystemResponse {
	sys, rows := configbuilder.RRToSystemConfig(full)
	return RRSystemResponse{System: full, Config: sys, Talkgroups: rows}
}

// handleConfigDocs answers GET /api/v1/config/docs — a map of config section →
// {instructions + documentation link} the builder renders and opens in a new
// tab. Sourced from the shared configbuilder.Sections() registry so the web and
// terminal builders show identical section help.
func (s *Server) handleConfigDocs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, configDocs())
}

// handleConfigFieldMeta answers GET /api/v1/config/fieldmeta — the shared
// per-field metadata registry keyed "StructName.FieldName" (comprehensive help
// plus select options / Hz / freq-list flags). The web builder feeds the help
// into its field widgets so its per-field help is sourced from the same Go
// registry the terminal builder uses.
func (s *Server) handleConfigFieldMeta(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, configbuilder.FieldMetas())
}

// handleConfigMarshal answers POST /api/v1/config/marshal — render the
// posted config to YAML so the builder can show a live file preview. It
// does not validate or write anything.
func (s *Server) handleConfigMarshal(w http.ResponseWriter, r *http.Request) {
	var cfg config.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		s.writeError(w, http.StatusBadRequest, "config: "+err.Error())
		return
	}
	data, err := config.Marshal(cfg)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "config: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"yaml": string(data)})
}

// handleConfigTalkgroups answers GET /api/v1/config/file/talkgroups?path=...
// returning the rows of each system's TalkgroupFile sidecar so the builder
// can view/edit the talkgroups of an existing config. Sidecars that are
// missing, unreadable, or reference a path outside the allowed roots are
// skipped silently (the system simply has no editable rows yet).
func (s *Server) handleConfigTalkgroups(w http.ResponseWriter, r *http.Request) {
	path, err := s.configBuilder.resolvePath(r.URL.Query().Get("path"))
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "config: "+err.Error())
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "config: "+err.Error())
		return
	}
	var cfg config.Config
	if err := config.Unmarshal(data, &cfg); err != nil {
		s.writeError(w, http.StatusBadRequest, "config: parse: "+err.Error())
		return
	}
	dir := filepath.Dir(path)
	out := map[string][]TalkgroupCSVRow{}
	for _, sysm := range cfg.Trunking.Systems {
		rel := strings.TrimSpace(sysm.TalkgroupFile)
		if rel == "" {
			continue
		}
		csvPath, perr := s.configBuilder.resolveSidecarPath(filepath.Join(dir, rel))
		if perr != nil {
			continue
		}
		rows, rerr := readTalkgroupCSV(csvPath)
		if rerr != nil {
			continue
		}
		out[rel] = rows
	}
	writeJSON(w, http.StatusOK, map[string]any{"talkgroups": out})
}

// handleConfigDelete answers DELETE /api/v1/config/file?path=... (gated).
func (s *Server) handleConfigDelete(w http.ResponseWriter, r *http.Request) {
	path, err := s.configBuilder.resolvePath(r.URL.Query().Get("path"))
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "config: "+err.Error())
		return
	}
	if err := os.Remove(path); err != nil {
		s.writeError(w, http.StatusInternalServerError, "config: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"deleted": path})
}

// handleConfigRename answers POST /api/v1/config/file/rename (gated). Both
// paths are allow-list validated; refuses to clobber an existing target.
func (s *Server) handleConfigRename(w http.ResponseWriter, r *http.Request) {
	var req struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "config: "+err.Error())
		return
	}
	from, err := s.configBuilder.resolvePath(req.From)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "config: from: "+err.Error())
		return
	}
	to, err := s.configBuilder.resolvePath(req.To)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "config: to: "+err.Error())
		return
	}
	if _, statErr := os.Stat(to); statErr == nil {
		s.writeError(w, http.StatusConflict, "config: target already exists: "+to)
		return
	}
	if err := os.Rename(from, to); err != nil {
		s.writeError(w, http.StatusInternalServerError, "config: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"from": from, "to": to})
}

// handleConfigMkdir answers POST /api/v1/config/dir (gated) — create a
// subdirectory inside an allowed root so the operator can organize configs.
func (s *Server) handleConfigMkdir(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "config: "+err.Error())
		return
	}
	dir, err := s.configBuilder.resolveDir(req.Path)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "config: "+err.Error())
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		s.writeError(w, http.StatusInternalServerError, "config: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": dir})
}

// readTalkgroupCSV / writeTalkgroupCSV delegate to the shared configbuilder
// implementation so the web and TUI builders read/write identical sidecars.
func readTalkgroupCSV(path string) ([]TalkgroupCSVRow, error) {
	return configbuilder.ReadTalkgroupCSV(path)
}

func writeTalkgroupCSV(path string, rows []TalkgroupCSVRow) error {
	return configbuilder.WriteTalkgroupCSV(path, rows)
}

// yamlUnmarshalConfig parses YAML into a Config without running Load's
// validation (the builder validates separately and may want to load an
// invalid file to fix it).
func yamlUnmarshalConfig(data []byte, out *config.Config) error {
	return config.Unmarshal(data, out)
}
