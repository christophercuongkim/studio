package pipeline

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/christophercuongkim/studio/internal/chapters"
	"github.com/christophercuongkim/studio/internal/manifest"
	"github.com/christophercuongkim/studio/internal/videoyaml"
)

func write(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func saveManifest(t *testing.T, dir string, m *manifest.Manifest) {
	t.Helper()
	if err := manifest.Save(dir, m); err != nil {
		t.Fatal(err)
	}
}

// TestDetectProgression walks a project through every stage and checks that Next
// advances correctly at each point.
func TestDetectProgression(t *testing.T) {
	dir := t.TempDir()

	// 0. Only video.yaml (fresh `new`): nothing ingested → next is ingest.
	vy := videoyaml.Default("Lake Trip", "27", "unlisted")
	videoyaml.Save(dir, &vy)
	if got := Detect(dir).Next; got != "ingest" {
		t.Errorf("fresh: next = %q, want ingest", got)
	}

	// 1. Ingested with a pending clip → next review.
	take := 0
	_ = take
	man := &manifest.Manifest{
		SchemaVersion: manifest.SchemaVersion,
		Shoot:         manifest.Shoot{Root: dir, Title: "Lake Trip", CamCode: "DJI"},
		Clips: []manifest.Clip{
			{ID: "c-001", Seq: 1, Stem: "a", Files: manifest.Files{Original: "originals/a.mp4"},
				ProxyInfo: manifest.ProxyInfo{Source: manifest.ProxyGenerated}, Review: manifest.Review{Status: manifest.StatusPending}},
		},
	}
	saveManifest(t, dir, man)
	if got := Detect(dir).Next; got != "review" {
		t.Errorf("ingested: next = %q, want review", got)
	}

	// 2. Reviewed + applied → next scaffold.
	man.Clips[0].Review.Status = manifest.StatusKept
	man.Clips[0].Review.Desc = "keeper"
	man.Clips[0].Applied = manifest.Applied{Done: true, FinalStem: "20260827_DJI001_keeper"}
	saveManifest(t, dir, man)
	if got := Detect(dir).Next; got != "scaffold" {
		t.Errorf("applied: next = %q, want scaffold", got)
	}

	// 3. Scaffolded → next render.
	write(t, dir, "lake-trip.kdenlive", "<mlt/>")
	if got := Detect(dir).Next; got != "render" {
		t.Errorf("scaffolded: next = %q, want render", got)
	}

	// 4. Rendered → next chapters.
	write(t, dir, "render/final.mp4", "video")
	if got := Detect(dir).Next; got != "chapters" {
		t.Errorf("rendered: next = %q, want chapters", got)
	}

	// 5. Chapters written → next qc.
	vy.Description = chapters.StartSentinel + "\n0:00 Intro\n" + chapters.EndSentinel
	videoyaml.Save(dir, &vy)
	if got := Detect(dir).Next; got != "qc" {
		t.Errorf("chaptered: next = %q, want qc", got)
	}

	// 6. QC passed → next thumbs.
	write(t, dir, "qc-report.json", `{"passed":true}`)
	if got := Detect(dir).Next; got != "thumbs" {
		t.Errorf("qc: next = %q, want thumbs", got)
	}

	// 7. Thumbnail chosen → next upload.
	write(t, dir, "thumbs/thumbnail.png", "png")
	if got := Detect(dir).Next; got != "upload" {
		t.Errorf("thumbed: next = %q, want upload", got)
	}

	// 8. Uploaded → next archive.
	id := "abc123"
	vy.YouTube.VideoID = &id
	videoyaml.Save(dir, &vy)
	if got := Detect(dir).Next; got != "archive" {
		t.Errorf("uploaded: next = %q, want archive", got)
	}

	// 9. Archived → complete.
	now := time.Now()
	man.ArchivedAt = &now
	saveManifest(t, dir, man)
	s := Detect(dir)
	if s.Next != "" {
		t.Errorf("archived: next = %q, want complete", s.Next)
	}
	// Every step should read done.
	for _, step := range s.Steps {
		if !step.Done {
			t.Errorf("step %s not done at completion", step.Name)
		}
	}
}

func TestFindProjects(t *testing.T) {
	root := t.TempDir()
	// Two projects (one with manifest, one with only video.yaml) and a non-project.
	write(t, filepath.Join(root, "2026-08-01_a"), manifest.FileName, `{"schemaVersion":1,"shoot":{},"clips":[],"archivedAt":null,"archivePath":""}`)
	vy := videoyaml.Default("B", "27", "private")
	videoyaml.Save(mustMkdir(t, filepath.Join(root, "2026-08-02_b")), &vy)
	os.MkdirAll(filepath.Join(root, "not-a-project"), 0o755)

	got := FindProjects([]string{root})
	if len(got) != 2 {
		t.Fatalf("got %d projects, want 2: %v", len(got), got)
	}
}

func mustMkdir(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}
