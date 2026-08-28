package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/christophercuongkim/studio/internal/chapters"
	"github.com/christophercuongkim/studio/internal/kdenlive"
	"github.com/christophercuongkim/studio/internal/videoyaml"
)

func runChapters(args []string) error {
	fs := flag.NewFlagSet("chapters", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: studio chapters <project> [--project-file <x.kdenlive>] [--write] [--offset 0s]")
		fmt.Fprintln(fs.Output(), "\nTurn Kdenlive timeline guides into a YouTube chapter list.")
		fmt.Fprintln(fs.Output())
		fs.PrintDefaults()
	}
	projectFile := fs.String("project-file", "", "specific .kdenlive (default: newest in project)")
	write := fs.Bool("write", false, "write the chapters into video.yaml:description")
	offset := fs.String("offset", "0s", "shift all timestamps (e.g. 3s, 1m30s)")
	rest, err := parseFlags(fs, args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if len(rest) != 1 {
		return fmt.Errorf("expected exactly one <project> argument, got %d", len(rest))
	}
	projectDir := rest[0]

	offsetDur, err := time.ParseDuration(*offset)
	if err != nil {
		return fmt.Errorf("--offset: %w", err)
	}

	kdePath := *projectFile
	if kdePath == "" {
		kdePath, err = newestKdenlive(projectDir)
		if err != nil {
			return err
		}
	} else if !filepath.IsAbs(kdePath) {
		kdePath = filepath.Join(projectDir, kdePath)
	}

	data, err := os.ReadFile(kdePath)
	if err != nil {
		return fmt.Errorf("read %s: %w", kdePath, err)
	}
	root, err := kdenlive.Load(data)
	if err != nil {
		return err
	}
	res, err := chapters.FromProject(root, offsetDur.Seconds())
	if err != nil {
		return err
	}
	for _, w := range res.Warnings {
		fmt.Fprintf(os.Stderr, "warn: %s\n", w)
	}

	if !*write {
		fmt.Println(res.Text())
		return nil
	}

	vy, err := videoyaml.Load(projectDir)
	if err != nil {
		return err
	}
	vy.Description = chapters.InsertIntoDescription(vy.Description, res.Text())
	if err := videoyaml.Save(projectDir, vy); err != nil {
		return err
	}
	fmt.Printf("wrote %d chapters into video.yaml\n", len(res.Chapters))
	return nil
}

// newestKdenlive returns the most recently modified *.kdenlive in dir.
func newestKdenlive(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	type cand struct {
		path string
		mod  time.Time
	}
	var cands []cand
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".kdenlive") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		cands = append(cands, cand{filepath.Join(dir, e.Name()), info.ModTime()})
	}
	if len(cands) == 0 {
		return "", fmt.Errorf("no .kdenlive file in %s (run 'studio scaffold' first)", dir)
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].mod.After(cands[j].mod) })
	return cands[0].path, nil
}
