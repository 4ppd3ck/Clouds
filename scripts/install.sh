#!/usr/bin/env bash
# Install clouds from GitHub Releases into ~/.local/bin (override with
# CLOUDS_BIN). Repo and version are overridable too:
#
#   curl -fsSL https://raw.githubusercontent.com/OWNER/REPO/main/scripts/install.sh | bash
#   CLOUDS_VERSION=v1.2.3 curl -fsSL ... | bash
set -euo pipefail

REPO="${CLOUDS_REPO:-GITHUB_REPO/clouds}"
VERSION="${CLOUDS_VERSION:-latest}"

os=$(uname -s)
case "$os" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *) echo "unsupported OS: $os — on Windows run scripts/install.ps1, or grab cloud-windows-amd64.zip from the releases page" >&2; exit 1 ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  armv7l|armv6l|arm) arch=arm ;;
  *) echo "unsupported architecture: $arch" >&2; exit 1 ;;
esac

bin_dir="${CLOUDS_BIN:-$HOME/.local/bin}"
base="cloud-$os-$arch"
if [ "$VERSION" = "latest" ]; then
  url="https://github.com/$REPO/releases/latest/download"
else
  url="https://github.com/$REPO/releases/download/$VERSION"
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "downloading $base.tar.gz from $REPO ($VERSION)"
curl -fL --progress-bar -o "$tmp/$base.tar.gz" "$url/$base.tar.gz"

# Checksums when published. Only "release ships no SHA256SUMS" (HTTP 404)
# skips verification; a failed request or a sums file without an entry for
# this artifact is fatal, never a silent pass.
code=$(curl -Ls -o "$tmp/SHA256SUMS" -w '%{http_code}' "$url/SHA256SUMS") || true
if [ "$code" = 200 ]; then
  want=$(grep " $base.tar.gz\$" "$tmp/SHA256SUMS" | awk '{print $1}') || true
  if [ -z "$want" ]; then
    echo "error: SHA256SUMS has no entry for $base.tar.gz — refusing to install" >&2
    exit 1
  fi
  if command -v sha256sum >/dev/null 2>&1; then
    got=$(sha256sum "$tmp/$base.tar.gz" | awk '{print $1}')
  else
    got=$(shasum -a 256 "$tmp/$base.tar.gz" | awk '{print $1}')
  fi
  if [ "$got" != "$want" ]; then
    echo "checksum mismatch: expected $want, got $got" >&2
    exit 1
  fi
  echo "checksum ok"
elif [ "$code" = 404 ]; then
  echo "note: no SHA256SUMS published for this release, skipping checksum"
else
  echo "error: downloading SHA256SUMS failed (HTTP $code)" >&2
  exit 1
fi

mkdir -p "$bin_dir"
tar -xzf "$tmp/$base.tar.gz" -C "$tmp"
install -m 0755 "$tmp/$base" "$bin_dir/clouds"

# Desktop entry (Linux only, when the release ships one; opt out with
# CLOUDS_NO_DESKTOP=1).
if [ "$os" = linux ] && [ -f "$tmp/clouds.desktop" ] && [ -z "${CLOUDS_NO_DESKTOP:-}" ]; then
  xdg="${XDG_DATA_HOME:-$HOME/.local/share}"
  install -d -m 0755 "$xdg/applications" "$xdg/icons/hicolor/scalable/apps"
  install -m 0644 "$tmp/clouds.desktop" "$xdg/applications/clouds.desktop"
  if [ -f "$tmp/clouds.svg" ]; then
    install -m 0644 "$tmp/clouds.svg" "$xdg/icons/hicolor/scalable/apps/clouds.svg"
  fi
  echo "desktop entry installed (applications menu → clouds)"
fi

case ":$PATH:" in
  *":$bin_dir:"*) ;;
  *) echo "note: $bin_dir is not on your PATH — add it with: export PATH=\"$bin_dir:\$PATH\"" ;;
esac

"$bin_dir/clouds" --version
echo "installed → $bin_dir/clouds"
