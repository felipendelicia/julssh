#!/usr/bin/env bash
set -e

BINARY="julssh"
DEFAULT_DEST="$HOME/.local/bin"

# ── checks ────────────────────────────────────────────────────────────────────

if ! command -v go &>/dev/null; then
    echo "Error: Go no está instalado. Instalalo desde https://go.dev/dl/" >&2
    exit 1
fi

# ── destino ───────────────────────────────────────────────────────────────────

DEST="${1:-$DEFAULT_DEST}"

if [[ "$DEST" == "/usr/local/bin" && "$EUID" -ne 0 ]]; then
    echo "Error: instalar en /usr/local/bin requiere sudo." >&2
    echo "  Usá: sudo bash install.sh /usr/local/bin" >&2
    exit 1
fi

mkdir -p "$DEST"

# ── build ─────────────────────────────────────────────────────────────────────

echo "Compilando $BINARY..."
go build -ldflags="-s -w" -o "$DEST/$BINARY" .
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
