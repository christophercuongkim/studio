package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/christophercuongkim/studio/internal/apply"
)

func runUndo(args []string) error {
	fs := flag.NewFlagSet("undo", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: studio undo <project> [--log undo/<file>.jsonl]")
		fmt.Fprintln(fs.Output(), "\nReverse the renames from an apply journal (the latest by default).")
		fmt.Fprintln(fs.Output())
		fs.PrintDefaults()
	}
	logFlag := fs.String("log", "", "apply journal to reverse (default: latest in undo/)")
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

	logPath := *logFlag
	if logPath == "" {
		logPath, err = apply.LatestLog(projectDir)
		if err != nil {
			return err
		}
	} else if !filepath.IsAbs(logPath) {
		logPath = filepath.Join(projectDir, logPath)
	}

	res, err := apply.Undo(projectDir, logPath)
	if err != nil {
		return err
	}
	fmt.Printf("reversed %d rename(s) from %s\n", res.Reversed, filepath.Base(logPath))
	for _, s := range res.Skipped {
		fmt.Fprintf(os.Stderr, "warn: skipped (target gone): %s\n", s)
	}
	return nil
}
