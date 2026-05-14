package gdrive

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestTokenRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	token := &oauth2.Token{
		AccessToken:  "test-access",
		RefreshToken: "test-refresh",
		Expiry:       time.Now().Add(time.Hour),
	}

	if err := saveToken(path, token); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected 0600 perms, got %o", info.Mode().Perm())
	}

	loaded, err := tokenFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.AccessToken != "test-access" {
		t.Errorf("AccessToken: expected %q, got %q", "test-access", loaded.AccessToken)
	}
	if loaded.RefreshToken != "test-refresh" {
		t.Errorf("RefreshToken: expected %q, got %q", "test-refresh", loaded.RefreshToken)
	}
}

func TestTokenFromFileMissing(t *testing.T) {
	_, err := tokenFromFile("/nonexistent/path/token.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestTokenPathUsesConfigDir(t *testing.T) {
	p := tokenPath("/tmp/julssh")
	if p != "/tmp/julssh/gdrive-token.json" {
		t.Errorf("unexpected token path: %q", p)
	}
}
