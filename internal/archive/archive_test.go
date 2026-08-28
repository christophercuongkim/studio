package archive

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/christophercuongkim/studio/internal/hash"
	"github.com/christophercuongkim/studio/internal/manifest"
	"github.com/christophercuongkim/studio/internal/videoyaml"
)

// buildProject writes a project whose manifest checksums match the originals on
// disk, with tool-created proxy/ and thumbs/raw/ dirs and a published video.yaml.
func buildProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, "originals/a.mp4", "alpha contents")
	writeFile(t, dir, "originals/b.mp4", "bravo contents")
	writeFile(t, dir, "proxy/a.mp4", "proxy alpha")
	writeFile(t, dir, "thumbs/raw/0001.png", "scratch")

	ha, _ := hash.XXH64File(filepath.Join(dir, "originals/a.mp4"))
	hb, _ := hash.XXH64File(filepath.Join(dir, "originals/b.mp4"))
	man := &manifest.Manifest{
		SchemaVersion: manifest.SchemaVersion,
		Shoot:         manifest.Shoot{Root: dir, Title: "trip", CamCode: "DJI"},
		Clips: []manifest.Clip{
			appliedClip("c-001", "originals/a.mp4", "proxy/a.mp4", ha),
			appliedClip("c-002", "originals/b.mp4", "", hb),
		},
	}
	if err := manifest.Save(dir, man); err != nil {
		t.Fatal(err)
	}
	vy := videoyaml.Default("Trip", "27", "unlisted")
	id := "vid123"
	vy.YouTube.VideoID = &id
	if err := videoyaml.Save(dir, &vy); err != nil {
		t.Fatal(err)
	}
	return dir
}

func appliedClip(id, orig, proxy, xxh string) manifest.Clip {
	c := manifest.Clip{
		ID: id, Seq: 1, Stem: id,
		Files:     manifest.Files{Original: orig, Proxy: proxy},
		Media:     manifest.Media{XXH64: xxh, VCodec: "hevc"},
		ProxyInfo: manifest.ProxyInfo{Source: manifest.ProxyGenerated},
		Review:    manifest.Review{Status: manifest.StatusKept, Desc: "x"},
		Applied:   manifest.Applied{Done: true, FinalStem: id},
	}
	if proxy == "" {
		c.ProxyInfo = manifest.ProxyInfo{Source: manifest.ProxyNone, Note: "n/a"}
	}
	return c
}

func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func requireRsync(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("rsync"); err != nil {
		t.Skip("rsync not on PATH")
	}
}

func TestArchiveHappyPath(t *testing.T) {
	requireRsync(t)
	dir := buildProject(t)
	archiveRoot := t.TempDir()

	res, err := Run(context.Background(), Options{ProjectDir: dir, ArchiveRoot: archiveRoot})
	if err != nil {
		t.Fatal(err)
	}
	if res.Hashed != 2 {
		t.Errorf("hashed %d, want 2", res.Hashed)
	}
	// Archive copy has the originals.
	if _, err := os.Stat(filepath.Join(res.ArchivePath, "originals/a.mp4")); err != nil {
		t.Errorf("archive copy missing original: %v", err)
	}
	// proxy/ and thumbs/raw/ pruned from the source.
	if _, err := os.Stat(filepath.Join(dir, "proxy")); !os.IsNotExist(err) {
		t.Error("proxy/ should be pruned")
	}
	if _, err := os.Stat(filepath.Join(dir, "thumbs", "raw")); !os.IsNotExist(err) {
		t.Error("thumbs/raw should be pruned")
	}
	// Manifest stamped.
	man, _ := manifest.Load(dir)
	if man.ArchivedAt == nil || man.ArchivePath != res.ArchivePath {
		t.Errorf("manifest not stamped: %+v", man.ArchivedAt)
	}
}

func TestArchiveBitRotAborts(t *testing.T) {
	requireRsync(t)
	dir := buildProject(t)
	// Corrupt an original after the manifest recorded its hash.
	writeFile(t, dir, "originals/a.mp4", "alpha contents TAMPERED")

	_, err := Run(context.Background(), Options{ProjectDir: dir, ArchiveRoot: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum-mismatch abort, got %v", err)
	}
	// Nothing pruned or stamped.
	if _, err := os.Stat(filepath.Join(dir, "proxy")); err != nil {
		t.Error("proxy/ should remain after aborted archive")
	}
}

func TestArchiveUnappliedAborts(t *testing.T) {
	dir := buildProject(t)
	man, _ := manifest.Load(dir)
	man.Clips[0].Applied.Done = false
	manifest.Save(dir, man)

	_, err := Run(context.Background(), Options{ProjectDir: dir, ArchiveRoot: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "not all kept clips") {
		t.Fatalf("expected unapplied-clips abort, got %v", err)
	}
}

func TestArchiveUnpublishedNeedsForce(t *testing.T) {
	requireRsync(t)
	dir := buildProject(t)
	vy, _ := videoyaml.Load(dir)
	vy.YouTube.VideoID = nil
	videoyaml.Save(dir, vy)

	if _, err := Run(context.Background(), Options{ProjectDir: dir, ArchiveRoot: t.TempDir()}); err == nil {
		t.Fatal("unpublished archive should require --force")
	}
	if _, err := Run(context.Background(), Options{ProjectDir: dir, ArchiveRoot: t.TempDir(), Force: true}); err != nil {
		t.Fatalf("--force should allow archiving unpublished: %v", err)
	}
}

func TestArchiveKeepProxies(t *testing.T) {
	requireRsync(t)
	dir := buildProject(t)
	if _, err := Run(context.Background(), Options{ProjectDir: dir, ArchiveRoot: t.TempDir(), KeepProxies: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "proxy")); err != nil {
		t.Error("proxy/ should be kept with --keep-proxies")
	}
	// thumbs/raw is still pruned (never user media).
	if _, err := os.Stat(filepath.Join(dir, "thumbs", "raw")); !os.IsNotExist(err) {
		t.Error("thumbs/raw should still be pruned")
	}
}

func TestArchiveDryRunChangesNothing(t *testing.T) {
	dir := buildProject(t)
	res, err := Run(context.Background(), Options{ProjectDir: dir, ArchiveRoot: t.TempDir(), DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if !res.DryRun || res.Hashed != 2 {
		t.Errorf("dry run result wrong: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(dir, "proxy")); err != nil {
		t.Error("dry run should not prune proxy/")
	}
	man, _ := manifest.Load(dir)
	if man.ArchivedAt != nil {
		t.Error("dry run should not stamp the manifest")
	}
}

func TestArchiveDisabledWithoutRoot(t *testing.T) {
	dir := buildProject(t)
	if _, err := Run(context.Background(), Options{ProjectDir: dir, ArchiveRoot: ""}); err == nil {
		t.Fatal("empty archiveRoot should disable archiving")
	}
}
