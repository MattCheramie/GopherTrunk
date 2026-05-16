// Package web embeds the built SPA so the daemon binary can serve
// the operator console without a sibling `gophertrunk-web/`
// directory. Build the SPA with `make web-build` (or `cd web &&
// npm run build`) before `go build`; the embed picks up everything
// under `dist/` automatically.
//
// When `dist/` is empty (fresh checkout, dev build with no `make
// web-build` yet) the embed contains only the `.gitkeep` sentinel
// and (*FS).HasAssets reports false. The launcher's web-open path
// falls back to filesystem discovery in that case so dev workflows
// aren't blocked.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var rawDist embed.FS

// Assets returns the embed.FS sub-tree rooted at `dist`. Callers
// should treat it as a read-only fs.FS — internal/api wires it
// through http.FileServerFS.
func Assets() fs.FS {
	sub, err := fs.Sub(rawDist, "dist")
	if err != nil {
		return rawDist
	}
	return sub
}

// HasAssets returns true when the embed contains real build output
// (index.html in particular). A fresh checkout with only the
// .gitkeep sentinel returns false; the launcher then falls back to
// filesystem-search of `gophertrunk-web/` siblings.
func HasAssets() bool {
	_, err := fs.Stat(Assets(), "index.html")
	return err == nil
}
