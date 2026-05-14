# Google Drive Sync — Design Spec

**Date:** 2026-05-14  
**Version:** v0.5  
**Status:** Approved

## Summary

Sync `connections.json` with Google Drive. Manual push/pull via keyboard shortcuts. OAuth2 browser flow for auth. Merge-by-ID conflict resolution (same strategy as existing `ImportMerge`).

## Architecture

```
internal/
└── gdrive/
    ├── auth.go      — OAuth2 flow, token cache in ~/.config/julssh/gdrive-token.json
    ├── client.go    — Drive API client, upload/download file
    └── sync.go      — push/pull logic, wraps store.ExportAll / store.ImportMerge
```

### Credentials

- OAuth2 client ID + secret embedded as constants in `auth.go` (compiled into binary)
- Desktop App type — client secret exposure is acceptable per Google's installed-app OAuth spec
- User token stored at `~/.config/julssh/gdrive-token.json`, permissions 0600
- Drive file ID cached at `~/.config/julssh/gdrive-file-id` to avoid searching on each sync

### OAuth Scope

`https://www.googleapis.com/auth/drive.file` — minimal, julssh can only access files it created.

### Drive File

`julssh-connections.json` in Drive root. Visible to user, portable, manually accessible.

## Data Flow

### First auth (no token present)

1. User presses `S` or `L` in list view
2. Status bar: `"Abriendo browser para autenticar con Google..."`
3. julssh starts local HTTP server on random port, opens browser to Google consent URL
4. User authorizes → Google redirects to `localhost:PORT` with auth code
5. julssh exchanges code for token, saves to `gdrive-token.json`
6. Continues with push or pull

### Push (`S`)

1. `store.ExportAll` → serialize current connections to JSON bytes
2. If `gdrive-file-id` exists: update file via Drive API (PATCH)
3. If not: create new file, save returned ID to `gdrive-file-id`
4. Status bar: `"Sincronizado con Google Drive ✓"`

### Pull (`L`)

1. Read file ID from `gdrive-file-id`; if missing, search Drive for `julssh-connections.json`
2. Download file content from Drive
3. `store.ImportMerge` with downloaded content → merge by UUID, local wins on conflict
4. Status bar: `"X conexiones importadas desde Drive"` (or `"Sin conexiones nuevas"` if 0)

## Keybindings

Added to list view. No conflict with existing keys.

| Key | Action |
|-----|--------|
| `S` | Push → Drive |
| `L` | Pull ← Drive |

## Error Handling

| Situation | Behavior |
|-----------|----------|
| No internet / network error | Status bar error, local store untouched |
| Token expired | Auto-refresh via oauth2 library (transparent) |
| Token revoked by user | Delete `gdrive-token.json`, re-trigger auth flow on next sync |
| Drive file deleted remotely | Pull: show "no remote file found"; Push: creates new file |
| Drive quota exceeded | Status bar: `"Error: Drive sin espacio"` |
| Auth cancelled by user | Status bar: `"Autenticación cancelada"`, no-op |

## Dependencies

```
golang.org/x/oauth2
google.golang.org/api/drive/v3
```

## Testing

- `gdrive/auth_test.go` — token read/write/refresh from cache file
- `gdrive/client_test.go` — mock HTTP server simulates Drive API responses
- `gdrive/sync_test.go` — push/pull against temporary store, verify merge semantics

No real integration tests (require live credentials). Unit tests with mocks are sufficient.

## Out of Scope

- Auto-sync on open/close
- Conflict UI (diff display)
- Multiple Drive accounts
- Sync to Sheets or other Google services
