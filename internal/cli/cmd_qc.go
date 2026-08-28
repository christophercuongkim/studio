package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/christophercuongkim/studio/internal/fsutil"
	"github.com/christophercuongkim/studio/internal/qc"
	"github.com/christophercuongkim/studio/internal/script"
	"github.com/christophercuongkim/studio/internal/videoyaml"
)

func runQC(args []string) error {
	fs := flag.NewFlagSet("qc", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: studio qc <project> [--render <file>] [--profile 4k30|1080p30|1080p60]")
		fmt.Fprintln(fs.Output(), "\nPre-upload render checks. Exits non-zero on any FAIL.")
		fmt.Fprintln(fs.Output())
		fs.PrintDefaults()
	}
	render := fs.String("render", "", "render file to check (default: video.yaml render)")
	profile := fs.String("profile", "", "expected profile: 4k30|1080p30|1080p60")
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

	renderPath := *render
	if renderPath == "" {
		vy, err := videoyaml.Load(projectDir)
		if err != nil {
			return fmt.Errorf("no --render and %w", err)
		}
		renderPath = filepath.Join(projectDir, vy.Render)
	} else if !filepath.IsAbs(renderPath) {
		renderPath = filepath.Join(projectDir, renderPath)
	}
	if _, err := os.Stat(renderPath); err != nil {
		return fmt.Errorf("render not found: %s", renderPath)
	}

	targetSec := targetLengthSeconds(projectDir)

	rep, err := qc.Run(context.Background(), renderPath, *profile, targetSec, qc.Default())
	if err != nil {
		return err
	}

	printQCTable(rep)

	if rep.Failed() {
		return errors.New("QC failed")
	}
	// On pass, write qc-report.json for upload to consume.
	rep.Timestamp = time.Now().UTC().Format(time.RFC3339)
	data, _ := json.MarshalIndent(rep, "", "  ")
	if err := fsutil.WriteFileAtomic(filepath.Join(projectDir, "qc-report.json"), append(data, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Println("qc-report.json written")
	return nil
}

// targetLengthSeconds reads script.md's targetLength (e.g. "8m") in seconds, or
// 0 if unavailable.
func targetLengthSeconds(projectDir string) float64 {
	data, err := os.ReadFile(filepath.Join(projectDir, "script.md"))
	if err != nil {
		return 0
	}
	sc, err := script.Parse(data)
	if err != nil || sc.TargetLength == "" {
		return 0
	}
	d, err := time.ParseDuration(sc.TargetLength)
	if err != nil {
		return 0
	}
	return d.Seconds()
}

func printQCTable(rep *qc.Report) {
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, c := range rep.Checks {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", c.Status, c.Name, c.Detail)
		if c.Repro != "" {
			fmt.Fprintf(tw, "\t\t%s\n", c.Repro)
		}
	}
	tw.Flush()
}
