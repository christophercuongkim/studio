package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/christophercuongkim/studio/internal/server"
)

func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: studio serve <project> [--addr 127.0.0.1:7723]")
		fmt.Fprintln(fs.Output(), "\nReview clips in a browser: rate, name, keep/reject. Binds localhost only.")
		fmt.Fprintln(fs.Output())
		fs.PrintDefaults()
	}
	addr := fs.String("addr", "127.0.0.1:7723", "address to bind (localhost only)")
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

	srv, err := server.New(rest[0])
	if err != nil {
		return err
	}
	defer srv.Close()

	httpSrv := &http.Server{Addr: *addr, Handler: srv.Handler()}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		fmt.Printf("studio serve — http://%s  (Ctrl-C to stop)\n", *addr)
		errCh <- httpSrv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		fmt.Println("\nshutting down, saving edits…")
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutCtx)
		return srv.Close() // flush any pending debounced save
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
