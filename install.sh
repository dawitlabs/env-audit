#!/bin/sh
set -e

REPO="dawitlabs/env-audit"
BIN="env-audit"

OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
  Linux)  os="linux" ;;
  Darwin) os="darwin" ;;
  *)      echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

case "$ARCH" in
  x86_64)          arch="amd64" ;;
  aarch64 | arm64) arch="arm64" ;;
  *)               echo "Unsupported arch: $ARCH" >&2; exit 1 ;;
esac

TAG=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
  | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')

if [ -z "$TAG" ]; then
  echo "Could not determine latest release" >&2; exit 1
fi

FILE="${BIN}-${os}-${arch}"
URL="https://github.com/${REPO}/releases/download/${TAG}/${FILE}"

echo "Installing $BIN $TAG ($os/$arch)..."
TMP=$(mktemp)
curl -fsSL "$URL" -o "$TMP"
chmod +x "$TMP"

if [ -w /usr/local/bin ]; then
  mv "$TMP" "/usr/local/bin/$BIN"
  echo "Installed to /usr/local/bin/$BIN"
else
  mkdir -p "$HOME/.local/bin"
  mv "$TMP" "$HOME/.local/bin/$BIN"
  echo "Installed to $HOME/.local/bin/$BIN"
  echo "Make sure $HOME/.local/bin is in your PATH."
fi

echo "Done. Run: $BIN"
