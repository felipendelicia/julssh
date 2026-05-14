# Themes & Fullscreen Layout — Design Spec

**Date:** 2026-05-14
**Version:** v0.8
**Status:** Approved

## Summary

Two related visual improvements: (1) a 5-theme system selectable from a dedicated `T` view with live preview, saved to `~/.config/julssh/config.json`; (2) all views fill the full terminal height so the footer sits at the bottom.

## Architecture

```
internal/styles/
└── theme.go         — Theme struct, 5 palettes, Active pointer, SetTheme(), style functions

internal/config/
└── config.go        — Load/Save config.json (theme key)

internal/model/
└── theme_view.go    — ThemeViewModel: list + live preview, ↑↓ Enter Esc
```

**Modified files:**
- `internal/styles/theme.go` — full rewrite: Theme struct, 5 palettes, style functions replacing style vars
- `internal/model/list.go` — add `T` key, fullscreen View()
- `internal/model/detail.go` — fullscreen View()
- `internal/model/form.go` — fullscreen View()
- `internal/model/gdrive_view.go` — fullscreen View()
- `internal/model/app.go` — handle MsgThemeChanged
- `main.go` — load config, call styles.SetTheme() before TUI

## internal/styles/theme.go

### Theme struct

```go
type Theme struct {
    Name      string
    Primary   lipgloss.Color
    Secondary lipgloss.Color
    Muted     lipgloss.Color
    ErrColor  lipgloss.Color
    Selected  lipgloss.Color
    Border    lipgloss.Color
    TagBg     lipgloss.Color
}
```

### 5 Palettes

| Key | Name | Primary | Secondary | TagBg |
|-----|------|---------|-----------|-------|
| `purple` | Purple | `#7C3AED` | `#A78BFA` | `#2D1B69` |
| `evergreen` | Evergreen | `#059669` | `#34D399` | `#064E3B` |
| `midnight` | Midnight | `#3B82F6` | `#93C5FD` | `#1E3A5F` |
| `sunset` | Sunset | `#EA580C` | `#FB923C` | `#431407` |
| `mono` | Mono | `#E5E5E5` | `#A3A3A3` | `#262626` |

All themes share: `Muted: #6B7280`, `ErrColor: #EF4444`.

### Active and SetTheme

```go
var Active *Theme  // set at startup via SetTheme()

func SetTheme(name string) {
    t, ok := Themes[name]
    if !ok {
        t = Themes["purple"]
    }
    Active = t
}

// ThemeOrder defines display order for the selector
var ThemeOrder = []string{"purple", "evergreen", "midnight", "sunset", "mono"}
```

### Style functions (replace style vars)

Every consumer calls `styles.Title()` instead of `styles.Title`. Each function reads `Active` at call time so theme changes take effect immediately on next render.

```go
func Title() lipgloss.Style       { return lipgloss.NewStyle().Foreground(Active.Primary).Bold(true).MarginBottom(1) }
func SelectedRow() lipgloss.Style { return lipgloss.NewStyle().Background(Active.Selected).Foreground(Active.Secondary).Bold(true) }
func NormalRow() lipgloss.Style   { return lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB")) }
func MutedText() lipgloss.Style   { return lipgloss.NewStyle().Foreground(Active.Muted) }
func ErrText() lipgloss.Style     { return lipgloss.NewStyle().Foreground(Active.ErrColor) }
func Tag() lipgloss.Style         { return lipgloss.NewStyle().Foreground(Active.Secondary).Background(Active.TagBg).Padding(0, 1) }
func Footer() lipgloss.Style      { return lipgloss.NewStyle().Foreground(Active.Muted).BorderTop(true).BorderStyle(lipgloss.NormalBorder()).BorderForeground(Active.Border).MarginTop(1).PaddingTop(1) }
func FieldLabel() lipgloss.Style  { return lipgloss.NewStyle().Foreground(Active.Secondary).Width(15) }
func ActiveInput() lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color("#F9FAFB")).Background(Active.Selected).Padding(0, 1) }
func InactiveInput() lipgloss.Style { return lipgloss.NewStyle().Foreground(Active.Muted).Padding(0, 1) }
func InlineErr() lipgloss.Style   { return lipgloss.NewStyle().Foreground(Active.ErrColor).MarginLeft(15) }
```

## internal/config/config.go

```go
type Config struct {
    Theme string `json:"theme"`
}

func Load(configDir string) (*Config, error)   // reads ~/.config/julssh/config.json
func Save(configDir string, cfg *Config) error  // writes ~/.config/julssh/config.json
```

Default when file missing or theme invalid: `"purple"`.

## internal/model/theme_view.go

### Layout

Left column (fixed ~16 chars): theme list with `▶` cursor on active.
Right column: live preview showing title, two fake connection rows (one selected), footer with keys.

```
Temas                    │ PREVIEW
▶ Purple                 │ julssh :: jd
  Evergreen              │
  Midnight               │ > prod-server    [tag]
  Sunset                 │   dev-box
  Mono                   │
                         │ [S]↑ [L]↓ [G]Drive [q]salir
─────────────────────────────────────────────────────
[↑↓] navegar  [Enter] aplicar  [Esc] volver sin cambiar
```

### Messages

```go
type MsgThemeChanged struct{ Name string }  // sent on Enter, handled in app.go to save config
```

### Keys

| Key | Action |
|-----|--------|
| `↑` / `k` | Previous theme (wraps) |
| `↓` / `j` | Next theme (wraps) |
| `Enter` | Apply theme, emit MsgThemeChanged, pop view |
| `Esc` / `q` | Restore original theme, pop view |

**On navigate:** calls `styles.SetTheme(name)` immediately for live preview. On `Esc`, restores the theme that was active when the view opened.

### State

```go
type ThemeViewModel struct {
    cursor   int     // index into ThemeOrder
    original string  // theme name when view opened (for Esc restore)
    width    int
    height   int
}
```

## main.go changes

```go
import "github.com/felipem/julssh/internal/config"

cfg, _ := config.Load(configDir)   // errors → default purple
styles.SetTheme(cfg.Theme)

// then start TUI as before
```

## app.go changes

Handle `MsgThemeChanged`: save config via `config.Save`. On error → status bar error.

```go
case MsgThemeChanged:
    cfg := &config.Config{Theme: msg.Name}
    if err := config.Save(a.configDir, cfg); err != nil {
        a.statusMsg = "config: " + err.Error()
        a.statusOK = false
        return a, clearStatusCmd()
    }
    return a, nil
```

`AppModel` gains a `configDir string` field. `NewApp` signature changes to `NewApp(s *store.Store, configDir string)`. `main.go` already has `configDir` available and passes it in.

## list.go changes

Add `T` key:
```go
case "T":
    view := NewThemeView(m.width, m.height)
    return m, func() tea.Msg { return MsgPushView{View: view} }
```

Update footer to include `[T]tema`.

## Fullscreen Layout

All `View()` methods fill terminal height. Pattern:

```go
func (m SomeModel) View() string {
    var b strings.Builder
    // ... render content into b ...
    content := b.String()
    contentLines := strings.Count(content, "\n") + 1
    footerLines := 3  // border + padding + text (varies per view)
    padding := m.height - contentLines - footerLines
    if padding > 0 {
        content += strings.Repeat("\n", padding)
    }
    content += styles.Footer().Render(footerText)
    return content
}
```

Applied to: `list.go`, `detail.go`, `form.go`, `gdrive_view.go`, `theme_view.go`.

Each view already stores `width` and `height` via `WindowSizeMsg` — no model changes needed.

## Error Handling

| Situation | Behavior |
|-----------|----------|
| config.json missing | Default purple, no error |
| Unknown theme in config | Default purple, no error |
| config.Save fails | Status bar error in app.go, theme stays applied in memory |

## Testing

`internal/config/config_test.go`:
- `TestLoadMissingFile` — missing file → default purple config
- `TestLoadInvalidTheme` — unknown theme name → Load returns it as-is (validation is in SetTheme)
- `TestSaveAndLoad` — save then load → same values

`internal/styles/theme_test.go`:
- `TestSetThemeKnown` — known name → Active matches expected palette
- `TestSetThemeUnknown` — unknown name → falls back to purple
- `TestSetThemeDefault` — empty string → falls back to purple

`theme_view.go` — no unit tests. Verified manually.

## Out of Scope

- Custom color picker (user-defined palette)
- Light mode
- Per-connection theme overrides
- Theme import/export
