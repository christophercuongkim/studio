package manifest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/christophercuongkim/studio/internal/fsutil"
)

// Load reads and validates the manifest in a project folder. It checks both the
// schema (structure, enums) and that every referenced file actually exists on
// disk (plan §4: "validates all relative paths exist").
func Load(projectDir string) (*Manifest, error) {
	path := filepath.Join(projectDir, FileName)
	m, err := LoadFile(path)
	if err != nil {
		return nil, err
	}
	if err := m.ValidatePaths(projectDir); err != nil {
		return nil, err
	}
	return m, nil
}

// LoadFile reads and schema-validates a manifest from an explicit path without
// checking file existence. Useful for read-only tooling (e.g. search) that may
// tolerate a moved tree, and for tests.
func LoadFile(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest %s: %w", path, err)
	}
	var m Manifest
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("parse manifest %s: %w", path, err)
	}
	if err := m.ValidateSchema(); err != nil {
		return nil, fmt.Errorf("invalid manifest %s: %w", path, err)
	}
	return &m, nil
}

// Save writes the manifest atomically into a project folder.
func Save(projectDir string, m *Manifest) error {
	if err := m.ValidateSchema(); err != nil {
		return fmt.Errorf("refusing to save invalid manifest: %w", err)
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	data = append(data, '\n')
	path := filepath.Join(projectDir, FileName)
	return fsutil.WriteFileAtomic(path, data, 0o644)
}

// ValidateSchema checks the manifest's structure and field invariants without
// touching the filesystem.
func (m *Manifest) ValidateSchema() error {
	if m.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported schemaVersion %d (this build understands %d)", m.SchemaVersion, SchemaVersion)
	}
	seen := make(map[string]bool, len(m.Clips))
	for i := range m.Clips {
		c := &m.Clips[i]
		if c.ID == "" {
			return fmt.Errorf("clip %d: empty id", i)
		}
		if seen[c.ID] {
			return fmt.Errorf("duplicate clip id %q", c.ID)
		}
		seen[c.ID] = true

		switch c.Review.Status {
		case StatusPending, StatusKept, StatusRejected:
		default:
			return fmt.Errorf("clip %s: invalid review.status %q", c.ID, c.Review.Status)
		}

		switch c.ProxyInfo.Source {
		case ProxyCamera, ProxyGenerated:
		case ProxyNone:
			if c.ProxyInfo.Note == "" {
				return fmt.Errorf("clip %s: proxyInfo.source=none requires a non-empty note", c.ID)
			}
		default:
			return fmt.Errorf("clip %s: invalid proxyInfo.source %q", c.ID, c.ProxyInfo.Source)
		}

		if c.Review.Rating < 0 || c.Review.Rating > 5 {
			return fmt.Errorf("clip %s: rating %d out of range 0–5", c.ID, c.Review.Rating)
		}
	}
	return nil
}

// ValidatePaths checks that every file the manifest references exists under
// root. It assumes ValidateSchema has already passed.
func (m *Manifest) ValidatePaths(root string) error {
	for i := range m.Clips {
		c := &m.Clips[i]
		must := []string{c.Files.Original}
		if c.Files.Proxy != "" {
			must = append(must, c.Files.Proxy)
		}
		must = append(must, c.Files.Sidecars...)
		for _, rel := range must {
			if rel == "" {
				continue
			}
			p := filepath.Join(root, rel)
			if _, err := os.Stat(p); err != nil {
				return fmt.Errorf("clip %s references missing file %s: %w", c.ID, rel, err)
			}
		}
	}
	return nil
}
