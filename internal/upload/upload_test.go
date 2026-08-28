package upload

import (
	"context"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/christophercuongkim/studio/internal/videoyaml"
)

func baseVY() *videoyaml.VideoYAML {
	vy := videoyaml.Default("My Video", "27", "private")
	vy.Render = "render/final.mp4"
	vy.Thumbnail = "" // opt out of thumbnail checks unless a test sets one
	return &vy
}

// project writes a project dir with the given video.yaml and a render file.
func project(t *testing.T, vy *videoyaml.VideoYAML) string {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "render"), 0o755)
	os.WriteFile(filepath.Join(dir, "render", "final.mp4"), []byte("video"), 0o644)
	if err := videoyaml.Save(dir, vy); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestValidate(t *testing.T) {
	dir := project(t, baseVY())

	if err := Validate(baseVY(), dir); err != nil {
		t.Errorf("valid should pass: %v", err)
	}

	longTitle := baseVY()
	longTitle.Title = strings.Repeat("x", 101)
	if err := Validate(longTitle, dir); err == nil {
		t.Error("long title should fail")
	}

	bigTags := baseVY()
	bigTags.Tags = []string{strings.Repeat("a", 400), strings.Repeat("b", 200)}
	if err := Validate(bigTags, dir); err == nil {
		t.Error("oversized tags should fail")
	}

	badPrivacy := baseVY()
	badPrivacy.Privacy = "secret"
	if err := Validate(badPrivacy, dir); err == nil {
		t.Error("bad privacy should fail")
	}

	pubUnlisted := baseVY()
	when := "2026-09-01T00:00:00Z"
	pubUnlisted.Privacy = "unlisted"
	pubUnlisted.PublishAt = &when
	if err := Validate(pubUnlisted, dir); err == nil {
		t.Error("publishAt with non-private should fail")
	}

	missingRender := baseVY()
	missingRender.Render = "render/missing.mp4"
	if err := Validate(missingRender, dir); err == nil {
		t.Error("missing render should fail")
	}
}

func TestValidateThumbnail(t *testing.T) {
	vy := baseVY()
	vy.Thumbnail = "thumbs/thumbnail.png"
	dir := project(t, vy)

	// Wrong dimensions → fail.
	writePNGSize(t, filepath.Join(dir, "thumbs", "thumbnail.png"), 640, 360)
	if err := Validate(vy, dir); err == nil {
		t.Error("wrong-size thumbnail should fail")
	}
	// Correct dimensions → pass.
	writePNGSize(t, filepath.Join(dir, "thumbs", "thumbnail.png"), 1280, 720)
	if err := Validate(vy, dir); err != nil {
		t.Errorf("1280x720 thumbnail should pass: %v", err)
	}
}

func TestBuildVideoAndDryRun(t *testing.T) {
	vy := baseVY()
	vy.Tags = []string{"go", "video"}
	v := buildVideo(vy, "unlisted") // privacy override
	if v.Status.PrivacyStatus != "unlisted" {
		t.Errorf("privacy override not applied: %s", v.Status.PrivacyStatus)
	}
	if v.Snippet.Title != "My Video" || v.Snippet.CategoryId != "27" {
		t.Errorf("snippet wrong: %+v", v.Snippet)
	}
	out := DryRun(vy, "")
	for _, want := range []string{"My Video", "private", "go, video", "render/final.mp4"} {
		if !strings.Contains(out, want) {
			t.Errorf("dry-run missing %q:\n%s", want, out)
		}
	}
}

func TestCheckQCGate(t *testing.T) {
	dir := t.TempDir()
	if err := CheckQCGate(dir, false); err == nil {
		t.Error("missing qc-report should fail the gate")
	}
	if err := CheckQCGate(dir, true); err != nil {
		t.Error("--skip-qc should bypass the gate")
	}
	os.WriteFile(filepath.Join(dir, "qc-report.json"), []byte(`{"passed":false}`), 0o644)
	if err := CheckQCGate(dir, false); err == nil {
		t.Error("failing qc-report should fail the gate")
	}
	os.WriteFile(filepath.Join(dir, "qc-report.json"), []byte(`{"passed":true}`), 0o644)
	if err := CheckQCGate(dir, false); err != nil {
		t.Errorf("passing qc-report should pass: %v", err)
	}
}

// These reach the idempotency/gate branches before any network call.
func TestRunIdempotencyAndGate(t *testing.T) {
	ctx := context.Background()

	// Already uploaded → refuse without --update (no network reached).
	uploaded := baseVY()
	id := "abc123"
	uploaded.YouTube.VideoID = &id
	dir := project(t, uploaded)
	if _, err := Run(ctx, Options{ProjectDir: dir}); err == nil || !strings.Contains(err.Error(), "already uploaded") {
		t.Errorf("expected already-uploaded refusal, got %v", err)
	}

	// --update with no videoId → error before network.
	fresh := project(t, baseVY())
	if _, err := Run(ctx, Options{ProjectDir: fresh, Update: true}); err == nil || !strings.Contains(err.Error(), "already-uploaded") {
		t.Errorf("expected update-needs-video error, got %v", err)
	}

	// No qc-report → gate fails before network.
	if _, err := Run(ctx, Options{ProjectDir: fresh}); err == nil || !strings.Contains(err.Error(), "qc-report") {
		t.Errorf("expected qc gate failure, got %v", err)
	}

	// Dry-run prints payload and never touches network even without qc-report.
	if res, err := Run(ctx, Options{ProjectDir: fresh, DryRun: true}); err != nil || !res.DryRun {
		t.Errorf("dry-run should succeed offline: %v", err)
	}
}

func writePNGSize(t *testing.T, path string, w, h int) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0o755)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
}
