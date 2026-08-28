package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFromSeedsDefaultsOnFirstRun(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "studio", "config.yaml")

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	// The file must now exist and carry comments (i.e. it's the commented
	// template, not a bare Marshal).
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("seeded file missing: %v", err)
	}
	if !strings.Contains(string(data), "# studio configuration") {
		t.Error("seeded config lacks header comment")
	}

	if cfg.CamCode != "DJI" || cfg.Player != "mpv" {
		t.Errorf("defaults not applied: %+v", cfg)
	}
	if cfg.UploadDefaults.Privacy != "private" || cfg.UploadDefaults.CategoryID != "27" {
		t.Errorf("upload defaults wrong: %+v", cfg.UploadDefaults)
	}
	if cfg.ArchiveRoot != "" {
		t.Errorf("archiveRoot should default empty, got %q", cfg.ArchiveRoot)
	}
}

func TestLoadFromExpandsTilde(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg, err := LoadFrom(path) // seeds defaults, which use ~/videos
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	wantProjects := filepath.Join(home, "videos")
	if cfg.ProjectsRoot != wantProjects {
		t.Errorf("ProjectsRoot = %q, want %q", cfg.ProjectsRoot, wantProjects)
	}
	if len(cfg.SearchRoots) != 1 || cfg.SearchRoots[0] != wantProjects {
		t.Errorf("SearchRoots = %v, want [%q]", cfg.SearchRoots, wantProjects)
	}
	if strings.HasPrefix(cfg.KdenliveTemplate, "~") {
		t.Errorf("KdenliveTemplate not expanded: %q", cfg.KdenliveTemplate)
	}
}

func TestLoadFromPartialOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	// Only override camCode; everything else must fall back to defaults.
	if err := os.WriteFile(path, []byte("camCode: A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.CamCode != "A" {
		t.Errorf("CamCode = %q, want A", cfg.CamCode)
	}
	if cfg.Player != "mpv" {
		t.Errorf("Player = %q, want default mpv", cfg.Player)
	}
}

func TestLoadFromRejectsBadYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("camCode: [unterminated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFrom(path); err == nil {
		t.Error("expected YAML parse error, got nil")
	}
}
