// Package webserver is the optional rfscope web console backend: a
// self-contained HTTP server that stages an uploaded IQ capture, runs the
// rfscope segmentation + analyzers over it, and returns the structured Scene
// (plus a cryptolab `ks` frames file for any unknown payloads). It is the
// browser counterpart to the `rfscope` CLI/TUI, modeled on the cryptolab and
// siglab web consoles.
//
// It depends only on the rfscope package and the standard library, and is
// imported only by the `rfscope serve` command.
package webserver

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/rfscope"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// Options configure a Server.
type Options struct {
	// Assets is the embedded SPA file system (served at /). Nil serves the API
	// only with a short placeholder at /.
	Assets fs.FS
	// TempDir stages uploads. Empty ⇒ a fresh temp dir the server owns and
	// removes on Close.
	TempDir string
	// MaxUploadBytes bounds one upload. 0 ⇒ 512 MiB.
	MaxUploadBytes int64
	// AnalyzeTimeout bounds one analysis. 0 ⇒ 120s.
	AnalyzeTimeout time.Duration
	// Logger receives server-level logs. Nil ⇒ slog.Default.
	Logger *slog.Logger
}

const (
	defaultMaxUpload     = 512 << 20
	defaultAnalyzeTimout = 120 * time.Second
)

// Server is the rfscope web console HTTP handler.
type Server struct {
	assets    fs.FS
	fixedDir  string
	maxUpload int64
	timeout   time.Duration
	log       *slog.Logger

	dirOnce sync.Once
	tempDir string
	ownTemp bool
	dirErr  error
}

// New constructs a Server. The staging temp directory is created lazily on the
// first upload, so merely registering the routes costs nothing on disk.
func New(opts Options) (*Server, error) {
	max := opts.MaxUploadBytes
	if max <= 0 {
		max = defaultMaxUpload
	}
	to := opts.AnalyzeTimeout
	if to <= 0 {
		to = defaultAnalyzeTimout
	}
	log := opts.Logger
	if log == nil {
		log = slog.Default()
	}
	return &Server{assets: opts.Assets, fixedDir: opts.TempDir, maxUpload: max, timeout: to, log: log}, nil
}

func (s *Server) ensureTempDir() (string, error) {
	s.dirOnce.Do(func() {
		if s.fixedDir != "" {
			s.dirErr = os.MkdirAll(s.fixedDir, 0o755)
			s.tempDir = s.fixedDir
			return
		}
		d, err := os.MkdirTemp("", "gophertrunk-rfscope-")
		if err != nil {
			s.dirErr = err
			return
		}
		s.tempDir, s.ownTemp = d, true
	})
	return s.tempDir, s.dirErr
}

// Handler returns a standalone HTTP mux: the rfscope API plus the embedded SPA
// at /. Used by `rfscope serve`.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.RegisterAPI(mux, nil)
	mux.Handle("GET /", s.spaHandler())
	return mux
}

// RegisterAPI mounts the /api/v1/rfscope/* routes onto mux. Mutating routes are
// wrapped with gate when non-nil so a daemon can apply the same auth middleware;
// the standalone server passes nil.
func (s *Server) RegisterAPI(mux *http.ServeMux, gate func(http.HandlerFunc) http.HandlerFunc) {
	if gate == nil {
		gate = func(h http.HandlerFunc) http.HandlerFunc { return h }
	}
	mux.HandleFunc("GET /api/v1/rfscope/analyzers", s.handleAnalyzers)
	mux.HandleFunc("POST /api/v1/rfscope/analyze", gate(s.handleAnalyze))
}

// Close removes the temp directory if the server created it.
func (s *Server) Close() error {
	if s.ownTemp && s.tempDir != "" {
		return os.RemoveAll(s.tempDir)
	}
	return nil
}

type analyzerDTO struct {
	Name      string   `json:"name"`
	Synopsis  string   `json:"synopsis"`
	DependsOn []string `json:"depends_on,omitempty"`
}

func (s *Server) handleAnalyzers(w http.ResponseWriter, _ *http.Request) {
	out := make([]analyzerDTO, 0)
	for _, a := range rfscope.Analyzers() {
		out = append(out, analyzerDTO{Name: a.Name(), Synopsis: a.Synopsis(), DependsOn: a.DependsOn()})
	}
	writeJSON(w, http.StatusOK, map[string]any{"analyzers": out, "formats": []string{"u8", "f32"}})
}

// analyzeResponse is the body of a successful analyze: the Scene plus the
// cryptolab `ks` frames recovered from any unknown payloads.
type analyzeResponse struct {
	Scene      *rfscope.Scene `json:"scene"`
	FrameCount int            `json:"frame_count"`
	Frames     string         `json:"frames,omitempty"`
}

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, s.maxUpload)
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		writeErr(w, http.StatusBadRequest, "rfscope: "+err.Error())
		return
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "rfscope: a `file` part is required")
		return
	}
	defer file.Close()

	format, err := siglab.ParseSampleFormat(r.FormValue("format"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "rfscope: "+err.Error())
		return
	}
	rate, _ := strconv.ParseFloat(r.FormValue("sample_rate_hz"), 64)
	if rate <= 0 {
		writeErr(w, http.StatusBadRequest, "rfscope: sample_rate_hz is required")
		return
	}
	freq64, _ := strconv.ParseUint(r.FormValue("freq"), 10, 32)

	dir, err := s.ensureTempDir()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "rfscope: temp dir: "+err.Error())
		return
	}
	id := randomID()
	path := filepath.Join(dir, id+".iq")
	dst, err := os.Create(path)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "rfscope: "+err.Error())
		return
	}
	_, copyErr := io.Copy(dst, file)
	closeErr := dst.Close()
	defer os.Remove(path)
	if copyErr != nil || closeErr != nil {
		writeErr(w, http.StatusBadRequest, "rfscope: upload write failed")
		return
	}

	src, err := rfscope.OpenFile(path, format, uint32(freq64), rate)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "rfscope: "+err.Error())
		return
	}
	defer src.Close()

	ctx, cancel := context.WithTimeout(r.Context(), s.timeout)
	defer cancel()

	scene, in, err := rfscope.Segment(ctx, src, s.segConfig(r))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "rfscope: "+err.Error())
		return
	}
	scene.Source = hdr.Filename
	if err := rfscope.Run(ctx, scene, in, s.selectedAnalyzers(r)...); err != nil {
		writeErr(w, http.StatusInternalServerError, "rfscope: "+err.Error())
		return
	}
	scene.Sanitize()

	var frames bytes.Buffer
	n, _ := rfscope.EmitFrames(scene, &frames)
	writeJSON(w, http.StatusOK, analyzeResponse{Scene: scene, FrameCount: n, Frames: frames.String()})
}

func (s *Server) segConfig(r *http.Request) rfscope.SegmentConfig {
	cfg := rfscope.SegmentConfig{}
	if v, err := strconv.Atoi(r.FormValue("fft")); err == nil {
		cfg.FFTSize = v
	}
	if v, err := strconv.ParseFloat(r.FormValue("channel_rate"), 64); err == nil {
		cfg.ChannelTargetRateHz = v
	}
	if v, err := strconv.ParseUint(r.FormValue("min_spacing"), 10, 32); err == nil {
		cfg.MinSpacingHz = uint32(v)
	}
	if win, err := strconv.ParseFloat(r.FormValue("window"), 64); err == nil && win > 0 {
		if rate, err := strconv.ParseFloat(r.FormValue("sample_rate_hz"), 64); err == nil && rate > 0 {
			cfg.MaxSamples = int(win * rate)
		}
	}
	return cfg
}

func (s *Server) selectedAnalyzers(r *http.Request) []string {
	raw := strings.TrimSpace(r.FormValue("analyzers"))
	if raw == "" {
		return nil
	}
	var out []string
	for _, a := range strings.Split(raw, ",") {
		if a = strings.TrimSpace(a); a != "" {
			out = append(out, a)
		}
	}
	return out
}

func (s *Server) spaHandler() http.Handler {
	if s.assets == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			io.WriteString(w, "rfscope web console: SPA not bundled (run `make rfscope-web-build`).")
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

func randomID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
