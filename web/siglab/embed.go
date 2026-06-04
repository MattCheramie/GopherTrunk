// Package siglabweb embeds the built standalone Signal Lab SPA so the
// gophertrunk binary can serve the offline signal-analysis console (via
// `gophertrunk siglab serve`, and from the daemon) without a sibling source
// tree. Build the SPA with `make siglab-web-build` (or `cd web/siglab &&
// npm run build`) before `go build`; the embed picks up everything under
// `dist/` automatically.
//
// When `dist/` is empty (fresh checkout, dev build with no build yet) the
// embed contains only the `.gitkeep` sentinel and HasAssets reports false;
// the serve command then prints an explanatory message instead of a blank
// page.
package siglabweb

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var rawDist embed.FS

// Assets returns the embed.FS sub-tree rooted at `dist`, suitable for
// http.FileServerFS / the api package's WebAssets option.
func Assets() fs.FS {
	sub, err := fs.Sub(rawDist, "dist")
	if err != nil {
		return rawDist
	}
	return sub
}

// HasAssets returns true when the embed contains a real build (index.html).
func HasAssets() bool {
	_, err := fs.Stat(Assets(), "index.html")
	return err == nil
}
