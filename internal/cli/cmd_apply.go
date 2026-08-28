package cli

import (
	"errors"
	"flag"
	"fmt"
	"time"

	"github.com/christophercuongkim/studio/internal/apply"
	"github.com/christophercuongkim/studio/internal/manifest"
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
	projectDir := rest[0]

	man, err := manifest.Load(projectDir)
	if err != nil {
		return err
	}
	plan, err := apply.Build(man, projectDir)
	if err != nil {
		return err
	}

	if *dryRun {
		fmt.Println(plan.DryRunTable())
		return nil
	}
	if plan.Empty() {
		fmt.Println("nothing to apply (no kept, unapplied clips)")
		return nil
	}

	logPath, err := apply.Execute(man, projectDir, plan, time.Now().UTC())
	if err != nil {
		return err
	}
	fmt.Printf("applied %d rename(s); journal: %s\n", len(plan.Renames), logPath)
	return nil
}
