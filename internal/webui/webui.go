// Package webui embeds the browser frontends served by studio's local servers.
// Bundling them into the binary keeps studio a single static executable (plan
// §2) with no build step and no external asset paths.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed serve/index.html serve/app.js serve/app.css
var files embed.FS

// ServeFS returns the file tree for the `studio serve` review UI, rooted so
// index.html is served at "/".
func ServeFS() fs.FS {
	sub, err := fs.Sub(files, "serve")
	if err != nil {
		panic(err) // embed path is a compile-time constant; can't fail at runtime
	}
	return sub
}
