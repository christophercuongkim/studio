package cli

import (
	"errors"
	"flag"
	"fmt"

	"github.com/christophercuongkim/studio/internal/config"
	"github.com/christophercuongkim/studio/internal/steps"
)

func runScaffold(args []string) error {
	fs := flag.NewFlagSet("scaffold", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: studio scaffold <project> [--template <path>] [--out <path>]")
		fmt.Fprintln(fs.Output(), "\nGenerate a Kdenlive project from a template with applied clips in the bin.")
		fmt.Fprintln(fs.Output())
		fs.PrintDefaults()
	}
	tmpl := fs.String("template", "", "empty Kdenlive template (default: config kdenliveTemplate)")
	out := fs.String("out", "", "output .kdenlive path (default: <project>/<title>.kdenlive; must not exist)")
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

	templatePath := *tmpl
	if templatePath == "" {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("no --template and cannot read config: %w", err)
		}
		templatePath = cfg.KdenliveTemplate
	}

	res, err := steps.Scaffold(rest[0], templatePath, *out)
	if err != nil {
		return err
	}
	fmt.Println(res.Text())
	return nil
}
