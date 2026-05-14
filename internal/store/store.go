package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("connection not found")

type Connection struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Host            string     `json:"host"`
	Port            int        `json:"port"`
	User            string     `json:"user"`
	IdentityFile    string     `json:"identity_file"`
	Domain          string     `json:"domain"`
	Password        string     `json:"password,omitempty"`
	Description     string     `json:"description"`
	Tags            []string   `json:"tags"`
	Type            string     `json:"type"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	LastConnectedAt *time.Time `json:"last_connected_at,omitempty"`
}

type Store struct {
	path        string
	Connections []Connection
}

type storeFile struct {
	Connections []Connection `json:"connections"`
}

func Load(path string) (*Store, error) {
	s := &Store{path: path, Connections: []Connection{}}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return nil, err
		}
		return s, s.save()
	}
	if err != nil {
		return nil, err
	}
	var f storeFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	if f.Connections != nil {
		s.Connections = f.Connections
	}
	for i := range s.Connections {
		if s.Connections[i].Type == "" {
			s.Connections[i].Type = "ssh"
		}
	}
	return s, nil
}

func (s *Store) save() error {
	f := storeFile{Connections: s.Connections}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) Add(c Connection) error {
	c.ID = uuid.New().String()
	now := time.Now()
	c.CreatedAt = now
	c.UpdatedAt = now
	s.Connections = append(s.Connections, c)
	return s.save()
}

func (s *Store) Update(c Connection) error {
	for i, existing := range s.Connections {
		if existing.ID == c.ID {
			c.CreatedAt = existing.CreatedAt
			c.UpdatedAt = time.Now()
			s.Connections[i] = c
			return s.save()
		}
	}
	return ErrNotFound
}

func (s *Store) Delete(id string) error {
	filtered := make([]Connection, 0, len(s.Connections))
	for _, c := range s.Connections {
		if c.ID != id {
			filtered = append(filtered, c)
		}
	}
	if len(filtered) == len(s.Connections) {
		return ErrNotFound
	}
	s.Connections = filtered
	return s.save()
}

func (s *Store) Filter(query string) []Connection {
	if query == "" {
		return s.Connections
	}
	q := strings.ToLower(query)
	var result []Connection
	for _, c := range s.Connections {
		if matchesQuery(c, q) {
			result = append(result, c)
		}
	}
	return result
}

func (s *Store) RecordConnect(id string) error {
	for i, c := range s.Connections {
		if c.ID == id {
			now := time.Now()
			s.Connections[i].LastConnectedAt = &now
			return s.save()
		}
	}
	return nil
}

func (s *Store) ExportAll(path string) error {
	f := storeFile{Connections: s.Connections}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func (s *Store) ImportMerge(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var f storeFile
	if err := json.Unmarshal(data, &f); err != nil {
		return 0, err
	}
	existing := make(map[string]bool, len(s.Connections))
	for _, c := range s.Connections {
		existing[c.ID] = true
	}
	added := 0
	for _, c := range f.Connections {
		if !existing[c.ID] {
			if c.Type == "" {
				c.Type = "ssh"
			}
			s.Connections = append(s.Connections, c)
			added++
		}
	}
	if added > 0 {
		return added, s.save()
	}
	return 0, nil
}

func matchesQuery(c Connection, q string) bool {
	fields := []string{c.Name, c.Host, c.User, c.IdentityFile, c.Description, c.Type, c.Domain}
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), q) {
			return true
		}
	}
	for _, tag := range c.Tags {
		if strings.Contains(strings.ToLower(tag), q) {
			return true
		}
	}
	return false
}
