# Auto-Updater — Design Spec

**Date:** 2026-05-14
**Version:** v0.7
**Status:** Approved

## Summary

At startup, julssh checks GitHub Releases for a newer version. If found, it prompts the user ("¿Actualizar? [s/N]"), downloads the correct binary for the current architecture, replaces the running executable, and relaunches. Network errors are reported and ignored — the TUI starts regardless.

## Architecture

```
internal/updater/
└── updater.go       — Check, fetchLatest, download, selfReplace
```

**Modified files:**
- `main.go` — call `updater.Check(version)` before `tea.NewProgram`

## updater.go

```go
func Check(currentVersion string)
```

Called from `main.go`. If `currentVersion == "dev"`, returns immediately (no-op for development builds).

### Flow

```
Check(currentVersion)
  ├── currentVersion == "dev" → return
  ├── GET https://api.github.com/repos/felipendelicia/julssh/releases/latest (3s timeout)
  │     └── error → print "No se pudo verificar actualizaciones: <err>" → return
  ├── semver.Compare(latestTag, currentVersion) <= 0 → return (no update)
  └── new version available
        ├── print "Nueva versión <tag> disponible. ¿Actualizar? [s/N]"
        ├── read one line from stdin
        ├── input != "s" → return
        └── download julssh_linux_<arch>.tar.gz from release assets
              → extract julssh binary from tar
              → write to temp file in same dir as os.Executable()
              → os.Rename(temp, execPath)   (atomic on Linux)
              → syscall.Exec(execPath, os.Args, os.Environ())
              → any error → print error → return (TUI continues)
```

### Architecture detection

`runtime.GOARCH` maps to asset filename:
- `amd64` → `julssh_linux_amd64.tar.gz`
- `arm64` → `julssh_linux_arm64.tar.gz`

Asset download URL comes from `assets[].browser_download_url` in the GitHub API response.

### Version comparison

Manual semver parse: strip leading `v`, split on `.`, compare major/minor/patch as integers. No new dependencies. If the GitHub tag is not valid semver (`vX.Y.Z`), the update check is skipped silently.

### Self-replace steps

1. `os.CreateTemp` — temp file in same directory as executable
2. Download `.tar.gz` into temp file
3. Open temp file, decompress gzip, iterate tar entries until `julssh` binary found
4. Write binary to a second temp file with mode `0755`
5. `os.Rename(binaryTemp, execPath)` — atomic replace
6. `syscall.Exec(execPath, os.Args, os.Environ())` — replace current process

## main.go change

```go
import "github.com/felipem/julssh/internal/updater"

// before tea.NewProgram:
updater.Check(version)
```

## Error Handling

| Situation | Behavior |
|-----------|----------|
| No network / API timeout (3s) | Print "No se pudo verificar actualizaciones: <err>", continue |
| GitHub API returns non-200 | Print error, continue |
| Invalid semver tag | Skip silently, continue |
| Download fails | Print error, continue |
| os.Rename fails (permissions) | Print error, continue |
| User answers N / Enter / other | Continue without updating |

## Testing

`internal/updater/updater_test.go` using `httptest.NewServer` to mock GitHub API:

- `TestNoUpdateWhenDev` — `version == "dev"` → returns without HTTP call
- `TestNoUpdateWhenSameVersion` — same version tag → returns without prompting
- `TestDetectsNewerVersion` — newer tag in mock response → returns release info
- `TestAssetURLByArch` — amd64 and arm64 produce correct asset URLs
- `TestAPIError` — mock server returns 500 → error returned, no panic

Download + self-replace: verified manually (too coupled to real filesystem to test reliably).

## Out of Scope

- Windows / macOS support
- Automatic background update check (no prompt)
- Rollback on failed update
- Checksum verification of downloaded binary
