package ssh

import (
	"testing"

	"github.com/felipem/julssh/internal/store"
)

func TestBuildArgs(t *testing.T) {
	cases := []struct {
		name     string
		conn     store.Connection
		expected []string
	}{
		{
			name:     "host only, default port",
			conn:     store.Connection{Host: "myhost", Port: 22},
			expected: []string{"myhost"},
		},
		{
			name:     "with user",
			conn:     store.Connection{Host: "myhost", Port: 22, User: "alice"},
			expected: []string{"alice@myhost"},
		},
		{
			name:     "non-default port",
			conn:     store.Connection{Host: "myhost", Port: 2222, User: "alice"},
			expected: []string{"alice@myhost", "-p", "2222"},
		},
		{
			name:     "with identity file",
			conn:     store.Connection{Host: "myhost", Port: 22, User: "alice", IdentityFile: "/home/user/.ssh/id_ed25519"},
			expected: []string{"alice@myhost", "-i", "/home/user/.ssh/id_ed25519"},
		},
		{
			name:     "all options",
			conn:     store.Connection{Host: "myhost", Port: 2222, User: "alice", IdentityFile: "/home/user/.ssh/id_rsa"},
			expected: []string{"alice@myhost", "-p", "2222", "-i", "/home/user/.ssh/id_rsa"},
		},
		{
			name:     "port 0 treated as default",
			conn:     store.Connection{Host: "myhost", Port: 0, User: "bob"},
			expected: []string{"bob@myhost"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildArgs(tc.conn)
			if len(got) != len(tc.expected) {
				t.Fatalf("expected args %v, got %v", tc.expected, got)
			}
			for i := range got {
				if got[i] != tc.expected[i] {
					t.Errorf("arg[%d]: expected %q, got %q", i, tc.expected[i], got[i])
				}
			}
		})
	}
}
