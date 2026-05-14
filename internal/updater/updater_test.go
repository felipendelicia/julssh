package updater

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func mockServer(t *testing.T, tag string, assets []asset) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(release{TagName: tag, Assets: assets})
	}))
}

func TestNoUpdateWhenDev(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("HTTP call made for dev version")
	}))
	defer srv.Close()
	old := apiURL
	apiURL = srv.URL
	defer func() { apiURL = old }()
	Check("dev")
}

func TestNoUpdateWhenSameVersion(t *testing.T) {
	srv := mockServer(t, "v0.6.0", nil)
	defer srv.Close()

	rel, err := fetchLatest(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	newer, err := isNewer(rel.TagName, "v0.6.0")
	if err != nil {
		t.Fatal(err)
	}
	if newer {
		t.Error("expected no update when versions are equal")
	}
}

func TestDetectsNewerVersion(t *testing.T) {
	srv := mockServer(t, "v0.7.0", nil)
	defer srv.Close()

	rel, err := fetchLatest(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	newer, err := isNewer(rel.TagName, "v0.6.0")
	if err != nil {
		t.Fatal(err)
	}
	if !newer {
		t.Error("expected newer=true when latest=v0.7.0 and current=v0.6.0")
	}
}

func TestAssetURLByArch(t *testing.T) {
	rel := &release{
		TagName: "v0.7.0",
		Assets: []asset{
			{Name: "julssh_linux_amd64.tar.gz", BrowserDownloadURL: "https://example.com/amd64.tar.gz"},
			{Name: "julssh_linux_arm64.tar.gz", BrowserDownloadURL: "https://example.com/arm64.tar.gz"},
		},
	}
	if url := findAssetURL(rel, "julssh_linux_amd64.tar.gz"); url != "https://example.com/amd64.tar.gz" {
		t.Errorf("amd64 URL = %q", url)
	}
	if url := findAssetURL(rel, "julssh_linux_arm64.tar.gz"); url != "https://example.com/arm64.tar.gz" {
		t.Errorf("arm64 URL = %q", url)
	}
	if url := findAssetURL(rel, "julssh_linux_missing.tar.gz"); url != "" {
		t.Errorf("expected empty URL for missing asset, got %q", url)
	}
}

func TestAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := fetchLatest(srv.URL)
	if err == nil {
		t.Error("expected error on HTTP 500")
	}
}

func TestParseSemver(t *testing.T) {
	cases := []struct {
		in   string
		want [3]int
		fail bool
	}{
		{"v1.2.3", [3]int{1, 2, 3}, false},
		{"v0.6.0", [3]int{0, 6, 0}, false},
		{"1.2.3", [3]int{1, 2, 3}, false},
		{"invalid", [3]int{}, true},
		{"v1.2", [3]int{}, true},
		{"v1.2.x", [3]int{}, true},
	}
	for _, c := range cases {
		got, err := parseSemver(c.in)
		if c.fail {
			if err == nil {
				t.Errorf("parseSemver(%q): expected error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseSemver(%q): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("parseSemver(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v0.7.0", "v0.6.0", true},
		{"v0.6.0", "v0.6.0", false},
		{"v0.5.0", "v0.6.0", false},
		{"v1.0.0", "v0.9.9", true},
		{"v0.6.1", "v0.6.0", true},
		{"v0.6.0", "v0.6.1", false},
	}
	for _, c := range cases {
		got, err := isNewer(c.latest, c.current)
		if err != nil {
			t.Errorf("isNewer(%q, %q): %v", c.latest, c.current, err)
			continue
		}
		if got != c.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}
