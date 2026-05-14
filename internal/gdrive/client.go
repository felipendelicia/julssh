package gdrive

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const driveFileName = "julssh-connections.json"

type Client struct {
	svc        *drive.Service
	fileIDPath string
}

func newClient(httpClient *http.Client, idPath string) (*Client, error) {
	svc, err := drive.NewService(context.Background(), option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create drive service: %w", err)
	}
	return &Client{svc: svc, fileIDPath: idPath}, nil
}

func fileIDPath(configDir string) string {
	return filepath.Join(configDir, "gdrive-file-id")
}

func (c *Client) readFileID() string {
	data, err := os.ReadFile(c.fileIDPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func (c *Client) writeFileID(id string) {
	_ = os.WriteFile(c.fileIDPath, []byte(id), 0600)
}

// Upload creates or updates julssh-connections.json on Drive.
func (c *Client) Upload(data []byte) error {
	fileID := c.readFileID()
	meta := &drive.File{Name: driveFileName, MimeType: "application/json"}

	if fileID != "" {
		_, err := c.svc.Files.Update(fileID, meta).
			Media(bytes.NewReader(data)).
			Do()
		if err != nil {
			// File was deleted remotely — create fresh.
			if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "notFound") {
				c.writeFileID("")
				return c.create(data)
			}
			return fmt.Errorf("drive update: %w", err)
		}
		return nil
	}

	return c.create(data)
}

func (c *Client) create(data []byte) error {
	meta := &drive.File{Name: driveFileName, MimeType: "application/json"}
	f, err := c.svc.Files.Create(meta).
		Media(bytes.NewReader(data)).
		Fields("id").
		Do()
	if err != nil {
		return fmt.Errorf("drive create: %w", err)
	}
	c.writeFileID(f.Id)
	return nil
}

// Download fetches julssh-connections.json from Drive.
func (c *Client) Download() ([]byte, error) {
	fileID := c.readFileID()

	if fileID == "" {
		list, err := c.svc.Files.List().
			Q(fmt.Sprintf("name = '%s' and trashed = false", driveFileName)).
			Fields("files(id)").
			Do()
		if err != nil {
			return nil, fmt.Errorf("drive list: %w", err)
		}
		if len(list.Files) == 0 {
			return nil, fmt.Errorf("no remote file found — push first to create it")
		}
		fileID = list.Files[0].Id
		c.writeFileID(fileID)
	}

	resp, err := c.svc.Files.Get(fileID).Download()
	if err != nil {
		return nil, fmt.Errorf("drive download: %w", err)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
