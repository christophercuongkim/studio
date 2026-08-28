package ingest

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/christophercuongkim/studio/internal/manifest"
	"github.com/christophercuongkim/studio/internal/probe"
	"github.com/christophercuongkim/studio/internal/proxy"
)

// ErrManifestExists is returned when a manifest is already present and --append
// was not given.
var ErrManifestExists = errors.New("manifest.json already exists in project (use --append to add new footage)")

// ErrInterrupted signals the run was cancelled (e.g. Ctrl-C). A manifest with
// the work completed so far is still written.
var ErrInterrupted = errors.New("ingest interrupted")

// Options configures a single ingest run.
type Options struct {
	DumpDir     string
	ProjectDir  string
	CamCode     string    // camera code for the shoot (falls back to "CAM")
	Copy        bool      // copy instead of move; leaves the dump untouched
	ClearSource bool      // with Copy: after a verified copy, delete the sources (empty the card)
	Jobs        int       // parallelism for checksum + proxy work; <1 means auto
	Append      bool      // add to an existing manifest
	Now         time.Time // shoot/creation timestamp; zero means time.Now()
}

// Summary reports what an ingest run did, for the CLI to print.
type Summary struct {
	NewClips          int
	AdoptedProxies    int
	GeneratedProxies  int
	FailedProxies     int
	SkippedExisting   int // append: groups already in the manifest
	ProbeFailures     []string
	UnmatchedSidecars []string
	ZeroByteSkipped   []string
	RejectedProxies   []string
	TotalBytes        int64
	Elapsed           time.Duration
	Interrupted       bool
	SourceCleared     int      // sources deleted after a verified copy (--clear-source)
	SourceKept        []string // sources kept because their copy didn't verify
}

// jobs returns the effective parallelism (default runtime.NumCPU()/2, min 1).
func (o Options) jobs() int {
	if o.Jobs >= 1 {
		return o.Jobs
	}
	if n := runtime.NumCPU() / 2; n >= 1 {
		return n
	}
	return 1
}

func (o Options) now() time.Time {
	if o.Now.IsZero() {
		return time.Now().UTC()
	}
	return o.Now.UTC()
}

// Run executes the ingest pipeline (plan §6). It makes no filesystem changes
// until every pre-move step (scan, group, probe, checksum, collision check) has
// succeeded.
func Run(ctx context.Context, opts Options) (*Summary, error) {
	start := opts.now()
	sum := &Summary{}

	info, err := os.Stat(opts.DumpDir)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("dump directory %s is not accessible", opts.DumpDir)
	}

	manifestPath := filepath.Join(opts.ProjectDir, manifest.FileName)
	var man *manifest.Manifest
	if _, err := os.Stat(manifestPath); err == nil {
		if !opts.Append {
			return nil, ErrManifestExists
		}
		man, err = manifest.Load(opts.ProjectDir)
		if err != nil {
			return nil, fmt.Errorf("load existing manifest for --append: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	// --- pre-move discovery (no changes to disk) ---
	scanRes, err := scan(opts.DumpDir)
	if err != nil {
		return nil, fmt.Errorf("scan dump: %w", err)
	}
	sum.ZeroByteSkipped = scanRes.zeroBytes

	groups, unmatched := groupFiles(scanRes.files)
	for _, u := range unmatched {
		sum.UnmatchedSidecars = append(sum.UnmatchedSidecars, u.path)
	}

	// Probe originals; drop (and leave in the dump) any group whose original
	// fails to probe.
	type probed struct {
		g    group
		orig probe.Result
	}
	var kept []probed
	for _, g := range groups {
		res, err := probe.File(ctx, g.original.path)
		if err != nil {
			sum.ProbeFailures = append(sum.ProbeFailures, fmt.Sprintf("%s: %v", g.original.path, err))
			continue
		}
		kept = append(kept, probed{g: g, orig: res})
	}

	// Checksum originals in parallel.
	sums := make([]string, len(kept))
	cerrs := make([]error, len(kept))
	parallel(opts.jobs(), len(kept), func(i int) {
		sums[i], cerrs[i] = xxh64File(kept[i].g.original.path)
	})
	for _, e := range cerrs {
		if e != nil {
			return nil, e // checksum failure is fatal; nothing has moved yet
		}
	}

	// --append: drop groups whose checksum already exists in the manifest.
	existing := map[string]bool{}
	nextSeq := 1
	if man != nil {
		for _, c := range man.Clips {
			existing[c.Media.XXH64] = true
			if c.Seq >= nextSeq {
				nextSeq = c.Seq + 1
			}
		}
		var filtered []probed
		filteredSums := sums[:0:0]
		for i, p := range kept {
			if existing[sums[i]] {
				sum.SkippedExisting++
				continue
			}
			filtered = append(filtered, p)
			filteredSums = append(filteredSums, sums[i])
		}
		kept, sums = filtered, filteredSums
	}

	if len(kept) == 0 {
		// Nothing to do, but still surface warnings. Don't touch the manifest.
		sum.Elapsed = opts.now().Sub(start)
		return sum, nil
	}

	// Plan moves for originals + sidecars + extras into originals/, and detect
	// flattening collisions before anything is moved.
	originalsDir := filepath.Join(opts.ProjectDir, "originals")
	var pairs []movePair
	for _, p := range kept {
		pairs = append(pairs, movePair{src: p.g.original.path, dst: filepath.Join(originalsDir, filepath.Base(p.g.original.path))})
		for _, s := range p.g.sidecars {
			pairs = append(pairs, movePair{src: s.path, dst: filepath.Join(originalsDir, filepath.Base(s.path))})
		}
		for _, e := range p.g.extras {
			pairs = append(pairs, movePair{src: e.path, dst: filepath.Join(originalsDir, filepath.Base(e.path))})
		}
	}
	if clashes := detectCollisions(pairs); len(clashes) > 0 {
		var b strings.Builder
		b.WriteString("flattened-name collisions (aborting before any move):\n")
		for _, c := range clashes {
			fmt.Fprintf(&b, "  %s\n    <- %s\n    <- %s\n", filepath.Base(c.dst), c.srcA, c.srcB)
		}
		return nil, errors.New(b.String())
	}
	// Also refuse if any destination already exists on disk (e.g. --append
	// re-importing a same-named file from a different card).
	for _, p := range pairs {
		if _, err := os.Stat(p.dst); err == nil {
			return nil, fmt.Errorf("destination already exists: %s (from %s)", p.dst, p.src)
		}
	}

	// --- from here on, the filesystem is mutated ---
	for _, d := range []string{originalsDir, filepath.Join(opts.ProjectDir, "proxy"), filepath.Join(opts.ProjectDir, "undo")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}

	// Move originals + sidecars + extras.
	moved := make([]movedGroup, len(kept))
	for i, p := range kept {
		mg := movedGroup{stem: p.g.stem}
		mg.originalDst = filepath.Join(originalsDir, filepath.Base(p.g.original.path))
		if err := moveFile(p.g.original.path, mg.originalDst, opts.Copy); err != nil {
			return nil, err
		}
		for _, s := range p.g.sidecars {
			dst := filepath.Join(originalsDir, filepath.Base(s.path))
			if err := moveFile(s.path, dst, opts.Copy); err != nil {
				return nil, err
			}
			mg.sidecarDsts = append(mg.sidecarDsts, dst)
		}
		for _, e := range p.g.extras {
			dst := filepath.Join(originalsDir, filepath.Base(e.path))
			if err := moveFile(e.path, dst, opts.Copy); err != nil {
				return nil, err
			}
			mg.sidecarDsts = append(mg.sidecarDsts, dst)
		}
		mg.proxyCand = p.g.proxyCand
		moved[i] = mg
	}

	// Set unmatched sidecars aside (never dropped, never in the manifest).
	if len(unmatched) > 0 {
		unmatchedDir := filepath.Join(originalsDir, "_unmatched")
		for _, u := range unmatched {
			_ = moveFile(u.path, filepath.Join(unmatchedDir, filepath.Base(u.path)), opts.Copy)
		}
	}

	// Resolve proxies in parallel; ffmpeg stderr is teed to the ingest log.
	logPath := filepath.Join(opts.ProjectDir, "undo", fmt.Sprintf("ingest-%s.log", start.Format("20060102T150405Z")))
	lg := &runLog{path: logPath}
	proxyInfos := make([]manifest.ProxyInfo, len(kept))
	proxyRels := make([]string, len(kept))
	rejected := make([]string, len(kept))
	interrupted := false
	var mu sync.Mutex

	parallel(opts.jobs(), len(kept), func(i int) {
		if ctx.Err() != nil {
			mu.Lock()
			interrupted = true
			mu.Unlock()
			proxyInfos[i] = manifest.ProxyInfo{Source: manifest.ProxyNone, Note: "interrupted before proxy resolution"}
			return
		}
		info, rel, rej := resolveProxy(ctx, opts, moved[i], kept[i].orig, lg)
		proxyInfos[i] = info
		proxyRels[i] = rel
		rejected[i] = rej
	})
	for _, r := range rejected {
		if r != "" {
			sum.RejectedProxies = append(sum.RejectedProxies, r)
		}
	}

	// Tally proxy outcomes.
	for _, pi := range proxyInfos {
		switch pi.Source {
		case manifest.ProxyCamera:
			sum.AdoptedProxies++
		case manifest.ProxyGenerated:
			sum.GeneratedProxies++
		case manifest.ProxyNone:
			sum.FailedProxies++
		}
	}

	// Build the manifest.
	if man == nil {
		man = &manifest.Manifest{
			SchemaVersion: manifest.SchemaVersion,
			Shoot: manifest.Shoot{
				Root:      absOrSame(opts.ProjectDir),
				Title:     deriveTitle(opts.ProjectDir),
				CreatedAt: start,
				CamCode:   camOrDefault(opts.CamCode),
			},
		}
	}
	for i, p := range kept {
		seq := nextSeq + i
		clip := manifest.Clip{
			ID:   fmt.Sprintf("c-%03d", seq),
			Seq:  seq,
			Stem: p.g.stem,
			Files: manifest.Files{
				Original: relPath(opts.ProjectDir, moved[i].originalDst),
				Proxy:    proxyRels[i],
				Sidecars: relPaths(opts.ProjectDir, moved[i].sidecarDsts),
			},
			Media: manifest.Media{
				DurationSec: p.orig.DurationSec,
				Width:       p.orig.Width,
				Height:      p.orig.Height,
				FPS:         p.orig.FPS(),
				VCodec:      p.orig.VCodec,
				HasAudio:    p.orig.HasAudio,
				CreatedAt:   createdAt(p.orig, moved[i].originalDst),
				SizeBytes:   p.g.original.size,
				XXH64:       sums[i],
			},
			ProxyInfo: proxyInfos[i],
			Review:    manifest.Review{Status: manifest.StatusPending},
			Applied:   manifest.Applied{Done: false},
		}
		man.Clips = append(man.Clips, clip)
		sum.NewClips++
		sum.TotalBytes += p.g.original.size
	}

	if err := manifest.Save(opts.ProjectDir, man); err != nil {
		return nil, fmt.Errorf("save manifest: %w", err)
	}

	// Empty the card after a verified copy (--clear-source): re-hash each copied
	// original against its recorded checksum and delete the source only on a
	// match. Other copied sources (sidecars, extras, proxy candidates, unmatched)
	// were size-verified during the copy. Never runs on move (sources already
	// gone), after an interruption, or on a probe-failed group (left in the dump).
	if opts.Copy && opts.ClearSource && !interrupted && ctx.Err() == nil {
		for i := range kept {
			if got, err := xxh64File(moved[i].originalDst); err != nil || got != sums[i] {
				sum.SourceKept = append(sum.SourceKept, kept[i].g.original.path)
			} else if os.Remove(kept[i].g.original.path) == nil {
				sum.SourceCleared++
			}
			for _, s := range kept[i].g.sidecars {
				if os.Remove(s.path) == nil {
					sum.SourceCleared++
				}
			}
			for _, e := range kept[i].g.extras {
				if os.Remove(e.path) == nil {
					sum.SourceCleared++
				}
			}
			if kept[i].g.proxyCand != nil {
				if os.Remove(kept[i].g.proxyCand.path) == nil {
					sum.SourceCleared++
				}
			}
		}
		for _, u := range unmatched {
			if os.Remove(u.path) == nil {
				sum.SourceCleared++
			}
		}
	}

	sum.Interrupted = interrupted || ctx.Err() != nil
	sum.Elapsed = opts.now().Sub(start)
	if sum.Interrupted {
		return sum, ErrInterrupted
	}
	return sum, nil
}

// movedGroup records where a group's files landed.
type movedGroup struct {
	stem        string
	originalDst string
	sidecarDsts []string
	proxyCand   *foundFile
}

// resolveProxy adopts the camera proxy if suitable, else generates one (plan §6
// step 6). It returns the manifest ProxyInfo and the manifest-relative proxy
// path ("" when there is no proxy).
func resolveProxy(ctx context.Context, opts Options, mg movedGroup, orig probe.Result, lg *runLog) (info manifest.ProxyInfo, rel, rejected string) {
	proxyDst := filepath.Join(opts.ProjectDir, "proxy", mg.stem+".mp4")
	proxyRel := relPath(opts.ProjectDir, proxyDst)

	if mg.proxyCand != nil {
		cand, err := probe.File(ctx, mg.proxyCand.path)
		if err == nil {
			if ok, reason := proxyAdoptable(cand, orig); ok {
				if err := moveFile(mg.proxyCand.path, proxyDst, opts.Copy); err == nil {
					return manifest.ProxyInfo{Source: manifest.ProxyCamera}, proxyRel, ""
				}
				// fall through to generation on move failure
			} else {
				rejected = fmt.Sprintf("%s (%s)", filepath.Base(mg.proxyCand.path), reason)
				lg.printf("reject camera proxy %s: %s", filepath.Base(mg.proxyCand.path), reason)
			}
		} else {
			rejected = fmt.Sprintf("%s (probe failed: %v)", filepath.Base(mg.proxyCand.path), err)
			lg.printf("probe camera proxy %s failed: %v", filepath.Base(mg.proxyCand.path), err)
		}
		// Rejected candidate is set aside, never deleted.
		rejDir := filepath.Join(opts.ProjectDir, "originals", "_rejected-proxies")
		_ = moveFile(mg.proxyCand.path, filepath.Join(rejDir, filepath.Base(mg.proxyCand.path)), opts.Copy)
	}

	stderr, err := proxy.Generate(ctx, mg.originalDst, proxyDst)
	lg.write(fmt.Sprintf("=== ffmpeg proxy %s ===\n", mg.stem), stderr)
	if err != nil {
		return manifest.ProxyInfo{Source: manifest.ProxyNone, Note: ffmpegNote(stderr)}, "", rejected
	}
	return manifest.ProxyInfo{Source: manifest.ProxyGenerated}, proxyRel, rejected
}

// parallel runs fn(0..n-1) with at most `jobs` concurrent goroutines. fn must
// only write to slot i of any shared slice (distinct indices are race-free).
func parallel(jobs, n int, fn func(i int)) {
	if jobs < 1 {
		jobs = 1
	}
	sem := make(chan struct{}, jobs)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			fn(i)
		}(i)
	}
	wg.Wait()
}

// runLog is a mutex-guarded appender for the per-run ingest log.
type runLog struct {
	mu   sync.Mutex
	path string
}

func (l *runLog) write(header string, body []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(header)
	f.Write(body)
	if len(body) > 0 && body[len(body)-1] != '\n' {
		f.WriteString("\n")
	}
}

func (l *runLog) printf(format string, args ...any) {
	l.write(fmt.Sprintf(format, args...)+"\n", nil)
}

// --- small helpers ---

var dateSlugRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}_(.+)$`)

func deriveTitle(projectDir string) string {
	base := filepath.Base(strings.TrimRight(projectDir, string(os.PathSeparator)))
	if m := dateSlugRe.FindStringSubmatch(base); m != nil {
		return m[1]
	}
	return base
}

func camOrDefault(cam string) string {
	if cam == "" {
		return "CAM"
	}
	return cam
}

func absOrSame(p string) string {
	if a, err := filepath.Abs(p); err == nil {
		return a
	}
	return p
}

func relPath(root, p string) string {
	if r, err := filepath.Rel(root, p); err == nil {
		return filepath.ToSlash(r)
	}
	return filepath.ToSlash(p)
}

func relPaths(root string, ps []string) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = relPath(root, p)
	}
	sort.Strings(out)
	return out
}

// createdAt prefers the probed creation_time, falling back to the file mtime
// (plan §3.2: date from probe, fallback file mtime).
func createdAt(r probe.Result, path string) time.Time {
	if !r.CreatedAt.IsZero() {
		return r.CreatedAt
	}
	if fi, err := os.Stat(path); err == nil {
		return fi.ModTime().UTC()
	}
	return time.Time{}
}

// ffmpegNote condenses ffmpeg stderr to a short note for proxyInfo.note.
func ffmpegNote(stderr []byte) string {
	lines := strings.Split(strings.TrimRight(string(stderr), "\n"), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if s := strings.TrimSpace(lines[i]); s != "" {
			return "ffmpeg: " + s
		}
	}
	return "ffmpeg failed with no stderr"
}
