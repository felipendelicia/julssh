# julssh ♥

Administrador de conexiones SSH para la terminal. Navegable con flechas y Enter.

*Dedicado a Julieta.*

---

## Features

- Guardá conexiones SSH con nombre, usuario, puerto, clave privada, descripción y tags
- Navegá con `↑/↓` o `j/k`, abrí con `Enter`
- Filtrá por cualquier campo en tiempo real con `/`
- Conectate con `c` — el TUI se suspende, SSH toma el terminal, al salir volvés a julssh
- CRUD completo: agregar, editar, borrar con confirmación
- Datos guardados en `~/.config/julssh/connections.json`

## Instalación

### Opción 1: copiar el binario (recomendada, sin dependencias)

```bash
git clone <repo>
cd julssh
go build -o julssh .
cp julssh ~/.local/bin/
```

Listo. Desde cualquier terminal:

```bash
julssh
```

### Opción 2: instalar en `/usr/local/bin` (requiere sudo)

```bash
sudo cp julssh /usr/local/bin/
```

### Opción 3: `go install`

```bash
go install github.com/felipem/julssh@latest
```

> Requiere que `~/go/bin` esté en tu `$PATH`. Agregá esto a tu `.bashrc` o `.zshrc`:
> ```bash
> export PATH="$HOME/go/bin:$PATH"
> ```

## Requisitos

- Linux
- Go 1.21+ (solo para compilar — el binario no requiere Go instalado)

## Keybindings

| Tecla | Acción |
|-------|--------|
| `↑/↓` o `j/k` | Navegar lista |
| `Enter` | Abrir detalle |
| `n` | Nueva conexión |
| `e` | Editar conexión |
| `d` | Borrar (pide confirmación) |
| `c` | Conectar vía SSH |
| `/` | Filtrar por texto |
| `Esc` | Volver / cancelar |
| `q` | Salir |

## Datos

Las conexiones se guardan en:

```
~/.config/julssh/connections.json
```

El archivo se crea automáticamente al primer uso.
