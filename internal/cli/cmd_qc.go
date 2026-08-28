package cli

import (
	"errors"
	"flag"
	"fmt"

	"github.com/christophercuongkim/studio/internal/steps"
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

	res, err := steps.QC(rest[0], *render, *profile)
	if err != nil {
		return err
	}
	fmt.Println(res.Text())
	if res.Failed {
		return errors.New("QC failed")
	}
	return nil
}
