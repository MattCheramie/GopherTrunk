//go:build cryptolab

package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/webserver"
	cryptolabweb "github.com/MattCheramie/GopherTrunk/web/cryptolab"

	// Register the cryptolab tools and subjects so the daemon-mounted console
	// (like the standalone `cryptolab serve`) sees the full tool schema.
	_ "github.com/MattCheramie/GopherTrunk/internal/cryptolab/subjects/motorola"
	_ "github.com/MattCheramie/GopherTrunk/internal/cryptolab/tools"
)

// mountCryptolab registers the cryptolab research console on the daemon mux:
// the /api/v1/cryptolab/* routes (mutations gated like every other daemon
// write route) and, when the SPA is bundled, the console at /cryptolab/. It is
// linked only in builds tagged `cryptolab`, so the default daemon binary is
// unaffected. The webserver service owns a temp dir that shutdown() reclaims
// via cryptolabCloser.
func (s *Server) mountCryptolab(mux *http.ServeMux) {
	svc, _ := s.cryptolabCloser.(*webserver.Server)
	if svc == nil {
		var err error
		svc, err = webserver.New(webserver.Options{Logger: s.log})
		if err != nil {
			if s.log != nil {
				s.log.Warn("cryptolab console disabled", "err", err)
			}
			return
		}
		s.cryptolabCloser = svc
	}
	svc.RegisterAPI(mux, s.gate)

	// Secondary SPA at /cryptolab/ (daemon path); the standalone `cryptolab
	// serve` serves it at / instead. The /cryptolab/ matcher is more specific
	// than the GET / catch-all, so both trees coexist.
	if cryptolabweb.HasAssets() {
		mux.Handle("GET /cryptolab/", s.cryptolabSpaHandler())
		mux.HandleFunc("GET /cryptolab", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/cryptolab/", http.StatusMovedPermanently)
		})
	}
}

// cryptolabSpaHandler serves the cryptolab SPA mounted under /cryptolab/,
// stripping the prefix and falling back to index.html for client-side routes
// (mirrors configSpaHandler).
func (s *Server) cryptolabSpaHandler() http.Handler {
	assets := cryptolabweb.Assets()
	fileSrv := http.FileServerFS(assets)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/cryptolab/"), "/")
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
