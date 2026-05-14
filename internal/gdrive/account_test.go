package gdrive

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestIsLoggedInFalse(t *testing.T) {
	dir := t.TempDir()
	if isLoggedIn(dir) {
		t.Error("expected false when no token file exists")
	}
}

func TestIsLoggedInTrue(t *testing.T) {
	dir := t.TempDir()
	token := &oauth2.Token{
		AccessToken:  "test-access",
		RefreshToken: "test-refresh",
		Expiry:       time.Now().Add(time.Hour),
	}
	if err := saveToken(tokenPath(dir), token); err != nil {
		t.Fatal(err)
	}
	if !isLoggedIn(dir) {
		t.Error("expected true when token file exists")
	}
}

func TestLogoutDeletesBothFiles(t *testing.T) {
	dir := t.TempDir()

	token := &oauth2.Token{AccessToken: "test", RefreshToken: "r"}
	if err := saveToken(tokenPath(dir), token); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fileIDPath(dir), []byte("file-123"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := logout(dir); err != nil {
		t.Fatalf("logout: %v", err)
	}

	if _, err := os.Stat(tokenPath(dir)); !os.IsNotExist(err) {
		t.Error("expected token file to be deleted")
	}
	if _, err := os.Stat(fileIDPath(dir)); !os.IsNotExist(err) {
		t.Error("expected file ID to be deleted")
	}
}

func TestLogoutIdempotent(t *testing.T) {
	dir := t.TempDir()
	if err := logout(dir); err != nil {
		t.Errorf("logout on empty dir must not error: %v", err)
	}
}

func TestLogoutPartialState(t *testing.T) {
	dir := t.TempDir()
	token := &oauth2.Token{AccessToken: "test", RefreshToken: "r"}
	if err := saveToken(tokenPath(dir), token); err != nil {
		t.Fatal(err)
	}
	// No file ID file — logout must still succeed.
	if err := logout(dir); err != nil {
		t.Errorf("logout with only token file must not error: %v", err)
	}
	if _, err := os.Stat(tokenPath(dir)); !os.IsNotExist(err) {
		t.Error("expected token file to be deleted")
	}
}

func TestUserInfoPathHelper(t *testing.T) {
	p := filepath.Join("/tmp/julssh", "gdrive-token.json")
	if tokenPath("/tmp/julssh") != p {
		t.Errorf("unexpected token path: %q", tokenPath("/tmp/julssh"))
	}
}
