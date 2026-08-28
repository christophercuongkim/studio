package apply

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/christophercuongkim/studio/internal/manifest"
)

// UndoResult reports what an undo did.
type UndoResult struct {
	LogPath  string
	Reversed int
	Skipped  []string // targets that no longer existed
}

// LatestLog returns the newest apply journal in a project's undo/ dir. Names
// are RFC3339-stamped so lexical order is chronological.
func LatestLog(projectDir string) (string, error) {
	undoDir := filepath.Join(projectDir, "undo")
	entries, err := os.ReadDir(undoDir)
	if err != nil {
		return "", fmt.Errorf("no undo logs: %w", err)
	}
	var logs []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "apply-") && strings.HasSuffix(e.Name(), ".jsonl") {
			logs = append(logs, e.Name())
		}
	}
	if len(logs) == 0 {
		return "", fmt.Errorf("no apply-*.jsonl logs in %s", undoDir)
	}
	sort.Strings(logs)
	return filepath.Join(undoDir, logs[len(logs)-1]), nil
}

// Undo reverses the renames in logPath (newest first when logPath is a project
// dir's latest), renaming each `to` back to its `from`. Lines whose target no
// longer exists are skipped with a warning. The manifest's file paths and
// applied state are reverted for every affected clip.
func Undo(projectDir, logPath string) (*UndoResult, error) {
	renames, err := readLog(logPath)
	if err != nil {
		return nil, err
	}
	// Load schema-only, not with path validation: after a partial apply failure
	// the manifest's paths are intentionally inconsistent with disk (that's the
	// state undo exists to repair), so the validating loader would reject it.
	man, err := manifest.LoadFile(filepath.Join(projectDir, manifest.FileName))
	if err != nil {
		return nil, err
	}

	res := &UndoResult{LogPath: logPath}
	// Per group, map new path → original path so we can revert the manifest.
	revert := map[string]map[string]string{}

	// Reverse order so a group's files unwind in the opposite sequence.
	for i := len(renames) - 1; i >= 0; i-- {
		r := renames[i]
		to := filepath.Join(projectDir, r.To)
		from := filepath.Join(projectDir, r.From)
		if _, err := os.Stat(to); err != nil {
			res.Skipped = append(res.Skipped, r.To)
			continue
		}
		if err := os.Rename(to, from); err != nil {
			return res, fmt.Errorf("undo rename %s → %s failed: %w", r.To, r.From, err)
		}
		res.Reversed++
		if revert[r.Group] == nil {
			revert[r.Group] = map[string]string{}
		}
		revert[r.Group][r.To] = r.From
	}

	// Revert manifest file paths and applied state for affected clips.
	for group, m := range revert {
		c := man.ClipByID(group)
		if c == nil {
			continue
		}
		c.Files.Original = revertPath(c.Files.Original, m)
		c.Files.Proxy = revertPath(c.Files.Proxy, m)
		for j, s := range c.Files.Sidecars {
			c.Files.Sidecars[j] = revertPath(s, m)
		}
		c.Applied = manifest.Applied{Done: false, FinalStem: "", AppliedAt: nil}
	}
	if err := manifest.Save(projectDir, man); err != nil {
		return res, fmt.Errorf("files reverted but manifest save failed: %w", err)
	}
	return res, nil
}

// revertPath maps a post-apply path back to its original via m, or returns it
// unchanged if not present.
func revertPath(p string, m map[string]string) string {
	if orig, ok := m[p]; ok {
		return orig
	}
	return p
}

func readLog(logPath string) ([]rename, error) {
	f, err := os.Open(logPath)
	if err != nil {
		return nil, fmt.Errorf("open apply log: %w", err)
	}
	defer f.Close()

	var out []rename
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var r rename
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			return nil, fmt.Errorf("corrupt apply log line: %w", err)
		}
		out = append(out, r)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
