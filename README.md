# julssh

Administrador de conexiones SSH para la terminal. Navegable con flechas y Enter.

*Dedicado a mi hermana Julieta.*

---

## Features

- Guardá conexiones SSH con nombre, usuario, puerto, clave privada, descripción y tags
- Navegá con `↑/↓` o `j/k`, abrí con `Enter`
- Filtrá por cualquier campo en tiempo real con `/`
- Conectate con `c` — el TUI se suspende, SSH toma el terminal, al salir volvés a julssh
- CRUD completo: agregar, editar, borrar con confirmación
- Datos guardados en `~/.config/julssh/connections.json`

## Instalación

### Sin Go (recomendado)

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/felipendelicia/julssh/main/install.sh) --download
```

Descarga el binario pre-compilado del último [GitHub Release](https://github.com/felipendelicia/julssh/releases). No requiere Go instalado.

### Desde fuente

Requiere Go 1.21+.

```bash
git clone https://github.com/felipendelicia/julssh.git
cd julssh
bash install.sh
```

### Desde .deb (Debian/Ubuntu amd64)

Descargá el `.deb` del último [release](https://github.com/felipendelicia/julssh/releases) e instalá:

```bash
sudo dpkg -i julssh_amd64.deb
```

Instala en `/usr/local/bin/julssh`.

---

Por defecto instala en `~/.local/bin/`. Para instalación global:

```bash
sudo bash install.sh /usr/local/bin
```

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
