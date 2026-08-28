package cli

import (
	"errors"
	"flag"
	"fmt"

	"github.com/christophercuongkim/studio/internal/manifest"
)

func runStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: studio status <project>")
		fmt.Fprintln(fs.Output(), "\nShow review and apply counts for a project.")
	}
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

	man, err := manifest.Load(rest[0])
	if err != nil {
		return err
	}

	var pending, kept, rejected, applied int
	for _, c := range man.Clips {
		switch c.Review.Status {
		case manifest.StatusKept:
			kept++
		case manifest.StatusRejected:
			rejected++
		default:
			pending++
		}
		if c.Applied.Done {
			applied++
		}
	}

	fmt.Printf("%s — %d clip(s)\n", man.Shoot.Title, len(man.Clips))
	fmt.Printf("  pending %d · kept %d · rejected %d\n", pending, kept, rejected)
	fmt.Printf("  applied %d/%d kept\n", applied, kept)
	if man.ArchivedAt != nil {
		fmt.Printf("  archived %s → %s\n", man.ArchivedAt.Format("2006-01-02"), man.ArchivePath)
	}
	return nil
}
