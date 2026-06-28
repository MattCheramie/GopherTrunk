// Package cryptolabweb embeds the built standalone Crypto Lab SPA so the
// gophertrunk binary can serve the cryptolab web console (via `gophertrunk
// cryptolab serve`) without a sibling source tree. Build the SPA with `make
// cryptolab-web-build` (or `cd web/cryptolab && npm run build`) before `go
// build`; the embed picks up everything under `dist/` automatically.
//
// When `dist/` is empty (fresh checkout, dev build with no build yet) the
// embed contains only the `.gitkeep` sentinel and HasAssets reports false;
// the serve command then prints an explanatory message instead of a blank
// page.
package cryptolabweb

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var rawDist embed.FS

// Assets returns the embed.FS sub-tree rooted at `dist`, suitable for the
// cryptolab webserver's Options.Assets.
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
