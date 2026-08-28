// Package config loads studio's global configuration from
// ~/.config/studio/config.yaml (plan §2.1). On first run it writes a
// commented-defaults file so the user has something to edit.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the global studio configuration.
type Config struct {
	ProjectsRoot     string         `yaml:"projectsRoot"`     // where `studio new` creates projects
	SearchRoots      []string       `yaml:"searchRoots"`      // dirs scanned for manifest.json
	KdenliveTemplate string         `yaml:"kdenliveTemplate"` // default scaffold template
	CamCode          string         `yaml:"camCode"`          // default camera code
	Player           string         `yaml:"player"`           // `search --play` player binary
	ArchiveRoot      string         `yaml:"archiveRoot"`      // empty = archive disabled
	UploadDefaults   UploadDefaults `yaml:"uploadDefaults"`
}

// UploadDefaults seed video.yaml on `studio new`.
type UploadDefaults struct {
	Privacy    string `yaml:"privacy"`
	CategoryID string `yaml:"categoryId"`
}

// defaultYAML is written verbatim on first run. It is hand-authored (rather than
// yaml.Marshal of Default()) so the file ships with explanatory comments —
// Marshal would drop them. Keep it in sync with Default() and the struct tags.
const defaultYAML = `# studio configuration — https://github.com/christophercuongkim/studio
# Paths may start with ~ (expanded to your home directory).

projectsRoot: ~/videos            # where 'studio new' creates projects
searchRoots: [~/videos]           # dirs scanned for manifest.json (recursive)
kdenliveTemplate: ~/videos/templates/empty-25.12.kdenlive
camCode: DJI                      # default camera code; per-run --cam overrides
player: mpv                       # used by 'studio search --play'
archiveRoot: ""                   # empty = 'studio archive' disabled
uploadDefaults:
  privacy: private                # private | unlisted | public
  categoryId: "27"                # 27 = Education
`

// Default returns the built-in defaults, matching defaultYAML. Paths here are
// still tilde-form; Load expands them.
func Default() Config {
	return Config{
		ProjectsRoot:     "~/videos",
		SearchRoots:      []string{"~/videos"},
		KdenliveTemplate: "~/videos/templates/empty-25.12.kdenlive",
		CamCode:          "DJI",
		Player:           "mpv",
		ArchiveRoot:      "",
		UploadDefaults:   UploadDefaults{Privacy: "private", CategoryID: "27"},
	}
}

// Path is the config file location, honoring XDG_CONFIG_HOME.
func Path() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "studio", "config.yaml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home directory: %w", err)
	}
	return filepath.Join(home, ".config", "studio", "config.yaml"), nil
}

// Load reads the config from its default path, creating a commented defaults
// file on first run. Missing individual fields fall back to Default(). All path
// fields are tilde-expanded before returning.
func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return Config{}, err
	}
	return LoadFrom(path)
}

// LoadFrom reads (and, if absent, seeds) the config at an explicit path.
func LoadFrom(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if err := seed(path); err != nil {
			return Config{}, err
		}
		data = []byte(defaultYAML)
	} else if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}

	// Start from defaults so a partial file only overrides what it sets.
	cfg := Default()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}

	if err := cfg.expandPaths(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// seed writes the commented defaults file, creating parent dirs.
func seed(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(defaultYAML), 0o644); err != nil {
		return fmt.Errorf("write default config: %w", err)
	}
	return nil
}

// expandPaths replaces a leading ~ with the home directory across every path
// field. Player is a binary name and intentionally left untouched.
func (c *Config) expandPaths() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("locate home directory: %w", err)
	}
	c.ProjectsRoot = expandTilde(c.ProjectsRoot, home)
	c.KdenliveTemplate = expandTilde(c.KdenliveTemplate, home)
	c.ArchiveRoot = expandTilde(c.ArchiveRoot, home)
	for i, r := range c.SearchRoots {
		c.SearchRoots[i] = expandTilde(r, home)
	}
	return nil
}

// expandTilde expands a leading ~ or ~/ to home. Empty stays empty (used as a
// sentinel for a disabled archiveRoot).
func expandTilde(p, home string) string {
	if p == "" {
		return ""
	}
	if p == "~" {
		return home
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:])
	}
	return p
}
