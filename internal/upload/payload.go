package upload

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/christophercuongkim/studio/internal/videoyaml"
	"google.golang.org/api/youtube/v3"
)

// buildVideo maps video.yaml to a YouTube API video resource. privacyOverride,
// when non-empty, replaces video.yaml's privacy.
func buildVideo(vy *videoyaml.VideoYAML, privacyOverride string) *youtube.Video {
	privacy := vy.Privacy
	if privacyOverride != "" {
		privacy = privacyOverride
	}
	v := &youtube.Video{
		Snippet: &youtube.VideoSnippet{
			Title:       vy.Title,
			Description: vy.Description,
			Tags:        vy.Tags,
			CategoryId:  vy.CategoryID,
		},
		Status: &youtube.VideoStatus{
			PrivacyStatus: privacy,
		},
	}
	if vy.PublishAt != nil && *vy.PublishAt != "" {
		v.Status.PublishAt = *vy.PublishAt
	}
	return v
}

// DryRun renders a human-readable summary of what would be uploaded, without
// touching the network.
func DryRun(vy *videoyaml.VideoYAML, privacyOverride string) string {
	v := buildVideo(vy, privacyOverride)
	var b strings.Builder
	fmt.Fprintln(&b, "would upload (dry run):")
	fmt.Fprintf(&b, "  title:       %s\n", v.Snippet.Title)
	fmt.Fprintf(&b, "  privacy:     %s\n", v.Status.PrivacyStatus)
	fmt.Fprintf(&b, "  categoryId:  %s\n", v.Snippet.CategoryId)
	fmt.Fprintf(&b, "  tags:        %s\n", strings.Join(v.Snippet.Tags, ", "))
	if v.Status.PublishAt != "" {
		fmt.Fprintf(&b, "  publishAt:   %s\n", v.Status.PublishAt)
	}
	if len(vy.PlaylistIDs) > 0 {
		fmt.Fprintf(&b, "  playlists:   %s\n", strings.Join(vy.PlaylistIDs, ", "))
	}
	fmt.Fprintf(&b, "  render:      %s\n", vy.Render)
	fmt.Fprintf(&b, "  thumbnail:   %s\n", vy.Thumbnail)
	desc := v.Snippet.Description
	if len(desc) > 200 {
		desc = desc[:200] + "…"
	}
	fmt.Fprintf(&b, "  description: %s\n", strings.ReplaceAll(desc, "\n", "\n               "))
	return strings.TrimRight(b.String(), "\n")
}

// qcReport is the subset of qc-report.json the gate needs.
type qcReport struct {
	Passed bool `json:"passed"`
}

// CheckQCGate requires a passing qc-report.json unless skip is set (plan §15).
func CheckQCGate(projectDir string, skip bool) error {
	if skip {
		return nil
	}
	path := filepath.Join(projectDir, "qc-report.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("no qc-report.json — run 'studio qc' first (or pass --skip-qc): %w", err)
	}
	var r qcReport
	if err := json.Unmarshal(data, &r); err != nil {
		return fmt.Errorf("unreadable qc-report.json: %w", err)
	}
	if !r.Passed {
		return fmt.Errorf("qc-report.json shows a failing QC — fix and re-run 'studio qc' (or pass --skip-qc)")
	}
	return nil
}
