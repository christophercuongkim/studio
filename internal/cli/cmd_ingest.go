package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/christophercuongkim/studio/internal/config"
	"github.com/christophercuongkim/studio/internal/drives"
	"github.com/christophercuongkim/studio/internal/ingest"
)

func runIngest(args []string) error {
	fs := flag.NewFlagSet("ingest", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: studio ingest <dump-dir> --project <dir> [flags]")
		fmt.Fprintln(fs.Output(), "\nImport a camera-card dump into a project's layout and write manifest.json.")
		fmt.Fprintln(fs.Output())
		fs.PrintDefaults()
	}
	project := fs.String("project", "", "project folder to populate (required)")
	cam := fs.String("cam", "", "camera code for the shoot (default: config camCode)")
	clearSource := fs.Bool("clear-source", false, "after a verified copy, delete the sources to empty the card")
	pick := fs.Bool("pick", false, "choose the source card from a list of connected external drives")
	jobs := fs.Int("jobs", 0, "parallel workers for checksum/proxy (default: NumCPU/2)")
	appendMode := fs.Bool("append", false, "add new footage to an existing manifest")
	rest, err := parseFlags(fs, args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	// The source is either a picked drive or the positional <dump-dir>.
	var dumpDir string
	switch {
	case *pick:
		d, err := drives.Prompt(os.Stdout, os.Stdin, drives.External())
		if err != nil {
			return err
		}
		dumpDir = d.Path
		if len(rest) > 0 {
			return errors.New("give a <dump-dir> or --pick, not both")
		}
	case len(rest) == 1:
		dumpDir = rest[0]
	default:
		return fmt.Errorf("expected one <dump-dir> argument (or --pick), got %d; see 'studio ingest --help'", len(rest))
	}
	if *project == "" {
		return errors.New("--project is required")
	}

	// Ingest always copies; the dump is never mutated except by --clear-source,
	// which empties the card only after each copy verifies.
	if *clearSource {
		fmt.Printf("copying, then emptying the card (sources deleted only after their copy verifies)\n")
	}

	camCode := *cam
	if camCode == "" {
		if cfg, err := config.Load(); err == nil {
			camCode = cfg.CamCode
		}
	}

	// Ctrl-C cancels the run; ingest still writes a manifest for work done.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	sum, err := ingest.Run(ctx, ingest.Options{
		DumpDir:     dumpDir,
		ProjectDir:  *project,
		Copy:        true,
		CamCode:     camCode,
		ClearSource: *clearSource,
		Jobs:        *jobs,
		Append:      *appendMode,
		Progress:    func(line string) { fmt.Println(line) },
	})
	if sum != nil {
		printIngestSummary(sum)
	}
	if errors.Is(err, ingest.ErrInterrupted) {
		fmt.Fprintf(os.Stderr, "\ninterrupted — manifest saved with completed work; re-run with --append to finish.\n")
		return nil
	}
	return err
}

func printIngestSummary(s *ingest.Summary) {
	fmt.Printf("ingested %d clip(s): %d camera proxies adopted, %d generated, %d failed\n",
		s.NewClips, s.AdoptedProxies, s.GeneratedProxies, s.FailedProxies)
	if s.SkippedExisting > 0 {
		fmt.Printf("  skipped %d already-ingested clip(s) (--append dedupe)\n", s.SkippedExisting)
	}
	if s.SourceCleared > 0 {
		fmt.Printf("  cleared %d source file(s) from the card\n", s.SourceCleared)
	}
	for _, k := range s.SourceKept {
		fmt.Fprintf(os.Stderr, "warn: kept on card (copy didn't verify): %s\n", k)
	}
	if s.TotalBytes > 0 {
		fmt.Printf("  %.2f GiB of originals in %s\n", float64(s.TotalBytes)/(1<<30), s.Elapsed.Round(1e6))
	}
	for _, w := range s.ZeroByteSkipped {
		fmt.Fprintf(os.Stderr, "warn: skipped zero-byte file %s\n", w)
	}
	for _, w := range s.UnmatchedSidecars {
		fmt.Fprintf(os.Stderr, "warn: unmatched sidecar moved to originals/_unmatched: %s\n", w)
	}
	for _, w := range s.RejectedProxies {
		fmt.Fprintf(os.Stderr, "warn: rejected camera proxy: %s\n", w)
	}
	for _, w := range s.ProbeFailures {
		fmt.Fprintf(os.Stderr, "warn: probe failed, left in dump: %s\n", w)
	}
}
