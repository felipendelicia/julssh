# Auto-Updater Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** At startup, check GitHub Releases for a newer version and offer to download, replace, and relaunch automatically.

**Architecture:** New `internal/updater/updater.go` with `Check(currentVersion string)` called from `main.go` before the TUI starts. Internal helpers (`fetchLatest`, `isNewer`, `parseSemver`, `findAssetURL`, `doUpdate`) are unexported and tested directly. No new dependencies — stdlib only.

**Tech Stack:** Go stdlib (`net/http`, `archive/tar`, `compress/gzip`, `syscall`, `os`), `httptest` for tests, GitHub Releases API.

---

## File Map

| Action | Path | Responsibility |
|--------|------|----------------|
| Create | `internal/updater/updater.go` | Check, fetchLatest, parseSemver, isNewer, assetFilename, findAssetURL, doUpdate |
| Create | `internal/updater/updater_test.go` | Tests for all pure/HTTP functions |
| Modify | `main.go` | Call `updater.Check(version)` before TUI |

---

## Task 1: internal/updater — Core logic

**Files:**
- Create: `internal/updater/updater.go`
- Create: `internal/updater/updater_test.go`

- [ ] **Step 1: Write failing tests in `updater_test.go`**

  Create `internal/updater/updater_test.go`:

  ```go
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
  ```

- [ ] **Step 2: Run to confirm failure**

  ```bash
  cd /home/felipe/Documents/Repositories/julssh
  go test ./internal/updater/... -v
  ```

  Expected: `FAIL` — package `updater` does not exist.

- [ ] **Step 3: Create `internal/updater/updater.go`**

  ```go
  package updater

  import (
      "archive/tar"
      "compress/gzip"
      "encoding/json"
      "fmt"
      "io"
      "net/http"
      "os"
      "path/filepath"
      "runtime"
      "strconv"
      "strings"
      "syscall"
      "time"
  )

  var apiURL = "https://api.github.com/repos/felipendelicia/julssh/releases/latest"

  type release struct {
      TagName string  `json:"tag_name"`
      Assets  []asset `json:"assets"`
  }

  type asset struct {
      Name               string `json:"name"`
      BrowserDownloadURL string `json:"browser_download_url"`
  }

  // Check looks up the latest GitHub release. If newer than currentVersion,
  // it prompts the user, downloads, replaces the binary, and relaunches.
  // Returns immediately if currentVersion == "dev".
  func Check(currentVersion string) {
      if currentVersion == "dev" {
          return
      }

      rel, err := fetchLatest(apiURL)
      if err != nil {
          fmt.Fprintf(os.Stderr, "No se pudo verificar actualizaciones: %v\n", err)
          return
      }

      newer, err := isNewer(rel.TagName, currentVersion)
      if err != nil || !newer {
          return
      }

      url := findAssetURL(rel, assetFilename())
      if url == "" {
          return
      }

      fmt.Printf("Nueva versión %s disponible. ¿Actualizar? [s/N] ", rel.TagName)
      var answer string
      fmt.Scanln(&answer)
      if strings.ToLower(strings.TrimSpace(answer)) != "s" {
          return
      }

      fmt.Println("Descargando actualización...")
      if err := doUpdate(url); err != nil {
          fmt.Fprintf(os.Stderr, "Error al actualizar: %v\n", err)
          return
      }

      execPath, err := os.Executable()
      if err != nil {
          fmt.Fprintf(os.Stderr, "Error al relanzar: %v\n", err)
          return
      }
      _ = syscall.Exec(execPath, os.Args, os.Environ())
  }

  func fetchLatest(url string) (*release, error) {
      client := &http.Client{Timeout: 3 * time.Second}
      resp, err := client.Get(url)
      if err != nil {
          return nil, err
      }
      defer resp.Body.Close()
      if resp.StatusCode != http.StatusOK {
          return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
      }
      var rel release
      if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
          return nil, err
      }
      return &rel, nil
  }

  func parseSemver(v string) ([3]int, error) {
      v = strings.TrimPrefix(v, "v")
      parts := strings.Split(v, ".")
      if len(parts) != 3 {
          return [3]int{}, fmt.Errorf("invalid semver: %q", v)
      }
      var nums [3]int
      for i, p := range parts {
          n, err := strconv.Atoi(p)
          if err != nil {
              return [3]int{}, fmt.Errorf("invalid semver part %q: %w", p, err)
          }
          nums[i] = n
      }
      return nums, nil
  }

  func isNewer(latest, current string) (bool, error) {
      l, err := parseSemver(latest)
      if err != nil {
          return false, err
      }
      c, err := parseSemver(current)
      if err != nil {
          return false, err
      }
      for i := range l {
          if l[i] > c[i] {
              return true, nil
          }
          if l[i] < c[i] {
              return false, nil
          }
      }
      return false, nil
  }

  func assetFilename() string {
      return fmt.Sprintf("julssh_linux_%s.tar.gz", runtime.GOARCH)
  }

  func findAssetURL(rel *release, name string) string {
      for _, a := range rel.Assets {
          if a.Name == name {
              return a.BrowserDownloadURL
          }
      }
      return ""
  }

  func doUpdate(url string) error {
      execPath, err := os.Executable()
      if err != nil {
          return fmt.Errorf("obtener ruta ejecutable: %w", err)
      }

      resp, err := http.Get(url)
      if err != nil {
          return fmt.Errorf("descarga: %w", err)
      }
      defer resp.Body.Close()
      if resp.StatusCode != http.StatusOK {
          return fmt.Errorf("descarga HTTP %d", resp.StatusCode)
      }

      gr, err := gzip.NewReader(resp.Body)
      if err != nil {
          return fmt.Errorf("gzip: %w", err)
      }
      defer gr.Close()

      tr := tar.NewReader(gr)
      var binaryData []byte
      for {
          hdr, err := tr.Next()
          if err == io.EOF {
              break
          }
          if err != nil {
              return fmt.Errorf("tar: %w", err)
          }
          if hdr.Name == "julssh" || strings.HasSuffix(hdr.Name, "/julssh") {
              binaryData, err = io.ReadAll(tr)
              if err != nil {
                  return fmt.Errorf("leer binario: %w", err)
              }
              break
          }
      }
      if binaryData == nil {
          return fmt.Errorf("binario julssh no encontrado en el archivo")
      }

      dir := filepath.Dir(execPath)
      tmp, err := os.CreateTemp(dir, "julssh-update-*")
      if err != nil {
          return fmt.Errorf("crear temp: %w", err)
      }
      tmpName := tmp.Name()

      if err := tmp.Chmod(0755); err != nil {
          tmp.Close()
          os.Remove(tmpName)
          return fmt.Errorf("chmod: %w", err)
      }
      if _, err := tmp.Write(binaryData); err != nil {
          tmp.Close()
          os.Remove(tmpName)
          return fmt.Errorf("escribir temp: %w", err)
      }
      if err := tmp.Close(); err != nil {
          os.Remove(tmpName)
          return fmt.Errorf("cerrar temp: %w", err)
      }

      if err := os.Rename(tmpName, execPath); err != nil {
          os.Remove(tmpName)
          return fmt.Errorf("reemplazar ejecutable: %w", err)
      }

      return nil
  }
  ```

- [ ] **Step 4: Run tests**

  ```bash
  go test ./internal/updater/... -v
  ```

  Expected: all tests PASS.

- [ ] **Step 5: Build to verify no compile errors**

  ```bash
  go build ./...
  ```

  Expected: no output.

- [ ] **Step 6: Commit**

  ```bash
  git add internal/updater/updater.go internal/updater/updater_test.go
  git commit -m "feat(updater): add auto-update check against GitHub Releases"
  ```

---

## Task 2: Wire updater in main.go

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Add `updater.Check(version)` to `main.go`**

  Current `main.go` (relevant section):

  ```go
  func main() {
      if len(os.Args) > 1 && os.Args[1] == "--version" {
          fmt.Println("julssh", version)
          os.Exit(0)
      }

      configDir, err := os.UserConfigDir()
  ```

  Add the import and the call:

  ```go
  import (
      // existing imports...
      "github.com/felipem/julssh/internal/updater"
  )

  func main() {
      if len(os.Args) > 1 && os.Args[1] == "--version" {
          fmt.Println("julssh", version)
          os.Exit(0)
      }

      updater.Check(version)

      configDir, err := os.UserConfigDir()
  ```

- [ ] **Step 2: Build**

  ```bash
  go build -o julssh .
  ```

  Expected: no errors.

- [ ] **Step 3: Run full test suite**

  ```bash
  go test ./...
  ```

  Expected: all tests PASS.

- [ ] **Step 4: Manual smoke test**

  ```bash
  ./julssh
  ```

  Expected: TUI starts normally (current version == latest → no prompt, or prompt appears if a newer release exists on GitHub).

  To force-test the update prompt with a fake old version:

  ```bash
  # build with fake old version
  go build -ldflags="-X main.version=v0.1.0" -o julssh_old .
  ./julssh_old
  ```

  Expected: "Nueva versión v0.6.0 disponible. ¿Actualizar? [s/N]" printed. Press N → TUI starts. Press s → downloads, replaces binary, relaunches.

- [ ] **Step 5: Commit**

  ```bash
  git add main.go
  git commit -m "feat: run auto-update check on startup"
  ```
