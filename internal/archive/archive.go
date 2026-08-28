// Package archive verifies a finished project and moves it to cold storage
// (plan §16): re-hash originals against the manifest, rsync to the archive root,
// verify the copy, prune regenerable files, and stamp the manifest. This is the
// only place studio deletes anything, and only files it created.
package archive

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/christophercuongkim/studio/internal/hash"
	"github.com/christophercuongkim/studio/internal/manifest"
	"github.com/christophercuongkim/studio/internal/videoyaml"
)

// prunable are the tool-created directories archive may delete (plan §16).
var prunable = []string{"proxy", filepath.Join("thumbs", "raw")}

// Options configures an archive run.
type Options struct {
	ProjectDir  string
	ArchiveRoot string // from config; empty disables archiving
	DryRun      bool
	KeepProxies bool
	Force       bool // allow archiving an unpublished video
	Now         time.Time
}

// Result reports what archive did.
type Result struct {
	Hashed      int
	ArchivePath string
	Pruned      []string
	DryRun      bool
}

// Run executes the archive pipeline.
func Run(ctx context.Context, opts Options) (*Result, error) {
	if opts.ArchiveRoot == "" {
		return nil, fmt.Errorf("archiveRoot is not set in config — archiving is disabled")
	}
	man, err := manifest.Load(opts.ProjectDir)
	if err != nil {
		return nil, err
	}

	// Precondition: every kept clip must be applied.
	var unapplied []string
	for _, c := range man.Clips {
		if c.Review.Status == manifest.StatusKept && !c.Applied.Done {
			unapplied = append(unapplied, c.ID)
		}
	}
	if len(unapplied) > 0 {
		return nil, fmt.Errorf("not all kept clips are applied (%s); run 'studio apply' first", strings.Join(unapplied, ", "))
	}

	// Warn (via error unless --force) if the video isn't published.
	if !opts.Force {
		if vy, err := videoyaml.Load(opts.ProjectDir); err == nil {
			if vy.YouTube.VideoID == nil || *vy.YouTube.VideoID == "" {
				return nil, fmt.Errorf("video has no youtube.videoId (unpublished); archive with --force to proceed")
			}
		}
	}

	// Verify originals against the manifest before doing anything destructive.
	hashed, err := verifyOriginals(opts.ProjectDir, man)
	if err != nil {
		return nil, err
	}

	dest := filepath.Join(opts.ArchiveRoot, filepath.Base(strings.TrimRight(opts.ProjectDir, string(os.PathSeparator))))
	res := &Result{Hashed: hashed, ArchivePath: dest}

	if opts.DryRun {
		res.DryRun = true
		res.Pruned = pruneTargets(opts)
		return res, nil
	}

	// rsync the project to the archive, then verify the copy by re-hashing.
	if err := rsync(ctx, opts.ProjectDir, dest); err != nil {
		return nil, err
	}
	if _, err := verifyOriginals(dest, man); err != nil {
		return nil, fmt.Errorf("archive copy failed verification: %w", err)
	}

	// Prune regenerable, tool-created dirs.
	for _, rel := range pruneTargets(opts) {
		p := filepath.Join(opts.ProjectDir, rel)
		if err := os.RemoveAll(p); err != nil {
			return nil, fmt.Errorf("prune %s: %w", rel, err)
		}
		res.Pruned = append(res.Pruned, rel)
	}
	// If proxies were pruned, clear their manifest paths so the manifest stays
	// valid (the files are gone). proxyInfo.source is kept so a future
	// 'studio proxies' rebuild knows each clip had a proxy.
	if !opts.KeepProxies {
		for i := range man.Clips {
			man.Clips[i].Files.Proxy = ""
		}
	}

	// Stamp the manifest.
	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	man.ArchivedAt = &now
	man.ArchivePath = dest
	if err := manifest.Save(opts.ProjectDir, man); err != nil {
		return nil, err
	}
	return res, nil
}

// verifyOriginals re-hashes every original under root against the manifest and
// returns the count verified. Any mismatch aborts (bit-rot or tampering).
func verifyOriginals(root string, man *manifest.Manifest) (int, error) {
	var mismatches []string
	n := 0
	for _, c := range man.Clips {
		if c.Media.XXH64 == "" {
			continue
		}
		p := filepath.Join(root, c.Files.Original)
		got, err := hash.XXH64File(p)
		if err != nil {
			return 0, fmt.Errorf("hash %s: %w", c.Files.Original, err)
		}
		if got != c.Media.XXH64 {
			mismatches = append(mismatches, fmt.Sprintf("%s: manifest %s, disk %s", c.Files.Original, c.Media.XXH64, got))
			continue
		}
		n++
	}
	if len(mismatches) > 0 {
		return 0, fmt.Errorf("checksum mismatch — refusing to archive (investigate bit-rot/tampering):\n  - %s", strings.Join(mismatches, "\n  - "))
	}
	return n, nil
}

// pruneTargets is the list of dirs that would be pruned, honoring --keep-proxies.
func pruneTargets(opts Options) []string {
	if opts.KeepProxies {
		// Still prune thumbs/raw (never user media), just not proxy/.
		return []string{filepath.Join("thumbs", "raw")}
	}
	return append([]string(nil), prunable...)
}

// rsync copies src into dest with archive mode and checksum verification.
// Shelling out is deliberate (plan §16): remote archiveRoots work over ssh.
func rsync(ctx context.Context, src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	// Trailing slash on src copies its contents into dest.
	cmd := exec.CommandContext(ctx, "rsync", "-a", "--checksum", src+string(os.PathSeparator), dest+string(os.PathSeparator))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("rsync failed: %w\n%s", err, out)
	}
	return nil
}
