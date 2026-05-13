# julssh

TUI SSH connection manager para Linux. Dedicado a Julieta.

## Stack

- **Lenguaje:** Go
- **TUI:** [Bubble Tea](https://github.com/charmbracelet/bubbletea) (arquitectura Elm, arrow keys nativas)
- **Estilos:** [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- **Almacenamiento:** JSON en `~/.config/julssh/connections.json`
- **Distribución:** binario único, `go install` o descarga directa

## Modelo de datos

```go
type Connection struct {
    ID          string   `json:"id"`          // UUID
    Name        string   `json:"name"`        // nombre legible
    Host        string   `json:"host"`        // hostname o IP
    Port        int      `json:"port"`        // default 22
    User        string   `json:"user"`        // usuario SSH
    IdentityFile string  `json:"identity_file"` // path a clave privada (opcional)
    Description string   `json:"description"` // texto libre
    Tags        []string `json:"tags"`        // etiquetas para filtrar
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

## Arquitectura TUI (Bubble Tea)

```
main.go
├── model/
│   ├── app.go        — estado global, router de vistas
│   └── connection.go — CRUD de conexiones, persistencia JSON
├── view/
│   ├── list.go       — lista navegable (flechas + enter)
│   ├── form.go       — formulario agregar/editar conexión
│   └── detail.go     — detalle de una conexión
└── styles/
    └── theme.go      — colores y estilos lipgloss
```

## Vistas

1. **Lista** — vista principal, flechas arriba/abajo navegan, enter abre detalle
2. **Detalle** — muestra datos completos, teclas para editar/conectar/borrar
3. **Formulario** — agregar o editar conexión con campos tab-navigables

## Keybindings (convención)

| Tecla | Acción |
|-------|--------|
| `↑/↓` o `j/k` | navegar lista |
| `Enter` | abrir detalle / confirmar |
| `n` | nueva conexión |
| `e` | editar conexión seleccionada |
| `d` | borrar (con confirmación) |
| `c` | conectar (lanza `ssh`) |
| `/` | filtrar por nombre o tag |
| `q` / `Esc` | volver / salir |

## Features iniciales (v0.1)

- [ ] CRUD completo de conexiones
- [ ] Lista navegable con nombre + host + tags visibles
- [ ] Formulario con validación básica (host requerido)
- [ ] Filtro por texto (nombre o tag)
- [ ] Lanzar conexión SSH via `exec.Command("ssh", ...)`
- [ ] Persistencia JSON automática

## Convenciones

- Caveman mode activo en comunicación
- Sin comentarios obvios en código; solo WHY no-obvios
- Sin features extra fuera del scope acordado
- `go test ./...` antes de marcar tarea completa
- Binario: `julssh`
- No agregar "Co-Authored-By: Claude" ni ninguna firma de AI en los commits

## Roadmap

### v0.2 — Distribución
- GitHub Releases con binarios pre-compilados (amd64/arm64/darwin) via GoReleaser
- Paquete `.deb` para Ubuntu/Debian (incluido en el release)
- Actualizar `install.sh` para descargar binario del release en vez de compilar

### Backlog (a priorizar)
- Importar conexiones desde `~/.ssh/config`
- Grupos/carpetas para organizar conexiones
- Jump hosts / bastion support
- Port forwarding por conexión
- Historial: última conexión por entry
- Export/import JSON (backup y sync entre máquinas)

## Instalación dev

```bash
go mod init github.com/felipem/julssh
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/lipgloss
go get github.com/google/uuid
go build -o julssh .
```
