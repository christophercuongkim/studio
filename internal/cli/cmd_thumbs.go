package cli

import (
	"errors"
	"flag"
	"fmt"

	"github.com/christophercuongkim/studio/internal/steps"
)

func runThumbs(args []string) error {
	fs := flag.NewFlagSet("thumbs", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: studio thumbs <project> [--from render|clips] [--count 12]")
		fmt.Fprintln(fs.Output(), "\nExtract and rank thumbnail candidate frames; final art is a human job.")
		fmt.Fprintln(fs.Output())
		fs.PrintDefaults()
	}
	from := fs.String("from", "render", "frame source: render | clips")
	count := fs.Int("count", 12, "number of candidates to keep")
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

	res, err := steps.Thumbs(rest[0], *from, *count)
	if err != nil {
		return err
	}
	fmt.Println(res.Text())
	return nil
}
