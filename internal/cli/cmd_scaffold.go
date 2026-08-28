package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/christophercuongkim/studio/internal/config"
	"github.com/christophercuongkim/studio/internal/kdenlive"
	"github.com/christophercuongkim/studio/internal/manifest"
)

func runScaffold(args []string) error {
	fs := flag.NewFlagSet("scaffold", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: studio scaffold <project> [--template <path>] [--out <path>]")
		fmt.Fprintln(fs.Output(), "\nGenerate a Kdenlive project from a template with applied clips in the bin.")
		fmt.Fprintln(fs.Output())
		fs.PrintDefaults()
	}
	tmpl := fs.String("template", "", "empty Kdenlive template (default: config kdenliveTemplate)")
	out := fs.String("out", "", "output .kdenlive path (default: <project>/<title>.kdenlive; must not exist)")
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

	man, err := manifest.Load(projectDir)
	if err != nil {
		return err
	}

	templatePath := *tmpl
	if templatePath == "" {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("no --template and cannot read config: %w", err)
		}
		templatePath = cfg.KdenliveTemplate
	}
	if templatePath == "" {
		return errors.New("no Kdenlive template configured; pass --template or set kdenliveTemplate in config")
	}

	outPath := *out
	if outPath == "" {
		title := man.Shoot.Title
		if title == "" {
			title = "project"
		}
		outPath = filepath.Join(projectDir, title+".kdenlive")
	}
	if _, err := os.Stat(outPath); err == nil {
		return fmt.Errorf("output already exists: %s (scaffold never overwrites)", outPath)
	}

	// Gather applied, kept clips as bin producers (absolute original paths).
	absProject, _ := filepath.Abs(projectDir)
	var clips []kdenlive.ClipRef
	for _, c := range man.Clips {
		if c.Review.Status != manifest.StatusKept || !c.Applied.Done {
			continue
		}
		clips = append(clips, kdenlive.ClipRef{
			Resource:    filepath.Join(absProject, c.Files.Original),
			DurationSec: c.Media.DurationSec,
			Rating:      c.Review.Rating,
		})
	}
	if len(clips) == 0 {
		return errors.New("no applied, kept clips to place; run 'studio apply' first")
	}

	data, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("read template: %w", err)
	}
	root, err := kdenlive.Load(data)
	if err != nil {
		return err
	}
	if err := kdenlive.Scaffold(root, clips); err != nil {
		return err
	}
	if err := os.WriteFile(outPath, root.Render(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}

	fmt.Printf("scaffolded %s with %d clip(s)\n", outPath, len(clips))
	return nil
}
