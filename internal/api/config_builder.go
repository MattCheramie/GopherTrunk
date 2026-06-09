package api

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/configbuilder"
	"github.com/MattCheramie/GopherTrunk/internal/radioreference"
)

// newRRClient builds a RadioReference client. It's a package var so tests can
// stub a client pointed at a fake SOAP server (the handlers build clients
// internally, so there's otherwise no endpoint-injection seam).
var newRRClient = radioreference.NewClient

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
		dirs = []string{canonicalDir(abs)}
	} else {
		for _, d := range config.CandidateDirs() {
			if abs, err := filepath.Abs(d); err == nil {
				dirs = append(dirs, canonicalDir(abs))
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
// allow-list: after symlink resolution it must sit inside one of the
// service dirs (or a nested subdirectory) and carry a .yaml/.yml extension.
// Returns the canonical absolute path.
func (svc *configBuilderService) resolvePath(reqPath string) (string, error) {
	return svc.resolveAllowed(reqPath, ".yaml", ".yml")
}

// resolveSidecarPath is resolvePath for talkgroup/RID CSV (or JSON) sidecars
// written next to the config file.
func (svc *configBuilderService) resolveSidecarPath(p string) (string, error) {
	return svc.resolveAllowed(p, ".csv", ".json")
}

// resolveAllowed checks the extension, symlink-resolves the path, and
// confirms it lands inside an allowed (already-canonical) root.
func (svc *configBuilderService) resolveAllowed(reqPath string, exts ...string) (string, error) {
	reqPath = strings.TrimSpace(reqPath)
	if reqPath == "" {
		return "", errors.New("path is required")
	}
	ext := strings.ToLower(filepath.Ext(reqPath))
	allowed := false
	for _, e := range exts {
		if ext == e {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", fmt.Errorf("path must end in %s", strings.Join(exts, " or "))
	}
	canon := evalSymlinksBestEffort(reqPath)
	for _, root := range svc.dirs {
		if withinRoot(root, canon) {
			return canon, nil
		}
	}
	return "", errPathOutsideRoots
}

// resolveDir validates a directory path for mkdir: symlink-resolved, it
// must sit inside (or equal) an allowed root. No extension check.
func (svc *configBuilderService) resolveDir(reqPath string) (string, error) {
	reqPath = strings.TrimSpace(reqPath)
	if reqPath == "" {
		return "", errors.New("path is required")
	}
	canon := evalSymlinksBestEffort(reqPath)
	for _, root := range svc.dirs {
		if withinRoot(root, canon) {
			return canon, nil
		}
	}
	return "", errPathOutsideRoots
}

// withinRoot reports whether abs is root itself or nested beneath it,
// using filepath.Rel so ".." escapes are rejected portably.
func withinRoot(root, abs string) bool {
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// canonicalDir resolves symlinks in an allow-list root, falling back to the
// cleaned absolute path when the directory doesn't exist yet (so a not-yet-
// created discovery dir still serves as a stable root).
func canonicalDir(abs string) string {
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	return abs
}

// evalSymlinksBestEffort resolves symlinks in the deepest existing ancestor
// of a (possibly not-yet-created) path and re-joins the non-existent
// remainder, so a symlinked file or parent that escapes the allow-list is
// caught while a brand-new file still resolves to its real parent dir.
func evalSymlinksBestEffort(path string) string {
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return filepath.Clean(path)
	}
	remainder := ""
	cur := abs
	for {
		if resolved, err := filepath.EvalSymlinks(cur); err == nil {
			if remainder == "" {
				return resolved
			}
			return filepath.Join(resolved, remainder)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return abs // reached the FS root without resolving
		}
		remainder = filepath.Join(filepath.Base(cur), remainder)
		cur = parent
	}
}

// rrAuthFromRequest overlays the per-request RR credentials carried in the
// X-RR-Key / X-RR-User / X-RR-Pass headers (the username/password the operator
// is editing in the builder) over the server's startup auth, then applies the
// built-in default app key. Header values win so what the user types is what
// gets sent to RadioReference and verifies their subscription.
func (svc *configBuilderService) rrAuthFromRequest(r *http.Request) radioreference.Auth {
	auth := svc.rrAuth
	if v := strings.TrimSpace(r.Header.Get("X-RR-Key")); v != "" {
		auth.AppKey = v
	}
	if v := strings.TrimSpace(r.Header.Get("X-RR-User")); v != "" {
		auth.Username = v
	}
	if v := strings.TrimSpace(r.Header.Get("X-RR-Pass")); v != "" {
		auth.Password = v
	}
	return radioreference.ResolveAuth(auth) // backstop the key with env/built-in
}

// rrClientFor builds a RadioReference client from the request-merged
// credentials. Returns radioreference.ErrNoCredentials when no key is set
// (built-in included) so the caller can answer 503.
func (svc *configBuilderService) rrClientFor(r *http.Request) (*radioreference.Client, error) {
	return newRRClient(svc.rrAuthFromRequest(r))
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
// Aliased to the shared configbuilder type so the TUI and web builder use
// one definition and the JSON wire shape is identical.
type TalkgroupCSVRow = configbuilder.TalkgroupCSVRow

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

// sectionOf delegates to the shared configbuilder helper.
func sectionOf(msg string) string { return configbuilder.SectionOf(msg) }
