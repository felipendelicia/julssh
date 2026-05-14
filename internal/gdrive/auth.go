package gdrive

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

const (
	clientID     = "734718020535-oe4h4jf464qr7ch4s6r6922nlperoo9i.apps.googleusercontent.com"
	clientSecret = "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"
)

func oauthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{drive.DriveFileScope},
		Endpoint:     google.Endpoint,
	}
}

func tokenPath(configDir string) string {
	return filepath.Join(configDir, "gdrive-token.json")
}

func tokenFromFile(path string) (*oauth2.Token, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	t := &oauth2.Token{}
	return t, json.NewDecoder(f).Decode(t)
}

func saveToken(path string, token *oauth2.Token) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
}

// GetClient returns an authenticated HTTP client, triggering browser auth on first use.
func GetClient(ctx context.Context, configDir string) (*http.Client, error) {
	conf := oauthConfig()
	path := tokenPath(configDir)

	token, err := tokenFromFile(path)
	if err == nil {
		// conf.Client auto-refreshes using RefreshToken when access token expires.
		// If token is revoked, subsequent API calls fail with 401 — handled in sync.go.
		return conf.Client(ctx, token), nil
	}

	authCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	token, err = browserAuth(authCtx, conf)
	if err != nil {
		return nil, err
	}
	if err := saveToken(path, token); err != nil {
		return nil, fmt.Errorf("save token: %w", err)
	}
	return conf.Client(ctx, token), nil
}

func browserAuth(ctx context.Context, conf *oauth2.Config) (*oauth2.Token, error) {
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, fmt.Errorf("start local server: %w", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	conf.RedirectURL = fmt.Sprintf("http://localhost:%d", port)

	state := "julssh-gdrive"
	authURL := conf.AuthCodeURL(state, oauth2.AccessTypeOffline)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux}
	var once sync.Once
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() {
			if r.URL.Query().Get("state") != state {
				errCh <- fmt.Errorf("state mismatch")
				return
			}
			code := r.URL.Query().Get("code")
			if code == "" {
				errCh <- fmt.Errorf("no code in redirect")
				return
			}
			fmt.Fprintf(w, "<html><body><p>Autenticación exitosa. Podés cerrar esta pestaña.</p></body></html>")
			codeCh <- code
		})
	})

	go func() { _ = srv.Serve(ln) }()
	defer srv.Close()

	exec.Command("xdg-open", authURL).Start()

	select {
	case code := <-codeCh:
		return conf.Exchange(ctx, code)
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("auth timed out: %w", ctx.Err())
	}
}
