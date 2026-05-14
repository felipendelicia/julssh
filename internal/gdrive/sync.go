package gdrive

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/felipem/julssh/internal/store"
)

type driveFile interface {
	Upload(data []byte) error
	Download() ([]byte, error)
}

func julsshConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "julssh"), nil
}

func buildClient() (*Client, error) {
	configDir, err := julsshConfigDir()
	if err != nil {
		return nil, err
	}
	httpClient, err := GetClient(context.Background(), configDir)
	if err != nil {
		return nil, fmt.Errorf("auth: %w", err)
	}
	return newClient(httpClient, fileIDPath(configDir))
}

func push(df driveFile, s *store.Store) error {
	data, err := s.ExportBytes()
	if err != nil {
		return err
	}
	return df.Upload(data)
}

func pull(df driveFile, s *store.Store) (int, error) {
	data, err := df.Download()
	if err != nil {
		return 0, err
	}
	return s.ImportMergeBytes(data)
}

// Push uploads all connections to Google Drive.
func Push(s *store.Store) error {
	client, err := buildClient()
	if err != nil {
		return err
	}
	return push(client, s)
}

// Pull downloads connections from Google Drive and merges them locally.
func Pull(s *store.Store) (int, error) {
	client, err := buildClient()
	if err != nil {
		return 0, err
	}
	return pull(client, s)
}
