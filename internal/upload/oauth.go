package upload

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

// OAuth file locations under ~/.config/studio (plan §15).
const (
	clientSecretFile = "yt-client.json" // user-provided Google OAuth client
	tokenFile        = "yt-token.json"  // cached token (0600)
)

func configDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "studio"), nil
}

// service builds an authenticated YouTube service, acquiring and caching an
// OAuth token on first use via a loopback redirect. notify, when non-nil,
// receives the authorization URL (so a UI can surface it) instead of stdout.
func service(ctx context.Context, notify func(url string)) (*youtube.Service, error) {
	dir, err := configDir()
	if err != nil {
		return nil, err
	}
	secret, err := os.ReadFile(filepath.Join(dir, clientSecretFile))
	if err != nil {
		return nil, fmt.Errorf("missing %s in %s — create a Google OAuth client (see README): %w", clientSecretFile, dir, err)
	}
	cfg, err := google.ConfigFromJSON(secret, youtube.YoutubeUploadScope, youtube.YoutubeScope)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", clientSecretFile, err)
	}

	tok, err := loadToken(filepath.Join(dir, tokenFile))
	if err != nil {
		tok, err = acquireToken(ctx, cfg, notify)
		if err != nil {
			return nil, err
		}
		if err := saveToken(filepath.Join(dir, tokenFile), tok); err != nil {
			return nil, err
		}
	}

	return youtube.NewService(ctx, option.WithHTTPClient(cfg.Client(ctx, tok)))
}

func loadToken(path string) (*oauth2.Token, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tok oauth2.Token
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, err
	}
	return &tok, nil
}

func saveToken(path string, tok *oauth2.Token) error {
	data, err := json.MarshalIndent(tok, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// acquireToken runs the installed-app auth-code flow with a loopback redirect on
// 127.0.0.1, capturing the code the browser is redirected to. notify, when
// non-nil, receives the auth URL (for a UI); otherwise it's printed to stdout.
func acquireToken(ctx context.Context, cfg *oauth2.Config, notify func(url string)) (*oauth2.Token, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	defer ln.Close()
	cfg.RedirectURL = fmt.Sprintf("http://%s", ln.Addr().String())

	codeCh := make(chan string, 1)
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code != "" {
			fmt.Fprintln(w, "studio: authorization received — you can close this tab.")
			codeCh <- code
		} else {
			http.Error(w, "no code", http.StatusBadRequest)
		}
	})}
	go srv.Serve(ln)
	defer srv.Close()

	authURL := cfg.AuthCodeURL("state", oauth2.AccessTypeOffline)
	if notify != nil {
		notify(authURL)
	} else {
		fmt.Println("Open this URL to authorize studio, then return here:")
		fmt.Println("  " + authURL)
	}

	select {
	case code := <-codeCh:
		return cfg.Exchange(ctx, code)
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("timed out waiting for authorization")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
