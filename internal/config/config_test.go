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

func TestProjectsRootFor(t *testing.T) {
	// A connected external drive: its mount point (parent) exists.
	ssd := t.TempDir()
	external := filepath.Join(ssd, "videos") // subdir need not exist yet

	cfg := Config{ProjectsRoot: "/home/me/videos", ExternalRoot: external}

	// External chosen when the drive is mounted (parent dir exists).
	if root, src := cfg.ProjectsRootFor(""); root != external || src != RootExternal {
		t.Errorf("connected SSD: got %q/%s, want %q/external", root, src, external)
	}

	// Falls back to default when the drive is absent.
	gone := Config{ProjectsRoot: "/home/me/videos", ExternalRoot: "/run/media/me/GONE/videos"}
	if root, src := gone.ProjectsRootFor(""); root != "/home/me/videos" || src != RootDefault {
		t.Errorf("disconnected SSD: got %q/%s, want default", root, src)
	}

	// No external configured → default.
	plain := Config{ProjectsRoot: "/home/me/videos"}
	if _, src := plain.ProjectsRootFor(""); src != RootDefault {
		t.Errorf("no external: src = %s, want default", src)
	}

	// Explicit --root override wins over everything (and is tilde-expanded).
	if root, src := cfg.ProjectsRootFor("/mnt/other"); root != "/mnt/other" || src != RootFlag {
		t.Errorf("override: got %q/%s, want /mnt/other/flag", root, src)
	}
}
