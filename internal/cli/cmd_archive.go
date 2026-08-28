package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/christophercuongkim/studio/internal/archive"
	"github.com/christophercuongkim/studio/internal/config"
)

func runArchive(args []string) error {
	fs := flag.NewFlagSet("archive", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: studio archive <project> [--dry-run] [--keep-proxies] [--force]")
		fmt.Fprintln(fs.Output(), "\nVerify originals, rsync to the archive root, prune regenerable files.")
		fmt.Fprintln(fs.Output())
		fs.PrintDefaults()
	}
	dryRun := fs.Bool("dry-run", false, "report what would happen; change nothing")
	keepProxies := fs.Bool("keep-proxies", false, "keep proxy/ (don't prune it)")
	force := fs.Bool("force", false, "archive even if the video is unpublished")
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

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	res, err := archive.Run(context.Background(), archive.Options{
		ProjectDir:  rest[0],
		ArchiveRoot: cfg.ArchiveRoot,
		DryRun:      *dryRun,
		KeepProxies: *keepProxies,
		Force:       *force,
	})
	if err != nil {
		return err
	}

	if res.DryRun {
		fmt.Printf("dry run: %d originals verified\n", res.Hashed)
		fmt.Printf("  would rsync → %s\n", res.ArchivePath)
		for _, p := range res.Pruned {
			fmt.Printf("  would prune %s/\n", p)
		}
		return nil
	}
	fmt.Printf("archived %d originals → %s\n", res.Hashed, res.ArchivePath)
	for _, p := range res.Pruned {
		fmt.Printf("  pruned %s/\n", p)
	}
	return nil
}
