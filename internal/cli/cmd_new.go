package cli

import (
	"errors"
	"flag"
	"fmt"
	"time"

	"github.com/christophercuongkim/studio/internal/config"
	"github.com/christophercuongkim/studio/internal/project"
)

func runNew(args []string) error {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: studio new <slug> [--title \"…\"] [--date YYYY-MM-DD]")
		fmt.Fprintln(fs.Output(), "\nCreate a new project folder with script.md and video.yaml skeletons.")
		fmt.Fprintln(fs.Output())
		fs.PrintDefaults()
	}
	title := fs.String("title", "", "video title (default: de-slugged form)")
	date := fs.String("date", "", "shoot date YYYY-MM-DD (default: today)")
	root := fs.String("root", "", "where to create the project (default: external SSD if connected, else projectsRoot)")
	rest, err := parseFlags(fs, args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if len(rest) != 1 {
		return fmt.Errorf("expected exactly one <slug> argument, got %d", len(rest))
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	projectsRoot, source := cfg.ProjectsRootFor(*root)

	dir, err := project.Create(project.Options{
		ProjectsRoot: projectsRoot,
		Slug:         rest[0],
		Title:        *title,
		Date:         *date,
		CategoryID:   cfg.UploadDefaults.CategoryID,
		Privacy:      cfg.UploadDefaults.Privacy,
	}, time.Now())
	if err != nil {
		return err
	}

	switch source {
	case config.RootExternal:
		fmt.Printf("created %s  (external drive)\n\n", dir)
	default:
		fmt.Printf("created %s\n\n", dir)
	}
	fmt.Println("next steps:")
	fmt.Println("  script → shoot → ingest → serve → apply → scaffold → edit →")
	fmt.Println("  chapters → qc → thumbs → upload → archive")
	return nil
}
