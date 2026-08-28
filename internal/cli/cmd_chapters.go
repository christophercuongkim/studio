package cli

import (
	"errors"
	"flag"
	"fmt"
	"time"

	"github.com/christophercuongkim/studio/internal/steps"
)

func runChapters(args []string) error {
	fs := flag.NewFlagSet("chapters", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: studio chapters <project> [--project-file <x.kdenlive>] [--write] [--offset 0s]")
		fmt.Fprintln(fs.Output(), "\nTurn Kdenlive timeline guides into a YouTube chapter list.")
		fmt.Fprintln(fs.Output())
		fs.PrintDefaults()
	}
	projectFile := fs.String("project-file", "", "specific .kdenlive (default: newest in project)")
	write := fs.Bool("write", false, "write the chapters into video.yaml:description")
	offset := fs.String("offset", "0s", "shift all timestamps (e.g. 3s, 1m30s)")
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

	offsetDur, err := time.ParseDuration(*offset)
	if err != nil {
		return fmt.Errorf("--offset: %w", err)
	}

	res, err := steps.Chapters(rest[0], *projectFile, *write, offsetDur)
	if err != nil {
		return err
	}
	fmt.Println(res.Text())
	return nil
}
