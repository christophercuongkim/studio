package cli

import (
	"errors"
	"flag"
	"fmt"

	"github.com/christophercuongkim/studio/internal/config"
	"github.com/christophercuongkim/studio/internal/steps"
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

	res, err := steps.Archive(rest[0], cfg.ArchiveRoot, *dryRun, *keepProxies, *force)
	if err != nil {
		return err
	}
	fmt.Println(res.Text())
	return nil
}
