// Package rfscopeweb embeds the built standalone RF Scope SPA so the gophertrunk
// binary can serve the offline RF-analysis console (via `gophertrunk rfscope
// serve`) without a sibling source tree. Build the SPA with
// `make rfscope-web-build` (or `cd web/rfscope && npm run build`) before
// `go build`; the embed picks up everything under `dist/` automatically.
//
// When `dist/` holds only the `.gitkeep` sentinel (a fresh checkout with no
// build yet) HasAssets reports false and the serve command prints an
// explanatory placeholder instead of a blank page.
package rfscopeweb

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var rawDist embed.FS

// Assets returns the embed.FS sub-tree rooted at dist, suitable for
// http.FileServerFS / the webserver's Assets option.
func Assets() fs.FS {
	sub, err := fs.Sub(rawDist, "dist")
	if err != nil {
		return rawDist
	}
	return sub
}

// HasAssets reports whether the embed contains a real build (an index.html).
func HasAssets() bool {
	_, err := fs.Stat(Assets(), "index.html")
	return err == nil
}
