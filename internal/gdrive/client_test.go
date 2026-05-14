package gdrive

import (
	"path/filepath"
	"testing"
)

func TestClientFileIDRoundTrip(t *testing.T) {
	dir := t.TempDir()
	c := &Client{fileIDPath: filepath.Join(dir, "gdrive-file-id")}

	if id := c.readFileID(); id != "" {
		t.Errorf("expected empty for missing file, got %q", id)
	}

	c.writeFileID("file-abc-123")
	if id := c.readFileID(); id != "file-abc-123" {
		t.Errorf("expected 'file-abc-123', got %q", id)
	}
}

func TestClientFileIDPathHelper(t *testing.T) {
	p := fileIDPath("/home/user/.config/julssh")
	if p != "/home/user/.config/julssh/gdrive-file-id" {
		t.Errorf("unexpected path: %q", p)
	}
}
