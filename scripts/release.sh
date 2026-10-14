#!/usr/bin/env bash
# Cross-compile release binaries for every supported OS/arch into release/.
#
# Usage:
#   scripts/release.sh              # version from newest v* git tag, if any
#   scripts/release.sh v1.2.3       # explicit version
#
# Output: release/cloud-<os>-<arch>[.exe], a compressed archive per target,
# and SHA256SUMS. Commit + build date are always stamped into the binaries.
set -euo pipefail
cd "$(dirname "$0")/.."
ROOT="$(pwd)"

# Version: explicit arg > newest v* git tag > none (dev default 0.1.0).
VERSION="${1:-}"
if [ -z "$VERSION" ] && command -v git >/dev/null 2>&1 \
   && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  VERSION="$(git describe --tags --abbrev=0 --match 'v*' 2>/dev/null || true)"
fi

LDFLAGS="-s -w"
if [ -n "$VERSION" ]; then
  LDFLAGS="$LDFLAGS -X clouds/ui.Version=$VERSION"
fi
if COMMIT="$(git rev-parse --short HEAD 2>/dev/null)"; then
  # Commit date, not wall-clock: matches scripts/release.bat (%cs) so both
  # drivers stamp identical binaries for the same commit.
  LDFLAGS="$LDFLAGS -X clouds/ui.Commit=$COMMIT -X clouds/ui.Date=$(git log -1 --format=%cs)"
fi

OUT=release
rm -rf "$OUT"
mkdir -p "$OUT"

echo "generating desktop assets (icon set, .desktop entry)"
go run ./packaging icons packaging/icons

# Windows icon resources: clouds.ico plus a COFF .rsrc object per architecture.
# clouds_windows_<arch>.syso must sit in the main package directory; the Go
# linker links it into that architecture's windows build automatically.
echo "generating Windows icon resources (clouds.ico + clouds_windows_*.syso)"
go run ./packaging ico packaging/icons
go run ./packaging syso -arch amd64
go run ./packaging syso -arch arm64

APPVERSION="${VERSION:-0.1.0}"

# zip_windows <file> <zip> — the Windows distribution zip (bare exe at the
# archive root, same output as scripts/release.bat's Compress-Archive). zip
# is used where present; Git Bash, a common Windows host, ships no zip.exe,
# so PowerShell's Compress-Archive is the fallback — safe here because a
# Windows artifact needs no unix exec bits (unlike the darwin/linux
# tarballs, which tar keeps). There is deliberately no tar.gz fallback:
# install.ps1 links $base.zip by name, so a host with neither writer must
# fail loudly rather than ship an archive the installer cannot fetch.
zip_windows() {
    file=$1; zipout=$2
    if command -v zip >/dev/null 2>&1; then
        (cd "$(dirname "$file")" && zip -q "$(basename "$zipout")" "$(basename "$file")")
    elif command -v powershell.exe >/dev/null 2>&1; then
        # Paths via env vars (no shell quoting reaches PowerShell); cygpath
        # converts /c/... to C:\... (always present in Git Bash).
        ZIP_SRC="$(cygpath -w "$file" 2>/dev/null || printf '%s' "$file")" \
        ZIP_DST="$(cygpath -w "$zipout" 2>/dev/null || printf '%s' "$zipout")" \
        powershell.exe -NoProfile -Command 'Compress-Archive -Path $env:ZIP_SRC -DestinationPath $env:ZIP_DST -Force'
    else
        echo "error: cannot write $zipout — install zip (e.g. apt install zip) or run on Windows" >&2
        exit 1
    fi
}

# os arch pairs; GOARM=7 is applied to linux/arm (Raspberry Pi 2/3/4 32-bit).
TARGETS=(
  "darwin amd64"
  "darwin arm64"
  "linux amd64"
  "linux arm64"
  "linux arm"
  "windows amd64"
  "windows arm64"
)

for t in "${TARGETS[@]}"; do
  os=${t% *}
  arch=${t#* }
  base="cloud-$os-$arch"
  ext=""
  [ "$os" = "windows" ] && ext=".exe"
  printf 'building %s\n' "$base$ext"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" GOARM=7 \
    go build -trimpath -ldflags "$LDFLAGS" -o "$OUT/$base$ext" .

  # Compressed archive for distribution (Go binaries shrink ~65%).
  case "$os" in
    windows)
      # .zip is part of the release contract (install.ps1 links it) — no
      # tar.gz fallback; see zip_windows above.
      zip_windows "$ROOT/$OUT/$base$ext" "$ROOT/$OUT/$base.zip"
      ;;
    darwin)
      (cd "$OUT" && tar -czf "$base.tar.gz" "$base$ext")
      printf 'bundling %s-app.zip\n' "$base"
      go run ./packaging app -bin "$OUT/$base" -arch "$arch" \
        -version "$APPVERSION" -icons packaging/icons -out "$OUT/$base-app.zip"
      ;;
    *) # linux: ship the desktop entry + icon for shell integration
      # (absolute -C paths: tar's -C is cumulative, relative would nest)
      tar -czf "$OUT/$base.tar.gz" -C "$ROOT/$OUT" "$base$ext" \
        -C "$ROOT/packaging/icons" clouds.desktop clouds.svg
      ;;
  esac
done

echo "writing $OUT/SHA256SUMS"
if command -v sha256sum >/dev/null 2>&1; then
  (cd "$OUT" && sha256sum cloud-* > SHA256SUMS)
else
  (cd "$OUT" && shasum -a 256 cloud-* > SHA256SUMS)
fi

echo
echo "release folder ready:"
ls -lh "$OUT"
