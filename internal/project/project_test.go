package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/christophercuongkim/studio/internal/videoyaml"
)

var fixedNow = time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)

func TestCreateLayout(t *testing.T) {
	root := t.TempDir()
	dir, err := Create(Options{ProjectsRoot: root, Slug: "lake-trip", CategoryID: "27", Privacy: "private"}, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(dir) != "2026-08-27_lake-trip" {
		t.Errorf("dir = %s, want date_slug", filepath.Base(dir))
	}
	for _, sub := range subdirs {
		if fi, err := os.Stat(filepath.Join(dir, sub)); err != nil || !fi.IsDir() {
			t.Errorf("missing subdir %s", sub)
		}
	}

	// script.md skeleton has frontmatter + Hook/Outro with the default title.
	script, _ := os.ReadFile(filepath.Join(dir, "script.md"))
	s := string(script)
	for _, want := range []string{`title: "Lake Trip"`, "targetLength: 8m", "## Hook", "## Outro"} {
		if !strings.Contains(s, want) {
			t.Errorf("script.md missing %q:\n%s", want, s)
		}
	}

	// video.yaml round-trips and carries the seeded defaults.
	vy, err := videoyaml.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if vy.Title != "Lake Trip" || vy.CategoryID != "27" || vy.Privacy != "private" {
		t.Errorf("video.yaml defaults wrong: %+v", vy)
	}
	if vy.Render != "render/final.mp4" || vy.Thumbnail != "thumbs/thumbnail.png" {
		t.Errorf("video.yaml paths wrong: %+v", vy)
	}
	if vy.Tags == nil || vy.PlaylistIDs == nil {
		t.Error("tags/playlistIds should marshal as empty lists, not null")
	}
}

func TestCreateRefusesExisting(t *testing.T) {
	root := t.TempDir()
	_, err := Create(Options{ProjectsRoot: root, Slug: "dup"}, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Create(Options{ProjectsRoot: root, Slug: "dup"}, fixedNow)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected refuse-existing, got %v", err)
	}
}

func TestCreateValidates(t *testing.T) {
	root := t.TempDir()
	if _, err := Create(Options{ProjectsRoot: root, Slug: "Bad Slug"}, fixedNow); err == nil {
		t.Error("expected slug validation error")
	}
	if _, err := Create(Options{ProjectsRoot: root, Slug: "ok", Date: "2026/08/27"}, fixedNow); err == nil {
		t.Error("expected date format error")
	}
}

func TestCreateHonorsExplicitTitleAndDate(t *testing.T) {
	root := t.TempDir()
	dir, err := Create(Options{ProjectsRoot: root, Slug: "trip", Title: "My Trip", Date: "2025-01-02"}, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(dir) != "2025-01-02_trip" {
		t.Errorf("dir = %s, want explicit date", filepath.Base(dir))
	}
	vy, _ := videoyaml.Load(dir)
	if vy.Title != "My Trip" {
		t.Errorf("title = %q, want My Trip", vy.Title)
	}
}
