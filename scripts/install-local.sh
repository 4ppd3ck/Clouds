#!/usr/bin/env bash
# Install the CLI from the release ZIP this script was extracted from.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
os=$(uname -s)
case "$os" in
  Darwin) binary="$ROOT/clouds.app/Contents/Resources/clouds" ;;
  Linux) binary="$ROOT/clouds" ;;
  *) echo "unsupported OS: $os" >&2; exit 1 ;;
esac
if [ ! -f "$binary" ]; then
  echo "clouds binary not found beside this installer" >&2
  exit 1
fi

bin_dir="${CLOUDS_BIN:-$HOME/.local/bin}"
mkdir -p "$bin_dir"
install -m 0755 "$binary" "$bin_dir/clouds"

if [ "$os" = Linux ] && [ -f "$ROOT/clouds.desktop" ]; then
  xdg="${XDG_DATA_HOME:-$HOME/.local/share}"
  install -d -m 0755 "$xdg/applications" "$xdg/icons/hicolor/scalable/apps"
  install -m 0644 "$ROOT/clouds.desktop" "$xdg/applications/clouds.desktop"
  [ ! -f "$ROOT/clouds.svg" ] || install -m 0644 "$ROOT/clouds.svg" "$xdg/icons/hicolor/scalable/apps/clouds.svg"
fi

case ":$PATH:" in
  *":$bin_dir:"*) ;;
  *)
    path_files=()
    if [ "$bin_dir" = "$HOME/.local/bin" ]; then
      case "${SHELL##*/}" in
        zsh) path_files=("$HOME/.zshrc") ;;
        bash) path_files=("$HOME/.bashrc" "$HOME/.bash_profile" "$HOME/.profile") ;;
      esac
    fi
    if [ "${#path_files[@]}" -gt 0 ]; then
      for path_file in "${path_files[@]}"; do
        if ! grep -Fq '# clouds: add install directory to PATH' "$path_file" 2>/dev/null; then
          mkdir -p "$(dirname "$path_file")"
          cat >> "$path_file" <<'SHELL_PATH'

# clouds: add install directory to PATH
case ":$PATH:" in
  *":$HOME/.local/bin:"*) ;;
  *) export PATH="$HOME/.local/bin:$PATH" ;;
esac
SHELL_PATH
        fi
      done
      echo "added $bin_dir to shell startup files; open a new terminal or source your shell file"
    else
      echo "add this to your shell configuration: export PATH=\"$bin_dir:\$PATH\""
    fi
    ;;
esac

"$bin_dir/clouds" --version
echo "installed → $bin_dir/clouds"
