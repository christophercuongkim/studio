package cli

import (
	"errors"
	"flag"
	"fmt"

	"github.com/christophercuongkim/studio/internal/steps"
)

func runApply(args []string) error {
	fs := flag.NewFlagSet("apply", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: studio apply <project> [--dry-run]")
		fmt.Fprintln(fs.Output(), "\nRename kept clips to their final names, journaling each rename for undo.")
		fmt.Fprintln(fs.Output())
		fs.PrintDefaults()
	}
	dryRun := fs.Bool("dry-run", false, "print the old→new table and make no changes")
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

	res, err := steps.Apply(rest[0], *dryRun)
	if err != nil {
		return err
	}
	fmt.Println(res.Text())
	return nil
}
