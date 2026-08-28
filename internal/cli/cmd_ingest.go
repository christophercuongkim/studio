package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/christophercuongkim/studio/internal/config"
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
	copyMode := fs.Bool("copy", false, "copy files instead of moving; leaves the dump untouched")
	jobs := fs.Int("jobs", 0, "parallel workers for checksum/proxy (default: NumCPU/2)")
	appendMode := fs.Bool("append", false, "add new footage to an existing manifest")
	rest, err := parseFlags(fs, args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if len(rest) != 1 {
		return fmt.Errorf("expected exactly one <dump-dir> argument, got %d; see 'studio ingest --help'", len(rest))
	}
	if *project == "" {
		return errors.New("--project is required")
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
		DumpDir:    rest[0],
		ProjectDir: *project,
		CamCode:    camCode,
		Copy:       *copyMode,
		Jobs:       *jobs,
		Append:     *appendMode,
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
