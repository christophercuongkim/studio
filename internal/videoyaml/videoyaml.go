// Package videoyaml is the schema and load/save for video.yaml — the single
// source of truth for a video's publish metadata (plan §9.2). It is written by
// `new`, edited by `chapters --write`, read by `qc`, and updated by `upload`.
package videoyaml

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/christophercuongkim/studio/internal/fsutil"
)

// FileName is the fixed name within a project folder.
const FileName = "video.yaml"

// SchemaVersion is the current video.yaml schema.
const SchemaVersion = 1

// VideoYAML is the publish-metadata document.
type VideoYAML struct {
	SchemaVersion int      `yaml:"schemaVersion"`
	Title         string   `yaml:"title"`
	Description   string   `yaml:"description"`
	Tags          []string `yaml:"tags"`
	CategoryID    string   `yaml:"categoryId"`
	Privacy       string   `yaml:"privacy"` // private|unlisted|public
	PublishAt     *string  `yaml:"publishAt"`
	PlaylistIDs   []string `yaml:"playlistIds"`
	Thumbnail     string   `yaml:"thumbnail"`
	Render        string   `yaml:"render"`
	YouTube       YouTube  `yaml:"youtube"`
}

// YouTube holds fields written back by `studio upload`.
type YouTube struct {
	VideoID    *string `yaml:"videoId"`
	UploadedAt *string `yaml:"uploadedAt"`
}

// Default returns a fresh video.yaml seeded with the given title and upload
// defaults (plan §9.2).
func Default(title, categoryID, privacy string) VideoYAML {
	return VideoYAML{
		SchemaVersion: SchemaVersion,
		Title:         title,
		Tags:          []string{},
		CategoryID:    categoryID,
		Privacy:       privacy,
		PlaylistIDs:   []string{},
		Thumbnail:     "thumbs/thumbnail.png",
		Render:        "render/final.mp4",
	}
}

// Load reads video.yaml from a project folder.
func Load(projectDir string) (*VideoYAML, error) {
	path := filepath.Join(projectDir, FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", FileName, err)
	}
	var v VideoYAML
	if err := yaml.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("parse %s: %w", FileName, err)
	}
	if v.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("unsupported %s schemaVersion %d (want %d)", FileName, v.SchemaVersion, SchemaVersion)
	}
	return &v, nil
}

// Save writes video.yaml atomically into a project folder.
func Save(projectDir string, v *VideoYAML) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", FileName, err)
	}
	return fsutil.WriteFileAtomic(filepath.Join(projectDir, FileName), data, 0o644)
}
