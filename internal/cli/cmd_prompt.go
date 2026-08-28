package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/christophercuongkim/studio/internal/promptsrv"
	"github.com/christophercuongkim/studio/internal/script"
)

func runPrompt(args []string) error {
	fs := flag.NewFlagSet("prompt", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: studio prompt <project> [--addr 0.0.0.0:7724] [--log]")
		fmt.Fprintln(fs.Output(), "\nRecording teleprompter driven by script.md. Binds all interfaces by")
		fmt.Fprintln(fs.Output(), "default for second-device (iPad) use.")
		fmt.Fprintln(fs.Output())
		fs.PrintDefaults()
	}
	addr := fs.String("addr", "0.0.0.0:7724", "address to bind (all interfaces by default)")
	logFlag := fs.Bool("log", false, "append events to prompt-log.jsonl in the project")
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

	data, err := os.ReadFile(filepath.Join(projectDir, "script.md"))
	if err != nil {
		return fmt.Errorf("read script.md: %w", err)
	}
	sc, err := script.Parse(data)
	if err != nil {
		return err
	}

	logPath := ""
	if *logFlag {
		logPath = filepath.Join(projectDir, "prompt-log.jsonl")
	}
	srv := promptsrv.New(sc, logPath)
	httpSrv := &http.Server{Addr: *addr, Handler: srv.Handler()}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	fmt.Printf("studio prompt — %d sections\n", len(sc.Sections))
	if port := portOf(*addr); port != "" {
		for _, u := range promptsrv.InterfaceURLs(port) {
			fmt.Printf("  %s\n", u)
		}
	}
	fmt.Println("  (Ctrl-C to stop)")

	errCh := make(chan error, 1)
	go func() { errCh <- httpSrv.ListenAndServe() }()

	select {
	case <-ctx.Done():
		return httpSrv.Shutdown(context.Background())
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func portOf(addr string) string {
	if i := strings.LastIndex(addr, ":"); i >= 0 {
		return addr[i+1:]
	}
	return ""
}
