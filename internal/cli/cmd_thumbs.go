package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"path/filepath"

	"github.com/christophercuongkim/studio/internal/manifest"
	"github.com/christophercuongkim/studio/internal/thumbs"
	"github.com/christophercuongkim/studio/internal/videoyaml"
)

func runThumbs(args []string) error {
	fs := flag.NewFlagSet("thumbs", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: studio thumbs <project> [--from render|clips] [--count 12]")
		fmt.Fprintln(fs.Output(), "\nExtract and rank thumbnail candidate frames; final art is a human job.")
		fmt.Fprintln(fs.Output())
		fs.PrintDefaults()
	}
	from := fs.String("from", "render", "frame source: render | clips")
	count := fs.Int("count", 12, "number of candidates to keep")
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

	src := thumbs.Source(*from)
	if src != thumbs.FromRender && src != thumbs.FromClips {
		return fmt.Errorf("--from must be 'render' or 'clips', got %q", *from)
	}

	opts := thumbs.Options{ProjectDir: projectDir, Source: src, Count: *count}

	// The clips source needs the manifest; the render source needs video.yaml.
	var man *manifest.Manifest
	if src == thumbs.FromClips {
		man, err = manifest.Load(projectDir)
		if err != nil {
			return err
		}
	} else {
		vy, err := videoyaml.Load(projectDir)
		if err != nil {
			return err
		}
		opts.RenderPath = filepath.Join(projectDir, vy.Render)
	}

	res, err := thumbs.Run(context.Background(), opts, man)
	if err != nil {
		return err
	}
	fmt.Printf("scanned %d frames → %d candidates\n", res.Scanned, len(res.Candidates))
	for _, c := range res.Candidates {
		fmt.Printf("  %s\n", filepath.Base(c))
	}
	fmt.Printf("contact sheet: %s\n", res.ContactSheet)
	return nil
}
