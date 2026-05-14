package store

import (
	"bytes"
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

func TestRecordConnect(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.json")
	s, _ := Load(path)
	s.Add(Connection{Name: "srv", Host: "h", Port: 22, Type: "ssh"})
	id := s.Connections[0].ID

	if s.Connections[0].LastConnectedAt != nil {
		t.Error("LastConnectedAt should be nil before first connect")
	}
	if err := s.RecordConnect(id); err != nil {
		t.Fatalf("RecordConnect: %v", err)
	}
	if s.Connections[0].LastConnectedAt == nil {
		t.Error("LastConnectedAt should be set after RecordConnect")
	}

	s2, _ := Load(path)
	if s2.Connections[0].LastConnectedAt == nil {
		t.Error("LastConnectedAt should persist after reload")
	}

	if err := s.RecordConnect("nonexistent"); err != nil {
		t.Errorf("RecordConnect with unknown ID should not error, got: %v", err)
	}
}

func TestExportAll(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.json")
	s, _ := Load(path)
	s.Add(Connection{Name: "a", Host: "h1", Port: 22, Type: "ssh"})
	s.Add(Connection{Name: "b", Host: "h2", Port: 3389, Type: "rdp"})

	exportPath := filepath.Join(t.TempDir(), "export.json")
	if err := s.ExportAll(exportPath); err != nil {
		t.Fatalf("ExportAll: %v", err)
	}

	s2, err := Load(exportPath)
	if err != nil {
		t.Fatalf("Load exported file: %v", err)
	}
	if len(s2.Connections) != 2 {
		t.Errorf("exported store should have 2 connections, got %d", len(s2.Connections))
	}
}

func TestImportMerge(t *testing.T) {
	dir := t.TempDir()
	pathA := filepath.Join(dir, "a.json")
	pathB := filepath.Join(dir, "b.json")

	sA, _ := Load(pathA)
	sA.Add(Connection{Name: "one", Host: "h1", Port: 22, Type: "ssh"})
	sA.Add(Connection{Name: "two", Host: "h2", Port: 22, Type: "ssh"})
	idOne := sA.Connections[0].ID

	sB, _ := Load(pathB)
	duplicate := Connection{Name: "one-dup", Host: "hX", Port: 22, Type: "ssh"}
	sB.Add(duplicate)
	sB.Connections[0].ID = idOne
	sB.save()
	sB.Add(Connection{Name: "three", Host: "h3", Port: 22, Type: "ssh"})

	exportPath := filepath.Join(dir, "b-export.json")
	sB.ExportAll(exportPath)

	added, err := sA.ImportMerge(exportPath)
	if err != nil {
		t.Fatalf("ImportMerge: %v", err)
	}
	if added != 1 {
		t.Errorf("expected 1 new connection added, got %d", added)
	}
	if len(sA.Connections) != 3 {
		t.Errorf("expected 3 connections after merge, got %d", len(sA.Connections))
	}
	for _, c := range sA.Connections {
		if c.Name == "one-dup" {
			t.Error("duplicate connection (same ID) should not have been imported")
		}
	}

	added2, err := sA.ImportMerge(exportPath)
	if err != nil {
		t.Fatalf("second ImportMerge: %v", err)
	}
	if added2 != 0 {
		t.Errorf("second import should add 0, got %d", added2)
	}
}

func TestExportBytes(t *testing.T) {
	dir := t.TempDir()
	s, err := Load(filepath.Join(dir, "c.json"))
	if err != nil {
		t.Fatal(err)
	}
	_ = s.Add(Connection{Name: "alpha", Host: "host.com", Type: "ssh"})

	data, err := s.ExportBytes()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`"alpha"`)) {
		t.Errorf("expected connection name in output, got: %s", data)
	}
	if !bytes.Contains(data, []byte(`"connections"`)) {
		t.Error("expected 'connections' key in output")
	}
}

func TestImportMergeBytes(t *testing.T) {
	dir := t.TempDir()
	s, err := Load(filepath.Join(dir, "c.json"))
	if err != nil {
		t.Fatal(err)
	}
	_ = s.Add(Connection{Name: "existing", Host: "h1.com", Type: "ssh"})
	existingID := s.Connections[0].ID

	payload := []byte(`{"connections":[
		{"id":"brand-new-id","name":"new","host":"h2.com","type":"ssh"},
		{"id":"` + existingID + `","name":"existing","host":"h1.com","type":"ssh"}
	]}`)

	added, err := s.ImportMergeBytes(payload)
	if err != nil {
		t.Fatal(err)
	}
	if added != 1 {
		t.Errorf("expected 1 added, got %d", added)
	}
	if len(s.Connections) != 2 {
		t.Errorf("expected 2 connections, got %d", len(s.Connections))
	}
}

func TestImportMergeBytesBadJSON(t *testing.T) {
	dir := t.TempDir()
	s, _ := Load(filepath.Join(dir, "c.json"))
	_, err := s.ImportMergeBytes([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
