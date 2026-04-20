// Package frontendassets embeds the built PWA bundle (frontend/dist).
// The dist directory is produced by `npm run build` and must exist before
// building the Go binary.
package frontendassets

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// DistFS returns the embedded dist/ directory as a filesystem.
func DistFS() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
