// Package drives enumerates connected external/removable storage (SD cards,
// external SSDs) and offers a lightweight numbered picker. On Linux, udisks
// auto-mounts each volume at /run/media/<user>/<LABEL> (or /media/<user>/…), so
// the mount directory's name is the volume label and its path is the location —
// no external tools, no TUI framework.
package drives

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Drive is one mounted external volume.
type Drive struct {
	Label string // the volume name (the mount directory's name)
	Path  string // the mount point
}

// External returns the currently-mounted external volumes, sorted by label.
func External() []Drive {
	return scan(baseDirs())
}

// baseDirs are the auto-mount roots to scan (per-user udisks locations).
func baseDirs() []string {
	var b []string
	if u, err := user.Current(); err == nil && u.Username != "" {
		b = append(b, filepath.Join("/run/media", u.Username), filepath.Join("/media", u.Username))
	}
	return b
}

// scan lists the immediate subdirectories of each base dir as drives.
func scan(bases []string) []Drive {
	seen := map[string]bool{}
	var out []Drive
	for _, base := range bases {
		entries, err := os.ReadDir(base)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			p := filepath.Join(base, e.Name())
			if seen[p] {
				continue
			}
			seen[p] = true
			out = append(out, Drive{Label: e.Name(), Path: p})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Label < out[j].Label })
	return out
}

// Prompt prints the drives as a numbered list and reads a 1-based choice from r,
// writing prompts to w. Empty input selects the first drive. It errors if there
// are no drives or the choice is out of range.
func Prompt(w io.Writer, r io.Reader, ds []Drive) (Drive, error) {
	if len(ds) == 0 {
		return Drive{}, fmt.Errorf("no external drives mounted (plug in an SD card or SSD)")
	}
	fmt.Fprintln(w, "External drives:")
	for i, d := range ds {
		fmt.Fprintf(w, "  %d) %s  (%s)\n", i+1, d.Label, d.Path)
	}
	fmt.Fprint(w, "choose [1]: ")

	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil && line == "" {
		return Drive{}, fmt.Errorf("no selection read: %w", err)
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return ds[0], nil
	}
	n, err := strconv.Atoi(line)
	if err != nil || n < 1 || n > len(ds) {
		return Drive{}, fmt.Errorf("invalid choice %q (want 1–%d)", line, len(ds))
	}
	return ds[n-1], nil
}

// IsRemovable reports whether path lives on a per-user auto-mount root — a good
// proxy for "this is an SD card / external drive", used to default ingest to
// copy so the card is safe to remove.
func IsRemovable(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	for _, base := range baseDirs() {
		if abs == base || strings.HasPrefix(abs, base+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
