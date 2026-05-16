package api

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ImportSourceKind discriminates the upload's payload format.
type ImportSourceKind string

const (
	ImportSourcePDF ImportSourceKind = "pdf"
	ImportSourceCSV ImportSourceKind = "csv"
)

// ImportSource is one uploaded file in a multipart import request.
// Filename + content kept in memory so the handler can pipe them to
// the daemon's parsers without retaining a file descriptor.
type ImportSource struct {
	Filename string
	Kind     ImportSourceKind
	Data     []byte
}

// ParsedSystemDTO is the JSON projection of one parsed
// system / site / talkgroup tree. It deliberately mirrors the
// cmd/gophertrunk parsedSystem shape so the SPA / TUI can render the
// preview verbatim without learning a third schema.
type ParsedSystemDTO struct {
	Name        string                 `json:"name"`
	Location    string                 `json:"location,omitempty"`
	County      string                 `json:"county,omitempty"`
	SysID       string                 `json:"sysid,omitempty"`
	WACN        string                 `json:"wacn,omitempty"`
	SystemType  string                 `json:"system_type,omitempty"`
	Protocol    string                 `json:"protocol"`
	SiteCount   int                    `json:"site_count"`
	TalkgroupCt int                    `json:"talkgroup_count"`
	SourcePath  string                 `json:"source_path,omitempty"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
}

// Importer is the daemon-side import surface. Decoupled via interface
// so the api package doesn't have to reach into cmd/gophertrunk's
// parser internals. The daemon supplies an adapter that delegates to
// parsePDFFile, parseCSVFile, mergeIntoConfig.
type Importer interface {
	// Parse runs the relevant parser (PDF or CSV) against the
	// supplied source and returns a preview DTO.
	Parse(s ImportSource) (ParsedSystemDTO, error)
	// Commit finalises a previously-parsed batch by merging it
	// into config.yaml and refreshing the in-memory talkgroup DB.
	// The implementer is responsible for serialising commits with
	// any other config writer (settings PATCH) so two callers
	// can't race the on-disk file.
	Commit(sources []ImportSource, force bool) (ImportCommitResult, error)
}

// ImportCommitResult is the response shape for a successful commit.
type ImportCommitResult struct {
	SystemsAdded    []string `json:"systems_added"`
	SystemsReplaced []string `json:"systems_replaced"`
	CSVPaths        []string `json:"csv_paths,omitempty"`
	ConfigPath      string   `json:"config_path,omitempty"`
}

// ImportPreviewResponse is the response shape for POST /api/v1/import.
type ImportPreviewResponse struct {
	ID      string            `json:"id"`
	Systems []ParsedSystemDTO `json:"systems"`
}

// importStaging is the in-memory hold area for parsed uploads
// awaiting commit. A TTL sweep drops abandoned previews so memory
// stays bounded.
type importStaging struct {
	mu      sync.Mutex
	entries map[string]*stagingEntry
	ttl     time.Duration
}

type stagingEntry struct {
	created time.Time
	sources []ImportSource
	systems []ParsedSystemDTO
}

func newImportStaging(ttl time.Duration) *importStaging {
	return &importStaging{
		entries: make(map[string]*stagingEntry),
		ttl:     ttl,
	}
}

func (st *importStaging) put(sources []ImportSource, systems []ParsedSystemDTO) string {
	id := randomID(16)
	st.mu.Lock()
	defer st.mu.Unlock()
	st.entries[id] = &stagingEntry{
		created: time.Now(),
		sources: sources,
		systems: systems,
	}
	return id
}

func (st *importStaging) take(id string) (*stagingEntry, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	e, ok := st.entries[id]
	if !ok {
		return nil, false
	}
	delete(st.entries, id)
	return e, true
}

func (st *importStaging) sweep(now time.Time) {
	st.mu.Lock()
	defer st.mu.Unlock()
	for id, e := range st.entries {
		if now.Sub(e.created) > st.ttl {
			delete(st.entries, id)
		}
	}
}

// runStagingSweeper kicks a background sweep every ttl/4 until ctx
// cancels. Call once from the daemon's spawn fan.
func (st *importStaging) runStagingSweeper(stopCh <-chan struct{}) {
	t := time.NewTicker(st.ttl / 4)
	defer t.Stop()
	for {
		select {
		case <-stopCh:
			return
		case now := <-t.C:
			st.sweep(now)
		}
	}
}

// randomID returns a hex-encoded n-byte random string. Used for the
// staging ID; not security-critical (the endpoint is auth-gated).
func randomID(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		// Fall back to a timestamp; collisions are very unlikely
		// because the endpoint serialises commit requests.
		return fmt.Sprintf("ts-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

// handleImportUpload processes POST /api/v1/import. The body is a
// multipart/form-data envelope with one or more `files[]` parts; each
// file is parsed in memory and the resulting preview is stored under
// a fresh staging ID. The operator then POSTs to
// /api/v1/import/{id}/commit to finalise.
func (s *Server) handleImportUpload(w http.ResponseWriter, r *http.Request) {
	if s.importer == nil {
		writeError(w, http.StatusServiceUnavailable, "import: not wired (daemon started without a -config file)")
		return
	}
	// 20 MiB max — generous for the largest RadioReference PDFs and
	// a CSV bundle, while bounded enough that an accidental upload
	// can't OOM the daemon.
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "import: "+err.Error())
		return
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		writeError(w, http.StatusBadRequest, "import: at least one `files` part is required")
		return
	}

	sources := make([]ImportSource, 0, len(files))
	previews := make([]ParsedSystemDTO, 0, len(files))
	for _, fh := range files {
		kind := classifyImportFile(fh.Filename)
		if kind == "" {
			writeError(w, http.StatusBadRequest,
				fmt.Sprintf("import: %q: unsupported extension (need .pdf or .csv)", fh.Filename))
			return
		}
		f, err := fh.Open()
		if err != nil {
			writeError(w, http.StatusBadRequest,
				fmt.Sprintf("import: %q: %v", fh.Filename, err))
			return
		}
		body, err := io.ReadAll(f)
		_ = f.Close()
		if err != nil {
			writeError(w, http.StatusBadRequest,
				fmt.Sprintf("import: %q: %v", fh.Filename, err))
			return
		}
		src := ImportSource{Filename: fh.Filename, Kind: kind, Data: body}
		preview, err := s.importer.Parse(src)
		if err != nil {
			writeError(w, http.StatusBadRequest,
				fmt.Sprintf("import: %q: %v", fh.Filename, err))
			return
		}
		sources = append(sources, src)
		previews = append(previews, preview)
	}
	id := s.imports.put(sources, previews)
	writeJSON(w, http.StatusOK, ImportPreviewResponse{ID: id, Systems: previews})
}

// handleImportCommit finalises a staged upload by merging the parsed
// sources into config.yaml + writing the per-system CSVs. The
// staging entry is consumed on success.
func (s *Server) handleImportCommit(w http.ResponseWriter, r *http.Request) {
	if s.importer == nil {
		writeError(w, http.StatusServiceUnavailable, "import: not wired")
		return
	}
	id := r.PathValue("id")
	entry, ok := s.imports.take(id)
	if !ok {
		writeError(w, http.StatusNotFound, "import: staging id not found (expired or already committed)")
		return
	}
	force := r.URL.Query().Get("force") == "true"
	result, err := s.importer.Commit(entry.sources, force)
	if err != nil {
		// Re-stash so the operator can retry without re-uploading.
		s.imports.mu.Lock()
		s.imports.entries[id] = entry
		s.imports.mu.Unlock()
		writeError(w, http.StatusBadRequest, "import: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// handleImportDiscard drops a staged upload without committing.
func (s *Server) handleImportDiscard(w http.ResponseWriter, r *http.Request) {
	if s.importer == nil {
		writeError(w, http.StatusServiceUnavailable, "import: not wired")
		return
	}
	id := r.PathValue("id")
	if _, ok := s.imports.take(id); !ok {
		writeError(w, http.StatusNotFound, "import: staging id not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// classifyImportFile picks the parser based on filename extension.
// Returns "" for unsupported types so the handler can 400 fast.
func classifyImportFile(name string) ImportSourceKind {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".pdf":
		return ImportSourcePDF
	case ".csv":
		return ImportSourceCSV
	}
	return ""
}

// ensureImporterReady is used by handlers that need an Importer
// configured. Returns a typed error so the caller can write the
// right status code.
func (s *Server) ensureImporterReady() error {
	if s.importer == nil {
		return errors.New("api: no Importer configured")
	}
	return nil
}
