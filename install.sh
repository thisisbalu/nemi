#!/bin/sh
# Install the latest Nemi release binary.
#   curl -fsSL https://raw.githubusercontent.com/thisisbalu/nemi/main/install.sh | sh
# Override the install location with INSTALL_DIR=/path sh install.sh
set -e

REPO="thisisbalu/nemi"
BIN="nemi"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
  x86_64 | amd64) arch="amd64" ;;
  aarch64 | arm64) arch="arm64" ;;
  *) echo "nemi: unsupported architecture: $arch" >&2; exit 1 ;;
esac
case "$os" in
  linux | darwin) ;;
  *) echo "nemi: unsupported OS: $os (use the Windows zip from the Releases page)" >&2; exit 1 ;;
esac

tag=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
  | grep '"tag_name"' | head -1 | cut -d '"' -f4)
if [ -z "$tag" ]; then
  echo "nemi: could not determine the latest release" >&2
  exit 1
fi
version=${tag#v}

url="https://github.com/$REPO/releases/download/$tag/${BIN}_${version}_${os}_${arch}.tar.gz"
echo "Downloading nemi $tag ($os/$arch)…"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
curl -fsSL "$url" | tar -xz -C "$tmp"

if [ -w "$INSTALL_DIR" ]; then
  mv "$tmp/$BIN" "$INSTALL_DIR/$BIN"
else
  echo "Installing to $INSTALL_DIR (requires sudo)…"
  sudo mv "$tmp/$BIN" "$INSTALL_DIR/$BIN"
fi
chmod +x "$INSTALL_DIR/$BIN"

echo "Installed: $("$INSTALL_DIR/$BIN" version)"
