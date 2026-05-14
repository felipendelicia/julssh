# Google Account Management — Design Spec

**Date:** 2026-05-14
**Version:** v0.6
**Status:** Approved

## Summary

Dedicated `G` key in the list opens a Google Drive view showing account status (connected/disconnected), email, name, and actions: push, pull, logout, login. Builds on the existing `internal/gdrive` package without adding new OAuth scopes.

## Architecture

```
internal/gdrive/
└── account.go      — IsLoggedIn, GetUserInfo, Logout, Login

internal/model/
└── gdrive_view.go  — DriveView: Bubble Tea model for the Drive management screen
```

**Modified files:**
- `internal/model/list.go` — add `G` key → push DriveView
- `internal/model/app.go` — handle MsgDriveLogoutDone, MsgDriveLoginDone, MsgDriveUserInfo

## gdrive/account.go

```go
type UserInfo struct {
    Email string
    Name  string
}

func IsLoggedIn(configDir string) bool
func GetUserInfo(configDir string) (*UserInfo, error)  // uses Drive about.get, no new scopes
func Logout(configDir string) error                    // deletes token + file ID cache
func Login(configDir string) error                     // explicit browser auth flow
```

**GetUserInfo** calls `svc.About.Get().Fields("user").Do()` using the existing authenticated client. Returns `displayName` and `emailAddress` from the Drive API response. No new OAuth scopes required — `drive.file` is sufficient.

**Logout** deletes `~/.config/julssh/gdrive-token.json` and `~/.config/julssh/gdrive-file-id`.

**Login** calls `GetClient` which triggers `browserAuth` if no token is cached. After success, the token is saved and the view fetches user info.

## model/gdrive_view.go

### States

```
loading → connected   (token exists, user info loaded)
loading → disconnected (no token)
```

### View: Connected

```
  Google Drive

  Estado:  Conectado
  Cuenta:  felipe@gmail.com
  Nombre:  Felipe Delicia

  [S] Subir conexiones
  [L] Bajar conexiones
  [O] Desloguear

  [Esc] volver
```

### View: Disconnected

```
  Google Drive

  Estado:  Desconectado

  [L] Iniciar sesión con Google

  [Esc] volver
```

### View: Loading

```
  Google Drive

  Cargando...
```

### Keybindings

| Key | State | Action |
|-----|-------|--------|
| `S` | connected | Push → Drive (MsgDrivePushDone) |
| `L` | connected | Pull ← Drive (MsgDrivePullDone) |
| `L` | disconnected | Login → browser auth, then load user info |
| `O` | connected | Logout → delete token, transition to disconnected |
| `Esc` | any | Pop view (back to list) |

### Messages

```go
type MsgDriveUserInfo struct {
    Info *gdrive.UserInfo  // nil = not logged in
    Err  error
}
type MsgDriveLogoutDone struct{ Err error }
type MsgDriveLoginDone struct{ Err error }
```

`MsgDrivePushDone` and `MsgDrivePullDone` are reused from the existing Drive sync feature.

## list.go change

Add `G` key in `handleKey`, between `I` and `S`:

```go
case "G":
    return m, func() tea.Msg { return MsgPushView{View: NewDriveView(m.store)} }
```

Update footer to include `[G]Drive`.

## app.go changes

Handle `MsgDriveLogoutDone` and `MsgDriveLoginDone` in `AppModel.Update`:
- Logout error → status bar error
- Login error → status bar error (view handles state transition internally)
- Success cases → no status bar message needed (view itself shows state)

`MsgDriveUserInfo` is handled inside `DriveView.Update` directly (not in app.go).

## Error Handling

| Situation | Behavior |
|-----------|----------|
| GetUserInfo fails (no network) | DriveView shows connected state with empty fields, S/L/O still available |
| Logout fails (file permissions) | MsgDriveLogoutDone{Err} → app status bar error |
| Login cancelled / timeout | MsgDriveLoginDone{Err} → app status bar "Autenticación cancelada", view stays disconnected |
| S/L from DriveView | Same as from list — MsgDrivePushDone/MsgDrivePullDone handled in app.go |

## Testing

`internal/gdrive/account_test.go`:
- `TestIsLoggedInFalse` — returns false when token file absent
- `TestIsLoggedInTrue` — returns true when token file exists
- `TestLogout` — creates token + file-id files, calls Logout, verifies both deleted
- `TestLoginNotCalledOnMissingDir` — IsLoggedIn on empty dir returns false without panic

`model/gdrive_view.go` — no unit tests. Bubble Tea view behavior verified manually.

## Out of Scope

- Switching between multiple Google accounts
- Showing Drive storage quota
- Token expiry timer in the UI
