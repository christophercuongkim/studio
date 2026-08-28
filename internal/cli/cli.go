// Package cli is the command registry and dispatcher for the studio binary.
//
// Each subcommand is a *Command registered in commands.go. A milestone adds a
// real implementation by replacing that command's Run func — nothing in main.go
// or this dispatcher needs to change. Keeping the surface in one ordered list
// means `studio --help` always reflects the whole pipeline, even for commands
// that are not built yet.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
)

// Command is a single studio subcommand.
type Command struct {
	// Name is the token typed after `studio` (e.g. "ingest").
	Name string
	// Summary is the one-line description shown in `studio --help`.
	Summary string
	// Usage is the argument synopsis shown after the name in help.
	Usage string
	// Run executes the command with its own args (everything after Name).
	// A nil Run means the command is declared but not yet implemented.
	Run func(args []string) error
}

// errNotImplemented is returned for declared-but-unbuilt commands so the
// roadmap is visible in --help while the pipeline is still being built out.
var errNotImplemented = errors.New("not implemented yet")

// Run dispatches args[0] to a registered command. It returns the process exit
// code: 0 on success, 1 on a command error, 2 on a usage error.
func Run(version string, args []string) int {
	if len(args) == 0 {
		printUsage(os.Stderr, version)
		return 2
	}

	switch args[0] {
	case "-h", "--help", "help":
		printUsage(os.Stdout, version)
		return 0
	case "-v", "--version", "version":
		fmt.Fprintf(os.Stdout, "studio %s\n", version)
		return 0
	}

	cmd := lookup(args[0])
	if cmd == nil {
		fmt.Fprintf(os.Stderr, "studio: unknown command %q\n\n", args[0])
		printUsage(os.Stderr, version)
		return 2
	}
	if cmd.Run == nil {
		fmt.Fprintf(os.Stderr, "studio %s: %v\n", cmd.Name, errNotImplemented)
		return 2
	}

	if err := cmd.Run(args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "studio %s: %v\n", cmd.Name, err)
		return 1
	}
	return 0
}

// lookup returns the registered command with the given name, or nil.
func lookup(name string) *Command {
	for _, c := range commands {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// printUsage writes the top-level help: synopsis plus an aligned command list.
func printUsage(w io.Writer, version string) {
	fmt.Fprintf(w, "studio %s — the full life of a YouTube video, one binary.\n\n", version)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  studio <command> [arguments]")
	fmt.Fprintln(w, "\nCommands:")

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, c := range commands {
		name := c.Name
		if c.Usage != "" {
			name += " " + c.Usage
		}
		fmt.Fprintf(tw, "  %s\t%s\n", name, c.Summary)
	}
	tw.Flush()

	fmt.Fprintln(w, "\nRun \"studio <command> --help\" for command-specific help.")
}
