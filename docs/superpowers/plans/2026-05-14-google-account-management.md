# Google Account Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a dedicated Google Drive management view (key `G`) showing account status, email, name, and login/logout/sync actions.

**Architecture:** New `internal/gdrive/account.go` exposes `IsLoggedIn`, `GetUserInfo`, `Logout`, `Login` using internal helpers testable with temp dirs. New `internal/model/gdrive_view.go` is a Bubble Tea model with three states (loading/connected/disconnected). `app.go` is updated to call `Init()` on pushed views so DriveView can trigger async user-info load on open. `list.go` gets the `G` key.

**Tech Stack:** Go, Bubble Tea, `google.golang.org/api/drive/v3` (about.get endpoint), existing `internal/gdrive` auth/client.

---

## File Map

| Action | Path | Responsibility |
|--------|------|----------------|
| Create | `internal/gdrive/account.go` | UserInfo struct, IsLoggedIn, GetUserInfo, Logout, Login |
| Create | `internal/gdrive/account_test.go` | Tests for internal helpers using temp dirs |
| Create | `internal/model/gdrive_view.go` | DriveView Bubble Tea model (loading/connected/disconnected) |
| Modify | `internal/model/app.go` | Call `Init()` on MsgPushView so views can start async work |
| Modify | `internal/model/list.go` | Add `G` key, update footer |

---

## Task 1: gdrive/account.go — Account helpers

**Files:**
- Create: `internal/gdrive/account.go`
- Create: `internal/gdrive/account_test.go`

- [ ] **Step 1: Write failing tests in `account_test.go`**

  Create `internal/gdrive/account_test.go`:

  ```go
  package gdrive

  import (
      "os"
      "path/filepath"
      "testing"
      "time"

      "golang.org/x/oauth2"
  )

  func TestIsLoggedInFalse(t *testing.T) {
      dir := t.TempDir()
      if isLoggedIn(dir) {
          t.Error("expected false when no token file exists")
      }
  }

  func TestIsLoggedInTrue(t *testing.T) {
      dir := t.TempDir()
      token := &oauth2.Token{
          AccessToken:  "test-access",
          RefreshToken: "test-refresh",
          Expiry:       time.Now().Add(time.Hour),
      }
      if err := saveToken(tokenPath(dir), token); err != nil {
          t.Fatal(err)
      }
      if !isLoggedIn(dir) {
          t.Error("expected true when token file exists")
      }
  }

  func TestLogoutDeletesBothFiles(t *testing.T) {
      dir := t.TempDir()

      token := &oauth2.Token{AccessToken: "test", RefreshToken: "r"}
      if err := saveToken(tokenPath(dir), token); err != nil {
          t.Fatal(err)
      }
      if err := os.WriteFile(fileIDPath(dir), []byte("file-123"), 0600); err != nil {
          t.Fatal(err)
      }

      if err := logout(dir); err != nil {
          t.Fatalf("logout: %v", err)
      }

      if _, err := os.Stat(tokenPath(dir)); !os.IsNotExist(err) {
          t.Error("expected token file to be deleted")
      }
      if _, err := os.Stat(fileIDPath(dir)); !os.IsNotExist(err) {
          t.Error("expected file ID to be deleted")
      }
  }

  func TestLogoutIdempotent(t *testing.T) {
      dir := t.TempDir()
      if err := logout(dir); err != nil {
          t.Errorf("logout on empty dir must not error: %v", err)
      }
  }

  func TestLogoutPartialState(t *testing.T) {
      dir := t.TempDir()
      token := &oauth2.Token{AccessToken: "test", RefreshToken: "r"}
      if err := saveToken(tokenPath(dir), token); err != nil {
          t.Fatal(err)
      }
      // No file ID file — logout must still succeed.
      if err := logout(dir); err != nil {
          t.Errorf("logout with only token file must not error: %v", err)
      }
      if _, err := os.Stat(tokenPath(dir)); !os.IsNotExist(err) {
          t.Error("expected token file to be deleted")
      }
  }

  func TestUserInfoPathHelper(t *testing.T) {
      p := filepath.Join("/tmp/julssh", "gdrive-token.json")
      if tokenPath("/tmp/julssh") != p {
          t.Errorf("unexpected token path: %q", tokenPath("/tmp/julssh"))
      }
  }
  ```

- [ ] **Step 2: Run to confirm failure**

  ```bash
  cd /home/felipe/Documents/Repositories/julssh
  go test ./internal/gdrive/... -run "TestIsLoggedIn|TestLogout|TestUserInfo" -v
  ```

  Expected: `FAIL` — `isLoggedIn`, `logout` not defined.

- [ ] **Step 3: Create `internal/gdrive/account.go`**

  ```go
  package gdrive

  import (
      "context"
      "fmt"
      "os"

      "google.golang.org/api/drive/v3"
      "google.golang.org/api/option"
  )

  // UserInfo holds the display name and email of the authenticated Google account.
  type UserInfo struct {
      Email string
      Name  string
  }

  func isLoggedIn(configDir string) bool {
      _, err := tokenFromFile(tokenPath(configDir))
      return err == nil
  }

  func getUserInfo(configDir string) (*UserInfo, error) {
      httpClient, err := GetClient(context.Background(), configDir)
      if err != nil {
          return nil, fmt.Errorf("auth: %w", err)
      }
      svc, err := drive.NewService(context.Background(), option.WithHTTPClient(httpClient))
      if err != nil {
          return nil, fmt.Errorf("drive service: %w", err)
      }
      about, err := svc.About.Get().Fields("user").Do()
      if err != nil {
          return nil, fmt.Errorf("drive about: %w", err)
      }
      return &UserInfo{
          Email: about.User.EmailAddress,
          Name:  about.User.DisplayName,
      }, nil
  }

  func logout(configDir string) error {
      if err := os.Remove(tokenPath(configDir)); err != nil && !os.IsNotExist(err) {
          return fmt.Errorf("remove token: %w", err)
      }
      if err := os.Remove(fileIDPath(configDir)); err != nil && !os.IsNotExist(err) {
          return fmt.Errorf("remove file ID: %w", err)
      }
      return nil
  }

  func login(configDir string) error {
      _, err := GetClient(context.Background(), configDir)
      return err
  }

  // IsLoggedIn reports whether a cached token exists for the current user.
  func IsLoggedIn() bool {
      dir, err := julsshConfigDir()
      if err != nil {
          return false
      }
      return isLoggedIn(dir)
  }

  // GetUserInfo returns the Google account email and display name.
  func GetUserInfo() (*UserInfo, error) {
      dir, err := julsshConfigDir()
      if err != nil {
          return nil, err
      }
      return getUserInfo(dir)
  }

  // Logout deletes the cached token and Drive file ID.
  func Logout() error {
      dir, err := julsshConfigDir()
      if err != nil {
          return err
      }
      return logout(dir)
  }

  // Login triggers the browser OAuth flow and caches the resulting token.
  func Login() error {
      dir, err := julsshConfigDir()
      if err != nil {
          return err
      }
      return login(dir)
  }
  ```

- [ ] **Step 4: Run tests**

  ```bash
  go test ./internal/gdrive/... -v
  ```

  Expected: all 15 tests PASS (9 existing + 6 new).

- [ ] **Step 5: Commit**

  ```bash
  git add internal/gdrive/account.go internal/gdrive/account_test.go
  git commit -m "feat(gdrive): add IsLoggedIn, GetUserInfo, Logout, Login"
  ```

---

## Task 2: model/gdrive_view.go — Drive management view

**Files:**
- Create: `internal/model/gdrive_view.go`

- [ ] **Step 1: Create `internal/model/gdrive_view.go`**

  No TDD here — Bubble Tea views are tested manually. Implement directly.

  ```go
  package model

  import (
      "strings"

      tea "github.com/charmbracelet/bubbletea"
      "github.com/felipem/julssh/internal/gdrive"
      "github.com/felipem/julssh/internal/store"
      "github.com/felipem/julssh/internal/styles"
  )

  type driveViewState int

  const (
      driveStateLoading      driveViewState = iota
      driveStateConnected
      driveStateDisconnected
  )

  // msgDriveUserInfo carries the result of an async user-info load.
  type msgDriveUserInfo struct {
      info *gdrive.UserInfo
      err  error
  }

  // msgDriveLogoutDone carries the result of an async logout.
  type msgDriveLogoutDone struct{ err error }

  // msgDriveLoginDone carries the result of an async login.
  type msgDriveLoginDone struct{ err error }

  // DriveViewModel is the Google Drive management screen.
  type DriveViewModel struct {
      store   *store.Store
      state   driveViewState
      info    *gdrive.UserInfo
      infoErr string
  }

  func NewDriveView(s *store.Store) DriveViewModel {
      return DriveViewModel{store: s, state: driveStateLoading}
  }

  // Init fires an async cmd to check login state and load user info.
  func (m DriveViewModel) Init() tea.Cmd {
      return func() tea.Msg {
          if !gdrive.IsLoggedIn() {
              return msgDriveUserInfo{}
          }
          info, err := gdrive.GetUserInfo()
          return msgDriveUserInfo{info: info, err: err}
      }
  }

  func (m DriveViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
      switch msg := msg.(type) {
      case tea.WindowSizeMsg:
          return m, nil

      case msgDriveUserInfo:
          if msg.info == nil && msg.err == nil {
              m.state = driveStateDisconnected
              return m, nil
          }
          m.state = driveStateConnected
          m.info = msg.info
          if msg.err != nil {
              m.infoErr = msg.err.Error()
          }
          return m, nil

      case msgDriveLogoutDone:
          if msg.err != nil {
              return m, func() tea.Msg { return MsgError{Err: msg.err} }
          }
          m.state = driveStateDisconnected
          m.info = nil
          m.infoErr = ""
          return m, nil

      case msgDriveLoginDone:
          if msg.err != nil {
              return m, func() tea.Msg { return MsgError{Err: msg.err} }
          }
          m.state = driveStateLoading
          return m, m.Init()

      case tea.KeyMsg:
          return m.handleKey(msg)
      }
      return m, nil
  }

  func (m DriveViewModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
      switch msg.String() {
      case "esc", "q":
          return m, func() tea.Msg { return MsgPopView{} }

      case "S":
          if m.state != driveStateConnected {
              return m, nil
          }
          s := m.store
          return m, func() tea.Msg {
              if err := gdrive.Push(s); err != nil {
                  return MsgDrivePushDone{Err: err}
              }
              return MsgDrivePushDone{}
          }

      case "L":
          switch m.state {
          case driveStateDisconnected:
              return m, func() tea.Msg {
                  if err := gdrive.Login(); err != nil {
                      return msgDriveLoginDone{err: err}
                  }
                  return msgDriveLoginDone{}
              }
          case driveStateConnected:
              s := m.store
              return m, func() tea.Msg {
                  added, err := gdrive.Pull(s)
                  return MsgDrivePullDone{Added: added, Err: err}
              }
          }

      case "O":
          if m.state != driveStateConnected {
              return m, nil
          }
          return m, func() tea.Msg {
              if err := gdrive.Logout(); err != nil {
                  return msgDriveLogoutDone{err: err}
              }
              return msgDriveLogoutDone{}
          }
      }
      return m, nil
  }

  func (m DriveViewModel) View() string {
      var b strings.Builder

      b.WriteString(styles.Title.Render("Google Drive"))
      b.WriteString("\n\n")

      switch m.state {
      case driveStateLoading:
          b.WriteString(styles.MutedText.Render("  Cargando...") + "\n")

      case driveStateDisconnected:
          b.WriteString(styles.FieldLabel.Render("Estado:"))
          b.WriteString("  " + styles.ErrText.Render("Desconectado") + "\n\n")
          b.WriteString(styles.Footer.Render("[L] Iniciar sesión con Google  [Esc] volver"))

      case driveStateConnected:
          b.WriteString(styles.FieldLabel.Render("Estado:"))
          b.WriteString("  " + styles.MutedText.Render("Conectado") + "\n")
          if m.info != nil {
              b.WriteString(styles.FieldLabel.Render("Cuenta:"))
              b.WriteString("  " + m.info.Email + "\n")
              b.WriteString(styles.FieldLabel.Render("Nombre:"))
              b.WriteString("  " + m.info.Name + "\n")
          } else if m.infoErr != "" {
              b.WriteString(styles.ErrText.Render("  Error al cargar info: "+m.infoErr) + "\n")
          }
          b.WriteString("\n")
          b.WriteString(styles.Footer.Render("[S] Subir conexiones  [L] Bajar conexiones  [O] Desloguear  [Esc] volver"))
      }

      return b.String()
  }
  ```

- [ ] **Step 2: Build to verify compilation**

  ```bash
  cd /home/felipe/Documents/Repositories/julssh && go build ./...
  ```

  Expected: no errors.

- [ ] **Step 3: Commit**

  ```bash
  git add internal/model/gdrive_view.go
  git commit -m "feat(model): add DriveView for Google account management"
  ```

---

## Task 3: Wiring — app.go + list.go

**Files:**
- Modify: `internal/model/app.go` — call `Init()` when pushing a view
- Modify: `internal/model/list.go` — add `G` key, update footer

- [ ] **Step 1: Update `MsgPushView` handler in `app.go` to call `Init()`**

  In `app.go`, find the `MsgPushView` case:

  ```go
  case MsgPushView:
      a.stack = append(a.stack, msg.View)
      return a, nil
  ```

  Replace with:

  ```go
  case MsgPushView:
      a.stack = append(a.stack, msg.View)
      return a, msg.View.Init()
  ```

  This calls `Init()` on every pushed view. Existing views (`ListModel`, `DetailModel`, `FormModel`, `InstallView`) all return `nil` from `Init()` so this is a no-op for them. `DriveViewModel.Init()` returns the user-info load cmd.

- [ ] **Step 2: Add `G` key in `list.go`**

  In `handleKey`, add after the `"L"` case (Drive pull) and before `"q"`:

  ```go
  case "G":
      view := NewDriveView(m.store)
      return m, func() tea.Msg { return MsgPushView{View: view} }
  ```

- [ ] **Step 3: Update footer in `list.go`**

  Find:

  ```go
  footer := "[n]ueva  [e]editar  [c]conectar  [/]filtrar  [X]exportar  [I]importar  [S]Drive↑  [L]Drive↓  [q]salir"
  ```

  Replace with:

  ```go
  footer := "[n]ueva  [e]editar  [c]conectar  [/]filtrar  [X]exportar  [I]importar  [G]Drive  [S]↑  [L]↓  [q]salir"
  ```

- [ ] **Step 4: Build**

  ```bash
  cd /home/felipe/Documents/Repositories/julssh && go build -o julssh .
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

  - Press `G` → DriveView opens, shows "Cargando..." briefly then account info (or "Desconectado" if not logged in)
  - If connected: press `O` → logout → view switches to disconnected
  - If disconnected: press `L` → browser opens → after auth → view shows account info
  - Press `Esc` → back to list

- [ ] **Step 7: Commit**

  ```bash
  git add internal/model/app.go internal/model/list.go
  git commit -m "feat(tui): wire G key to DriveView, call Init() on view push"
  ```
