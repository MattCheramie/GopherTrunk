package api

import (
	"io/fs"
	"net/http"
	"strings"

	rfscopeweb "github.com/MattCheramie/GopherTrunk/internal/rfscope/webserver"
)

// RFScopeOptions configures the optional RF Scope console. When Enabled,
// NewServer wires the RF Scope web subsystem and routes() mounts the
// /api/v1/rfscope/* analysis routes plus — when Assets carries an index.html —
// the RF Scope SPA at /rfscope/. The daemon sets it so operators can reach the
// offline RF-analysis console from the main UI in a new tab; the standalone
// `gophertrunk rfscope serve` command keeps using its own webserver at /.
type RFScopeOptions struct {
	Enabled bool
	// Assets is the embedded RF Scope SPA file tree. nil mounts the API only.
	Assets fs.FS
}

// mountRFScope registers the RF Scope console on the daemon mux: the
// /api/v1/rfscope/* routes (the analyze mutation gated like every other write
// route) and, when the SPA is bundled, the console at /rfscope/. A no-op when
// the subsystem was not enabled. The webserver owns a lazily-created temp dir
// that shutdown() reclaims via rfscopeCloser. Advertised via
// RuntimeDTO.RFScopeConsole so the other consoles only link here when the SPA
// is actually reachable.
func (s *Server) mountRFScope(mux *http.ServeMux) {
	if !s.rfscopeOpts.Enabled {
		return
	}
	svc, err := rfscopeweb.New(rfscopeweb.Options{Assets: s.rfscopeOpts.Assets, Logger: s.log})
	if err != nil {
		if s.log != nil {
			s.log.Warn("rfscope console disabled", "err", err)
		}
		return
	}
	s.rfscopeCloser = svc
	svc.RegisterAPI(mux, s.gate)

	// Secondary SPA at /rfscope/ (daemon path); the standalone `rfscope serve`
	// serves it at / instead. The /rfscope/ matcher is more specific than the
	// GET / catch-all, so both trees coexist.
	if s.rfscopeOpts.Assets != nil {
		if _, err := fs.Stat(s.rfscopeOpts.Assets, "index.html"); err == nil {
			mux.Handle("GET /rfscope/", s.rfscopeSpaHandler())
			mux.HandleFunc("GET /rfscope", func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "/rfscope/", http.StatusMovedPermanently)
			})
			s.rfscopeConsole = true
		}
	}
}

// rfscopeSpaHandler serves the RF Scope SPA mounted under /rfscope/ on the
// daemon, stripping the prefix and falling back to index.html for client-side
// routes (mirrors siglabSpaHandler / cryptolabSpaHandler).
func (s *Server) rfscopeSpaHandler() http.Handler {
	assets := s.rfscopeOpts.Assets
	fileSrv := http.FileServerFS(assets)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/rfscope/"), "/")
		if rel != "" {
			if _, err := fs.Stat(assets, rel); err == nil {
				r2 := r.Clone(r.Context())
				r2.URL.Path = "/" + rel
				fileSrv.ServeHTTP(w, r2)
				return
			}
		}
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/"
		fileSrv.ServeHTTP(w, r2)
	})
}
