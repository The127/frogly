// Package web holds frogly's frontend, a Svelte app. `just web` builds it
// into dist, and the frogly binary carries that build.
package web

import (
	"embed"
	"io/fs"
)

// dist holds only .gitkeep until `just web` ran, so Go compiles frogly on
// a fresh clone. The binary then serves no frontend, only the API. Vite
// empties dist on every build and copies .gitkeep back from public.
//
//go:embed all:dist
var dist embed.FS

// Files is the built frontend, with index.html at its root.
func Files() fs.FS {
	files, err := fs.Sub(dist, "dist")
	if err != nil {
		// fs.Sub only fails for an invalid path, and "dist" is a valid one
		panic(err)
	}

	return files
}
