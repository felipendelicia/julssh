# julssh v0.1 — Design Spec

**Date:** 2026-05-13
**Status:** Approved

## Overview

TUI SSH connection manager para Linux. Navegable con flechas y Enter. Permite guardar conexiones SSH con nombre, descripción, tags y lanzarlas como subproceso desde el TUI.

Dedicado a Julieta.

---

## Stack

| Componente | Decisión |
|-----------|---------|
| Lenguaje | Go |
| TUI framework | Bubble Tea (arquitectura Elm) |
| Estilos | Lip Gloss |
| Almacenamiento | JSON en `~/.config/julssh/connections.json` |
| Distribución | Binario único (`go build` / `go install`) |

---

## Estructura de archivos

```
julssh/
├── main.go
├── go.mod
├── internal/
│   ├── store/
│   │   ├── store.go         — load/save JSON, CRUD Connection
│   │   └── store_test.go
│   ├── model/
│   │   ├── app.go           — AppModel: router con stack de vistas
│   │   ├── list.go          — ListModel: lista navegable + filtro
│   │   ├── detail.go        — DetailModel: vista de una conexión
│   │   └── form.go          — FormModel: crear/editar conexión
│   ├── ssh/
│   │   ├── connect.go       — lanzar ssh como subproceso
│   │   └── connect_test.go
│   └── styles/
│       └── theme.go         — colores y layouts lipgloss
└── docs/
    └── superpowers/specs/
```

---

## Modelo de datos

```go
type Connection struct {
    ID           string    `json:"id"`
    Name         string    `json:"name"`
    Host         string    `json:"host"`
    Port         int       `json:"port"`          // default 22
    User         string    `json:"user"`
    IdentityFile string    `json:"identity_file"` // opcional
    Description  string    `json:"description"`   // opcional
    Tags         []string  `json:"tags"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}
```

Tags se ingresan en el formulario como texto separado por comas (`produccion, aws, backend`).

---

## Store

Archivo: `~/.config/julssh/connections.json`

- Si no existe al arrancar → se crea automáticamente vacío (sin mensaje al usuario)
- Escritura atómica: write a archivo temporal + rename
- Operaciones: `Load`, `Save`, `Add`, `Update`, `Delete`, `Filter`

**Filtro:** `Filter(query string) []Connection` — busca `query` en todos los campos string (Name, Host, User, IdentityFile, Description, Tags) via `strings.Contains` case-insensitive.

---

## Arquitectura TUI

`AppModel` mantiene un `stack []tea.Model`. Navegación:

```
Push(view)  → agrega vista al stack, se muestra la del tope
Pop()       → elimina vista del tope, vuelve a la anterior
```

Mensajes del router:
```go
type MsgPushView struct{ View tea.Model }
type MsgPopView  struct{}
type MsgConnectSSH struct{ Conn store.Connection }
```

Flujo de navegación:
```
[ListModel] --Enter--> [DetailModel] --e--> [FormModel]
     ^                      |                    |
     |                   Esc/q                 Esc/q
     +<---------------------+--------------------+
```

---

## Vistas

### ListModel

- Tabla: `[Nombre]  [user@host:port]  [tags]`
- `↑/↓` o `j/k`: navegar
- `Enter`: push DetailModel con conexión seleccionada
- `/`: activar filtro inline (filtra en tiempo real, `Esc` limpia)
- `n`: push FormModel vacío
- `q`: salir de la app

### DetailModel

- Muestra todos los campos de la conexión
- Footer con keybindings: `[c]conectar  [e]editar  [d]borrar  [Esc]volver`
- `d`: confirmación inline `"Borrar [nombre]? (s/n)"` antes de eliminar
- `c`: emite `MsgConnectSSH`
- `e`: push FormModel pre-llenado
- `Esc`: pop

### FormModel

- Campos en orden: Name, Host, Port, User, IdentityFile, Description, Tags
- `Tab` / `Shift+Tab` / `Enter`: avanzar al siguiente campo
- `Enter` en último campo (Tags): guardar y pop
- Validaciones:
  - Host: no vacío
  - Port: número entre 1 y 65535 (default 22 si vacío)
- Error inline bajo el campo inválido
- Si viene de edición: campos pre-llenados

---

## SSH como subproceso

```go
func Connect(p *tea.Program, conn store.Connection) error {
    args := buildArgs(conn)   // ["user@host", "-p", "22", ...]
    p.ReleaseTerminal()
    cmd := exec.Command("ssh", args...)
    cmd.Stdin  = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    err := cmd.Run()
    p.RestoreTerminal()
    return err
}
```

`buildArgs` agrega `-p PORT` si != 22, `-i IdentityFile` si no vacío.

---

## Manejo de errores

| Escenario | Manejo |
|-----------|--------|
| JSON corrupto al cargar | `log.Fatal` con mensaje claro, exit 1 |
| Falla al guardar JSON | Status bar con error, app sigue |
| `ssh` no encontrado en PATH | Status bar con error en DetailModel |
| `ssh` falla (exit code != 0) | Status bar con error al volver al TUI |
| Port inválido en form | Validación inline, no llega al store |

**Status bar:** línea al fondo del TUI. Errores transitorios desaparecen a los 3 segundos via `tea.Tick`.

---

## Keybindings globales

| Tecla | Acción |
|-------|--------|
| `↑/↓` o `j/k` | Navegar lista |
| `Enter` | Abrir detalle / confirmar |
| `Tab` / `Shift+Tab` | Navegar campos del form |
| `n` | Nueva conexión |
| `e` | Editar conexión seleccionada |
| `d` | Borrar (con confirmación) |
| `c` | Conectar vía SSH |
| `/` | Filtrar |
| `Esc` | Volver / cancelar filtro |
| `q` | Salir (solo desde ListModel) |

---

## Testing

- `store_test.go`: Load, Save, Add, Update, Delete, Filter con archivo temp en `t.TempDir()`
- `connect_test.go`: verifica que `buildArgs` construye args correctos para distintas combinaciones (con/sin port, con/sin identity file)
- No tests de UI para v0.1 (Bubble Tea no tiene modo headless estable)

---

## Instalación dev

```bash
go mod init github.com/felipem/julssh
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/lipgloss
go get github.com/google/uuid
go build -o julssh .
./julssh
```
