#!/usr/bin/env sh
# Build and package every supported Clouds release target.
# Usage: ./release.sh [version] [os|os/arch ...]
# Examples: ./release.sh v1.2.3 darwin/arm64
#           ./release.sh darwin/arm64
#           ./release.sh linux
# Writes only cloud-<os>-<arch>.zip packages and SHA256SUMS to release/.
set -eu
cd "$(dirname "$0")"
ROOT=$(pwd)
VERSION=
if [ "$#" -gt 0 ]; then
    case "$1" in
        darwin|linux|windows|darwin/*|linux/*|windows/*) ;;
        *) VERSION=$1; shift ;;
    esac
fi
if [ -z "$VERSION" ] && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    VERSION=$(git describe --tags --abbrev=0 --match 'v*' 2>/dev/null || true)
fi
# Git tags and user input conventionally include v; ui.Version is stored
# without it because the UI and --version output add the display prefix.
VERSION=${VERSION#v}
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || true)
BUILD_DATE=
if [ -n "$COMMIT" ]; then BUILD_DATE=$(git log -1 --format=%cs 2>/dev/null || true); fi
LDFLAGS="-s -w"
[ -z "$VERSION" ] || LDFLAGS="$LDFLAGS -X clouds/ui.Version=$VERSION"
if [ -n "$COMMIT" ]; then LDFLAGS="$LDFLAGS -X clouds/ui.Commit=$COMMIT -X clouds/ui.Date=$BUILD_DATE"; fi
APPVERSION=${VERSION:-0.1.0}
OUT="$ROOT/release"
BINDIR=
STAGE=
cleanup() {
    [ -z "$BINDIR" ] || rm -rf "$BINDIR"
    [ -z "$STAGE" ] || rm -rf "$STAGE"
}
trap cleanup EXIT HUP INT TERM
rm -rf "$OUT"
mkdir -p "$OUT"

ALL_TARGETS="darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 linux/arm windows/amd64 windows/arm64"
TARGETS=
for arg in "$@"; do
    case "$arg" in
        darwin|linux|windows)
            for target in $ALL_TARGETS; do
                case "$target" in
                    "$arg"/*) TARGETS="$TARGETS $target" ;;
                esac
            done
            ;;
        darwin/*|linux/*|windows/*)
            case " $ALL_TARGETS " in
                *" $arg "*) TARGETS="$TARGETS $arg" ;;
                *) echo "error: unknown target '$arg'; supported: $ALL_TARGETS" >&2; exit 1 ;;
            esac
            ;;
        *)
            echo "error: unknown argument '$arg'; use a version, OS, or os/arch target" >&2
            exit 1
            ;;
    esac
done
if [ -z "$TARGETS" ]; then TARGETS=$ALL_TARGETS; fi
# Remove duplicate targets while preserving the requested order.
DEDUPED=
for target in $TARGETS; do
    case " $DEDUPED " in
        *" $target "*) ;;
        *) DEDUPED="$DEDUPED $target" ;;
    esac
done
TARGETS=$DEDUPED

echo "generating icons and Windows resources"
go run ./packaging icons packaging/icons
go run ./packaging ico packaging/icons
go run ./packaging syso -arch amd64
go run ./packaging syso -arch arm64

zip_tree() {
    dir=$1
    archive=$2
    if command -v zip >/dev/null 2>&1; then
        (cd "$dir" && zip -q -r "$archive" .)
    elif command -v powershell.exe >/dev/null 2>&1; then
        stage_win=$(cygpath -w "$dir" 2>/dev/null || printf '%s' "$dir")
        archive_win=$(cygpath -w "$archive" 2>/dev/null || printf '%s' "$archive")
        ZIP_STAGE="$stage_win" ZIP_OUT="$archive_win" powershell.exe -NoProfile -Command 'Compress-Archive -Path (Join-Path $env:ZIP_STAGE "*") -DestinationPath $env:ZIP_OUT -Force'
    else
        echo "error: zip is required to create $archive" >&2
        exit 1
    fi
}

for target in $TARGETS; do
    os=${target%/*}
    arch=${target#*/}
    base="cloud-$os-$arch"
    ext=
    [ "$os" != windows ] || ext=.exe
    echo "building $base$ext"
    BINDIR=$(mktemp -d)
    bin="$BINDIR/$base$ext"
    goarm=
    [ "$arch" != arm ] || goarm=7
    if [ -n "$goarm" ]; then
        CGO_ENABLED=0 GOOS=$os GOARCH=$arch GOARM=$goarm go build -trimpath -ldflags "$LDFLAGS" -o "$bin" .
    else
        CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath -ldflags "$LDFLAGS" -o "$bin" .
    fi
    case "$os" in
        darwin)
            if command -v codesign >/dev/null 2>&1; then codesign --force --sign - "$bin" >/dev/null 2>&1 || true; fi
            go run ./packaging app -bin "$bin" -arch "$arch" -version "$APPVERSION" -icons packaging/icons -out "$OUT/$base.zip"
            ;;
        linux)
            STAGE=$(mktemp -d)
            cp "$bin" "$STAGE/clouds"
            chmod 0755 "$STAGE/clouds"
            cp packaging/icons/clouds.desktop packaging/icons/clouds.svg "$STAGE/"
            cp scripts/install-local.sh "$STAGE/install.sh"
            chmod 0755 "$STAGE/install.sh"
            chmod 0644 "$STAGE/clouds.desktop" "$STAGE/clouds.svg"
            zip_tree "$STAGE" "$OUT/$base.zip"
            rm -rf "$STAGE"; STAGE=
            ;;
        windows)
            STAGE=$(mktemp -d)
            cp "$bin" "$STAGE/$base.exe"
            cp packaging/icons/clouds.ico "$STAGE/clouds.ico"
            cp scripts/install-local.ps1 "$STAGE/install.ps1"
            zip_tree "$STAGE" "$OUT/$base.zip"
            rm -rf "$STAGE"; STAGE=
            ;;
    esac
    rm -rf "$BINDIR"; BINDIR=
done

echo "writing release/SHA256SUMS"
if command -v sha256sum >/dev/null 2>&1; then
    (cd "$OUT" && sha256sum cloud-*.zip > SHA256SUMS)
else
    (cd "$OUT" && shasum -a 256 cloud-*.zip > SHA256SUMS)
fi
ls -lh "$OUT"
