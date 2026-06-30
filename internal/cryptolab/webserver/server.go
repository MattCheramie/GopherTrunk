// Package webserver is the optional cryptolab web console backend: a
// self-contained HTTP server that stages uploaded files, runs a selected
// cryptolab tool/mode with form-supplied settings, streams progress, and
// returns the structured Result and any artifacts. It is the browser
// counterpart to the `cryptolab` CLI.
//
// It is imported only by the tag-gated `cryptolab serve` command, so it is
// never linked into the default gophertrunk binary. It depends only on the
// cryptolab engine package and the standard library.
package webserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"
)

// Options configure a Server.
type Options struct {
	// Assets is the embedded SPA file system (served at /). Nil serves the
	// API only with a short placeholder at /.
	Assets fs.FS
	// TempDir stages uploads and job artifacts. Empty ⇒ a fresh temp dir the
	// server owns and removes on Close.
	TempDir string
	// MaxUploadBytes bounds one upload. 0 ⇒ 512 MiB.
	MaxUploadBytes int64
	// Logger receives server-level logs. Nil ⇒ slog.Default.
	Logger *slog.Logger
}

const defaultMaxUpload = 512 << 20

// Server is the cryptolab web console HTTP handler.
type Server struct {
	assets    fs.FS
	fixedDir  string // explicit temp dir from Options (empty ⇒ owned/lazy)
	maxUpload int64
	log       *slog.Logger

	dirOnce sync.Once
	tempDir string
	ownTemp bool
	dirErr  error

	mu    sync.Mutex
	files map[string]*upload
	jobs  map[string]*job
}

type upload struct {
	ID, Name, Path string
	Size           int64
	Created        time.Time
}

// New constructs a Server. The staging temp directory is created lazily on
// the first upload/run (not at construction), so merely registering the
// routes — as the daemon does for every request mux — costs nothing on disk.
func New(opts Options) (*Server, error) {
	max := opts.MaxUploadBytes
	if max <= 0 {
		max = defaultMaxUpload
	}
	log := opts.Logger
	if log == nil {
		log = slog.Default()
	}
	return &Server{
		assets:    opts.Assets,
		fixedDir:  opts.TempDir, // empty ⇒ create an owned temp dir lazily
		maxUpload: max,
		log:       log,
		files:     map[string]*upload{},
		jobs:      map[string]*job{},
	}, nil
}

// ensureTempDir creates the staging directory on first use and returns it.
func (s *Server) ensureTempDir() (string, error) {
	s.dirOnce.Do(func() {
		if s.fixedDir != "" {
			s.dirErr = os.MkdirAll(s.fixedDir, 0o755)
			s.tempDir = s.fixedDir
			return
		}
		d, err := os.MkdirTemp("", "gophertrunk-cryptolab-")
		if err != nil {
			s.dirErr = err
			return
		}
		s.tempDir, s.ownTemp = d, true
	})
	return s.tempDir, s.dirErr
}

// Handler returns a standalone HTTP mux: the cryptolab API plus the embedded
// SPA at /. Used by `cryptolab serve`.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.RegisterAPI(mux, nil)
	mux.Handle("GET /", s.spaHandler())
	return mux
}

// RegisterAPI mounts the /api/v1/cryptolab/* routes onto mux. Mutating routes
// (file upload, run) are wrapped with gate when non-nil, so the daemon can
// apply the same auth middleware it uses for its other write routes; the
// standalone server passes nil. This lets the same service back both the
// standalone `cryptolab serve` and the daemon-mounted console at /cryptolab/.
func (s *Server) RegisterAPI(mux *http.ServeMux, gate func(http.HandlerFunc) http.HandlerFunc) {
	if gate == nil {
		gate = func(h http.HandlerFunc) http.HandlerFunc { return h }
	}
	mux.HandleFunc("GET /api/v1/cryptolab/tools", s.handleTools)
	mux.HandleFunc("POST /api/v1/cryptolab/files", gate(s.handleUpload))
	mux.HandleFunc("POST /api/v1/cryptolab/run", gate(s.handleRun))
	mux.HandleFunc("GET /api/v1/cryptolab/jobs/{id}", s.handleJob)
	mux.HandleFunc("GET /api/v1/cryptolab/jobs/{id}/events", s.handleJobEvents)
	mux.HandleFunc("GET /api/v1/cryptolab/jobs/{id}/artifacts/{name}", s.handleArtifact)
	mux.HandleFunc("GET /api/v1/cryptolab/recipe/ops", s.handleRecipeOps)
	mux.HandleFunc("POST /api/v1/cryptolab/recipe", gate(s.handleRecipe))
}

// Close removes the temp directory if the server created it.
func (s *Server) Close() error {
	if s.ownTemp && s.tempDir != "" {
		return os.RemoveAll(s.tempDir)
	}
	return nil
}

func (s *Server) handleTools(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, cryptolab.Schema())
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, s.maxUpload)
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		writeErr(w, http.StatusBadRequest, "upload: "+err.Error())
		return
	}
	f, hdr, err := r.FormFile("file")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "upload: a `file` part is required")
		return
	}
	defer f.Close()

	dir, err := s.ensureTempDir()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "upload: temp dir: "+err.Error())
		return
	}
	id := randomID()
	path := filepath.Join(dir, id+".bin")
	dst, err := os.Create(path)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "upload: "+err.Error())
		return
	}
	n, copyErr := io.Copy(dst, f)
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(path)
		writeErr(w, http.StatusBadRequest, "upload: write failed")
		return
	}
	u := &upload{ID: id, Name: hdr.Filename, Path: path, Size: n, Created: time.Now()}
	s.mu.Lock()
	s.files[id] = u
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"id": u.ID, "name": u.Name, "size": u.Size})
}

type runRequest struct {
	Tool   string            `json:"tool"`
	Mode   string            `json:"mode"`
	Params map[string]string `json:"params"`
	Files  map[string]string `json:"files"` // param name -> upload id
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	var req runRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "run: "+err.Error())
		return
	}
	if _, _, err := cryptolab.LookupMode(req.Tool, req.Mode); err != nil {
		writeErr(w, http.StatusBadRequest, "run: "+err.Error())
		return
	}
	schema, ok := schemaFor(req.Tool, req.Mode)
	if !ok {
		writeErr(w, http.StatusBadRequest, "run: no schema for tool/mode")
		return
	}

	dir, err := s.ensureTempDir()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "run: temp dir: "+err.Error())
		return
	}
	id := randomID()
	jobDir := filepath.Join(dir, "job-"+id)
	if err := os.MkdirAll(jobDir, 0o755); err != nil {
		writeErr(w, http.StatusInternalServerError, "run: "+err.Error())
		return
	}
	args, err := s.buildArgs(schema, req, jobDir)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "run: "+err.Error())
		return
	}

	j := newJob(id, req.Tool, req.Mode, jobDir)
	s.mu.Lock()
	s.jobs[id] = j
	s.mu.Unlock()

	_, mode, _ := cryptolab.LookupMode(req.Tool, req.Mode)
	env := cryptolab.Env{
		Logger: slog.New(j.handler()),
		OutDir: jobDir,
		Format: cryptolab.FormatText,
		Stdout: io.Discard,
	}
	// Run on a background context: the request returns immediately with the
	// job id, so the run must outlive it (the SSE handler uses its own
	// request context for streaming).
	go func() {
		res, runErr := mode.Run(context.Background(), args, env)
		j.finish(res, runErr)
	}()
	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": id})
}

// buildArgs translates the form submission into the mode's CLI args, staging
// file params to their uploaded paths and assigning output-file params a path
// under the job directory.
func (s *Server) buildArgs(schema cryptolab.ModeSchema, req runRequest, jobDir string) ([]string, error) {
	var args []string
	for _, p := range schema.Params {
		switch p.Kind {
		case "file":
			uid := req.Files[p.Name]
			if uid == "" {
				if p.Required {
					return nil, fmt.Errorf("%s: a file is required", p.Name)
				}
				continue
			}
			s.mu.Lock()
			u, ok := s.files[uid]
			s.mu.Unlock()
			if !ok {
				return nil, fmt.Errorf("%s: unknown upload %q", p.Name, uid)
			}
			args = append(args, "-"+p.Name, u.Path)
		case "outfile":
			name := sanitizeName(req.Params[p.Name])
			if name == "" {
				name = sanitizeName(p.Default)
			}
			if name == "" {
				name = p.Name + ".out"
			}
			args = append(args, "-"+p.Name, filepath.Join(jobDir, name))
		case "bool":
			if req.Params[p.Name] == "true" {
				args = append(args, "-"+p.Name)
			}
		default:
			if v := strings.TrimSpace(req.Params[p.Name]); v != "" {
				args = append(args, "-"+p.Name, v)
			}
		}
	}
	return args, nil
}

func (s *Server) handleJob(w http.ResponseWriter, r *http.Request) {
	j, ok := s.job(r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "job not found")
		return
	}
	writeJSON(w, http.StatusOK, j.snapshot())
}

func (s *Server) handleJobEvents(w http.ResponseWriter, r *http.Request) {
	j, ok := s.job(r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "job not found")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, ": cryptolab events\n\n")
	flusher.Flush()

	ctx := r.Context()
	go func() {
		<-ctx.Done()
		j.wake()
	}()
	cursor := 0
	for {
		pending, state, errMsg := j.eventsFrom(&cursor, ctx)
		if ctx.Err() != nil {
			return
		}
		for _, ev := range pending {
			payload, _ := json.Marshal(ev)
			fmt.Fprintf(w, "event: log\ndata: %s\n\n", payload)
		}
		flusher.Flush()
		if state != "running" {
			done, _ := json.Marshal(map[string]string{"state": state, "error": errMsg})
			fmt.Fprintf(w, "event: done\ndata: %s\n\n", done)
			flusher.Flush()
			return
		}
	}
}

func (s *Server) handleArtifact(w http.ResponseWriter, r *http.Request) {
	j, ok := s.job(r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "job not found")
		return
	}
	name := sanitizeName(r.PathValue("name"))
	if name == "" {
		writeErr(w, http.StatusBadRequest, "bad artifact name")
		return
	}
	path := filepath.Join(j.dir, name)
	f, err := os.Open(path)
	if err != nil {
		writeErr(w, http.StatusNotFound, "artifact not found")
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
	io.Copy(w, f)
}

func (s *Server) spaHandler() http.Handler {
	if s.assets == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			io.WriteString(w, "cryptolab web console: SPA not bundled (run `make cryptolab-web-build`).")
		})
	}
	fileSrv := http.FileServerFS(s.assets)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		clean := strings.TrimPrefix(r.URL.Path, "/")
		if clean != "" {
			if _, err := fs.Stat(s.assets, clean); err != nil {
				r = r.Clone(r.Context())
				r.URL.Path = "/" // SPA fallback to index.html
			}
		}
		fileSrv.ServeHTTP(w, r)
	})
}

func (s *Server) job(id string) (*job, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[id]
	return j, ok
}

func schemaFor(tool, mode string) (cryptolab.ModeSchema, bool) {
	for _, ms := range cryptolab.Schema() {
		if ms.Tool == tool && ms.Mode == mode {
			return ms, true
		}
	}
	return cryptolab.ModeSchema{}, false
}

func randomID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// sanitizeName keeps a download/output name to a safe single path element.
func sanitizeName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == ".." || name == "/" {
		return ""
	}
	return name
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
