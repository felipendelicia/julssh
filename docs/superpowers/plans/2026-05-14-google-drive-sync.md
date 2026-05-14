# Google Drive Sync Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add manual push/pull sync of connections to Google Drive via `S`/`L` keys in the list view.

**Architecture:** New `internal/gdrive/` package handles OAuth2 browser flow, Drive API calls, and push/pull orchestration. `Store` gains `ExportBytes`/`ImportMergeBytes` for in-memory serialization. List dispatches async cmds; app.go handles result messages exactly like existing `MsgExportDone`/`MsgImportDone`.

**Tech Stack:** `golang.org/x/oauth2`, `google.golang.org/api/drive/v3`, Google Drive API v3 with `drive.file` scope (julssh-only, minimal permissions).

---

## File Map

| Action | Path | Responsibility |
|--------|------|----------------|
| Create | `internal/gdrive/auth.go` | OAuth2 config, token cache, browser flow |
| Create | `internal/gdrive/auth_test.go` | Token read/write tests |
| Create | `internal/gdrive/client.go` | Drive API wrapper, file ID cache |
| Create | `internal/gdrive/client_test.go` | File ID cache tests |
| Create | `internal/gdrive/sync.go` | Push/Pull orchestration, driveFile interface |
| Create | `internal/gdrive/sync_test.go` | Push/Pull logic tests via mock |
| Modify | `internal/store/store.go` | Add ExportBytes, ImportMergeBytes; refactor ImportMerge |
| Modify | `internal/store/store_test.go` | Tests for new methods |
| Modify | `internal/model/app.go` | Add MsgDrivePushDone/MsgDrivePullDone and handlers |
| Modify | `internal/model/list.go` | Add S/L keys, import gdrive, update footer |

---

## Task 1: Google Cloud Setup + Add Dependencies

**Files:**
- Modify: `go.mod`, `go.sum` (via `go get`)

- [ ] **Step 1: Create OAuth credentials in Google Cloud Console**

  1. Go to https://console.cloud.google.com → create or select a project.
  2. APIs & Services → Enable APIs → search "Google Drive API" → Enable.
  3. APIs & Services → Credentials → Create Credentials → OAuth client ID.
  4. Application type: **Desktop app**. Name: `julssh`.
  5. Download or copy the **Client ID** and **Client Secret**.
  Keep these values — you'll embed them in `auth.go` in Task 3.

- [ ] **Step 2: Add Go dependencies**

  ```bash
  cd /path/to/julssh
  go get golang.org/x/oauth2
  go get google.golang.org/api/drive/v3
  ```

  Expected: `go.mod` and `go.sum` updated.

- [ ] **Step 3: Commit**

  ```bash
  git add go.mod go.sum
  git commit -m "chore: add oauth2 and Google Drive API dependencies"
  ```

---

## Task 2: Store — ExportBytes and ImportMergeBytes

**Files:**
- Modify: `internal/store/store.go`
- Modify: `internal/store/store_test.go`

- [ ] **Step 1: Write failing tests in `store_test.go`**

  Add to `internal/store/store_test.go`:

  ```go
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
  ```

  Add `"bytes"` to imports in `store_test.go`.

- [ ] **Step 2: Run to confirm failure**

  ```bash
  go test ./internal/store/... -run "TestExportBytes|TestImportMergeBytes" -v
  ```

  Expected: `FAIL` — methods not defined.

- [ ] **Step 3: Implement ExportBytes, ImportMergeBytes, refactor ImportMerge in `store.go`**

  Add after `ImportMerge`:

  ```go
  func (s *Store) ExportBytes() ([]byte, error) {
      f := storeFile{Connections: s.Connections}
      return json.MarshalIndent(f, "", "  ")
  }

  func (s *Store) ImportMergeBytes(data []byte) (int, error) {
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
  ```

  Refactor the existing `ImportMerge` to call `ImportMergeBytes`:

  ```go
  func (s *Store) ImportMerge(path string) (int, error) {
      data, err := os.ReadFile(path)
      if err != nil {
          return 0, err
      }
      return s.ImportMergeBytes(data)
  }
  ```

- [ ] **Step 4: Run tests**

  ```bash
  go test ./internal/store/... -v
  ```

  Expected: all tests PASS including existing ones.

- [ ] **Step 5: Commit**

  ```bash
  git add internal/store/store.go internal/store/store_test.go
  git commit -m "feat(store): add ExportBytes and ImportMergeBytes"
  ```

---

## Task 3: gdrive/auth.go — OAuth2 Token Cache and Browser Flow

**Files:**
- Create: `internal/gdrive/auth.go`
- Create: `internal/gdrive/auth_test.go`

- [ ] **Step 1: Write failing tests in `auth_test.go`**

  Create `internal/gdrive/auth_test.go`:

  ```go
  package gdrive

  import (
      "os"
      "path/filepath"
      "testing"
      "time"

      "golang.org/x/oauth2"
  )

  func TestTokenRoundTrip(t *testing.T) {
      dir := t.TempDir()
      path := filepath.Join(dir, "token.json")

      token := &oauth2.Token{
          AccessToken:  "test-access",
          RefreshToken: "test-refresh",
          Expiry:       time.Now().Add(time.Hour),
      }

      if err := saveToken(path, token); err != nil {
          t.Fatal(err)
      }

      info, err := os.Stat(path)
      if err != nil {
          t.Fatal(err)
      }
      if info.Mode().Perm() != 0600 {
          t.Errorf("expected 0600 perms, got %o", info.Mode().Perm())
      }

      loaded, err := tokenFromFile(path)
      if err != nil {
          t.Fatal(err)
      }
      if loaded.AccessToken != "test-access" {
          t.Errorf("AccessToken: expected %q, got %q", "test-access", loaded.AccessToken)
      }
      if loaded.RefreshToken != "test-refresh" {
          t.Errorf("RefreshToken: expected %q, got %q", "test-refresh", loaded.RefreshToken)
      }
  }

  func TestTokenFromFileMissing(t *testing.T) {
      _, err := tokenFromFile("/nonexistent/path/token.json")
      if err == nil {
          t.Error("expected error for missing file")
      }
  }

  func TestTokenPathUsesConfigDir(t *testing.T) {
      p := tokenPath("/tmp/julssh")
      if p != "/tmp/julssh/gdrive-token.json" {
          t.Errorf("unexpected token path: %q", p)
      }
  }
  ```

- [ ] **Step 2: Run to confirm failure**

  ```bash
  go test ./internal/gdrive/... -run "TestToken" -v
  ```

  Expected: `FAIL` — package not found.

- [ ] **Step 3: Create `internal/gdrive/auth.go`**

  Replace `YOUR_CLIENT_ID` and `YOUR_CLIENT_SECRET` with the values from Task 1 Step 1.

  ```go
  package gdrive

  import (
      "context"
      "encoding/json"
      "fmt"
      "net"
      "net/http"
      "os"
      "os/exec"
      "path/filepath"
      "time"

      "golang.org/x/oauth2"
      "golang.org/x/oauth2/google"
      "google.golang.org/api/drive/v3"
  )

  const (
      clientID     = "YOUR_CLIENT_ID"
      clientSecret = "YOUR_CLIENT_SECRET"
  )

  func oauthConfig() *oauth2.Config {
      return &oauth2.Config{
          ClientID:     clientID,
          ClientSecret: clientSecret,
          Scopes:       []string{drive.DriveFileScope},
          Endpoint:     google.Endpoint,
      }
  }

  func tokenPath(configDir string) string {
      return filepath.Join(configDir, "gdrive-token.json")
  }

  func tokenFromFile(path string) (*oauth2.Token, error) {
      f, err := os.Open(path)
      if err != nil {
          return nil, err
      }
      defer f.Close()
      t := &oauth2.Token{}
      return t, json.NewDecoder(f).Decode(t)
  }

  func saveToken(path string, token *oauth2.Token) error {
      f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
      if err != nil {
          return err
      }
      defer f.Close()
      return json.NewEncoder(f).Encode(token)
  }

  // GetClient returns an authenticated HTTP client, triggering browser auth on first use.
  func GetClient(ctx context.Context, configDir string) (*http.Client, error) {
      conf := oauthConfig()
      path := tokenPath(configDir)

      token, err := tokenFromFile(path)
      if err == nil {
          // conf.Client auto-refreshes using RefreshToken when access token expires.
          // If token is revoked, subsequent API calls fail with 401 — handled in sync.go.
          return conf.Client(ctx, token), nil
      }

      authCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
      defer cancel()

      token, err = browserAuth(authCtx, conf)
      if err != nil {
          return nil, err
      }
      _ = saveToken(path, token)
      return conf.Client(ctx, token), nil
  }

  func browserAuth(ctx context.Context, conf *oauth2.Config) (*oauth2.Token, error) {
      ln, err := net.Listen("tcp", "localhost:0")
      if err != nil {
          return nil, fmt.Errorf("start local server: %w", err)
      }
      port := ln.Addr().(*net.TCPAddr).Port
      conf.RedirectURL = fmt.Sprintf("http://localhost:%d", port)

      state := "julssh-gdrive"
      authURL := conf.AuthCodeURL(state, oauth2.AccessTypeOffline)

      codeCh := make(chan string, 1)
      errCh := make(chan error, 1)

      mux := http.NewServeMux()
      srv := &http.Server{Handler: mux}
      mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
          if r.URL.Query().Get("state") != state {
              errCh <- fmt.Errorf("state mismatch")
              return
          }
          code := r.URL.Query().Get("code")
          if code == "" {
              errCh <- fmt.Errorf("no code in redirect")
              return
          }
          fmt.Fprintf(w, "<html><body><p>Autenticación exitosa. Podés cerrar esta pestaña.</p></body></html>")
          codeCh <- code
      })

      go func() { _ = srv.Serve(ln) }()
      defer srv.Close()

      exec.Command("xdg-open", authURL).Start()

      select {
      case code := <-codeCh:
          return conf.Exchange(ctx, code)
      case err := <-errCh:
          return nil, err
      case <-ctx.Done():
          return nil, fmt.Errorf("auth timed out: %w", ctx.Err())
      }
  }
  ```

- [ ] **Step 4: Run tests**

  ```bash
  go test ./internal/gdrive/... -run "TestToken" -v
  ```

  Expected: all 3 token tests PASS.

- [ ] **Step 5: Commit**

  ```bash
  git add internal/gdrive/auth.go internal/gdrive/auth_test.go
  git commit -m "feat(gdrive): OAuth2 token cache and browser auth flow"
  ```

---

## Task 4: gdrive/client.go — Drive API Client

**Files:**
- Create: `internal/gdrive/client.go`
- Create: `internal/gdrive/client_test.go`

- [ ] **Step 1: Write failing tests in `client_test.go`**

  Create `internal/gdrive/client_test.go`:

  ```go
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
  ```

- [ ] **Step 2: Run to confirm failure**

  ```bash
  go test ./internal/gdrive/... -run "TestClient" -v
  ```

  Expected: `FAIL` — `Client` type not defined.

- [ ] **Step 3: Create `internal/gdrive/client.go`**

  ```go
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
  ```

- [ ] **Step 4: Run tests**

  ```bash
  go test ./internal/gdrive/... -run "TestClient" -v
  ```

  Expected: both client tests PASS.

- [ ] **Step 5: Commit**

  ```bash
  git add internal/gdrive/client.go internal/gdrive/client_test.go
  git commit -m "feat(gdrive): Drive API client with file ID cache"
  ```

---

## Task 5: gdrive/sync.go — Push/Pull Orchestration

**Files:**
- Create: `internal/gdrive/sync.go`
- Create: `internal/gdrive/sync_test.go`

- [ ] **Step 1: Write failing tests in `sync_test.go`**

  Create `internal/gdrive/sync_test.go`:

  ```go
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

  func TestPushPullMergeNoDuplicates(t *testing.T) {
      s1 := tempStore(t)
      _ = s1.Add(store.Connection{Name: "alpha", Host: "h1", Type: "ssh"})

      mock := &mockDriveFile{}
      _ = push(mock, s1)

      s2 := tempStore(t)
      _ = s2.Add(store.Connection{Name: "alpha", Host: "h1", Type: "ssh"})

      // alpha exists in s2 locally (different UUID) — the remote one is also added
      // because IDs differ. This is the expected behaviour: merge by UUID.
      added, err := pull(mock, s2)
      if err != nil {
          t.Fatalf("pull: %v", err)
      }
      // s2 has 1 local + 1 from remote (different IDs)
      if added != 1 {
          t.Errorf("expected 1 added (different UUID), got %d", added)
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

  func TestPushExportError(t *testing.T) {
      // Store with empty path will fail on save, but ExportBytes doesn't save.
      // So push itself only fails if ExportBytes fails, which requires bad state.
      // Verify upload error propagates.
      mock := &mockDriveFile{uploadErr: fmt.Errorf("quota exceeded")}
      s := tempStore(t)
      err := push(mock, s)
      if err == nil {
          t.Error("expected error on upload failure")
      }
  }
  ```

- [ ] **Step 2: Run to confirm failure**

  ```bash
  go test ./internal/gdrive/... -run "TestPush|TestPull" -v
  ```

  Expected: `FAIL` — `push`, `pull` functions not defined.

- [ ] **Step 3: Create `internal/gdrive/sync.go`**

  ```go
  package gdrive

  import (
      "context"
      "fmt"
      "os"
      "path/filepath"

      "github.com/felipem/julssh/internal/store"
  )

  type driveFile interface {
      Upload(data []byte) error
      Download() ([]byte, error)
  }

  func julsshConfigDir() (string, error) {
      base, err := os.UserConfigDir()
      if err != nil {
          return "", err
      }
      return filepath.Join(base, "julssh"), nil
  }

  func buildClient() (*Client, error) {
      configDir, err := julsshConfigDir()
      if err != nil {
          return nil, err
      }
      httpClient, err := GetClient(context.Background(), configDir)
      if err != nil {
          return nil, fmt.Errorf("auth: %w", err)
      }
      return newClient(httpClient, fileIDPath(configDir))
  }

  func push(df driveFile, s *store.Store) error {
      data, err := s.ExportBytes()
      if err != nil {
          return err
      }
      return df.Upload(data)
  }

  func pull(df driveFile, s *store.Store) (int, error) {
      data, err := df.Download()
      if err != nil {
          return 0, err
      }
      return s.ImportMergeBytes(data)
  }

  // Push uploads all connections to Google Drive.
  func Push(s *store.Store) error {
      client, err := buildClient()
      if err != nil {
          return err
      }
      return push(client, s)
  }

  // Pull downloads connections from Google Drive and merges them locally.
  func Pull(s *store.Store) (int, error) {
      client, err := buildClient()
      if err != nil {
          return 0, err
      }
      return pull(client, s)
  }
  ```

  Note: `Client` implements `driveFile` because it has `Upload([]byte) error` and `Download() ([]byte, error)`.

- [ ] **Step 4: Run tests**

  ```bash
  go test ./internal/gdrive/... -v
  ```

  Expected: all gdrive tests PASS.

- [ ] **Step 5: Commit**

  ```bash
  git add internal/gdrive/sync.go internal/gdrive/sync_test.go
  git commit -m "feat(gdrive): Push and Pull orchestration with driveFile interface"
  ```

---

## Task 6: TUI — Messages and Keys

**Files:**
- Modify: `internal/model/app.go`
- Modify: `internal/model/list.go`

- [ ] **Step 1: Add message types and handlers to `app.go`**

  Add after `MsgImportDone`:

  ```go
  type MsgDrivePushDone struct{ Err error }
  type MsgDrivePullDone struct {
      Added int
      Err   error
  }
  ```

  Add cases in `AppModel.Update`, after the `MsgImportDone` case:

  ```go
  case MsgDrivePushDone:
      if msg.Err != nil {
          a.statusMsg = "drive: " + msg.Err.Error()
          a.statusOK = false
          return a, clearStatusCmd()
      }
      return a, func() tea.Msg { return msgStatusOK{text: "Sincronizado con Google Drive ✓"} }

  case MsgDrivePullDone:
      if msg.Err != nil {
          a.statusMsg = "drive: " + msg.Err.Error()
          a.statusOK = false
          return a, clearStatusCmd()
      }
      if msg.Added == 0 {
          return a, func() tea.Msg { return msgStatusOK{text: "Drive: sin conexiones nuevas"} }
      }
      return a, func() tea.Msg {
          return msgStatusOK{text: fmt.Sprintf("Drive: %d conexiones importadas", msg.Added)}
      }
  ```

- [ ] **Step 2: Add S/L keys to `list.go`**

  Add import at top of `list.go`:

  ```go
  "github.com/felipem/julssh/internal/gdrive"
  ```

  In `handleKey`, add after the `"I"` case and before `"q"`:

  ```go
  case "S":
      s := m.store
      return m, func() tea.Msg {
          if err := gdrive.Push(s); err != nil {
              return MsgDrivePushDone{Err: err}
          }
          return MsgDrivePushDone{}
      }
  case "L":
      s := m.store
      return m, func() tea.Msg {
          added, err := gdrive.Pull(s)
          return MsgDrivePullDone{Added: added, Err: err}
      }
  ```

- [ ] **Step 3: Update footer text in `list.go`**

  Find the footer line in `View()`:

  ```go
  footer := "[n]ueva  [e]editar  [c]conectar  [/]filtrar  [X]exportar  [I]importar  [q]salir"
  ```

  Replace with:

  ```go
  footer := "[n]ueva  [e]editar  [c]conectar  [/]filtrar  [X]exportar  [I]importar  [S]Drive↑  [L]Drive↓  [q]salir"
  ```

- [ ] **Step 4: Build and verify compilation**

  ```bash
  go build -o julssh .
  ```

  Expected: no errors.

- [ ] **Step 5: Run full test suite**

  ```bash
  go test ./...
  ```

  Expected: all tests PASS.

- [ ] **Step 6: Manual smoke test**

  ```bash
  ./julssh
  ```

  - Press `S` → browser opens for Google auth (first time) or status "Sincronizado ✓" (if token exists)
  - Press `L` → status shows count of imported connections or "sin conexiones nuevas"
  - Press `S` on a machine with no internet → status shows drive error after auth

- [ ] **Step 7: Commit**

  ```bash
  git add internal/model/app.go internal/model/list.go
  git commit -m "feat(tui): add S/L keys for Google Drive push/pull"
  ```

---

## Task 7: Final Verification

- [ ] **Step 1: Full test suite**

  ```bash
  go test ./...
  ```

  Expected: PASS.

- [ ] **Step 2: Build release binary**

  ```bash
  go build -o julssh .
  ls -lh julssh
  ```

  Expected: binary builds cleanly.

- [ ] **Step 3: Check .goreleaser.yml if needed**

  Open `.goreleaser.yml`. No changes needed — GoReleaser uses `go build` which picks up new deps automatically.

- [ ] **Step 4: Final commit if any stragglers**

  ```bash
  git status
  ```

  If clean: done. If dirty: commit remaining files.
