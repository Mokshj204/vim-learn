#!/usr/bin/env sh
set -eu

REPO="${VIMLEARN_REPO:-Mokshj204/vim-learn}"
BIN_NAME="vimlearn"
INSTALL_DIR="${VIMLEARN_INSTALL_DIR:-$HOME/.local/bin}"

OS_RAW="$(uname -s)"
case "$OS_RAW" in
  Linux)  OS="linux" ;;
  Darwin) OS="darwin" ;;
  *)
    echo "Unsupported OS: $OS_RAW" >&2
    exit 1
    ;;
esac

ARCH_RAW="$(uname -m)"
case "$ARCH_RAW" in
  x86_64|amd64)  ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH_RAW" >&2
    exit 1
    ;;
esac

ASSET="vimlearn-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"

mkdir -p "$INSTALL_DIR"
TMP_FILE="$(mktemp)"
trap 'rm -f "$TMP_FILE"' EXIT

echo "Downloading ${ASSET} from ${URL}..."
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$URL" -o "$TMP_FILE"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$TMP_FILE" "$URL"
else
  echo "Error: curl or wget is required." >&2
  exit 1
fi

chmod +x "$TMP_FILE"
mv "$TMP_FILE" "${INSTALL_DIR}/${BIN_NAME}"

echo "Installed ${BIN_NAME} to ${INSTALL_DIR}/${BIN_NAME}"
case ":$PATH:" in
  *":${INSTALL_DIR}:"*)
    echo "Run 'vimlearn' from any directory to start!"
    ;;
  *)
    echo "Add ${INSTALL_DIR} to your PATH to run 'vimlearn' from anywhere:"
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    ;;
esac
