// Package upload publishes a finished render to YouTube (plan §15): validate
// video.yaml against YouTube's hard limits, gate on a passing qc-report.json,
// then resumable-insert the video, set the thumbnail, and add it to playlists —
// writing the resulting videoId back to video.yaml.
package upload

import (
	"fmt"
	"image"
	_ "image/jpeg" // register decoders for thumbnail dimension checks
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/christophercuongkim/studio/internal/videoyaml"
)

// YouTube hard limits (plan §15).
const (
	maxTitleLen    = 100
	maxTagsTotal   = 500
	maxThumbBytes  = 2 << 20 // 2 MiB
	thumbW, thumbH = 1280, 720
)

// Validate checks video.yaml against YouTube's limits and that the render (and,
// if present, the thumbnail) is acceptable. It returns all problems at once.
func Validate(vy *videoyaml.VideoYAML, projectDir string) error {
	var problems []string

	if vy.Title == "" {
		problems = append(problems, "title is empty")
	}
	if len(vy.Title) > maxTitleLen {
		problems = append(problems, fmt.Sprintf("title is %d chars (max %d)", len(vy.Title), maxTitleLen))
	}
	if total := tagsTotal(vy.Tags); total > maxTagsTotal {
		problems = append(problems, fmt.Sprintf("tags total %d chars (max %d)", total, maxTagsTotal))
	}
	switch vy.Privacy {
	case "private", "unlisted", "public":
	default:
		problems = append(problems, fmt.Sprintf("privacy %q must be private|unlisted|public", vy.Privacy))
	}
	if vy.PublishAt != nil && *vy.PublishAt != "" && vy.Privacy != "private" {
		problems = append(problems, "publishAt requires privacy: private")
	}

	// Render must exist.
	renderPath := filepath.Join(projectDir, vy.Render)
	if _, err := os.Stat(renderPath); err != nil {
		problems = append(problems, fmt.Sprintf("render not found: %s", vy.Render))
	}

	// Thumbnail is optional, but if present must meet YouTube's limits.
	if vy.Thumbnail != "" {
		thumbPath := filepath.Join(projectDir, vy.Thumbnail)
		if fi, err := os.Stat(thumbPath); err == nil {
			if fi.Size() > maxThumbBytes {
				problems = append(problems, fmt.Sprintf("thumbnail is %d bytes (max %d)", fi.Size(), maxThumbBytes))
			}
			if w, h, ok := imageDims(thumbPath); ok && (w != thumbW || h != thumbH) {
				problems = append(problems, fmt.Sprintf("thumbnail is %dx%d (want %dx%d)", w, h, thumbW, thumbH))
			}
		}
	}

	if len(problems) > 0 {
		return fmt.Errorf("video.yaml is not ready to upload:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return nil
}

// tagsTotal counts characters across all tags (YouTube's ~500-char budget).
func tagsTotal(tags []string) int {
	n := 0
	for _, t := range tags {
		n += len(t)
	}
	return n
}

func imageDims(path string) (w, h int, ok bool) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, false
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, false
	}
	return cfg.Width, cfg.Height, true
}
