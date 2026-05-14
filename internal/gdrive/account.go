package gdrive

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// UserInfo holds the display name and email of the authenticated Google account.
type UserInfo struct {
	Email string
	Name  string
}

func isLoggedIn(configDir string) bool {
	_, err := tokenFromFile(tokenPath(configDir))
	return err == nil
}

func getUserInfo(configDir string) (*UserInfo, error) {
	httpClient, err := GetClient(context.Background(), configDir)
	if err != nil {
		return nil, fmt.Errorf("auth: %w", err)
	}
	svc, err := drive.NewService(context.Background(), option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("drive service: %w", err)
	}
	about, err := svc.About.Get().Fields("user").Do()
	if err != nil {
		return nil, fmt.Errorf("drive about: %w", err)
	}
	return &UserInfo{
		Email: about.User.EmailAddress,
		Name:  about.User.DisplayName,
	}, nil
}

func logout(configDir string) error {
	if err := os.Remove(tokenPath(configDir)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove token: %w", err)
	}
	if err := os.Remove(fileIDPath(configDir)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove file ID: %w", err)
	}
	return nil
}

func login(configDir string) error {
	_, err := GetClient(context.Background(), configDir)
	return err
}

// IsLoggedIn reports whether a cached token exists for the current user.
func IsLoggedIn() bool {
	dir, err := julsshConfigDir()
	if err != nil {
		return false
	}
	return isLoggedIn(dir)
}

// GetUserInfo returns the Google account email and display name.
func GetUserInfo() (*UserInfo, error) {
	dir, err := julsshConfigDir()
	if err != nil {
		return nil, err
	}
	return getUserInfo(dir)
}

// Logout deletes the cached token and Drive file ID.
func Logout() error {
	dir, err := julsshConfigDir()
	if err != nil {
		return err
	}
	return logout(dir)
}

// Login triggers the browser OAuth flow and caches the resulting token.
func Login() error {
	dir, err := julsshConfigDir()
	if err != nil {
		return err
	}
	return login(dir)
}
