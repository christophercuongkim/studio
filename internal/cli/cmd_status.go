package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/christophercuongkim/studio/internal/config"
	"github.com/christophercuongkim/studio/internal/pipeline"
)

func runStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: studio status [project]")
		fmt.Fprintln(fs.Output(), "\nShow a project's pipeline stage and what's next. With no argument,")
		fmt.Fprintln(fs.Output(), "summarize every project under searchRoots.")
	}
	rest, err := parseFlags(fs, args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	switch len(rest) {
	case 1:
		printProject(pipeline.Detect(rest[0]))
		return nil
	case 0:
		return statusOverview()
	default:
		return fmt.Errorf("expected at most one [project] argument, got %d", len(rest))
	}
}

// printProject prints one project's pipeline checklist and next step.
func printProject(s *pipeline.State) {
	fmt.Println(s.Title)
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, step := range s.Steps {
		box := "[ ]"
		if step.Done {
			box = "[x]"
		}
		fmt.Fprintf(tw, "  %s %s\t%s\n", box, step.Name, step.Detail)
	}
	tw.Flush()
	if s.Next == "" {
		fmt.Println("  ✓ complete")
	} else {
		fmt.Printf("  next: %s\n", s.Next)
	}
}

// statusOverview lists every project with its current stage.
func statusOverview() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	dirs := pipeline.FindProjects(cfg.SearchRoots)
	if len(dirs) == 0 {
		fmt.Println("no projects found under searchRoots")
		return nil
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "project\tstage")
	for _, dir := range dirs {
		s := pipeline.Detect(dir)
		stage := "→ " + s.Next
		if s.Next == "" {
			stage = "✓ complete"
		}
		fmt.Fprintf(tw, "%s\t%s\n", filepath.Base(dir), stage)
	}
	tw.Flush()
	return nil
}
