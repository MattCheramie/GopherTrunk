package api

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/radioreference"
)

// ConfigBuilderOptions configure the web Config Builder/Editor subsystem.
type ConfigBuilderOptions struct {
	// Enabled mounts the /api/v1/config/* routes and the builder SPA.
	Enabled bool
	// ConfigDir, when set, is the single directory the builder lists and
	// constrains saves to. When empty the builder uses
	// config.CandidateDirs() (the same locations the daemon discovers).
	ConfigDir string
	// Assets is the builder SPA, served at /config/ when the daemon mounts
	// the subsystem as a secondary tree. The standalone `config serve`
	// command instead puts the SPA in ServerOptions.WebAssets (served at
	// /), leaving this nil.
	Assets fs.FS
	// RadioReference holds the credentials for the RR browse/import
	// routes. When AppKey is empty the RR routes return 503.
	RadioReference radioreference.Auth
}

// configBuilderService owns the resolved save/load roots, the RR
// credentials, and the SPA assets for the subsystem.
type configBuilderService struct {
	dirs   []string
	rrAuth radioreference.Auth
	assets fs.FS
}

func newConfigBuilderService(opts ConfigBuilderOptions) (*configBuilderService, error) {
	var dirs []string
	if strings.TrimSpace(opts.ConfigDir) != "" {
		abs, err := filepath.Abs(opts.ConfigDir)
		if err != nil {
			return nil, fmt.Errorf("config builder: resolve config dir: %w", err)
		}
		dirs = []string{abs}
	} else {
		for _, d := range config.CandidateDirs() {
			if abs, err := filepath.Abs(d); err == nil {
				dirs = append(dirs, abs)
			}
		}
	}
	if len(dirs) == 0 {
		return nil, errors.New("config builder: no usable config directories")
	}
	return &configBuilderService{
		dirs:   dirs,
		rrAuth: opts.RadioReference,
		assets: opts.Assets,
	}, nil
}

// errPathOutsideRoots is returned when a requested load/save path escapes
// the configured allow-list of directories.
var errPathOutsideRoots = errors.New("path is outside the allowed config directories")

// resolvePath validates a client-supplied config path against the
// allow-list: it must clean to an absolute path inside one of the service
// dirs and carry a .yaml/.yml extension. It returns the cleaned absolute
// path. mustExist is advisory — existence is checked by the caller — but
// the directory containing a new file must itself be an allowed root.
func (svc *configBuilderService) resolvePath(reqPath string) (string, error) {
	reqPath = strings.TrimSpace(reqPath)
	if reqPath == "" {
		return "", errors.New("path is required")
	}
	switch strings.ToLower(filepath.Ext(reqPath)) {
	case ".yaml", ".yml":
	default:
		return "", errors.New("path must end in .yaml or .yml")
	}
	abs, err := filepath.Abs(filepath.Clean(reqPath))
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(abs)
	for _, root := range svc.dirs {
		if dir == root {
			return abs, nil
		}
	}
	return "", errPathOutsideRoots
}

// resolveSidecarPath validates a talkgroup/RID CSV (or JSON) sidecar path
// against the allow-list: it must live directly in one of the service dirs
// and carry a .csv/.json extension. Returns the cleaned absolute path.
func (svc *configBuilderService) resolveSidecarPath(p string) (string, error) {
	abs, err := filepath.Abs(filepath.Clean(p))
	if err != nil {
		return "", err
	}
	switch strings.ToLower(filepath.Ext(abs)) {
	case ".csv", ".json":
	default:
		return "", errors.New("sidecar must be .csv or .json")
	}
	dir := filepath.Dir(abs)
	for _, root := range svc.dirs {
		if dir == root {
			return abs, nil
		}
	}
	return "", errPathOutsideRoots
}

// rrClient builds a RadioReference client from the configured credentials.
// Returns radioreference.ErrNoCredentials when no key is set so the caller
// can answer 503.
func (svc *configBuilderService) rrClient() (*radioreference.Client, error) {
	return radioreference.NewClient(svc.rrAuth)
}

// ---- Wire DTOs -----------------------------------------------------------

// ConfigFileInfo describes one discovered config file.
type ConfigFileInfo struct {
	Path     string `json:"path"`
	Dir      string `json:"dir"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Modified string `json:"modified"` // RFC3339
	Valid    bool   `json:"valid"`
	Error    string `json:"error,omitempty"`
}

// ConfigListResponse is the body of GET /api/v1/config/files.
type ConfigListResponse struct {
	Dirs  []string         `json:"dirs"`
	Files []ConfigFileInfo `json:"files"`
}

// ConfigLoadResponse is the body of GET /api/v1/config/file.
type ConfigLoadResponse struct {
	Path       string           `json:"path"`
	Config     config.Config    `json:"config"`
	Validation ValidationResult `json:"validation"`
	Mtime      int64            `json:"mtime"`
}

// ValidationError is one section-keyed config error.
type ValidationError struct {
	Section string `json:"section"`
	Message string `json:"message"`
}

// ValidationResult is the structured outcome of a validate call.
type ValidationResult struct {
	OK     bool              `json:"ok"`
	Errors []ValidationError `json:"errors"`
}

// ConfigValidateRequest is the body of POST /api/v1/config/validate. When
// Section is empty the whole config is validated (one error per section).
type ConfigValidateRequest struct {
	Config  config.Config `json:"config"`
	Section string        `json:"section,omitempty"`
}

// ConfigSaveRequest is the body of POST /api/v1/config/file.
type ConfigSaveRequest struct {
	Path      string        `json:"path"`
	Config    config.Config `json:"config"`
	Mtime     int64         `json:"mtime,omitempty"`
	Overwrite bool          `json:"overwrite"`
	// Talkgroups optionally writes Trunk Recorder–style CSV sidecars
	// keyed by the relative TalkgroupFile path referenced in the config.
	Talkgroups map[string][]TalkgroupCSVRow `json:"talkgroups,omitempty"`
}

// TalkgroupCSVRow is one row of a talkgroup CSV sidecar written on save.
type TalkgroupCSVRow struct {
	Decimal     uint32 `json:"decimal"`
	AlphaTag    string `json:"alpha_tag,omitempty"`
	Description string `json:"description,omitempty"`
	Tag         string `json:"tag,omitempty"`
	Group       string `json:"group,omitempty"`
	Mode        string `json:"mode,omitempty"`
}

// ConfigSaveResponse is the body returned after a successful save.
type ConfigSaveResponse struct {
	Path          string   `json:"path"`
	Mtime         int64    `json:"mtime"`
	TalkgroupCSVs []string `json:"talkgroup_csvs,omitempty"`
}

// validationErrorsFrom maps section errors to the wire shape, parsing the
// leading "section..." token off each message to fill Section.
func validationErrorsFrom(errs []error) ValidationResult {
	if len(errs) == 0 {
		return ValidationResult{OK: true}
	}
	out := make([]ValidationError, 0, len(errs))
	for _, e := range errs {
		msg := e.Error()
		out = append(out, ValidationError{Section: sectionOf(msg), Message: msg})
	}
	return ValidationResult{OK: false, Errors: out}
}

// sectionOf extracts the top-level config section from a validation error
// message (everything before the first '.', ':', or '[' in the leading
// token). E.g. "trunking.systems[0]: name required" → "trunking".
func sectionOf(msg string) string {
	end := len(msg)
	for i, r := range msg {
		if r == '.' || r == ':' || r == '[' || r == ' ' {
			end = i
			break
		}
	}
	return msg[:end]
}
