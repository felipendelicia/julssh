package gdrive

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/felipem/julssh/internal/store"
)

type mockDriveFile struct {
	data        []byte
	uploadErr   error
	downloadErr error
}

func (m *mockDriveFile) Upload(data []byte) error {
	if m.uploadErr != nil {
		return m.uploadErr
	}
	m.data = make([]byte, len(data))
	copy(m.data, data)
	return nil
}

func (m *mockDriveFile) Download() ([]byte, error) {
	return m.data, m.downloadErr
}

func tempStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Load(filepath.Join(t.TempDir(), "c.json"))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestPushPullRoundTrip(t *testing.T) {
	s1 := tempStore(t)
	_ = s1.Add(store.Connection{Name: "server1", Host: "10.0.0.1", Type: "ssh"})

	mock := &mockDriveFile{}

	if err := push(mock, s1); err != nil {
		t.Fatalf("push: %v", err)
	}
	if len(mock.data) == 0 {
		t.Error("expected data uploaded to mock")
	}

	s2 := tempStore(t)
	added, err := pull(mock, s2)
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	if added != 1 {
		t.Errorf("expected 1 added, got %d", added)
	}
	if s2.Connections[0].Name != "server1" {
		t.Errorf("expected 'server1', got %q", s2.Connections[0].Name)
	}
}

func TestPushPullMergeByUUID(t *testing.T) {
	s1 := tempStore(t)
	_ = s1.Add(store.Connection{Name: "alpha", Host: "h1", Type: "ssh"})

	mock := &mockDriveFile{}
	_ = push(mock, s1)

	s2 := tempStore(t)
	_ = s2.Add(store.Connection{Name: "alpha", Host: "h1", Type: "ssh"})

	// s2 has its own "alpha" with a different UUID than s1's "alpha".
	// Pull adds s1's alpha (different UUID) — merge is by UUID not by name.
	added, err := pull(mock, s2)
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	if added != 1 {
		t.Errorf("expected 1 added (different UUID), got %d", added)
	}
	if len(s2.Connections) != 2 {
		t.Errorf("expected 2 connections, got %d", len(s2.Connections))
	}
}

func TestPullDownloadError(t *testing.T) {
	mock := &mockDriveFile{downloadErr: fmt.Errorf("network failure")}
	s := tempStore(t)
	_, err := pull(mock, s)
	if err == nil {
		t.Error("expected error on download failure")
	}
}

func TestPushUploadError(t *testing.T) {
	mock := &mockDriveFile{uploadErr: fmt.Errorf("quota exceeded")}
	s := tempStore(t)
	err := push(mock, s)
	if err == nil {
		t.Error("expected error on upload failure")
	}
}
