#!/bin/sh
# Install the latest ppm release on Linux or macOS:
#   curl -fsSL https://raw.githubusercontent.com/rahulcodepython/PPM/main/install.sh | sh
#
# Installs to ~/.local/bin (/usr/local/bin as root) and adds it to PATH.
# Options (environment variables):
#   PPM_VERSION=v2.0.0          install a specific version
#   PPM_INSTALL_DIR=/opt/bin    install somewhere else
set -eu

REPO="rahulcodepython/PPM"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
  linux|darwin) ;;
  *) echo "unsupported OS: $os (on Windows use install.ps1)" >&2; exit 1 ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) echo "unsupported architecture: $arch" >&2; exit 1 ;;
esac

if [ -n "${PPM_VERSION:-}" ]; then
  base="https://github.com/$REPO/releases/download/$PPM_VERSION"
else
  base="https://github.com/$REPO/releases/latest/download"
fi
archive="ppm_${os}_${arch}.tar.gz"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading $archive ..."
curl -fsSL "$base/$archive" -o "$tmp/$archive"
curl -fsSL "$base/checksums.txt" -o "$tmp/checksums.txt"

expected=$(grep " $archive\$" "$tmp/checksums.txt" | cut -d' ' -f1)
if command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "$tmp/$archive" | cut -d' ' -f1)
else
  actual=$(shasum -a 256 "$tmp/$archive" | cut -d' ' -f1)
fi
if [ -z "$expected" ] || [ "$expected" != "$actual" ]; then
  echo "checksum mismatch for $archive" >&2
  exit 1
fi

tar -xzf "$tmp/$archive" -C "$tmp" ppm
chmod +x "$tmp/ppm"

# ppm copies itself to the install folder and adds it to PATH.
if [ -n "${PPM_INSTALL_DIR:-}" ]; then
  "$tmp/ppm" self install --dir "$PPM_INSTALL_DIR"
else
  "$tmp/ppm" self install
fi
