// Package webui embeds the browser frontends served by studio's local servers.
// Bundling them into the binary keeps studio a single static executable (plan
// §2) with no build step and no external asset paths.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed serve/index.html serve/app.js serve/app.css
//go:embed prompt/index.html prompt/app.js prompt/app.css
//go:embed seakim/styles.css seakim/tokens/*.css seakim/fonts/*.woff2
var files embed.FS

// ServeFS returns the file tree for the `studio serve` review UI, rooted so
// index.html is served at "/".
func ServeFS() fs.FS {
	return sub("serve")
}

// PromptFS returns the file tree for the `studio prompt` teleprompter UI.
func PromptFS() fs.FS {
	return sub("prompt")
}

// SeakimFS returns the vendored SeaKim design-system runtime (styles.css +
// tokens + fonts), served under /seakim/ so both UIs can link it. Keeping it in
// the binary preserves studio's no-external-assets invariant.
func SeakimFS() fs.FS {
	return sub("seakim")
}

func sub(dir string) fs.FS {
	f, err := fs.Sub(files, dir)
	if err != nil {
		panic(err) // embed path is a compile-time constant; can't fail at runtime
	}
	return f
}
