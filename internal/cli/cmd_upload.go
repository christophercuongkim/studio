package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/christophercuongkim/studio/internal/upload"
)

func runUpload(args []string) error {
	fs := flag.NewFlagSet("upload", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: studio upload <project> [--dry-run] [--privacy …] [--update] [--skip-qc]")
		fmt.Fprintln(fs.Output(), "\nUpload the render to YouTube (or update its metadata).")
		fmt.Fprintln(fs.Output())
		fs.PrintDefaults()
	}
	dryRun := fs.Bool("dry-run", false, "print the request payload and make no network calls")
	privacy := fs.String("privacy", "", "override privacy: private|unlisted|public")
	update := fs.Bool("update", false, "metadata-only update of an already-uploaded video")
	skipQC := fs.Bool("skip-qc", false, "skip the passing-qc-report requirement")
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

	res, err := upload.Run(context.Background(), upload.Options{
		ProjectDir: rest[0],
		DryRun:     *dryRun,
		Privacy:    *privacy,
		Update:     *update,
		SkipQC:     *skipQC,
	})
	if err != nil {
		return err
	}
	switch {
	case res.DryRun:
		// payload already printed
	case res.Updated:
		fmt.Printf("updated https://youtu.be/%s\n", res.VideoID)
	default:
		fmt.Printf("uploaded https://youtu.be/%s\n", res.VideoID)
	}
	return nil
}
