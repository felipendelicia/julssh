package store

import (
	"path/filepath"
	"testing"
)

func TestLoadCreatesFileIfNotExists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.json")
	s, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(s.Connections) != 0 {
		t.Errorf("expected 0 connections, got %d", len(s.Connections))
	}
}

func TestAddAndPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.json")
	s, _ := Load(path)
	err := s.Add(Connection{Name: "test", Host: "localhost", Port: 22})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	// reload from disk to verify persistence
	s2, _ := Load(path)
	if len(s2.Connections) != 1 {
		t.Fatalf("expected 1 connection after reload, got %d", len(s2.Connections))
	}
	c := s2.Connections[0]
	if c.Name != "test" {
		t.Errorf("expected name 'test', got '%s'", c.Name)
	}
	if c.ID == "" {
		t.Error("expected non-empty ID after Add")
	}
	if c.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestUpdate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.json")
	s, _ := Load(path)
	s.Add(Connection{Name: "original", Host: "localhost", Port: 22})
	conn := s.Connections[0]
	conn.Name = "updated"
	if err := s.Update(conn); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if s.Connections[0].Name != "updated" {
		t.Errorf("expected 'updated', got '%s'", s.Connections[0].Name)
	}
	if s.Connections[0].CreatedAt != conn.CreatedAt {
		t.Error("Update must preserve CreatedAt")
	}
}

func TestDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.json")
	s, _ := Load(path)
	s.Add(Connection{Name: "to-delete", Host: "localhost", Port: 22})
	id := s.Connections[0].ID
	if err := s.Delete(id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(s.Connections) != 0 {
		t.Errorf("expected 0 connections after delete, got %d", len(s.Connections))
	}
}

func TestFilter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.json")
	s, _ := Load(path)
	s.Add(Connection{Name: "prod-server", Host: "10.0.0.1", Port: 22, User: "admin", Tags: []string{"produccion"}})
	s.Add(Connection{Name: "dev-box", Host: "192.168.1.1", Port: 22, Tags: []string{"desarrollo"}})

	cases := []struct {
		query    string
		expected int
	}{
		{"prod", 1},
		{"192", 1},
		{"desarrollo", 1},
		{"PRODUCCION", 1}, // case-insensitive
		{"admin", 1},
		{"", 2},
		{"nada-que-matchee", 0},
	}
	for _, tc := range cases {
		result := s.Filter(tc.query)
		if len(result) != tc.expected {
			t.Errorf("Filter(%q): expected %d, got %d", tc.query, tc.expected, len(result))
		}
	}
}

func TestTypeMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.json")
	s, _ := Load(path)
	s.Add(Connection{Name: "legacy", Host: "h1", Port: 22})
	// simulate legacy JSON without type
	s.Connections[0].Type = ""
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	s2, _ := Load(path)
	if s2.Connections[0].Type != "ssh" {
		t.Errorf("expected type 'ssh' after migration, got %q", s2.Connections[0].Type)
	}
}

func TestFilterByType(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.json")
	s, _ := Load(path)
	s.Add(Connection{Name: "srv1", Host: "h1", Port: 22, Type: "ssh"})
	s.Add(Connection{Name: "win1", Host: "h2", Port: 3389, Type: "rdp"})
	s.Add(Connection{Name: "desk1", Host: "h3", Port: 5900, Type: "vnc"})

	if got := s.Filter("rdp"); len(got) != 1 || got[0].Name != "win1" {
		t.Errorf("Filter('rdp') unexpected: %v", got)
	}
	if got := s.Filter("vnc"); len(got) != 1 {
		t.Errorf("Filter('vnc') expected 1, got %d", len(got))
	}
}

func TestFilterByDomain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.json")
	s, _ := Load(path)
	s.Add(Connection{Name: "w", Host: "h", Port: 3389, Type: "rdp", Domain: "CORP"})
	s.Add(Connection{Name: "s", Host: "h2", Port: 22, Type: "ssh"})

	if got := s.Filter("corp"); len(got) != 1 {
		t.Errorf("Filter('corp') expected 1, got %d", len(got))
	}
}
