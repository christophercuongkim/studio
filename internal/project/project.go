// Package project creates a new video project folder: the directory layout,
// script.md skeleton, and video.yaml (plan §3.1, §9.1). It is the `studio new`
// command's logic, kept out of the CLI so it can be tested directly.
package project

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/christophercuongkim/studio/internal/videoyaml"
)

// subdirs are the per-project folders created by `new` (plan §3.1).
var subdirs = []string{"undo", "originals", "proxy", "render", "thumbs"}

// slugRe validates a project slug: lowercase alphanumerics and hyphens.
var slugRe = regexp.MustCompile(`^[a-z0-9-]+$`)

// Options configures project creation.
type Options struct {
	ProjectsRoot string
	Slug         string
	Title        string // defaults to a de-slugged form
	Date         string // YYYY-MM-DD; empty = today
	CategoryID   string // from config uploadDefaults
	Privacy      string // from config uploadDefaults
}

// Create builds the project folder and returns its path. It refuses to touch an
// existing directory (plan §9.1).
func Create(opts Options, now time.Time) (string, error) {
	if !slugRe.MatchString(opts.Slug) {
		return "", fmt.Errorf("slug %q must be lowercase letters, digits, and hyphens", opts.Slug)
	}

	date := opts.Date
	if date == "" {
		date = now.Format("2006-01-02")
	} else if _, err := time.Parse("2006-01-02", date); err != nil {
		return "", fmt.Errorf("--date must be YYYY-MM-DD: %w", err)
	}

	title := opts.Title
	if title == "" {
		title = deslug(opts.Slug)
	}

	dir := filepath.Join(opts.ProjectsRoot, date+"_"+opts.Slug)
	if _, err := os.Stat(dir); err == nil {
		return "", fmt.Errorf("project already exists: %s", dir)
	} else if !os.IsNotExist(err) {
		return "", err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	for _, d := range subdirs {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
			return "", err
		}
	}

	if err := os.WriteFile(filepath.Join(dir, "script.md"), []byte(scriptSkeleton(title)), 0o644); err != nil {
		return "", err
	}

	catID := opts.CategoryID
	if catID == "" {
		catID = "27"
	}
	privacy := opts.Privacy
	if privacy == "" {
		privacy = "private"
	}
	vy := videoyaml.Default(title, catID, privacy)
	if err := videoyaml.Save(dir, &vy); err != nil {
		return "", err
	}

	return dir, nil
}

// scriptSkeleton is the starter script.md (plan §9.1, §10.1 format).
func scriptSkeleton(title string) string {
	return fmt.Sprintf(`---
title: %q
targetLength: 8m
---

## Hook
-

## Outro
-
`, title)
}

// deslug turns "lake-trip" into "Lake Trip" for a default title.
func deslug(slug string) string {
	words := strings.Split(slug, "-")
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
