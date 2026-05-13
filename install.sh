#!/usr/bin/env bash
set -e

BINARY="julssh"
DEFAULT_DEST="$HOME/.local/bin"
REPO="felipendelicia/julssh"

# ── args ──────────────────────────────────────────────────────────────────────

DOWNLOAD=false
DEST="$DEFAULT_DEST"

for arg in "$@"; do
    case "$arg" in
        --download) DOWNLOAD=true ;;
        *) DEST="$arg" ;;
    esac
done

# ── checks ────────────────────────────────────────────────────────────────────

if [[ "$DEST" == "/usr/local/bin" && "$EUID" -ne 0 ]]; then
    echo "Error: instalar en /usr/local/bin requiere sudo." >&2
    echo "  Usá: sudo bash install.sh /usr/local/bin" >&2
    exit 1
fi

mkdir -p "$DEST"

# ── install ───────────────────────────────────────────────────────────────────

if $DOWNLOAD; then
    if ! command -v curl &>/dev/null; then
        echo "Error: curl no está instalado." >&2
        exit 1
    fi

    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64)          ARCH="amd64" ;;
        aarch64 | arm64) ARCH="arm64" ;;
        *)
            echo "Error: arquitectura no soportada: $ARCH" >&2
            exit 1
            ;;
    esac

    echo "Obteniendo última versión de $REPO..."
    DOWNLOAD_URL=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
        | grep "browser_download_url" \
        | grep "linux_${ARCH}\.tar\.gz" \
        | cut -d '"' -f 4)

    if [[ -z "$DOWNLOAD_URL" ]]; then
        echo "Error: no se encontró binario para linux/$ARCH en el último release." >&2
        exit 1
    fi

    echo "Descargando $DOWNLOAD_URL..."
    TMP=$(mktemp -d)
    trap 'rm -rf "$TMP"' EXIT
    curl -fsSL "$DOWNLOAD_URL" | tar -xz -C "$TMP"
    mv "$TMP/$BINARY" "$DEST/$BINARY"
    chmod +x "$DEST/$BINARY"
else
    if ! command -v go &>/dev/null; then
        echo "Error: Go no está instalado." >&2
        echo "  Instalalo desde https://go.dev/dl/" >&2
        echo "  O usá --download para instalar sin Go." >&2
        exit 1
    fi

    echo "Compilando $BINARY..."
    go build -ldflags="-s -w" -o "$DEST/$BINARY" .
fi

echo "Instalado en $DEST/$BINARY"

# ── PATH warning ──────────────────────────────────────────────────────────────

if ! echo "$PATH" | tr ':' '\n' | grep -qx "$DEST"; then
    echo ""
    echo "Advertencia: $DEST no está en tu \$PATH."
    echo "Agregá esto a tu ~/.bashrc o ~/.zshrc:"
    echo ""
    echo "  export PATH=\"$DEST:\$PATH\""
    echo ""
fi

echo "Listo. Ejecutá: $BINARY"
