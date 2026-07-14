package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/gtbundle"
)

// bundleInfoDTO is the web-facing projection of a bundle's manifest: enough for
// a console to render a case summary and its file inventory without unpacking
// the archive.
type bundleInfoDTO struct {
	Name          string          `json:"name"`
	Title         string          `json:"title,omitempty"`
	Format        string          `json:"format"`
	SchemaVersion int             `json:"schema_version"`
	CaptureIntent string          `json:"capture_intent"`
	CreatedAt     string          `json:"created_at"`
	Generator     string          `json:"generator"`
	Protocol      string          `json:"protocol,omitempty"`
	Source        string          `json:"source,omitempty"`
	FutureSchema  bool            `json:"future_schema,omitempty"`
	Files         []bundleFileDTO `json:"files"`
	Counts        map[string]int  `json:"role_counts"`
}

type bundleFileDTO struct {
	Path       string `json:"path"`
	Role       string `json:"role"`
	MediaType  string `json:"media_type"`
	SizeBytes  int64  `json:"size_bytes"`
	SHA256     string `json:"sha256"`
	ProducedBy string `json:"produced_by,omitempty"`
}

// bundleVerifyDTO reports the per-file integrity outcome of a bundle.
type bundleVerifyDTO struct {
	OK        bool                  `json:"ok"`
	Files     []bundleVerifyFileDTO `json:"files"`
	Failures  int                   `json:"failures"`
	Untracked int                   `json:"untracked"`
}

type bundleVerifyFileDTO struct {
	Path      string `json:"path"`
	Role      string `json:"role,omitempty"`
	OK        bool   `json:"ok"`
	Untracked bool   `json:"untracked,omitempty"`
	Error     string `json:"error,omitempty"`
}

// resolveBundlePath validates and resolves a client-supplied bundle path against
// the server's allowed roots. It requires a .gtb / .tar.gz suffix and, when
// bundleRoot is configured, that the path stays under it — so the endpoint can't
// be used to read arbitrary files off the host.
func (s *Server) resolveBundlePath(p string) (string, bool) {
	if p == "" {
		return "", false
	}
	if !strings.HasSuffix(p, ".gtb.tar.gz") && !strings.HasSuffix(p, ".tar.gz") && !strings.HasSuffix(p, ".gtb") {
		return "", false
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", false
	}
	if s.bundleRoot != "" {
		rel, err := filepath.Rel(s.bundleRoot, abs)
		if err != nil || strings.HasPrefix(rel, "..") {
			return "", false
		}
	}
	if fi, err := os.Stat(abs); err != nil || fi.IsDir() {
		return "", false
	}
	return abs, true
}

// handleBundleInfo serves GET /api/v1/bundle/info?path=… — the manifest summary
// of a bundle reachable on the daemon host.
func (s *Server) handleBundleInfo(w http.ResponseWriter, r *http.Request) {
	path, ok := s.resolveBundlePath(r.URL.Query().Get("path"))
	if !ok {
		http.Error(w, "invalid or disallowed bundle path", http.StatusBadRequest)
		return
	}
	rd, err := gtbundle.Open(path)
	if err != nil {
		http.Error(w, "open bundle: "+err.Error(), http.StatusBadRequest)
		return
	}
	m := rd.Manifest()
	dto := bundleInfoDTO{
		Name:          m.Name,
		Title:         m.Title,
		Format:        m.Format,
		SchemaVersion: m.SchemaVersion,
		CaptureIntent: string(m.CaptureIntent),
		CreatedAt:     m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Generator:     m.Generator.Tool + " " + m.Generator.Version,
		Protocol:      m.Provenance.Protocol,
		Source:        m.Provenance.Source,
		FutureSchema:  rd.FutureSchema(),
		Counts:        map[string]int{},
	}
	for _, f := range m.Files {
		dto.Files = append(dto.Files, bundleFileDTO{
			Path:       f.Path,
			Role:       string(f.Role),
			MediaType:  f.MediaType,
			SizeBytes:  f.SizeBytes,
			SHA256:     f.SHA256,
			ProducedBy: f.ProducedBy,
		})
		dto.Counts[string(f.Role)]++
	}
	writeJSON(w, http.StatusOK, dto)
}

// handleBundleVerify serves POST /api/v1/bundle/verify with a JSON body
// {"path": "..."} — recompute and check every file's sha256.
func (s *Server) handleBundleVerify(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	path, ok := s.resolveBundlePath(body.Path)
	if !ok {
		http.Error(w, "invalid or disallowed bundle path", http.StatusBadRequest)
		return
	}
	rd, err := gtbundle.Open(path)
	if err != nil {
		http.Error(w, "open bundle: "+err.Error(), http.StatusBadRequest)
		return
	}
	out := bundleVerifyDTO{OK: true}
	for _, v := range rd.Verify() {
		fd := bundleVerifyFileDTO{Path: v.Path, Role: string(v.Role), OK: v.OK, Untracked: v.Untracked}
		if v.Err != nil {
			fd.Error = v.Err.Error()
		}
		if v.Untracked {
			out.Untracked++
		} else if !v.OK {
			out.Failures++
			out.OK = false
		}
		out.Files = append(out.Files, fd)
	}
	writeJSON(w, http.StatusOK, out)
}

// handleBundleDownload serves GET /api/v1/bundle/download?path=… — stream a
// bundle to the client for download.
func (s *Server) handleBundleDownload(w http.ResponseWriter, r *http.Request) {
	path, ok := s.resolveBundlePath(r.URL.Query().Get("path"))
	if !ok {
		http.Error(w, "invalid or disallowed bundle path", http.StatusBadRequest)
		return
	}
	f, err := os.Open(path)
	if err != nil {
		http.Error(w, "open bundle: "+err.Error(), http.StatusNotFound)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filepath.Base(path)+"\"")
	http.ServeContent(w, r, filepath.Base(path), fileModTime(f), f)
}

func fileModTime(f *os.File) time.Time {
	if fi, err := f.Stat(); err == nil {
		return fi.ModTime()
	}
	return time.Time{}
}
