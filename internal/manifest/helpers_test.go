package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile writes content to dir/name, creating parent dirs.
func writeFile(dir, name, content string) error {
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(content), 0o644)
}

// writeReferencedFiles touches every file a manifest points at, so
// ValidatePaths passes in round-trip tests.
func writeReferencedFiles(t *testing.T, dir string, m *Manifest) {
	t.Helper()
	for _, c := range m.Clips {
		paths := []string{c.Files.Original}
		if c.Files.Proxy != "" {
			paths = append(paths, c.Files.Proxy)
		}
		paths = append(paths, c.Files.Sidecars...)
		for _, rel := range paths {
			if rel == "" {
				continue
			}
			if err := writeFile(dir, rel, "x"); err != nil {
				t.Fatal(err)
			}
		}
	}
}
