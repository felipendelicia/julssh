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
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Host         string    `json:"host"`
	Port         int       `json:"port"`
	User         string    `json:"user"`
	IdentityFile string    `json:"identity_file"`
	Description  string    `json:"description"`
	Tags         []string  `json:"tags"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
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

func matchesQuery(c Connection, q string) bool {
	fields := []string{c.Name, c.Host, c.User, c.IdentityFile, c.Description}
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
