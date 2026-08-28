// Package ingest turns a raw camera-card dump into a studio project layout plus
// a manifest.json (plan §6). It walks and groups files, probes and checksums
// originals, moves everything into place, and resolves a proxy per clip.
package ingest

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// File kinds, by extension.
type kind int

const (
	kindOther   kind = iota // ignored
	kindMedia               // an original: .mp4 .mov .mxf .avi .mts .m4v
	kindProxy               // a camera low-res proxy candidate: .lrf (DJI) .lrv (GoPro)
	kindSidecar             // metadata kept alongside: .srt .thm .xml .wav
)

// extKind maps a lowercased extension (with dot) to its kind. The .lrf/.lrv
// files are split out from the other sidecars because, unlike .srt/.thm, they
// are candidate proxies to be probed and possibly adopted (plan §6 step 6).
var extKind = map[string]kind{
	".mp4": kindMedia, ".mov": kindMedia, ".mxf": kindMedia,
	".avi": kindMedia, ".mts": kindMedia, ".m4v": kindMedia,

	".lrf": kindProxy, ".lrv": kindProxy,

	".srt": kindSidecar, ".thm": kindSidecar,
	".xml": kindSidecar, ".wav": kindSidecar,
}

// foundFile is one relevant file discovered under the dump directory.
type foundFile struct {
	path string // absolute path
	base string // filename without extension (original case)
	ext  string // lowercased extension, with dot
	kind kind
	size int64
}

// scanResult is the outcome of walking the dump directory.
type scanResult struct {
	files     []foundFile
	zeroBytes []string // paths of skipped zero-byte files (warned)
}

// scan walks dumpDir recursively, collecting media/proxy/sidecar files. Hidden
// files (dot-prefixed) and zero-byte files are skipped; zero-byte paths are
// reported so the caller can warn.
func scan(dumpDir string) (scanResult, error) {
	var res scanResult
	err := filepath.WalkDir(dumpDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			// Skip hidden directories entirely (e.g. macOS .Spotlight-V100).
			if path != dumpDir && strings.HasPrefix(name, ".") {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") {
			return nil // hidden file
		}
		ext := strings.ToLower(filepath.Ext(name))
		k, ok := extKind[ext]
		if !ok {
			return nil // irrelevant extension
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Size() == 0 {
			res.zeroBytes = append(res.zeroBytes, path)
			return nil
		}
		res.files = append(res.files, foundFile{
			path: path,
			base: strings.TrimSuffix(name, filepath.Ext(name)),
			ext:  ext,
			kind: k,
			size: info.Size(),
		})
		return nil
	})
	if err != nil {
		return scanResult{}, err
	}
	return res, nil
}
